package utils

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/i582/cfmt"
	"github.com/jung-kurt/gofpdf"
	"github.com/olekukonko/tablewriter"
	"github.com/pixfid/luft/data"
	"github.com/pixfid/luft/usbid"
	"github.com/thoas/go-funk"
	"io/fs"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func GetSub(r *regexp.Regexp, s string, i int) string {
	match := r.FindStringSubmatch(s)
	if match != nil {
		return r.FindStringSubmatch(s)[i]
	}
	return ""
}

func GetActionType(logLine string) string {
	if strings.Contains(logLine, "New USB device found") {
		return "c"
	} else if strings.Contains(logLine, "Product: ") {
		return "c"
	} else if strings.Contains(logLine, "Manufacturer: ") {
		return "c"
	} else if strings.Contains(logLine, "SerialNumber: ") {
		return "c"
	} else if strings.Contains(logLine, "USB Mass Storage device detected") {
		return "c"
	} else if strings.Contains(logLine, "disconnect") {
		return "d"
	}
	return ""
}

func FilterEvents(params data.ParseParams, events []data.Event) []data.Event {
	var filtered []data.Event
	//filter only mass devices
	if params.OnlyMass {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter only mass storage devices}}::green", time.Now().Format(time.Stamp)))
		filtered = funk.Filter(events, func(event data.Event) bool {
			return event.IsMassStorage
		}).([]data.Event)
	}

	//check by whitelist
	if params.CheckWl {
		var tmp []data.Event
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Checking devices by white list}}::green", time.Now().Format(time.Stamp)))
		for _, event := range filtered {
			if funk.Contains(whiteList, event.SerialNumber) {
				event.Trusted = true
			}
			tmp = append(tmp, event)
		}
		filtered = tmp
	}

	//filter Untrusted
	if params.Untrusted {
		filtered = funk.Filter(filtered, func(event data.Event) bool {
			return !event.Trusted
		}).([]data.Event)
	}

	var tmp []data.Event
	for _, event := range filtered {
		manufactStr, productStr := usbid.FindDevice(event.Vid, event.Pid)
		event.ManufacturerName = manufactStr
		if len(productStr) != 0 {
			event.ProductName = productStr
		}
		tmp = append(tmp, event)
	}
	filtered = tmp

	sort.Slice(filtered, func(i, j int) bool {
		switch params.SortBy {
		case "desc":
			return filtered[i].ConnectedTime.After(filtered[j].ConnectedTime)
		case "asc":
			return filtered[i].ConnectedTime.Before(filtered[j].ConnectedTime)
		}
		return filtered[i].ConnectedTime.Before(filtered[j].ConnectedTime)
	})

	if params.Number != 0 {
		return filtered[0 : params.Number+1]
	}
	return filtered
}

func RemoveDuplicates(events []data.Event) []data.Event {
	var clearEvents []data.Event
	for _, event := range events {
		if !InSlice(clearEvents, event) {
			clearEvents = append(clearEvents, event)
		}
	}
	return clearEvents
}

func InSlice(arr []data.Event, val data.Event) bool {
	for _, v := range arr {
		if v.ConnectedTime == val.ConnectedTime {
			return true
		}
	}
	return false
}

func TimeStampToTime(timeStampString string) time.Time {
	layout := "Jan _2 15:04:05"
	pTime, _ := time.Parse(layout, timeStampString)
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

func ClearLogs(list []string) []string {
	var filtered []string
	for _, s := range list {
		if !strings.Contains(s, "parsec") {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func PrintEvents(e []data.Event) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Connected", "Host", "VID", "PID", "Product", "Manufacturer", "Serial Number"}) //, "Port", "Disconnected"})

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor}, //connection date
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},   //host
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},   //vid
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},   //pid
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},   //product
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},   //manufacturer
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},   //serial
		//tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //port
		//tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},   //disconnected
	)

	for _, event := range e {
		if event.Trusted {
			table.Rich([]string{
				event.ConnectedTime.Format("Jan _2 15:04:05"),
				event.Host,
				event.Vid,
				event.Pid,
				event.ProductName,
				event.ManufacturerName,
				event.SerialNumber,
				//event.Port,
				//event.Disconn
			},
				[]tablewriter.Colors{
					{tablewriter.Bold, tablewriter.FgHiGreenColor}, //connection date
					{tablewriter.Bold, tablewriter.FgBlackColor},   //host
					{tablewriter.Bold, tablewriter.FgBlackColor},   //vid
					{tablewriter.Bold, tablewriter.FgBlackColor},   //pid
					{tablewriter.Bold, tablewriter.FgBlackColor},   //product
					{tablewriter.Bold, tablewriter.FgBlackColor},   //manufacturer
					{tablewriter.Bold, tablewriter.FgHiGreenColor}, //serial
					//{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //port
					//{tablewriter.Bold, tablewriter.FgHiRedColor},   //disconnected
				})
		} else {
			table.Append([]string{event.ConnectedTime.Format("Jan _2 15:04:05"), event.Host, event.Vid, event.Pid, event.ProductName, event.ManufacturerName, event.SerialNumber}) //event.Port, event.Disconn})
		}
	}
	table.SetColumnSeparator("║")
	table.SetRowSeparator("═")
	table.SetCenterSeparator("╬")

	table.SetBorder(true) // Set Border to false
	table.SetAutoMergeCells(false)
	table.SetRowLine(true)
	table.Render()
}

func GenerateReport(events []data.Event, fn string) {
	pdf := newReport()
	pdf = image(pdf)
	pdf = header(pdf)
	pdf = table(pdf, events)

	if pdf.Err() {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed creating PDF report: %s}}::red", time.Now().Format(time.Stamp), pdf.Err()))
	}

	err := savePDF(pdf, fn)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cannot save PDF: %s}}::red", time.Now().Format(time.Stamp), pdf.Err()))
	}
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
	pdf.CellFormat(colWidths["PR"], rowHeight, "PRODUCT", "1", 0, "", true, 0, "")
	pdf.CellFormat(colWidths["M"], rowHeight, "MANUFACTURER", "1", 0, "", true, 0, "")
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

func ExportData(events []data.Event, format string) {
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: %s }}::green", time.Now().Format(time.Stamp), format))
	var data []byte
	var fn string
	switch format {
	case "json":
		fn = fmt.Sprintf("events_data.%s", "json")
		data, _ = json.MarshalIndent(events, "", " ")
	case "xml":
		fn = fmt.Sprintf("events_data.%s", "xml")
		data, _ = xml.MarshalIndent(events, "", " ")
	case "pdf":
		GenerateReport(events, fmt.Sprintf("events_data.%s", "pdf"))
	}
	if data != nil {
		err := ioutil.WriteFile(fn, data, fs.ModePerm)
		if err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Events exported to: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		}
	}
}