package main

import (
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/core"
	"github.com/pixfid/luft/usbid"
	"github.com/thatisuday/commando"
	"time"
)

const (
	ver = "v0.1"
	url = "https://github.com/pixfid/luft"
)

func main() {
	PrintBanner()
	_, _ = cfmt.Println(cfmt.Sprintf("[*] Starting at: %v", time.Now().Format(time.Stamp)))

	commando.
		SetExecutableName("luft").
		SetVersion(ver).
		SetDescription("Is a simple forensics tool with command line interface that lets you keep track of" +
			" USB device artifacts (i.e., USB event history) on Linux machines.")

	commando.
		Register("ids").
		SetDescription("working with usb.ids database").
		SetShortDescription("usb.ids database").
		AddArgument("action", "actions search or download usb.ids <search|download>", "").
		AddFlag("vid, v", "vendor ID", commando.String, "1").
		AddFlag("pid, p", "product ID", commando.String, "1").
		AddFlag("usbids, e", "external usb.ids path", commando.String, "usb.ids").
		AddFlag("external, U", "use external usb.ids", commando.Bool, false).
		SetAction(ids)

	commando.
		Register("local").
		SetDescription("get usb events history from local machine.").
		SetShortDescription("local events history.").
		AddArgument("action", "one of actions <history | export>", "").
		AddFlag("check, C", "check by whitelist", commando.Bool, false).
		AddFlag("all, a", "show all usb devices", commando.Bool, false).
		AddFlag("sort, S", "sort by ascending or descending: asc | desc", commando.String, "asc").
		AddFlag("format, F", "events export format: json | xml | pdf", commando.String, "json").
		AddFlag("untrusted, u", "show only untrusted devices", commando.Bool, false).
		AddFlag("log, L", "external log path", commando.String, "/var/log/").
		AddFlag("whitelist, W", "external whitelist path", commando.String, "/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules").
		AddFlag("usbids, E", "external usb.ids path", commando.String, "usb.ids").
		AddFlag("external, U", "use external usb.ids", commando.Bool, false).
		SetAction(local)

	commando.Register("remote").
		SetDescription("get events history from remote machine.").
		SetShortDescription("remote events history.").
		AddArgument("action", "one of actions <history | export>", "").
		AddFlag("check, C", "check by whitelist", commando.Bool, false).
		AddFlag("all, a", "show all usb devices", commando.Bool, false).
		AddFlag("sort, S", "sort by ascending or descending: asc | desc", commando.String, "asc").
		AddFlag("format, F", "events export format: json | xml | pdf", commando.String, "json").
		AddFlag("whitelist, W", "external whitelist path", commando.String, "/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules").
		AddFlag("server, s", "server", commando.String, "localhost").
		AddFlag("port, p", "port", commando.String, "20").
		AddFlag("login, L", "login", commando.String, "").
		AddFlag("password, P", "password", commando.String, "").
		AddFlag("untrusted, u", "show only untrusted devices", commando.Bool, false).
		SetAction(remote)

	commando.Parse(nil)

	_, _ = cfmt.Println(cfmt.Sprintf("[*] Shut down at: %v", time.Now().Format(time.Stamp)))
}

func local(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
	core.LOG_PATH = flags["log"].Value.(string)
	core.SORT_BY = flags["sort"].Value.(string)
	core.EXTERNAL_USBIDS = flags["external"].Value.(bool)
	core.EXTERNAL_USBIDS_PATH = flags["usbids"].Value.(string)
	core.WL_PATH = flags["whitelist"].Value.(string)
	core.CHECK_WL = flags["check"].Value.(bool)
	core.ONLY_MASS = flags["all"].Value.(bool)
	core.UNTRUSTED = flags["untrusted"].Value.(bool)

	if core.UNTRUSTED {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Will be print only untrusted devices}}::green", time.Now().Format(time.Stamp)))
	}

	if core.EXTERNAL_USBIDS {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using external usb.ids}}::green", time.Now().Format(time.Stamp)))
		err := usbid.LoadFromFile(core.EXTERNAL_USBIDS_PATH)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] External usb.ids loaded}}::green", time.Now().Format(time.Stamp)))
		if err != nil {
			_, _ = cfmt.Println("{{Error loading external usb.ids will be using embedded}}::red")
		}
	}

	if core.CHECK_WL {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using whitelist}}::green", time.Now().Format(time.Stamp)))
		if core.WL_PATH != "" {
			core.ParseWL(core.WL_PATH)
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] External whitelist loaded}}::green", time.Now().Format(time.Stamp)))
		} else {
			core.ParseWL("/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules")
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] System whitelist using}}::green", time.Now().Format(time.Stamp)))
		}

	}

	if args["action"].Value == "history" {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Preparing gathered events}}::green", time.Now().Format(time.Stamp)))
		core.GetLocalLogs()
	}

	if args["action"].Value == "export" {
		core.FORMAT = flags["format"].Value.(string)
		core.EXPORT = true
		core.GetLocalLogs()
	}
}

