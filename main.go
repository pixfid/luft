package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/config"
	"github.com/pixfid/luft/core/parsers"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/usbids"
	"github.com/umputun/go-flags"
)

const (
	ver = "v0.3"
	url = "https://github.com/pixfid/luft"
)

var opts struct {
	ConfigFile       string `long:"config" env:"LUFT_CONFIG" description:"path to config file (YAML)"`
	UpdateUSBIDs     bool   `long:"update-usbids" description:"download and update USB IDs database"`
	ClearCache       bool   `long:"clear-cache" description:"clear USB IDs cache and exit"`
	Streaming        bool   `long:"streaming" env:"STREAMING" description:"use streaming parser for memory-efficient processing of large logs"`
	Workers          int    `short:"w" long:"workers" env:"WORKERS" description:"number of worker threads for parallel parsing (default: CPU cores)" default:"0"`
	MassStorage      bool   `short:"m" long:"masstorage" env:"MASSTORAGE" description:"show only mass storage devices"`
	Untrusted        bool   `short:"u" long:"untrusted" env:"UNTRUSTED" description:"show only untrusted devices"`
	Number           int    `short:"n" long:"number" env:"NUMBER" description:"number of events to show"`
	Sort             string `short:"s" long:"sort" env:"SORT" choice:"asc" choice:"desc" description:"sort events" default:"asc"`
	Export           bool   `short:"e" long:"export" env:"EXPORT" description:"export events"`
	CheckByWhiteList bool   `short:"c" long:"check" env:"CHECK" description:"check devices for whitelist"`

	External struct {
		Whitelist string `short:"W" env:"WHITELIST" long:"whitelist" description:"external whitelist path"`
		UsbIds    string `short:"U" env:"USBIDS" long:"usbids" description:"usbids path" default:"/var/core/usbutils/usb.ids"`
	}

	Events struct {
		Source     string `short:"S" long:"source" choice:"local" choice:"remote" choice:"database" description:"events target"`
		RemoteHost string `long:"remote-host" env:"REMOTE_HOST" description:"remote host name from config file"`
		Export     struct {
			Format   string `short:"F" long:"format" choice:"json" choice:"xml" choice:"pdf" env:"FORMAT" description:"events export format" default:"pdf"`
			FileName string `short:"N" long:"filename" env:"FILENAME" description:"events export file name" default:"events_data"`
		} `group:"export" namespace:"export" env-namespace:"EXPORT"`
		Path   string `long:"path" description:"log directory" default:"/var/log/"`
		Remote struct {
			IP          string `short:"I" long:"ip" env:"IP" description:"ip address"`
			Port        string `long:"port" env:"port" description:"ssh port" default:"22"`
			Login       string `short:"L" long:"login" env:"LOGIN" description:"login"`
			Password    string `short:"P" long:"password" env:"PASSWORD" description:"password (deprecated, use SSH key instead)"`
			SSHKey      string `short:"K" long:"ssh-key" env:"SSH_KEY" description:"path to SSH private key (recommended)"`
			Timeout     int    `short:"T" long:"timeout" env:"TIMEOUT" description:"SSH connection timeout in seconds" default:"30"`
			InsecureSSH bool   `long:"insecure-ssh" env:"INSECURE_SSH" description:"skip SSH host key verification (NOT RECOMMENDED)"`
		} `group:"remote" namespace:"remote" env-namespace:"REMOTE"`
	} `group:"events" namespace:"events" env-namespace:"EVENTS"`
}

func PrintBanner() {
	_, _ = cfmt.Println(cfmt.Sprintf(`
{{┬  ┬ ┬┌─┐┌┬┐}}::bgLightRed
{{│  │ │├┤  │ }}::bgLightRed {{Linux Usb Forensic Tool %s}}::lightYellow
{{┴─┘└─┘└   ┴ }}::bgLightRed {{%s}}::lightBlue`, ver, url))
}

