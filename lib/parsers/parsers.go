package parsers

import (
	"bufio"
	"compress/gzip"
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/lib/utils"
	"log"
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
		panic(err)
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
	var logEvents []data.LogEvent
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if r.MatchString(line) || u.MatchString(line) {
			toTime := utils.TimeStampToTime(line[:15])
			eventType := utils.GetActionType(line)
			if eventType != "" {
				logEvents = append(logEvents, data.LogEvent{
					Date:       toTime,
					ActionType: eventType,
					LogLine:    line,
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
	}

	defer file.Close()

	return parseLine(bufio.NewScanner(file))
}

func parseGzipped(path string) []data.LogEvent {
	file, err := os.Open(path)

	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cannot read log file: %s}}::red", time.Now().Format(time.Stamp), path))
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	defer gz.Close()
	return parseLine(bufio.NewScanner(gz))
}

func ParseFiles(files []string) []data.LogEvent {
	var recordTypes []data.LogEvent

	for _, file := range files {
		switch filepath.Ext(file) {
		case ".gz":
			recordTypes = append(recordTypes, parseGzipped(file)...)
		case ".log":
			recordTypes = append(recordTypes, parseUGzipped(file)...)
		default:
			recordTypes = append(recordTypes, parseUGzipped(file)...)
		}
	}

	return recordTypes
}

func CollectEventsData(events []data.LogEvent) []data.Event {
	var curr = -1
	var link int
	var interrupted bool
	allEvents := make([]data.Event, 0)

	reVid := regexp.MustCompile(`idVendor=(\w+)`)
	rePid := regexp.MustCompile(`idProduct=(\w+)`)
	reProd := regexp.MustCompile(`Product: (.*?$)`)
	reManufact := regexp.MustCompile(`Manufacturer: (.*?$)`)
	reSerial := regexp.MustCompile(`SerialNumber: (.*?$)`)
	rePort := regexp.MustCompile(`(?m)usb (.*[0-9]):`)
	usStorage := regexp.MustCompile(`usb-storage (.*?$)`)

	for _, event := range events {
		if event.ActionType == "c" {
			if strings.Contains(event.LogLine, "New USB device found, ") {
				host := strings.Split(event.LogLine, ` `)[4]
				vid := utils.GetSub(reVid, event.LogLine, 1)
				pid := utils.GetSub(rePid, event.LogLine, 1)

				port := utils.GetSub(rePort, event.LogLine, 1)
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

			} else if !interrupted {
				switch {
				case link == 2:
					prod := utils.GetSub(reProd, event.LogLine, 1)
					if prod == "" {
						interrupted = true
					} else {
						allEvents[curr].ProductName = prod
						link = 3
					}
				case link == 3:
					manufact := utils.GetSub(reManufact, event.LogLine, 1)
					if manufact == "" {
						interrupted = true
					} else {
						allEvents[curr].ManufacturerName = manufact
						link = 4
					}
				case link == 4:
					serial := utils.GetSub(reSerial, event.LogLine, 1)
					if serial == "" {
						interrupted = true
					} else {
						allEvents[curr].SerialNumber = serial
						link = 5
					}
				case link == 5:
					storage := utils.GetSub(usStorage, event.LogLine, 1)
					if storage != "" {
						allEvents[curr].IsMassStorage = true
					}
					interrupted = true
				}
			} else {
				continue
			}
		} else if event.ActionType == "d" {
			port := utils.GetSub(rePort, event.LogLine, 1)
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
