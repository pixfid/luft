package core

import (
	"github.com/i582/cfmt"
	"github.com/jung-kurt/gofpdf"
	"github.com/olekukonko/tablewriter"
	"github.com/pixfid/luft/usbid"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type RecordType struct {
	Date       time.Time
	ActionType string
	LogLine    string
}

type Event struct {
	Conn     time.Time
	Host     string
	Vid      string
	Pid      string
	Prod     string
	Manufact string
	Serial   string
	Port     string
	Disconn  time.Time
	wl       bool
	Storage  bool
}

func PrintEvents(e []Event) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Connected", "Host", "VID", "PID", "Product", "Manufacturer", "Serial Number"}) //, "Port", "Disconnected"})

	table.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor}, //connection date
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //host
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //vid
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //pid
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //product
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //manufacturer
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},   //serial
		//tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //port
		//tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},   //disconnected
	)

	for _, event := range e {
		if event.wl {
			table.Rich([]string{
				event.Conn.Format("Jan _2 15:04:05"),
				event.Host,
				event.Vid,
				event.Pid,
				event.Prod,
				event.Manufact,
				event.Serial,
				//event.Port,
				//event.Disconn
			},
				[]tablewriter.Colors{
					{tablewriter.Bold, tablewriter.FgHiGreenColor}, //connection date
					{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //host
					{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //vid
					{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //pid
					{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //product
					{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //manufacturer
					{tablewriter.Bold, tablewriter.FgHiGreenColor}, //serial
					//{tablewriter.Bold, tablewriter.FgHiWhiteColor}, //port
					//{tablewriter.Bold, tablewriter.FgHiRedColor},   //disconnected
				})
		} else {
			table.Append([]string{event.Conn.Format("Jan _2 15:04:05"), event.Host, event.Vid, event.Pid, event.Prod, event.Manufact, event.Serial}) //event.Port, event.Disconn})
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

func FilterEvents(events []Event) []Event {
	filterEvents := make([]Event, 0)
	if ONLY_MASS {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter only mass storage devices}}::green", time.Now().Format(time.Stamp)))
	}

	if CHECK_WL {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Checking devices by white list}}::green", time.Now().Format(time.Stamp)))
	}

	for _, event := range events {
		if ONLY_MASS || event.Storage {
			if CHECK_WL {
				if IsInWhiteList(event.Serial) {
					event.wl = true
				}
			}
			manufactStr, productStr := usbid.FindDevice(event.Vid, event.Pid)
			event.Manufact = manufactStr
			if len(productStr) != 0 {
				event.Prod = productStr
			}

			if UNTRUSTED {
				if !event.wl {
					filterEvents = append(filterEvents, event)
				}
			} else {
				filterEvents = append(filterEvents, event)
			}
		}
	}
	sort.Slice(filterEvents, func(i, j int) bool {
		switch SORT_BY {
		case "desc":
			return filterEvents[i].Conn.After(filterEvents[j].Conn)
		case "asc":
			return filterEvents[i].Conn.Before(filterEvents[j].Conn)
		}
		return filterEvents[i].Conn.Before(filterEvents[j].Conn)
	})

	return filterEvents
}

func IsInWhiteList(serial string) bool {
	for _, s := range WHITE_LIST {
		if s == serial {
			return true
		}
	}
	return false
}

func RemoveDuplicates(events []Event) []Event {
	var clearEvents []Event
	for _, event := range events {
		if !InSlice(clearEvents, event) {
			clearEvents = append(clearEvents, event)
		}
	}
	return clearEvents
}

func InSlice(arr []Event, val Event) bool {
	for _, v := range arr {
		if v.Conn == val.Conn {
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

func Expand(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, path[1:]), nil
}

func GenerateReport(events []Event, fn string) {
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

//

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

func table(pdf *gofpdf.Fpdf, tbl []Event) *gofpdf.Fpdf {
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetFillColor(255, 255, 255)
	pdf.Ln(-1)

	for _, event := range tbl {
		pdf.SetTextColor(75, 177, 24)
		pdf.CellFormat(colWidths["C"], rowHeight, event.Conn.Format("Jan _2 15:04:05"), "1", 0, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(colWidths["H"], rowHeight, event.Host, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["V"], rowHeight, event.Vid, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["P"], rowHeight, event.Pid, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["PR"], rowHeight, event.Prod, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths["M"], rowHeight, event.Manufact, "1", 0, "L", false, 0, "")
		if event.wl {
			pdf.SetTextColor(0, 255, 0)
		} else {
			pdf.SetTextColor(255, 24, 0)
		}
		pdf.CellFormat(colWidths["S"], rowHeight, event.Serial, "1", 0, "L", false, 0, "")
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
