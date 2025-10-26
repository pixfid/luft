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
	// Get SSH authentication methods
	authMethods, err := utils.GetSSHAuthMethods(params.SSHKeyPath, params.Password)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to setup authentication: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		return
	}

	// Get host key callback
	hostKeyCallback, err := utils.GetHostKeyCallback(params.InsecureSSH)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to setup host key verification: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Hint: Use --insecure-ssh to skip verification (NOT RECOMMENDED)}}::yellow", time.Now().Format(time.Stamp)))
		return
	}

	config := &ssh.ClientConfig{
		User:            params.Login,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         time.Duration(params.SSHTimeout) * time.Second,
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Connecting to %s:%s with timeout %ds...}}::green",
		time.Now().Format(time.Stamp), params.IP, params.Port, params.SSHTimeout))

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", params.IP, params.Port), config)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to dial: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		return
	}
	defer conn.Close()

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Successfully connected to remote host}}::green", time.Now().Format(time.Stamp)))

	hostName := func(cmd string) string {
		session, err := conn.NewSession()
		if err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to create SSH session: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
			return "unknown"
		}
		defer session.Close()

		var stdoutBuf bytes.Buffer
		session.Stdout = &stdoutBuf
		err = session.Run(cmd)
		if err != nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to exec command '%s': %s}}::red", time.Now().Format(time.Stamp), cmd, err.Error()))
			return "unknown"
		}
		return strings.TrimSuffix(stdoutBuf.String(), "\n")
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to create SFTP client: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		return
	}
	defer client.Close()

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Starting on: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName(`hostname -f`)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] User login: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName(`who | grep " :0" | cut -d " " -f1`)))

	readFile := func(path []string, client *sftp.Client) {

		var recordTypes []data.LogEvent

		for _, s := range path {
			// Process each file in a separate function to ensure proper resource cleanup
			func(filePath string) {
				if filepath.Ext(filePath) == ".gz" {
					file, err := client.Open(filePath)
					if err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to open file %s: %s}}::red", time.Now().Format(time.Stamp), filePath, err.Error()))
						return
					}
					defer file.Close()

					gz, err := gzip.NewReader(file)
					if err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to create gzip reader for %s: %s}}::red", time.Now().Format(time.Stamp), filePath, err.Error()))
						return
					}
					defer gz.Close()

					scanner := bufio.NewScanner(gz)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					recordTypes = append(recordTypes, parseLine(scanner)...)

					if err := scanner.Err(); err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Scanner error for %s: %s}}::red", time.Now().Format(time.Stamp), filePath, err.Error()))
					}
				} else {
					file, err := client.Open(filePath)
					if err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed to open file %s: %s}}::red", time.Now().Format(time.Stamp), filePath, err.Error()))
						return
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					recordTypes = append(recordTypes, parseLine(scanner)...)

					if err := scanner.Err(); err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Scanner error for %s: %s}}::red", time.Now().Format(time.Stamp), filePath, err.Error()))
					}
				}
			}(s)
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
