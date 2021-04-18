package core

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var WHITE_LIST []string

func ParseWL(wlPath string) {
	file, err := os.Open(wlPath)
	if err == nil {
		reVid, _ := regexp.Compile(`ATTRS{serial}==(.*?$)`)
		scanner := bufio.NewScanner(file)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			result := GetSub(reVid, line, 1)
			if result != "" {
				result = strings.Trim(strings.Split(result, ",")[0], `"`)
				WHITE_LIST = append(WHITE_LIST, result)
			}
		}
	}
}
