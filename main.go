package main

import (
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/lib/parsers"
	"github.com/pixfid/luft/lib/utils"
	"github.com/pixfid/luft/usbids"
	"github.com/umputun/go-flags"
	"log"
	"os"
	"time"
)

const (
	ver = "v0.2"
	url = "https://github.com/pixfid/luft"
)

var opts struct {
	/*
		Ids struct {
			Update bool `long:"update" env:"UPDATE" description:"Update (download) the USB ID database." required:"false"`
			Search struct {
				Vid string `short:"v" long:"vid" env:"VID" description:"Vendor ID"`
				Pid string `short:"p" long:"pid" env:"PID" description:"Product ID"`
			} `group:"search" namespace:"search" env-namespace:"SEARCH"`
		} `group:"ids" namespace:"ids" env-namespace:"IDS"`
	*/

	MassStorage bool   `short:"m" long:"masstorage" env:"MASSTORAGE" description:"show only mass storage devices"`
	Untrusted   bool   `short:"u" long:"untrusted" env:"UNTRUSTED" description:"show only untrusted devices"`
	Number      int    `short:"n" long:"number" env:"NUMBER" description:"number of events to show"`
	Sort        string `short:"s" long:"sort" env:"SORT" choice:"asc" choice:"desc" description:"sort events" default:"asc"`
	Export      bool   `short:"e" long:"export" env:"EXPORT" description:"export events"`
	Check       bool   `short:"c" long:"check" env:"CHECK" description:"check devices for whitelist"`
	ExtUsbIds   bool   `short:"E" long:"extusbids" env:"EXTUSBIDS" description:"external usbids data base"`

	External struct {
		Whitelist string `short:"W" env:"WHITELIST" long:"whitelist" description:"whitelist path"`
		UsbIds    string `short:"U" env:"USBIDS" long:"usbids" description:"usbids path"`
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

func parseLocalEvents(pp data.ParseParams) {
	parsers.LocalEvents(pp)
}

func parseRemoteEvents(pp data.ParseParams) {
	parsers.RemoteEvents(pp)
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
			log.Printf("[ERROR] cli error: %v", err)
		}
		os.Exit(1)
	}

	var pp = data.ParseParams{
		LogPath:            opts.Events.Path,
		WlPath:             opts.External.Whitelist,
		OnlyMass:           opts.MassStorage,
		CheckWl:            opts.Check,
		Number:             opts.Number,
		Export:             opts.Export,
		Format:             opts.Events.Export.Format,
		ExternalUsbIds:     opts.ExtUsbIds,
		ExternalUsbIdsPath: opts.External.UsbIds,
		SortBy:             opts.Sort,
		Untrusted:          opts.Untrusted,
		Login:              opts.Events.Remote.Login,
		Password:           opts.Events.Remote.Password,
		Port:               opts.Events.Remote.Port,
		Ip:                 opts.Events.Remote.IP,
	}

	if opts.Check {
		if _, err := os.Stat(opts.External.Whitelist); !os.IsNotExist(err) {
			err := utils.ParseWL(opts.External.Whitelist)
			if err != nil {
				_, _ = cfmt.Println("{{Error loading external whitelist}}::red")
			}
		} else {
			err := utils.ParseWL("/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules")
			if err != nil {
				_, _ = cfmt.Println("{{Error loading external whitelist}}::red")
			}
		}
	}

	if opts.ExtUsbIds {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using external usb.ids}}::green", time.Now().Format(time.Stamp)))
		if _, err := os.Stat(opts.External.UsbIds); !os.IsNotExist(err) {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] External usb.ids loaded}}::green", time.Now().Format(time.Stamp)))
			err := usbids.LoadFromFile(opts.External.UsbIds)
			if err != nil {
				_, _ = cfmt.Println("{{Error loading external usb.ids will be using embedded}}::red")
			}
		} else {
			_, _ = cfmt.Println("{{Error loading external usb.ids will be using embedded}}::red")
		}
	}

	if opts.Untrusted {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Will be print only untrusted devices}}::green", time.Now().Format(time.Stamp)))
	}

	switch opts.Events.Source {
	case "local":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered events}}::green", time.Now().Format(time.Stamp)))
		parseLocalEvents(pp)
	case "remote":
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered events}}::green", time.Now().Format(time.Stamp)))
		parseRemoteEvents(pp)
	case "database":
	}

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Shut down at: %v", time.Now().Format(time.Stamp)))
}
