package utils

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/jung-kurt/gofpdf"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/usbids"
	"github.com/thoas/go-funk"
)

func Submatch(r *regexp.Regexp, logLine string, idx int) string {
	if match := r.FindStringSubmatch(logLine); match != nil {
		if subMatch := r.FindStringSubmatch(logLine)[idx]; subMatch != "" {
			return subMatch
		}
	}

	return ""
}

func GetActionType(logLine string) data.ActionType {
	switch {
	case strings.Contains(logLine, "New USB device found"):
		return data.Connected
	case strings.Contains(logLine, "Product: "):
		return data.Connected
	case strings.Contains(logLine, "Manufacturer: "):
		return data.Connected
	case strings.Contains(logLine, "SerialNumber: "):
		return data.Connected
	case strings.Contains(logLine, "USB Mass Storage device detected"):
		return data.Connected
	case strings.Contains(logLine, "disconnect"):
		return data.Disconnected
	}

	return data.Unknown
}

func FilterEvents(params data.ParseParams, events []data.Event) []data.Event {
	// Start with all events - critical fix to prevent empty result when OnlyMass=false
	filtered := events

	//filter only mass devices
	if params.OnlyMass {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter only mass storage devices}}::green", time.Now().Format(time.Stamp)))

		filtered = funk.Filter(events, func(event data.Event) bool {
			return event.IsMassStorage
		}).([]data.Event)
	}

	//check by whitelist
	if params.CheckWl {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Checking devices by white list}}::green", time.Now().Format(time.Stamp)))

		for i, event := range filtered {
			if IsInWhiteList(event.SerialNumber) {
				filtered[i].Trusted = true
			}
		}
	}

	//filter Untrusted
	if params.Untrusted {
		filtered = funk.Filter(filtered, func(event data.Event) bool {
			return !event.Trusted
		}).([]data.Event)
	}

	// Enrich with USB IDs database information
	for i, event := range filtered {
		manufactureStr, productStr := usbids.FindDevice(event.Vid, event.Pid)
		if len(productStr) != 0 {
			filtered[i].ProductName = productStr
		}

		if len(manufactureStr) != 0 {
			filtered[i].ManufacturerName = manufactureStr
		}
	}

	// Sort events
	sort.Slice(filtered, func(i, j int) bool {
		switch params.SortBy {
		case "desc":
			return filtered[i].ConnectedTime.After(filtered[j].ConnectedTime)
		case "asc":
			return filtered[i].ConnectedTime.Before(filtered[j].ConnectedTime)
		}
		return filtered[i].ConnectedTime.Before(filtered[j].ConnectedTime)
	})

	// Limit number of results with bounds checking
	if params.Number != 0 {
		if params.Number > len(filtered) {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: requested %d events but only %d available}}::yellow", time.Now().Format(time.Stamp), params.Number, len(filtered)))
			return filtered
		}
		return filtered[0:params.Number]
	}

	return filtered
}

func RemoveDuplicates(events []data.Event) []data.Event {
	// Use map for O(n) performance instead of O(n²)
	seen := make(map[time.Time]bool)
	clearEvents := make([]data.Event, 0, len(events))

	for _, event := range events {
		if !seen[event.ConnectedTime] {
			seen[event.ConnectedTime] = true
			clearEvents = append(clearEvents, event)
		}
	}

	return clearEvents
}

func TimeStampToTime(timeStampString string) time.Time {
	layout := "Jan _2 15:04:05"
	pTime, err := time.Parse(layout, timeStampString)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to parse timestamp '%s': %s}}::red", time.Now().Format(time.Stamp), timeStampString, err.Error()))
		return time.Time{}
	}

	// Linux syslog doesn't include year in timestamps, so we need to add it
	// Use current year, but if the parsed month is in the future, use previous year
	now := time.Now()
	year := now.Year()

	// If the event month is greater than current month, it must be from last year
	// e.g., current date is Jan 2024, but log shows Dec - must be Dec 2023
	if pTime.Month() > now.Month() {
		year--
	}

	// Reconstruct time with the correct year
	pTime = time.Date(year, pTime.Month(), pTime.Day(), pTime.Hour(), pTime.Minute(), pTime.Second(), 0, time.Local)

	return pTime
}

func ExpandPath(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(usr.HomeDir, path[1:]), nil
}

