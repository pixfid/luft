package parsers

import (
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/lib/utils"
	"os"
	"time"
)

func LocalEvents(params data.ParseParams) {
	hostName, _ := os.Hostname()
	if _, err := os.Stat(params.LogPath); !os.IsNotExist(err) {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Log directory missing: }}::red", time.Now().Format(time.Stamp)))
	}
	list := CollectLogs(params)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Starting on: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded %d logs files}}::green", time.Now().Format(time.Stamp), len(list)))

	recordTypes := ParseFiles(list)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d events records}}::green", time.Now().Format(time.Stamp), len(recordTypes)))
	events := CollectEventsData(recordTypes)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsed %d events}}::green", time.Now().Format(time.Stamp), len(events)))
	events = utils.RemoveDuplicates(events)
	events = utils.FilterEvents(params, events)

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter and remove duplicates complete, %d clear events found}}::green", time.Now().Format(time.Stamp), len(events)))

	if params.Export {
		utils.ExportData(events, params.Format)
	} else {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: table}}::green", time.Now().Format(time.Stamp)))
		utils.PrintEvents(events)
	}
}
