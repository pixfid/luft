package main

import (
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/core/parsers"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/usbids"
	"github.com/umputun/go-flags"
	"os"
	"time"
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
			Format string `short:"F" long:"format" choice:"json" choice:"xml" choice:"pdf" env:"FORMAT" description:"events export format" default:"pdf"`
		} `group:"export" namespace:"export" env-namespace:"EXPORT"`
		Path   string `long:"path" description:"log directory" default:"/var/log/"`
		Remote struct {
			IP       string `short:"I" long:"ip" env:"IP" description:"ip address"`
			Port     string `long:"port" env:"PORT" description:"ssh port" default:"22"`
			Login    string `short:"L" long:"login" env:"LOGIN" description:"login"`
			Password string `short:"P" long:"password" env:"PASSWORD" description:"password"`
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
		ExternalUsbIdsPath: opts.External.UsbIds,
		SortBy:             opts.Sort,
		Untrusted:          opts.Untrusted,
		Login:              opts.Events.Remote.Login,
		Password:           opts.Events.Remote.Password,
		Port:               opts.Events.Remote.Port,
		IP:                 opts.Events.Remote.IP,
	}

	if _, err := os.Stat(opts.External.Whitelist); !os.IsNotExist(err) {
		if err := utils.LoadWhiteList(opts.External.Whitelist); err != nil {
			_, _ = cfmt.Println("{{Error loading external whitelist}}::red")
		}
	} else {
		if err := utils.LoadWhiteList("/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules"); err != nil {
			_, _ = cfmt.Println("{{Error loading system udev whitelist}}::red")
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

	switch opts.Events.Source {
	case "local":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered local events}}::green", time.Now().Format(time.Stamp)))
		parsers.LocalEvents(parseParams)
	case "remote":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered remote events}}::green", time.Now().Format(time.Stamp)))
		parsers.RemoteEvents(parseParams)
	default:
	}

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Shut down at: %v", time.Now().Format(time.Stamp)))

}
