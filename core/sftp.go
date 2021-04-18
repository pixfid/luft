package core

import (
	"bufio"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/i582/cfmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func GetRemoteLogs(server, port, login, pass string) {
	config := &ssh.ClientConfig{
		User: login,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: trustedHostKeyCallback(""),
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", server, port), config)
	if err != nil {
		_, _ = cfmt.Println("{{Failed to dial!}}::red")
		return
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		_, _ = cfmt.Println("{{Failed to create client!}}::red")
	}
	// Close connection
	defer client.Close()

	readFile := func(path []string, client *sftp.Client) {

		var recordTypes []RecordType
		var scanner *bufio.Scanner

		parseLine := func(line string) {
			r, _ := regexp.Compile(`(?:]|:) usb (.*?): `)
			u, _ := regexp.Compile(`(?:]|:) usb-storage (.*?): `)

			if r.MatchString(line) || u.MatchString(line) {
				toTime := TimeStampToTime(line[:15])
				eventType := GetActionType(line)
				if eventType != "" {
					recordTypes = append(recordTypes, RecordType{
						Date:       toTime,
						ActionType: eventType,
						LogLine:    line,
					})
				}
			}
		}

		for _, s := range path {
			if filepath.Ext(s) == ".gz" {
				file, err := client.Open(s)

				if err != nil {
					log.Fatal(err)
				}

				gz, err := gzip.NewReader(file)

				if err != nil {
					log.Fatal(err)
				}

				defer file.Close()
				defer gz.Close()

				scanner = bufio.NewScanner(gz)
			} else {
				file, err := client.Open(s)
				if err != nil {
					log.Fatal(err)
				}

				defer file.Close()
				scanner = bufio.NewScanner(file)
			}
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)

			for scanner.Scan() {
				parseLine(scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}

		}
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d events records}}::green", time.Now().Format(time.Stamp), len(recordTypes)))
		events := ParseHistory(recordTypes)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsed %d events}}::green", time.Now().Format(time.Stamp), len(events)))
		filteredEvents := FilterEvents(events)
		clearEvents := RemoveDuplicates(filteredEvents)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter and remove duplicates complete, %d clear events found}}::green", time.Now().Format(time.Stamp), len(clearEvents)))

		if EXPORT {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: %s }}::green", time.Now().Format(time.Stamp), FORMAT))
			var data []byte
			var fn string
			switch FORMAT {
			case "json":
				fn = fmt.Sprintf("events_data.%s", "json")
				data, _ = json.MarshalIndent(clearEvents, "", " ")
			case "xml":
				fn = fmt.Sprintf("events_data.%s", "xml")
				data, _ = xml.MarshalIndent(clearEvents, "", " ")
			case "pdf":
				GenerateReport(clearEvents, fmt.Sprintf("events_data.%s", "pdf"))
			}
			if data != nil {
				err := ioutil.WriteFile(fn, data, fs.ModePerm)
				if err != nil {
					_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Events exported to: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
				}
			}
		} else {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: table}}::green", time.Now().Format(time.Stamp)))
			PrintEvents(clearEvents)
		}

	}

	var files []string

	d, _ := client.ReadDir("/var/log")
	for _, fileInfo := range d {
		if !fileInfo.IsDir() {
			if strings.Contains(fileInfo.Name(), "syslog") {
				files = append(files, fmt.Sprintf("/var/log/%s", fileInfo.Name()))
			} else if strings.Contains(fileInfo.Name(), "messages") {
				files = append(files, fmt.Sprintf("/var/log/%s", fileInfo.Name()))
			} else if strings.Contains(fileInfo.Name(), "kern") {
				files = append(files, fmt.Sprintf("/var/log/%s", fileInfo.Name()))
			} else if strings.Contains(fileInfo.Name(), "daemon") {
				files = append(files, fmt.Sprintf("/var/log/%s", fileInfo.Name()))
			}
		}
	}
	readFile(files, client)
}

func trustedHostKeyCallback(trustedKey string) ssh.HostKeyCallback {

	if trustedKey == "" {
		return func(_ string, _ net.Addr, k ssh.PublicKey) error {
			//log.Printf("WARNING: SSH-key verification is *NOT* in effect: to fix, add this trustedKey: %q", keyString(k))
			return nil
		}
	}

	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
		ks := keyString(k)
		if trustedKey != ks {
			return fmt.Errorf("SSH-key verification: expected %q but got %q", trustedKey, ks)
		}

		return nil
	}
}

func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal()) // e.g. "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...."
}
