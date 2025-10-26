package parsers

import (
	"bufio"
	"compress/gzip"
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func CollectLogs(params data.ParseParams) []string {
	var files []string

	path, err := utils.ExpandPath(params.LogPath)

	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filed expand path: %s}}::red", time.Now().Format(time.Stamp), path))
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		switch {
		case strings.Contains(path, "syslog"):
			files = append(files, path)
		case strings.Contains(path, "messages"):
			files = append(files, path)
		case strings.Contains(path, "kern"):
			files = append(files, path)
		case strings.Contains(path, "daemon"):
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return files
}

func parseLine(scanner *bufio.Scanner) []data.LogEvent {
	r := regexp.MustCompile(`(?:]|:) usb (.*?): `)
	u := regexp.MustCompile(`(?:]|:) usb-storage (.*?): `)
	d := regexp.MustCompile(`(\S+\s+\d+\s\d{2}:\d{2}:\d{2})`)

	var logEvents []data.LogEvent

	buf := make([]byte, 0, 64*1024)

	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		logLine := scanner.Text()
		if r.MatchString(logLine) || u.MatchString(logLine) {
			logTime := utils.Submatch(d, logLine, 1)
			dateTime := utils.TimeStampToTime(logTime)
			eventType := utils.GetActionType(logLine)

			if eventType != data.Unknown {
				logEvents = append(logEvents, data.LogEvent{
					Date:       dateTime,
					ActionType: eventType,
					LogLine:    logLine,
				})
			}
		}
	}
	return logEvents
}
func parseUGzipped(path string) []data.LogEvent {
	file, err := os.Open(path)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cannot read log file: %s}}::red", time.Now().Format(time.Stamp), path))
		return []data.LogEvent{}
	}

	defer file.Close()

	return parseLine(bufio.NewScanner(file))
}

func parseGzipped(path string) []data.LogEvent {
	file, err := os.Open(path)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cannot read log file: %s}}::red", time.Now().Format(time.Stamp), path))
		return []data.LogEvent{}
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cannot create gzip reader for %s: %s}}::red", time.Now().Format(time.Stamp), path, err.Error()))
		return []data.LogEvent{}
	}
	defer gz.Close()

	return parseLine(bufio.NewScanner(gz))
}

func ParseFiles(files []string) []data.LogEvent {
	var recordTypes []data.LogEvent

	for _, file := range files {
		switch filepath.Ext(file) {
		case ".gz":
			recordTypes = append(recordTypes, parseGzipped(file)...)
		default:
			recordTypes = append(recordTypes, parseUGzipped(file)...)
		}
	}

	return recordTypes
}

// CollectEventsData collect data from events logs.
func CollectEventsData(events []data.LogEvent) []data.Event {
	var curr = -1
	var link int
	var interrupted bool
	allEvents := make([]data.Event, 0)

	reVid := regexp.MustCompile(`idVendor=(\w+)`)
	rePid := regexp.MustCompile(`idProduct=(\w+)`)
	reProduct := regexp.MustCompile(`Product: (.*?$)`)
	reManufacture := regexp.MustCompile(`Manufacturer: (.*?$)`)
	reSerial := regexp.MustCompile(`SerialNumber: (.*?$)`)
	rePort := regexp.MustCompile(`(?m)usb (.*[0-9]):`)
	usStorage := regexp.MustCompile(`usb-storage (.*?$)`)
	reHost := regexp.MustCompile(`(.*:\d{2}\s)(.*) (.*:\s\[)`)

	for _, event := range events {
		if event.ActionType == data.Connected {
			switch {
			case strings.Contains(event.LogLine, "New USB device found, "):
				host := utils.Submatch(reHost, event.LogLine, 2)
				vid := utils.Submatch(reVid, event.LogLine, 1)
				pid := utils.Submatch(rePid, event.LogLine, 1)
				port := utils.Submatch(rePort, event.LogLine, 1)

				allEvents = append(allEvents, data.Event{
					ConnectedTime:     event.Date,
					Host:              host,
					Vid:               vid,
					Pid:               pid,
					ProductName:       "None",
					ManufacturerName:  "None",
					SerialNumber:      "None",
					ConnectionPort:    port,
					DisconnectionTime: time.Now(),
				})

				curr++
				link = 2
				interrupted = false

			case !interrupted:
				switch {
				case link == 2:
					prod := utils.Submatch(reProduct, event.LogLine, 1)
					if prod == "" {
						interrupted = true
					} else {
						allEvents[curr].ProductName = prod
						link = 3
					}
				case link == 3:
					manufacture := utils.Submatch(reManufacture, event.LogLine, 1)
					if manufacture == "" {
						interrupted = true
					} else {
						allEvents[curr].ManufacturerName = manufacture
						link = 4
					}
				case link == 4:
					serial := utils.Submatch(reSerial, event.LogLine, 1)
					if serial == "" {
						interrupted = true
					} else {
						allEvents[curr].SerialNumber = serial
						link = 5
					}
				case link == 5:
					storage := utils.Submatch(usStorage, event.LogLine, 1)
					if storage != "" {
						allEvents[curr].IsMassStorage = true
					}
					interrupted = true
				}
			default:
				continue
			}
		} else if event.ActionType == data.Disconnected {
			port := utils.Submatch(rePort, event.LogLine, 1)
			if port != "" {
				for i := range allEvents {
					if allEvents[i].ConnectionPort == port {
						allEvents[i].DisconnectionTime = event.Date
					}
				}
			}
		}
	}
	return allEvents
}
