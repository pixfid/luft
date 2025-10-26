package usbids

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
)

var (
	vendors = map[string]*Vendor{}

	version     = regexp.MustCompile(`Version: (\d{4}.\d{2}.\d{2})`)
	date        = regexp.MustCompile(`Date:\s+(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`)
	vendorLine  = regexp.MustCompile(`^([[:xdigit:]]{4})\s{2}(.+)$`)
	productLine = regexp.MustCompile(`\t([[:xdigit:]]{4})\s{2}(.+)$`)

	Ids     = []string{"/var/core/usbutils/usb.ids", "/usr/share/hwdata/usb.ids", "usb.ids"}
	Version = ""
	Date    = ""
)

type Vendor struct {
	ID      string
	Name    string
	Product map[string]*Product
}

type Product struct {
	ID   string
	Name string
}

func LoadFromFiles() error {
	for _, usbID := range Ids {
		if err := LoadFromFile(usbID); err != nil {
			continue
		}

		return nil
	}
	return nil
}

func ParseUsbIDs(file *os.File) error {
	scanner := bufio.NewScanner(file)

	emitVendor := func(vendors map[string]*Vendor, vendor Vendor) {
		vendors[vendor.ID] = &vendor
	}

	var (
		currVendor *Vendor
		prevVendor *Vendor
	)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.HasPrefix(line, `#`) {
			if result := version.FindStringSubmatch(line); len(result) != 0 {
				Version = result[1]
			}
			if result := date.FindStringSubmatch(line); len(result) != 0 {
				Date = result[1]
			}

			continue
		} else if result := vendorLine.FindStringSubmatch(line); len(result) != 0 {
			if vendor := prevVendor; vendor != nil {
				emitVendor(vendors, *vendor)
			}
			currVendor = &Vendor{
				Name:    result[2],
				ID:      result[1],
				Product: map[string]*Product{},
			}
			prevVendor = currVendor
		} else if result := productLine.FindStringSubmatch(line); len(result) != 0 {
			if currVendor := currVendor; currVendor != nil {
				currVendor.Product[result[1]] = &Product{
					ID:   result[1],
					Name: result[2],
				}
			}
		} else {
			break
		}
	}

	if scanner.Err() != nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{Error while parse usb.ids}}::red"))
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] usb.ids loaded from: %s, Version: %s, Date: %s}}::green",
		time.Now().Format(time.Stamp), file.Name(), Version, Date))

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] usb.ids %d vendors load}}::green", time.Now().Format(time.Stamp),
		len(vendors)))

	return nil
}

func LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return ParseUsbIDs(file)
}

func FindDevice(vid, pid string) (string, string) {
	if vendors := vendors; vendors != nil {
		if vendor := vendors[vid]; vendor != nil {
			if device := vendor.Product[pid]; device != nil {
				return vendor.Name, device.Name
			}

			return vendor.Name, ""
		}

		return "", ""
	}

	return "", ""
}

// ProgressReader tracks download progress
type ProgressReader struct {
	io.Reader
	Total      int64
	Current    int64
	OnProgress func(current, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)

	if pr.OnProgress != nil {
		pr.OnProgress(pr.Current, pr.Total)
	}

	return n, err
}

// UpdateUSBIDs downloads the latest USB IDs database
func UpdateUSBIDs(targetPath string) error {
	// USB IDs sources (in order of preference)
	sources := []string{
		"https://usb-ids.gowly.com/usb.ids",
		"https://raw.githubusercontent.com/gentoo/hwids/master/usb.ids",
		"http://www.linux-usb.org/usb.ids",
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Updating USB IDs database...}}::cyan", time.Now().Format(time.Stamp)))

	var lastErr error
	for _, source := range sources {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Trying source: %s}}::yellow", time.Now().Format(time.Stamp), source))

		err := downloadUSBIDs(source, targetPath)
		if err == nil {
			_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ USB IDs database successfully updated to: %s}}::green", time.Now().Format(time.Stamp), targetPath))

			// Try to load and display version info
			if loadErr := LoadFromFile(targetPath); loadErr == nil {
				_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Database version: %s, Date: %s}}::green", time.Now().Format(time.Stamp), Version, Date))
			}

			return nil
		}

		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Failed: %s}}::red", time.Now().Format(time.Stamp), err.Error()))
		lastErr = err
	}

	return fmt.Errorf("failed to download USB IDs from all sources: %w", lastErr)
}

// downloadUSBIDs downloads USB IDs file from a specific source
func downloadUSBIDs(url, targetPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Make request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Get content length for progress tracking
	contentLength := resp.ContentLength

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "usb.ids.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	// Create progress reader
	var lastProgress int
	progressReader := &ProgressReader{
		Reader: resp.Body,
		Total:  contentLength,
		OnProgress: func(current, total int64) {
			if total > 0 {
				progress := int(float64(current) / float64(total) * 100)
				// Update every 5%
				if progress-lastProgress >= 5 || progress == 100 {
					_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Downloaded: %d%%}}::cyan",
						time.Now().Format(time.Stamp), progress))
					lastProgress = progress
				}
			}
		},
	}

	// Download to temp file
	_, err = io.Copy(tmpFile, progressReader)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Move temp file to target location
	if err := os.Rename(tmpPath, targetPath); err != nil {
		// If rename fails (cross-device), try copy
		if copyErr := copyFile(tmpPath, targetPath); copyErr != nil {
			return fmt.Errorf("failed to move file to target: %w", copyErr)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