func remote(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
	core.SORT_BY = flags["sort"].Value.(string)
	core.CHECK_WL = flags["check"].Value.(bool)
	core.WL_PATH = flags["whitelist"].Value.(string)
	core.CHECK_WL = flags["check"].Value.(bool)
	core.ONLY_MASS = flags["all"].Value.(bool)
	core.UNTRUSTED = flags["untrusted"].Value.(bool)

	if core.UNTRUSTED {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Will be print only untrusted devices}}::green", time.Now().Format(time.Stamp)))
	}

	if core.CHECK_WL {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using whitelist}}::green", time.Now().Format(time.Stamp)))
		if core.WL_PATH != "" {
			core.ParseWL(core.WL_PATH)
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] External whitelist loaded}}::green", time.Now().Format(time.Stamp)))
		} else {
			core.ParseWL("/etc/udev/rules.d/99_PDAC_LOCAL_flash.rules")
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] System whitelist using}}::green", time.Now().Format(time.Stamp)))
		}

	}

	if args["action"].Value == "history" {
		server := flags["server"].Value.(string)
		port := flags["port"].Value.(string)
		login := flags["login"].Value.(string)
		password := flags["password"].Value.(string)
		core.GetRemoteLogs(server, port, login, password)
	}

	if args["action"].Value == "export" {
		core.FORMAT = flags["format"].Value.(string)
		core.EXPORT = true
		server := flags["server"].Value.(string)
		port := flags["port"].Value.(string)
		login := flags["login"].Value.(string)
		password := flags["password"].Value.(string)
		core.GetRemoteLogs(server, port, login, password)
	}

}

func ids(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
	if args["action"].Value == "download" {
		downloadUrl := flags["url"].Value.(string)
		err := usbid.DownloadUsbIds(downloadUrl)
		if err != nil {
			_, _ = cfmt.Println("{{%s}}::green", err.Error())
		} else {
			_, _ = cfmt.Println("{{USB.IDS will be updated}}::green")
		}
	}

	if args["action"].Value == "search" {
		core.EXTERNAL_USBIDS = flags["external"].Value.(bool)
		core.EXTERNAL_USBIDS_PATH = flags["usbids"].Value.(string)

		if core.EXTERNAL_USBIDS {
			err := usbid.LoadFromFile(core.EXTERNAL_USBIDS_PATH)
			if err != nil {
				_, _ = cfmt.Println("{{Error loading external usb.ids will be using embedded}}::red")
			}
		}

		vid := flags["vid"].Value.(string)
		pid := flags["pid"].Value.(string)
		manufactStr, productStr := usbid.FindDevice(vid, pid)
		_, _ = cfmt.Println(cfmt.Sprintf("Device -> Manufacturer: {{%s}}::green Product: {{%s}}::green", manufactStr, productStr))
	}
}

func PrintBanner() {
	_, _ = cfmt.Println(cfmt.Sprintf(`
{{┬  ┬ ┬┌─┐┌┬┐}}::bgLightRed
{{│  │ │├┤  │ }}::bgLightRed {{Linux Usb Forensic Tool %s}}::lightYellow
{{┴─┘└─┘└   ┴ }}::bgLightRed {{%s}}::lightBlue`, ver, url))
}
