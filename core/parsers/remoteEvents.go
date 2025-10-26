package parsers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/core/utils"
	"github.com/pixfid/luft/data"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func RemoteEvents(params data.ParseParams) error {
	// Get SSH authentication methods
	authMethods, err := utils.GetSSHAuthMethods(params.SSHKeyPath, params.Password)
	if err != nil {
		return fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Get host key callback
	hostKeyCallback, err := utils.GetHostKeyCallback(params.InsecureSSH)
	if err != nil {
		return fmt.Errorf("failed to setup host key verification (hint: use --insecure-ssh to skip verification, NOT RECOMMENDED): %w", err)
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
		return fmt.Errorf("failed to connect to %s:%s: %w", params.IP, params.Port, err)
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
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer client.Close()

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Starting on: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName(`hostname -f`)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] User login: }}::green {{%s}}::red", time.Now().Format(time.Stamp), hostName(`who | grep " :0" | cut -d " " -f1`)))

	readFile := func(path []string, client *sftp.Client) error {
		var recordTypes []data.LogEvent

		for _, s := range path {
			// Process each file in a separate function to ensure proper resource cleanup
			func(filePath string) {
				if filepath.Ext(filePath) == ".gz" {
					file, err := client.Open(filePath)
					if err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to open file %s: %s}}::yellow", time.Now().Format(time.Stamp), filePath, err.Error()))
						return
					}
					defer file.Close()

					gz, err := gzip.NewReader(file)
					if err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to create gzip reader for %s: %s}}::yellow", time.Now().Format(time.Stamp), filePath, err.Error()))
						return
					}
					defer gz.Close()

					scanner := bufio.NewScanner(gz)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					recordTypes = append(recordTypes, parseLine(scanner)...)

					if err := scanner.Err(); err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: scanner error for %s: %s}}::yellow", time.Now().Format(time.Stamp), filePath, err.Error()))
					}
				} else {
					file, err := client.Open(filePath)
					if err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to open file %s: %s}}::yellow", time.Now().Format(time.Stamp), filePath, err.Error()))
						return
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					recordTypes = append(recordTypes, parseLine(scanner)...)

					if err := scanner.Err(); err != nil {
						_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: scanner error for %s: %s}}::yellow", time.Now().Format(time.Stamp), filePath, err.Error()))
					}
				}
			}(s)
		}

		if len(recordTypes) == 0 {
			return fmt.Errorf("no USB events found in remote log files")
		}

		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d events records}}::green", time.Now().Format(time.Stamp), len(recordTypes)))
		events := CollectEventsData(recordTypes)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsed %d events}}::green", time.Now().Format(time.Stamp), len(events)))
		filteredEvents := utils.FilterEvents(params, events)
		clearEvents := utils.RemoveDuplicates(filteredEvents)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Filter and remove duplicates complete, %d clear events found}}::green", time.Now().Format(time.Stamp), len(clearEvents)))

		if params.Export {
			if err := utils.ExportData(clearEvents, params.Format, params.FileName); err != nil {
				return fmt.Errorf("failed to export events: %w", err)
			}
		} else {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Representation: table}}::green", time.Now().Format(time.Stamp)))
			utils.PrintEvents(clearEvents)
		}

		return nil
	}

	var files []string

	readDir, err := client.ReadDir("/var/log")
	if err != nil {
		return fmt.Errorf("failed to read remote /var/log directory: %w", err)
	}

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

	if len(files) == 0 {
		return fmt.Errorf("no relevant log files found in /var/log on remote host")
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Found %d log files to process}}::green", time.Now().Format(time.Stamp), len(files)))

	if err := readFile(files, client); err != nil {
		return fmt.Errorf("failed to process remote log files: %w", err)
	}

	return nil
}
