package parsers

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
)

// Compiled regular expressions for log parsing (performance optimization)
var (
	// parseLine regexes
	reUSB        = regexp.MustCompile(`(?:]|:) usb (.*?): `)
	reUSBStorage = regexp.MustCompile(`(?:]|:) usb-storage (.*?): `)
	reTimestamp  = regexp.MustCompile(`(\S+\s+\d+\s\d{2}:\d{2}:\d{2})`)

	// CollectEventsData regexes
	reVid             = regexp.MustCompile(`idVendor=(\w+)`)
	rePid             = regexp.MustCompile(`idProduct=(\w+)`)
	reProduct         = regexp.MustCompile(`Product: (.*?$)`)
	reManufacture     = regexp.MustCompile(`Manufacturer: (.*?$)`)
	reSerial          = regexp.MustCompile(`SerialNumber: (.*?$)`)
	rePort            = regexp.MustCompile(`(?m)usb (.*[0-9]):`)
	reUSBStorageMatch = regexp.MustCompile(`usb-storage (.*?$)`)
	reHost            = regexp.MustCompile(`(.*:\d{2}\s)(.*) (.*:\s\[)`)
)

func CollectLogs(params data.ParseParams) ([]string, error) {
	var files []string

	path, err := utils.ExpandPath(params.LogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path %s: %w", params.LogPath, err)
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/directories that we can't access
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: skipping %s: %s}}::yellow", time.Now().Format(time.Stamp), path, err.Error()))
			return nil
		}
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
		return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no log files found in %s", path)
	}

	return files, nil
}

func parseLine(scanner *bufio.Scanner) []data.LogEvent {
	var logEvents []data.LogEvent

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		logLine := scanner.Text()
		if reUSB.MatchString(logLine) || reUSBStorage.MatchString(logLine) {
			logTime := utils.Submatch(reTimestamp, logLine, 1)
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

// fileJob represents a file parsing job
type fileJob struct {
	path  string
	index int // to preserve order
}

// fileResult represents the result of parsing a file
type fileResult struct {
	events []data.LogEvent
	index  int
	err    error
}

// ParseFiles parses log files in parallel using a worker pool
func ParseFiles(files []string) []data.LogEvent {
	return ParseFilesWithWorkers(files, 0)
}

// ParseFilesWithWorkers parses log files in parallel with specified number of workers
// If workers <= 0, uses runtime.NumCPU()
func ParseFilesWithWorkers(files []string, workers int) []data.LogEvent {
	if len(files) == 0 {
		return []data.LogEvent{}
	}

	startTime := time.Now()

	// Determine worker count
	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	// Cap at file count (no point having more workers than files)
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	// If only one file or one worker, use sequential parsing for simplicity
	if numWorkers == 1 || len(files) == 1 {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsing %d log file(s) sequentially...}}::cyan",
			time.Now().Format(time.Stamp), len(files)))
		events := ParseFilesSequential(files)
		duration := time.Since(startTime)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Parsed %d events from %d file(s) in %v}}::green",
			time.Now().Format(time.Stamp), len(events), len(files), duration))
		return events
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsing %d log files using %d workers...}}::cyan",
		time.Now().Format(time.Stamp), len(files), numWorkers))

	// Create channels
	jobs := make(chan fileJob, len(files))
	results := make(chan fileResult, len(files))

	// Start workers
	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go parseWorker(w, jobs, results, &wg)
	}

	// Send jobs
	for i, file := range files {
		jobs <- fileJob{path: file, index: i}
	}
	close(jobs)

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	fileResults := make([]fileResult, 0, len(files))
	for result := range results {
		if result.err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to parse file: %s}}::yellow",
				time.Now().Format(time.Stamp), result.err.Error()))
		}
		fileResults = append(fileResults, result)
	}

	// Sort results by original index to preserve order
	sortFileResults(fileResults)

	// Aggregate all events
	totalEvents := 0
	for _, fr := range fileResults {
		totalEvents += len(fr.events)
	}

	allEvents := make([]data.LogEvent, 0, totalEvents)
	for _, fr := range fileResults {
		allEvents = append(allEvents, fr.events...)
	}

	duration := time.Since(startTime)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Parsed %d events from %d files in %v}}::green",
		time.Now().Format(time.Stamp), len(allEvents), len(files), duration))

	return allEvents
}

// parseWorker is a worker that processes file parsing jobs
func parseWorker(id int, jobs <-chan fileJob, results chan<- fileResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		var events []data.LogEvent
		var err error

		switch filepath.Ext(job.path) {
		case ".gz":
			events = parseGzipped(job.path)
		default:
			events = parseUGzipped(job.path)
		}

		// Check if parsing had errors (empty result might indicate error)
		if len(events) == 0 {
			// This is not necessarily an error, file might just be empty
			// but parseGzipped/parseUGzipped don't return errors
		}

		results <- fileResult{
			events: events,
			index:  job.index,
			err:    err,
		}
	}
}

// sortFileResults sorts file results by index to preserve original order
func sortFileResults(results []fileResult) {
	// Simple insertion sort for small slices (typically we have few files)
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		for j >= 0 && results[j].index > key.index {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}

// ParseFilesSequential parses files sequentially (legacy fallback)
func ParseFilesSequential(files []string) []data.LogEvent {
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
					storage := utils.Submatch(reUSBStorageMatch, event.LogLine, 1)
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