// mergeConfigWithFlags merges config file values with CLI flags
// CLI flags take precedence over config file values
func mergeConfigWithFlags(cfg *config.Config) {
	// Apply config values only if CLI flags are not set

	// Whitelist
	if opts.External.Whitelist == "" && cfg.Whitelist != "" {
		opts.External.Whitelist = cfg.Whitelist
	}

	// USB IDs
	if opts.External.UsbIds == "/var/core/usbutils/usb.ids" && cfg.UsbIds != "" {
		opts.External.UsbIds = cfg.UsbIds
	}

	// Log path
	if opts.Events.Path == "/var/log/" && cfg.LogPath != "" {
		opts.Events.Path = cfg.LogPath
	}

	// Export format
	if opts.Events.Export.Format == "pdf" && cfg.Export.Format != "" {
		opts.Events.Export.Format = cfg.Export.Format
	}

	// Mass storage filter
	if !opts.MassStorage && cfg.MassStorage {
		opts.MassStorage = cfg.MassStorage
	}

	// Untrusted filter
	if !opts.Untrusted && cfg.Untrusted {
		opts.Untrusted = cfg.Untrusted
	}

	// Check whitelist
	if !opts.CheckByWhiteList && cfg.CheckWl {
		opts.CheckByWhiteList = cfg.CheckWl
	}

	// Remote host from config
	if opts.Events.Source == "remote" && opts.Events.RemoteHost != "" {
		host, err := cfg.GetRemoteHost(opts.Events.RemoteHost)
		if err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] %s}}::red", time.Now().Format(time.Stamp), err.Error()))
			os.Exit(1)
		}

		// Apply remote host settings if CLI flags are not set
		if opts.Events.Remote.IP == "" {
			opts.Events.Remote.IP = host.IP
		}
		if opts.Events.Remote.Port == "22" {
			opts.Events.Remote.Port = host.Port
		}
		if opts.Events.Remote.Login == "" {
			opts.Events.Remote.Login = host.User
		}
		if opts.Events.Remote.SSHKey == "" {
			opts.Events.Remote.SSHKey = host.SSHKey
		}
		if opts.Events.Remote.Password == "" {
			opts.Events.Remote.Password = host.Password
		}
		if opts.Events.Remote.Timeout == 30 {
			opts.Events.Remote.Timeout = host.Timeout
		}
		if !opts.Events.Remote.InsecureSSH {
			opts.Events.Remote.InsecureSSH = host.InsecureSSH
		}

		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using remote host from config: %s (%s)}}::green", time.Now().Format(time.Stamp), host.Name, host.IP))
	}
}

// handleUSBIDsUpdate handles the --update-usbids flag
func handleUSBIDsUpdate() {
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] USB IDs Update Mode}}::cyan|bold", time.Now().Format(time.Stamp)))

	// Determine target path for USB IDs file
	targetPath := opts.External.UsbIds

	// If default path is not writable, try alternatives
	if !isWritable(targetPath) {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: %s is not writable, trying alternatives...}}::yellow", time.Now().Format(time.Stamp), targetPath))

		// Try user's home directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			targetPath = homeDir + "/.local/share/luft/usb.ids"
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using alternative path: %s}}::cyan", time.Now().Format(time.Stamp), targetPath))
		} else {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: Cannot determine writable location}}::red", time.Now().Format(time.Stamp)))
			os.Exit(1)
		}
	}

	// Update USB IDs
	if err := usbids.UpdateUSBIDs(targetPath); err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: Failed to update USB IDs: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		os.Exit(1)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Update completed successfully!}}::green|bold", time.Now().Format(time.Stamp)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] To use this database, run luft with: --usbids=%s}}::cyan", time.Now().Format(time.Stamp), targetPath))
}

// handleClearCache handles the --clear-cache flag
func handleClearCache() {
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cache Clear Mode}}::cyan|bold", time.Now().Format(time.Stamp)))

	// Get USB IDs path
	usbIdsPath := opts.External.UsbIds

	// Try to clear cache
	if err := usbids.ClearCache(usbIdsPath); err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: Failed to clear cache: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		os.Exit(1)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Cache cleared successfully!}}::green|bold", time.Now().Format(time.Stamp)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Next load will parse from source and rebuild cache}}::cyan", time.Now().Format(time.Stamp)))
}

