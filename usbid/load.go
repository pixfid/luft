// Copyright 2013 Google Inc.  All rights reserved.
// Copyright 2016 the core Authors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package usbid

import (
	"bufio"
	"fmt"
	"github.com/pixfid/luft/core"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// LinuxUsbDotOrg is one source of files in the format used by this package.
	LinuxUsbDotOrg = "http://www.linux-usb.org/usb.ids"
	usbIds         = "usb.ids"
)

var (
	// Vendors stores the vendor and product ID mappings.
	Vendors map[core.ID]*Vendor

	// Classes stores the class, subclass and protocol mappings.
	Classes map[core.Class]*Class
)

// LoadFromURL replaces the built-in vendor and class mappings with ones loaded
// from the given URL.
//
// This should usually only be necessary if the mappings in the library are
// stale.  The contents of this file as of February 2012 are embedded in the
// library itself.
func LoadFromURL(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ids, cls, err := ParseIDs(resp.Body)
	if err != nil {
		return err
	}

	Vendors = ids
	Classes = cls
	LastUpdate = time.Now()
	return nil
}

func DownloadUsbIds(customDownloadUrl string) error {
	downloadUrl := LinuxUsbDotOrg
	if customDownloadUrl != "" {
		downloadUrl = customDownloadUrl
	}
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if exists(usbIds) {
		err := os.Remove(usbIds)
		if err != nil {
			return err
		}
	}

	out, err := os.Create(usbIds)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func LoadFromFile(path string) error {
	file, err := os.Open(path)

	if err != nil {
		return err
	}

	defer file.Close()

	reader := bufio.NewReader(file)

	ids, cls, err := ParseIDs(reader)
	if err != nil {
		return err
	}

	Vendors = ids
	Classes = cls
	LastUpdate = time.Now()
	return nil
}

//go:generate go run regen/regen.go --template regen/load_data.go.tpl -o load_data.go

func init() {
	ids, cls, err := ParseIDs(strings.NewReader(usbIdListData))
	if err != nil {
		log.Printf("usbid: failed to parsers: %s", err)
		return
	}

	Vendors = ids
	Classes = cls
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func FindDevice(vid, pid string) (manufactStr string, productStr string) {
	v := hex2int(vid)
	p := hex2int(pid)

	vendor := Vendors[core.ID(v)]
	if vendor != nil {
		manufactStr = vendor.String()
		product := vendor.Product[core.ID(p)]
		if product != nil {
			productStr = product.String()
		}
	}

	return manufactStr, productStr
}

func hex2int(hex string) int64 {
	value, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		fmt.Printf("Conversion failed: %s\n", err)
	} else {
		return value
	}
	return 0
}
