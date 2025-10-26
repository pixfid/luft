package utils

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
)

var (
	udevRulesRegex = regexp.MustCompile(`\S+"(?P<serial>.*)"\S+"(?P<flag>.*)"\s#(?P<comment>.*)`)
	wl             = map[string]*Serial{}
)

type Serial struct {
	Serial     string
	IsIgnore   bool
	Commentary string
}

func IsInWhiteList(serial string) bool {
	return wl[serial] != nil
}

func WhiteListSerialInfo(serial string) *Serial {
	return wl[serial]
}

func LoadWhiteList(wlPath string) error {
	content, err := ioutil.ReadFile(wlPath)
	if err != nil {
		return err
	}

	return udevWhiteListParser(content)
}

func emitSerial(wl map[string]*Serial, serial Serial) {
	wl[serial.Serial] = &serial
}

func udevWhiteListParser(fileData []byte) error {
	content := string(fileData)

	re := udevRulesRegex.FindAllStringSubmatch(content, -1)

	if re == nil || len(re) == 0 {
		return fmt.Errorf("no valid whitelist entries found in file")
	}

	successCount := 0
	for i, fields := range re {
		if len(fields) < 4 {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: skipping malformed whitelist entry %d}}::yellow", time.Now().Format(time.Stamp), i+1))
			continue
		}

		result, err := strconv.ParseBool(fields[2])
		if err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: invalid boolean value in whitelist entry %d (serial: %s), defaulting to false}}::yellow", time.Now().Format(time.Stamp), i+1, fields[1]))
			result = false
		}

		emitSerial(wl, Serial{
			Serial:     fields[1],
			IsIgnore:   result,
			Commentary: strings.TrimSpace(fields[3]),
		})
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("failed to parse any valid whitelist entries")
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Loaded %d whitelist entries}}::green", time.Now().Format(time.Stamp), successCount))
	return nil
}
