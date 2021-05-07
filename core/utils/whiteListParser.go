package utils

import (
	"github.com/i582/cfmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
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

	if re := re; re != nil {
		for _, fields := range re {
			result, err := strconv.ParseBool(fields[2])
			if err != nil {
				_, _ = cfmt.Println(cfmt.Sprintf("{{Error while parse whitelist record}}::red"))
			}
			emitSerial(wl, Serial{
				Serial:     fields[1],
				IsIgnore:   result,
				Commentary: strings.TrimSpace(fields[3]),
			})
		}
	}
	return nil
}
