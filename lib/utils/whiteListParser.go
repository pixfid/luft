package utils

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var whiteList []string

func ParseWL(wlPath string) error {
	file, err := os.Open(wlPath)
	if err != nil {
		return err
	}
	reVid := regexp.MustCompile(`ATTRS{serial}==(.*?$)`)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		result := GetSub(reVid, line, 1)
		if result != "" {
			result = strings.Trim(strings.Split(result, ",")[0], `"`)
			whiteList = append(whiteList, result)
		}
	}
	return nil
}
