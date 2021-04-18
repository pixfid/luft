package core

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/i582/cfmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var Root = "/var/log/"

func GetLocalLogs() {
	if LOG_PATH != "" {
		Root = LOG_PATH
	}
	list := GetFilteredHistory()
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded %d logs files}}::green", time.Now().Format(time.Stamp), len(list)))
	recordTypes := ReadLogFile(list)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d events records}}::green", time.Now().Format(time.Stamp), len(recordTypes)))
	events := ParseHistory(recordTypes)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsed %d events}}::green", time.Now().Format(time.Stamp), len(events)))
	filteredEvents := FilterEvents(events)
	clearEvents := RemoveDuplicates(filteredEvents)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter and remove duplicates complete, %d clear events found}}::green", time.Now().Format(time.Stamp), len(clearEvents)))

	if EXPORT {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: %s }}::green", time.Now().Format(time.Stamp), FORMAT))
		var data []byte
		var fn string
		switch FORMAT {
		case "json":
			fn = fmt.Sprintf("events_data.%s", "json")
			data, _ = json.MarshalIndent(clearEvents, "", " ")
		case "xml":
			fn = fmt.Sprintf("events_data.%s", "xml")
			data, _ = xml.MarshalIndent(clearEvents, "", " ")
		case "pdf":
			GenerateReport(clearEvents, fmt.Sprintf("events_data.%s", "pdf"))
		}
		if data != nil {
			err := ioutil.WriteFile(fn, data, fs.ModePerm)
			if err != nil {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Events exported to: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
			}
		}
	} else {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: table}}::green", time.Now().Format(time.Stamp)))
		PrintEvents(clearEvents)
	}
}

func ReadLogFile(list []string) []RecordType {
	var recordTypes []RecordType
	var scanner *bufio.Scanner

	for _, s := range list {

		r, _ := regexp.Compile(`(?:]|:) usb (.*?): `)
		u, _ := regexp.Compile(`(?:]|:) usb-storage (.*?): `)

		if filepath.Ext(s) == ".gz" {
			file, err := os.Open(s)

			if err != nil {
				log.Fatal(err)
			}

			gz, err := gzip.NewReader(file)

			if err != nil {
				log.Fatal(err)
			}

			defer file.Close()
			defer gz.Close()
			scanner = bufio.NewScanner(gz)
		} else {
			file, err := os.Open(s)
			if err != nil {
				log.Fatal(err)
			}

			defer file.Close()
			scanner = bufio.NewScanner(file)
		}

		parseLine := func(line string) {
			if r.MatchString(line) || u.MatchString(line) {
				toTime := TimeStampToTime(line[:15])
				eventType := GetActionType(line)
				if eventType != "" {
					recordTypes = append(recordTypes, RecordType{
						Date:       toTime,
						ActionType: eventType,
						LogLine:    line,
					})
				}
			}
		}

		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			parseLine(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}

	return recordTypes
}

func GetFilteredHistory() []string {
	var files []string
	path, err := Expand(Root)

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "syslog") {
			files = append(files, path)
		} else if strings.Contains(path, "messages") {
			files = append(files, path)
		} else if strings.Contains(path, "kern") {
			files = append(files, path)
		} else if strings.Contains(path, "daemon") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

func ParseHistory(filteredHistory []RecordType) []Event {
	var curr = -1
	var link int
	var interrupted bool
	allEvents := make([]Event, 0)

	reVid, _ := regexp.Compile(`idVendor=(\w+)`)
	rePid, _ := regexp.Compile(`idProduct=(\w+)`)
	reProd, _ := regexp.Compile(`Product: (.*?$)`)
	reManufact, _ := regexp.Compile(`Manufacturer: (.*?$)`)
	reSerial, _ := regexp.Compile(`SerialNumber: (.*?$)`)
	rePort, _ := regexp.Compile(`(?m)usb (.*[0-9]):`)
	usStorage, _ := regexp.Compile(`usb-storage (.*?$)`)

	for _, event := range filteredHistory {
		if event.ActionType == "c" {
			if strings.Contains(event.LogLine, "New USB device found, ") {
				host := strings.Split(event.LogLine, ` `)[4]
				vid := GetSub(reVid, event.LogLine, 1)
				pid := GetSub(rePid, event.LogLine, 1)

				port := GetSub(rePort, event.LogLine, 1)
				allEvents = append(allEvents, Event{
					Conn:     event.Date,
					Host:     host,
					Vid:      vid,
					Pid:      pid,
					Prod:     "None",
					Manufact: "None",
					Serial:   "None",
					Port:     port,
					Disconn:  time.Now(),
				})
				curr++
				link = 2
				interrupted = false

			} else if !interrupted {
				if link == 2 {
					prod := GetSub(reProd, event.LogLine, 1)
					if prod == "" {
						interrupted = true
					} else {
						allEvents[curr].Prod = prod
						link = 3
					}
				} else if link == 3 {
					manufact := GetSub(reManufact, event.LogLine, 1)
					if manufact == "" {
						interrupted = true
					} else {
						allEvents[curr].Manufact = manufact
						link = 4
					}
				} else if link == 4 {
					serial := GetSub(reSerial, event.LogLine, 1)
					if serial == "" {
						interrupted = true
					} else {
						allEvents[curr].Serial = serial
						link = 5
					}
				} else if link == 5 {
					storage := GetSub(usStorage, event.LogLine, 1)
					if storage != "" {
						allEvents[curr].Storage = true
					}
					interrupted = true
				}
			} else {
				continue
			}
		} else if event.ActionType == "d" {
			port := GetSub(rePort, event.LogLine, 1)
			if port != "" {
				for i := range allEvents {
					if allEvents[i].Port == port {
						allEvents[i].Disconn = event.Date
					}
				}
			}
		}
	}
	return allEvents
}
