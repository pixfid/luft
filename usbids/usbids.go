package usbids

import (
	"bufio"
	"github.com/i582/cfmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	vendors                 = map[string]*Vendor{}
	classes                 = map[string]*DeviceClass{}
	audioClassTerminalTypes = map[string]*AudioClassTerminalType{}
	videoClassTerminalTypes = map[string]*VideClassTerminalType{}
	hids                    = map[string]*HID{}

	version            = regexp.MustCompile(`Version: (\d{4}.\d{2}.\d{2})`)
	date               = regexp.MustCompile(`Date:\s+(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`)
	vendorLine         = regexp.MustCompile(`^([[:xdigit:]]{4})\s{2}(.+)$`)
	deviceLine         = regexp.MustCompile(`\t([[:xdigit:]]{4})\s{2}(.+)$`)
	deviceClassLine    = regexp.MustCompile(`^(C)\s+([[:xdigit:]]{2})\s+(.*)`)
	deviceSubClassLine = regexp.MustCompile(`^\t([[:xdigit:]]{2})\s+(.*)`)
	deviceProtocolLine = regexp.MustCompile(`^\t\t([[:xdigit:]]{2})\s+(.*)`)
	actTypeLine        = regexp.MustCompile(`^(AT)\s+([[:xdigit:]]{4})\s+(.*)`)
	vctTypeLine        = regexp.MustCompile(`^(VT)\s+(\d{4})\s+(.*)`)
	hidLine            = regexp.MustCompile(`^(HID)\s+([[:xdigit:]]{2})\s+(.*)`)

	Ids     = []string{"/var/lib/usbutils/usb.ids", "/usr/share/hwdata/usb.ids", "usb.ids"}
	Version = ""
	Date    = ""
)

type HID struct {
	ID   string
	Name string
}

type AudioClassTerminalType struct {
	ID   string
	Name string
}

type VideClassTerminalType struct {
	ID   string
	Name string
}

type Vendor struct {
	ID     string
	Name   string
	Device map[string]*Device
}

type Device struct {
	ID        string
	Name      string
	Interface map[string]*Interface
}

type Interface struct {
	ID   string
	Name string
}

type DeviceClass struct {
	ID             string
	Name           string
	DeviceSubClass map[string]*DeviceSubClass
}

type DeviceSubClass struct {
	ID             string
	Name           string
	DeviceProtocol map[string]*DeviceProtocol
}

type DeviceProtocol struct {
	ID   string
	Name string
}

func LoadFromFiles() error {
	for _, usbID := range Ids {
		if err := LoadFromFile(usbID); err != nil {
			continue
		}
		return nil
	}
	return nil
}

func LoadFromFile(path string) error {
	file, err := os.Open(path)

	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)

	emitVendor := func(vendors map[string]*Vendor, vendor Vendor) {
		vendors[vendor.ID] = &vendor
	}

	emitClass := func(classes map[string]*DeviceClass, class DeviceClass) {
		classes[class.ID] = &class
	}

	emitActt := func(actts map[string]*AudioClassTerminalType, actt AudioClassTerminalType) {
		actts[actt.ID] = &actt
	}

	emitVctt := func(vctts map[string]*VideClassTerminalType, vctt VideClassTerminalType) {
		vctts[vctt.ID] = &vctt
	}

	emitHID := func(hids map[string]*HID, hid HID) {
		hids[hid.ID] = &hid
	}

	var currVendor *Vendor
	var prevVendor *Vendor

	var currClass *DeviceClass
	var prevClass *DeviceClass
	var classId string

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, `#`) {
			if result := version.FindStringSubmatch(line); len(result) != 0 {
				Version = result[1]
			}
			if result := date.FindStringSubmatch(line); len(result) != 0 {
				Date = result[1]
			}
			continue
		} else if result := vendorLine.FindStringSubmatch(line); len(result) != 0 {
			if vendor := prevVendor; vendor != nil {
				emitVendor(vendors, *vendor)
			}
			currVendor = &Vendor{
				Name:   result[2],
				ID:     result[1],
				Device: map[string]*Device{},
			}
			prevVendor = currVendor
		} else if result := deviceLine.FindStringSubmatch(line); len(result) != 0 {
			if currVendor := currVendor; currVendor != nil {
				currVendor.Device[result[1]] = &Device{
					ID:   result[1],
					Name: result[2],
				}
			}
		} else if result := deviceClassLine.FindStringSubmatch(line); len(result) != 0 {
			if class := prevClass; class != nil {
				emitClass(classes, *class)
			}
			currClass = &DeviceClass{
				ID:             result[2],
				Name:           result[3],
				DeviceSubClass: map[string]*DeviceSubClass{},
			}
			prevClass = currClass
		} else if result := deviceSubClassLine.FindStringSubmatch(line); len(result) != 0 {
			if currClass := currClass; currClass != nil {
				currClass.DeviceSubClass[result[1]] = &DeviceSubClass{
					ID:             result[1],
					Name:           result[2],
					DeviceProtocol: map[string]*DeviceProtocol{},
				}
				classId = result[1]
			}
		} else if result := deviceProtocolLine.FindStringSubmatch(line); len(result) != 0 {
			if currClass := currClass; currClass != nil {
				if currentSubClass := currClass.DeviceSubClass[classId]; currentSubClass != nil {
					currentSubClass.DeviceProtocol[result[1]] = &DeviceProtocol{
						ID:   result[1],
						Name: result[2],
					}
				}
			}
		} else if result := actTypeLine.FindStringSubmatch(line); len(result) != 0 { //List of Audio Class Terminal Types
			emitActt(audioClassTerminalTypes, AudioClassTerminalType{
				ID:   result[2],
				Name: result[3],
			})
		} else if result := vctTypeLine.FindStringSubmatch(line); len(result) != 0 { //List of Video Class Terminal Types
			emitVctt(videoClassTerminalTypes, VideClassTerminalType{
				ID:   result[2],
				Name: result[3],
			})
		} else if result := hidLine.FindStringSubmatch(line); len(result) != 0 { //List of HID Descriptor Types
			emitHID(hids, HID{
				ID:   result[2],
				Name: result[3],
			})
		} else {
			continue
		}
	}
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] usb.ids loaded from: %s, Version: %s, Date: %s}}::green", time.Now().Format(time.Stamp), path, Version, Date))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] usb.ids %d vendors load}}::green", time.Now().Format(time.Stamp), len(vendors)))
	return nil
}

func FindDevice(vid, pid string) (string, string) {
	if vendors := vendors; vendors != nil {
		vendor := vendors[vid]
		if vendor != nil {
			device := vendor.Device[pid]
			if device != nil {
				return vendor.Name, device.Name
			}
			return vendor.Name, ""
		}
		return "", ""
	}
	return "", ""
}
