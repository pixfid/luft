package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/core/parsers"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/usbids"
	"github.com/spf13/cobra"
)

var (
	// Source flags
	sourceType string
	logPath    string
	remoteHost string

	// Filter flags
	massStorage bool
	untrusted   bool
	checkWl     bool
	number      int
	sortBy      string
	whitelist   string
	usbidsPath  string

	// Export flags
	export       bool
	exportFormat string
	exportFile   string

	// Performance flags
	workers   int
	streaming bool

	// Remote flags
	remoteIP      string
	remotePort    string
	remoteLogin   string
	remotePass    string
	remoteSSHKey  string
	remoteTimeout int
	insecureSSH   bool
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Collect and analyze USB device events",
	Long: `Collect USB device connection events from local or remote systems.

Supports multiple sources:
  - local:    Analyze logs from the local system
  - remote:   Analyze logs from a remote system via SSH
  - database: Analyze logs from a database (future feature)

Examples:
  # Analyze local logs
  luft events --source local

  # Analyze remote host from config
  luft events --source remote --remote-host prod-server

  # Analyze with filters
  luft events --source local --mass-storage --untrusted --check-whitelist

  # Export to PDF
  luft events --source local --export --format pdf --output report`,
	RunE: runEvents,
}

func init() {
	rootCmd.AddCommand(eventsCmd)

	// Source flags
	eventsCmd.Flags().StringVarP(&sourceType, "source", "S", "", "event source (local, remote, database) [required]")
	eventsCmd.Flags().StringVar(&logPath, "path", "/var/log/", "log directory path")
	eventsCmd.Flags().StringVar(&remoteHost, "remote-host", "", "remote host name from config file")
	eventsCmd.MarkFlagRequired("source")

	// Filter flags
	eventsCmd.Flags().BoolVarP(&massStorage, "mass-storage", "m", false, "show only mass storage devices")
	eventsCmd.Flags().BoolVarP(&untrusted, "untrusted", "u", false, "show only untrusted devices")
	eventsCmd.Flags().BoolVarP(&checkWl, "check-whitelist", "c", false, "check devices against whitelist")
	eventsCmd.Flags().IntVarP(&number, "number", "n", 0, "number of events to show (0 = all)")
	eventsCmd.Flags().StringVarP(&sortBy, "sort", "s", "asc", "sort events (asc, desc)")
	eventsCmd.Flags().StringVarP(&whitelist, "whitelist", "W", "", "whitelist file path")
	eventsCmd.Flags().StringVarP(&usbidsPath, "usbids", "U", "/var/lib/usbutils/usb.ids", "USB IDs database path")

	// Export flags
	eventsCmd.Flags().BoolVarP(&export, "export", "e", false, "export events")
	eventsCmd.Flags().StringVarP(&exportFormat, "format", "F", "pdf", "export format (json, xml, pdf)")
	eventsCmd.Flags().StringVarP(&exportFile, "output", "o", "events_data", "export filename (without extension)")

	// Performance flags
	eventsCmd.Flags().IntVarP(&workers, "workers", "w", 0, "number of worker threads (0 = auto)")
	eventsCmd.Flags().BoolVar(&streaming, "streaming", false, "use streaming parser for large logs")

	// Remote flags
	eventsCmd.Flags().StringVarP(&remoteIP, "remote-ip", "I", "", "remote host IP address")
	eventsCmd.Flags().StringVar(&remotePort, "remote-port", "22", "remote SSH port")
	eventsCmd.Flags().StringVarP(&remoteLogin, "remote-login", "L", "", "remote login username")
	eventsCmd.Flags().StringVarP(&remotePass, "remote-password", "P", "", "remote password (deprecated, use SSH key)")
	eventsCmd.Flags().StringVarP(&remoteSSHKey, "remote-key", "K", "", "path to SSH private key (recommended)")
	eventsCmd.Flags().IntVarP(&remoteTimeout, "remote-timeout", "T", 30, "SSH connection timeout in seconds")
	eventsCmd.Flags().BoolVar(&insecureSSH, "insecure-ssh", false, "skip SSH host key verification (NOT RECOMMENDED)")
}

func runEvents(cmd *cobra.Command, args []string) error {
	// Merge config with flags
	mergeConfigWithFlags()

	// Build parse parameters
	params := data.ParseParams{
		Ctx:                rootCtx,
		LogPath:            logPath,
		WlPath:             whitelist,
		OnlyMass:           massStorage,
		CheckWl:            checkWl,
		Number:             number,
		Export:             export,
		Format:             exportFormat,
		FileName:           exportFile,
		ExternalUsbIdsPath: usbidsPath,
		SortBy:             sortBy,
		Untrusted:          untrusted,
		Login:              remoteLogin,
		Password:           remotePass,
		Port:               remotePort,
		IP:                 remoteIP,
		SSHKeyPath:         remoteSSHKey,
		SSHTimeout:         remoteTimeout,
		InsecureSSH:        insecureSSH,
		Workers:            workers,
		Streaming:          streaming,
	}

	// Load whitelist if needed
	if checkWl {
		if err := loadWhitelist(); err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: %s}}::yellow", time.Now().Format(time.Stamp), err.Error()))
		}
	}

	// Load USB IDs database
	if err := loadUSBIDs(); err != nil {
		return err
	}

	if untrusted {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filtering: only untrusted devices}}::green", time.Now().Format(time.Stamp)))
	}

	// Validate and execute based on source
	switch sourceType {
	case "local":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Collecting local events...}}::green", time.Now().Format(time.Stamp)))
		err := parsers.LocalEvents(params)
		if err != nil {
			if errors.Is(err, rootCtx.Err()) {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Operation cancelled by user}}::yellow", time.Now().Format(time.Stamp)))
				os.Exit(130)
			}
			return err
		}

	case "remote":
		if err := validateRemoteFlags(); err != nil {
			return err
		}
		showRemoteWarnings()

		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Collecting remote events...}}::green", time.Now().Format(time.Stamp)))
		err := parsers.RemoteEvents(params)
		if err != nil {
			if errors.Is(err, rootCtx.Err()) {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Operation cancelled by user}}::yellow", time.Now().Format(time.Stamp)))
				os.Exit(130)
			}
			return err
		}

	case "database":
		return fmt.Errorf("database source not yet implemented")

	default:
		return fmt.Errorf("unknown source type: %s (use: local, remote, database)", sourceType)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Completed at: %v", time.Now().Format(time.Stamp)))
	return nil
}

