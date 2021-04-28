package parsers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/i582/cfmt"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"path/filepath"
	"strings"
	"time"
)

func RemoteEvents(params data.ParseParams) {
	config := &ssh.ClientConfig{
		User: params.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(params.Password),
		},
		HostKeyCallback: utils.TrustedHostKeyCallback(""),
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", params.IP, params.Port), config)
	if err != nil {
		_, _ = cfmt.Println("{{Failed to dial}}::red")
		return
	}

	hostName := func(cmd string) string {
		session, _ := conn.NewSession()
		defer session.Close()

		var stdoutBuf bytes.Buffer
		session.Stdout = &stdoutBuf
		err := session.Run(cmd)
		if err != nil {
			_, _ = cfmt.Println("{{Failed to exec command}}::red")
		}
		return strings.TrimSuffix(stdoutBuf.String(), "\n")
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		_, _ = cfmt.Println("{{Failed to create client}}::red")
	}
	// Close connection
	defer client.Close()

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Starting on: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName(`hostname -f`)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] User login: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName(`who | grep " :0" | cut -d " " -f1`)))

	readFile := func(path []string, client *sftp.Client) {

		var recordTypes []data.LogEvent
		var scanner *bufio.Scanner

		for _, s := range path {
			if filepath.Ext(s) == ".gz" {
				file, err := client.Open(s)

				if err != nil {
					_, _ = cfmt.Println("{{Failed to open file}}::red")
				}

				gz, err := gzip.NewReader(file)

				if err != nil {
					_, _ = cfmt.Println("{{Failed to create Reader}}::red")
				}

				defer file.Close()
				defer gz.Close()

				scanner = bufio.NewScanner(gz)
			} else {
				file, err := client.Open(s)
				if err != nil {
					_, _ = cfmt.Println("{{Failed to open file}}::red")
				}

				defer file.Close()
				scanner = bufio.NewScanner(file)
			}
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)

			recordTypes = append(recordTypes, parseLine(scanner)...)

			if err := scanner.Err(); err != nil {
				_, _ = cfmt.Println(cfmt.Sprintf("{{Failed to %s }}::red", err.Error()))
			}
		}

		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d events records}}::green", time.Now().Format(time.Stamp), len(recordTypes)))
		events := CollectEventsData(recordTypes)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsed %d events}}::green", time.Now().Format(time.Stamp), len(events)))
		filteredEvents := utils.FilterEvents(params, events)
		clearEvents := utils.RemoveDuplicates(filteredEvents)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter and remove duplicates complete, %d clear events found}}::green", time.Now().Format(time.Stamp), len(clearEvents)))

		if params.Export {
			utils.ExportData(clearEvents, params.Format)
		} else {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: table}}::green", time.Now().Format(time.Stamp)))
			utils.PrintEvents(clearEvents)
		}
	}

	var files []string

	readDir, _ := client.ReadDir("/var/log")
	for _, fileInfo := range readDir {
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
