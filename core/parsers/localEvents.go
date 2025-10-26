package parsers

import (
	"fmt"
	"os"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
)

func LocalEvents(params data.ParseParams) error {
	path, err := utils.ExpandPath(params.LogPath)
	if err != nil {
		return fmt.Errorf("failed to expand log path: %w", err)
	}

	hostName, err := os.Hostname()
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to get hostname: %s}}::yellow", time.Now().Format(time.Stamp), err.Error()))
		hostName = "unknown"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("log directory does not exist: %s", path)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Starting on: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName))

	list, err := CollectLogs(params)
	if err != nil {
		return fmt.Errorf("failed to collect log files: %w", err)
	}
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded %d logs files}}::green", time.Now().Format(time.Stamp), len(list)))

	recordTypes := ParseFiles(list)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d events records}}::green", time.Now().Format(time.Stamp), len(recordTypes)))

	events := CollectEventsData(recordTypes)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsed %d events}}::green", time.Now().Format(time.Stamp), len(events)))

	events = utils.RemoveDuplicates(events)
	events = utils.FilterEvents(params, events)

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter and remove duplicates complete, %d clear events found}}::green", time.Now().Format(time.Stamp), len(events)))

	if params.Export {
		if err := utils.ExportData(events, params.Format, params.FileName); err != nil {
			return fmt.Errorf("failed to export events: %w", err)
		}
	} else {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: table}}::green", time.Now().Format(time.Stamp)))
		utils.PrintEvents(events)
	}

	return nil
}
