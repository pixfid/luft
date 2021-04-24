package usbids

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var (
	vendors    = map[string]*Vendor{}
	vendorLine = regexp.MustCompile(`^([[:xdigit:]]{4})\s{2}(.+)$`)
	deviceLine = regexp.MustCompile(`\t([[:xdigit:]]{4})\s{2}(.+)$`)
)

type Vendor struct {
	Name    string
	ID      string
	Product map[string]*Device
}

type Device struct {
	ID   string
	Name string
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

	var currVendor *Vendor
	var prevVendor *Vendor

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, `#`) {
			continue
		}
		if result := vendorLine.FindStringSubmatch(line); len(result) != 0 {
			if vendor := prevVendor; vendor != nil {
				emitVendor(vendors, *vendor)
			}
			currVendor = &Vendor{
				Name:    result[2],
				ID:      result[1],
				Product: map[string]*Device{},
			}
			prevVendor = currVendor
		} else if result := deviceLine.FindStringSubmatch(line); len(result) != 0 {
			if currVendor := currVendor; currVendor != nil {
				currVendor.Product[result[1]] = &Device{
					ID:   result[1],
					Name: result[2],
				}
			}
		} else {
			break
		}
	}
	return nil
}

func FindDevice(vid, pid string) (string, string) {
	if vendors := vendors; vendors != nil {
		vendor := vendors[vid]
		if vendor != nil {
			product := vendor.Product[pid]
			if product != nil {
				return vendor.Name, product.Name
			}
			return vendor.Name, ""
		}
		return "", ""
	}
	return "", ""
}

func init() {
	err := LoadFromFile("usb.ids")
	if err != nil {
		return
	}
}
