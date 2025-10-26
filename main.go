package main

import (
	"os"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
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
		Source string `short:"S" long:"source" choice:"local" choice:"remote" choice:"database" description:"events target" required:"true"`
		Export struct {
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

func main() {

	PrintBanner()

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Starting at: %v", time.Now().Format(time.Stamp)))

	p := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash|flags.HelpFlag)
	p.SubcommandsOptional = true

	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[ERROR] cli error: %v}}::red"), err)
		}
		os.Exit(1)
	}

	var parseParams = data.ParseParams{
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
	var err error
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
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ERROR: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		os.Exit(1)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Shut down at: %v", time.Now().Format(time.Stamp)))

}