// isWritable checks if a path is writable
func isWritable(path string) bool {
	// Check if file exists
	info, err := os.Stat(path)
	if err == nil {
		// File exists, check if writable
		testFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return false
		}
		testFile.Close()
		return true
	}

	// File doesn't exist, check if directory is writable
	dir := path
	if info == nil || !info.IsDir() {
		dir = filepath.Dir(path)
	}

	// Try to create a temp file in the directory
	testFile, err := os.CreateTemp(dir, ".luft-write-test-*")
	if err != nil {
		return false
	}
	testFile.Close()
	os.Remove(testFile.Name())
	return true
}

// setupSignalHandler creates a context that will be cancelled on SIGINT or SIGTERM
func setupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		_, _ = cfmt.Println(cfmt.Sprintf("\n{{[%v] Received signal: %v - initiating graceful shutdown...}}::yellow|bold", time.Now().Format(time.Stamp), sig))
		cancel()

		// Force exit if second signal received
		sig = <-sigChan
		_, _ = cfmt.Println(cfmt.Sprintf("\n{{[%v] Received second signal: %v - forcing immediate exit}}::red|bold", time.Now().Format(time.Stamp), sig))
		os.Exit(1)
	}()

	return ctx, cancel
}

func main() {

	PrintBanner()

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Starting at: %v", time.Now().Format(time.Stamp)))

	// Setup signal handler for graceful shutdown
	ctx, cancel := setupSignalHandler()
	defer cancel()

	p := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash|flags.HelpFlag)
	p.SubcommandsOptional = true

	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[ERROR] cli error: %v}}::red", err))
		}
		os.Exit(1)
	}

	// Handle USB IDs update flag
	if opts.UpdateUSBIDs {
		handleUSBIDsUpdate()
		return
	}

	// Handle cache clear flag
	if opts.ClearCache {
		handleClearCache()
		return
	}

	// Validate required flags for normal operation
	if opts.Events.Source == "" {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[ERROR] the required flag -S/--events.source was not specified}}::red"))
		_, _ = cfmt.Println(cfmt.Sprintf("{{Run 'luft --help' for usage information}}::yellow"))
		os.Exit(1)
	}

	// Load configuration file
	cfg, err := config.Load(opts.ConfigFile)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Error loading config: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		os.Exit(1)
	}

	// Merge config with CLI flags (CLI flags take precedence)
	mergeConfigWithFlags(cfg)

	var parseParams = data.ParseParams{
		Ctx:                ctx,
		LogPath:            opts.Events.Path,
		WlPath:             opts.External.Whitelist,
		OnlyMass:           opts.MassStorage,
		CheckWl:            opts.CheckByWhiteList,
		Number:             opts.Number,
		Export:             opts.Export,
		Format:             opts.Events.Export.Format,
		FileName:           opts.Events.Export.FileName,
		ExternalUsbIdsPath: opts.External.UsbIds,
		SortBy:             opts.Sort,
		Untrusted:          opts.Untrusted,
		Login:              opts.Events.Remote.Login,
		Password:           opts.Events.Remote.Password,
		Port:               opts.Events.Remote.Port,
		IP:                 opts.Events.Remote.IP,
		SSHKeyPath:         opts.Events.Remote.SSHKey,
		SSHTimeout:         opts.Events.Remote.Timeout,
		InsecureSSH:        opts.Events.Remote.InsecureSSH,
		Workers:            opts.Workers,
		Streaming:          opts.Streaming,
	}

	// Load whitelist if needed
	if opts.CheckByWhiteList {
		var whitelistLoaded bool
		if opts.External.Whitelist != "" {
			if _, err := os.Stat(opts.External.Whitelist); !os.IsNotExist(err) {
				if err := utils.LoadWhiteList(opts.External.Whitelist); err != nil {
					_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Error loading external whitelist %s: %s}}::red", time.Now().Format(time.Stamp), opts.External.Whitelist, err.Error()))
				} else {
					_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded whitelist from %s}}::green", time.Now().Format(time.Stamp), opts.External.Whitelist))
					whitelistLoaded = true
				}
			} else {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: whitelist file not found: %s}}::yellow", time.Now().Format(time.Stamp), opts.External.Whitelist))
			}
		}

		// Try default location if custom whitelist not loaded
		if !whitelistLoaded {
			defaultWhitelist := "/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules"
			if _, err := os.Stat(defaultWhitelist); !os.IsNotExist(err) {
				if err := utils.LoadWhiteList(defaultWhitelist); err != nil {
					_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Error loading system udev whitelist: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
				} else {
					_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded default whitelist from %s}}::green", time.Now().Format(time.Stamp), defaultWhitelist))
					whitelistLoaded = true
				}
			}
		}

		if !whitelistLoaded {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: no whitelist loaded, but whitelist checking is enabled}}::yellow", time.Now().Format(time.Stamp)))
		}
	}

	if _, err := os.Stat(opts.External.UsbIds); !os.IsNotExist(err) {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using external usb.ids}}::green", time.Now().Format(time.Stamp)))
		if err := usbids.LoadFromFile(opts.External.UsbIds); err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] {{Try load another one usb.ids}}::green", time.Now().Format(time.Stamp)))
			if err := usbids.LoadFromFiles(); err != nil {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Error loading any usb.ids}}::red", time.Now().Format(time.Stamp)))
			}
		}
	} else {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] %s}}::red", time.Now().Format(time.Stamp), err.Error()))
	}

	if opts.Untrusted {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Will be print only untrusted devices}}::green", time.Now().Format(time.Stamp)))
	}

	// Validate and show warnings for remote connections
	if opts.Events.Source == "remote" {
		// Validate required parameters
		if opts.Events.Remote.IP == "" {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: IP address is required for remote connection (use -I flag)}}::red", time.Now().Format(time.Stamp)))
			os.Exit(1)
		}
		if opts.Events.Remote.Login == "" {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: Login is required for remote connection (use -L flag)}}::red", time.Now().Format(time.Stamp)))
			os.Exit(1)
		}
		if opts.Events.Remote.Password == "" && opts.Events.Remote.SSHKey == "" {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: Either password (-P) or SSH key (-K) must be provided for remote connection}}::red", time.Now().Format(time.Stamp)))
			os.Exit(1)
		}

		// Security warnings
		if opts.Events.Remote.InsecureSSH {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ⚠️  WARNING: SSH host key verification is DISABLED! Connection is vulnerable to man-in-the-middle attacks.}}::bgRed|white|bold", time.Now().Format(time.Stamp)))
		}
		if opts.Events.Remote.Password != "" && opts.Events.Remote.SSHKey == "" {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ⚠️  WARNING: Using password authentication. SSH key authentication is more secure.}}::yellow|bold", time.Now().Format(time.Stamp)))
		}
	}

	// Validate number parameter
	if opts.Number < 0 {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: number of events cannot be negative}}::red", time.Now().Format(time.Stamp)))
		os.Exit(1)
	}

	// Execute based on source
	switch opts.Events.Source {
	case "local":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered local events}}::green", time.Now().Format(time.Stamp)))
		err = parsers.LocalEvents(parseParams)
	case "remote":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered remote events}}::green", time.Now().Format(time.Stamp)))
		err = parsers.RemoteEvents(parseParams)
	default:
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: unknown source type: %s}}::red", time.Now().Format(time.Stamp), opts.Events.Source))
		os.Exit(1)
	}

	if err != nil {
		// Check if error is due to context cancellation
		if errors.Is(err, context.Canceled) {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Operation cancelled by user}}::yellow", time.Now().Format(time.Stamp)))
			os.Exit(130) // Standard exit code for SIGINT
		}
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		os.Exit(1)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Shut down at: %v", time.Now().Format(time.Stamp)))

}
