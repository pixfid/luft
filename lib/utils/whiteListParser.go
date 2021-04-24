package utils

import (
	"io/ioutil"
	"log"
	"regexp"
)

var WhiteList WhiteListCache

type WhiteListCache struct {
	cache []string
}

func (c *WhiteListCache) Add(serial string) {
	c.cache = append(c.cache, serial)
}

func (c *WhiteListCache) Has(serial string) bool {
	for _, innerSerial := range c.cache {
		if innerSerial == serial {
			return true
		}
	}

	return false
}

func ParseWL(wlPath string) error {
	cache := WhiteListCache{}
	reVid := regexp.MustCompile(`ATTRS{serial}=="(.*?)"`)
	content, err := ioutil.ReadFile(wlPath)
	if err != nil {
		log.Fatal(err)
	}
	result := GetSubs(reVid, string(content), 1)
	for i := range result {
		cache.Add(result[i][1])
	}

	WhiteList = cache
	return err
}
