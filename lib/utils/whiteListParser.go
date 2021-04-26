package utils

import (
	"github.com/i582/cfmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

var (
	whiteListRegex = regexp.MustCompile(`\S+"(?P<serial>.*)"\S+"(?P<flag>.*)"\s#(?P<comment>.*)`)
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

	return parseWhiteList(string(content), whiteListRegex)
}

func parseWhiteList(fileContents string, whiteListRegex *regexp.Regexp) error {

	emitSerial := func(wl map[string]*Serial, serial Serial) {
		wl[serial.Serial] = &serial
	}

	re := whiteListRegex.FindAllStringSubmatch(fileContents, -1)

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