func mergeConfigWithFlags() {
	if configLoaded == nil {
		return
	}

	// Apply config values only if flags are at default values
	if whitelist == "" && configLoaded.Whitelist != "" {
		whitelist = configLoaded.Whitelist
	}
	if usbidsPath == "/var/lib/usbutils/usb.ids" && configLoaded.UsbIds != "" {
		usbidsPath = configLoaded.UsbIds
	}
	if logPath == "/var/log/" && configLoaded.LogPath != "" {
		logPath = configLoaded.LogPath
	}
	if exportFormat == "pdf" && configLoaded.Export.Format != "" {
		exportFormat = configLoaded.Export.Format
	}
	if !massStorage && configLoaded.MassStorage {
		massStorage = configLoaded.MassStorage
	}
	if !untrusted && configLoaded.Untrusted {
		untrusted = configLoaded.Untrusted
	}
	if !checkWl && configLoaded.CheckWl {
		checkWl = configLoaded.CheckWl
	}

	// Handle remote host from config
	if remoteHost != "" {
		host, err := configLoaded.GetRemoteHost(remoteHost)
		if err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: %s}}::yellow", time.Now().Format(time.Stamp), err.Error()))
			return
		}

		// Apply remote host settings
		if remoteIP == "" {
			remoteIP = host.IP
		}
		if remotePort == "22" {
			remotePort = host.Port
		}
		if remoteLogin == "" {
			remoteLogin = host.User
		}
		if remoteSSHKey == "" {
			remoteSSHKey = host.SSHKey
		}
		if remoteTimeout == 30 {
			remoteTimeout = host.Timeout
		}
		if !insecureSSH {
			insecureSSH = host.InsecureSSH
		}

		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using remote host from config: %s (%s)}}::green",
			time.Now().Format(time.Stamp), host.Name, host.IP))
	}
}

func loadWhitelist() error {
	var whitelistLoaded bool

	if whitelist != "" {
		if _, err := os.Stat(whitelist); !os.IsNotExist(err) {
			if err := utils.LoadWhiteList(whitelist); err != nil {
				return fmt.Errorf("failed to load whitelist %s: %w", whitelist, err)
			}
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded whitelist from %s}}::green",
				time.Now().Format(time.Stamp), whitelist))
			whitelistLoaded = true
		} else {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: whitelist file not found: %s}}::yellow",
				time.Now().Format(time.Stamp), whitelist))
		}
	}

	// Try default location if custom whitelist not loaded
	if !whitelistLoaded {
		defaultWhitelist := "/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules"
		if _, err := os.Stat(defaultWhitelist); !os.IsNotExist(err) {
			if err := utils.LoadWhiteList(defaultWhitelist); err == nil {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded default whitelist from %s}}::green",
					time.Now().Format(time.Stamp), defaultWhitelist))
				whitelistLoaded = true
			}
		}
	}

	if !whitelistLoaded {
		return fmt.Errorf("no whitelist loaded, but whitelist checking is enabled")
	}

	return nil
}

func loadUSBIDs() error {
	if _, err := os.Stat(usbidsPath); !os.IsNotExist(err) {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loading USB IDs database...}}::green", time.Now().Format(time.Stamp)))
		if err := usbids.LoadFromFile(usbidsPath); err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to load %s, trying alternatives...}}::yellow",
				time.Now().Format(time.Stamp), usbidsPath))
			if err := usbids.LoadFromFiles(); err != nil {
				return fmt.Errorf("failed to load USB IDs database: %w", err)
			}
		}
	} else {
		return fmt.Errorf("USB IDs file not found: %s", usbidsPath)
	}

	return nil
}

func validateRemoteFlags() error {
	if remoteIP == "" && remoteHost == "" {
		return fmt.Errorf("remote source requires --remote-ip or --remote-host")
	}
	if remoteLogin == "" {
		return fmt.Errorf("remote source requires --remote-login")
	}
	if remotePass == "" && remoteSSHKey == "" {
		return fmt.Errorf("remote source requires --remote-password or --remote-key")
	}
	return nil
}

func showRemoteWarnings() {
	if insecureSSH {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ⚠️  WARNING: SSH host key verification is DISABLED!}}::bgRed|white|bold",
			time.Now().Format(time.Stamp)))
	}
	if remotePass != "" && remoteSSHKey == "" {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ⚠️  WARNING: Using password authentication. SSH key is more secure.}}::yellow|bold",
			time.Now().Format(time.Stamp)))
	}
}