func PrintEvents(e []data.Event) {
	// Configure colorized renderer
	headerTint := renderer.Tint{
		FG: renderer.Colors{color.FgWhite, color.Bold},
	}

	columnTint := renderer.Tint{
		FG: renderer.Colors{color.FgWhite},
		Columns: []renderer.Tint{
			{FG: renderer.Colors{color.FgGreen}}, // Connected time
			{FG: renderer.Colors{color.FgWhite}}, // Host
			{FG: renderer.Colors{color.FgWhite}}, // VID
			{FG: renderer.Colors{color.FgWhite}}, // PID
			{FG: renderer.Colors{color.FgWhite}}, // Manufacturer
			{FG: renderer.Colors{color.FgWhite}}, // Product
			{FG: renderer.Colors{color.FgHiRed}}, // Serial Number (default red for untrusted)
		},
	}

	borderTint := renderer.Tint{
		FG: renderer.Colors{color.FgHiBlack},
	}

	config := renderer.ColorizedConfig{
		Header: headerTint,
		Column: columnTint,
		Border: borderTint,
	}

	// Create table with colorized renderer
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewColorized(config)),
	)

	// Set header
	table.Header("Connected", "Host", "VID", "PID", "Manufacturer", "Product", "Serial Number")

	// Add data rows
	greenSerial := color.New(color.FgGreen).SprintFunc()
	redSerial := color.New(color.FgHiRed).SprintFunc()

	for _, event := range e {
		serialNumber := event.SerialNumber
		// Color the serial number based on trust status
		if event.Trusted {
			serialNumber = greenSerial(event.SerialNumber)
		} else {
			serialNumber = redSerial(event.SerialNumber)
		}

		table.Append(
			event.ConnectedTime.Format("Jan _2 15:04:05"),
			event.Host,
			event.Vid,
			event.Pid,
			event.ManufacturerName,
			event.ProductName,
			serialNumber,
		)
	}

	// Render the table
	table.Render()
}

func GenerateReport(events []data.Event, fn string) error {
	pdf := newReport()
	pdf = image(pdf)
	pdf = header(pdf)
	pdf = table(pdf, events)

	if pdf.Err() {
		return fmt.Errorf("failed creating PDF report: %v", pdf.Error())
	}

	err := savePDF(pdf, fn)
	if err != nil {
		return fmt.Errorf("cannot save PDF to %s: %w", fn, err)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] PDF report saved to: %s}}::green", time.Now().Format(time.Stamp), fn))
	return nil
}

var colWidths = map[string]float64{"C": 30, "H": 30, "V": 10, "P": 10, "PR": 70, "M": 70, "S": 60}
var rowHeight = 6.5

func newReport() *gofpdf.Fpdf {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Times", "B", 20)
	pdf.SetTextColor(255, 24, 0)
	pdf.Cell(40, 10, "USB events history report")
	pdf.SetTextColor(0, 0, 255)
	pdf.Ln(12)
	pdf.SetFont("Times", "B", 15)
	pdf.Cell(40, 7, time.Now().Format("Mon Jan 2, 2006"))
	pdf.Ln(20)
	pdf.SetTextColor(0, 0, 0)
	return pdf
}

func header(pdf *gofpdf.Fpdf) *gofpdf.Fpdf {
	pdf.SetFont("Times", "B", 12)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(colWidths["C"], rowHeight, "CONNECTED", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["H"], rowHeight, "HOST", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["V"], rowHeight, "VID", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["P"], rowHeight, "PID", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["M"], rowHeight, "MANUFACTURER", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["PR"], rowHeight, "PRODUCT", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["S"], rowHeight, "SERIAL NUMBER", "1", 0, "", true, 0, "")
	return pdf
}

func table(pdf *gofpdf.Fpdf, tbl []data.Event) *gofpdf.Fpdf {
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetFillColor(255, 255, 255)
	pdf.Ln(-1)

	for _, event := range tbl {
		pdf.SetTextColor(75, 177, 24)
		pdf.CellFormat(colWidths["C"], rowHeight, event.ConnectedTime.Format("Jan _2 15:04:05"), "1", 0, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(colWidths["H"], rowHeight, event.Host, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["V"], rowHeight, event.Vid, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["P"], rowHeight, event.Pid, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["PR"], rowHeight, event.ProductName, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["M"], rowHeight, event.ManufacturerName, "1", 0, "L", false, 0, "")
		if event.Trusted {
			pdf.SetTextColor(0, 255, 0)
		} else {
			pdf.SetTextColor(255, 24, 0)
		}
		pdf.CellFormat(colWidths["S"], rowHeight, event.SerialNumber, "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	return pdf
}

func image(pdf *gofpdf.Fpdf) *gofpdf.Fpdf {
	pdf.ImageOptions("stats.png", 265, 10, 25, 25, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	return pdf
}

func savePDF(pdf *gofpdf.Fpdf, fn string) error {
	return pdf.OutputFileAndClose(fn)
}

func ExportData(events []data.Event, format string, fileName string) error {
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: %s }}::green", time.Now().Format(time.Stamp), format))

	var exportData []byte
	var fn string
	var err error

	switch format {
	case "json":
		fn = fmt.Sprintf("%s.%s", fileName, "json")
		exportData, err = json.MarshalIndent(events, "", " ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	case "xml":
		fn = fmt.Sprintf("%s.%s", fileName, "xml")
		exportData, err = xml.MarshalIndent(events, "", " ")
		if err != nil {
			return fmt.Errorf("failed to marshal XML: %w", err)
		}
	case "pdf":
		fn = fmt.Sprintf("%s.%s", fileName, "pdf")
		if err := GenerateReport(events, fn); err != nil {
			return fmt.Errorf("failed to generate PDF report: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown export format: %s", format)
	}

	if exportData != nil {
		err := os.WriteFile(fn, exportData, fs.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", fn, err)
		}
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Events exported to: %s}}::green", time.Now().Format(time.Stamp), fn))
	}

	return nil
}
