package usbids

import (
	"bufio"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/schollz/progressbar/v3"
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

// CacheData represents cached USB IDs data
type CacheData struct {
	Vendors     map[string]*Vendor
	Version     string
	Date        string
	SourceHash  string    // MD5 hash of source file
	CachedAt    time.Time // When cache was created
	SourceMTime time.Time // Modification time of source file
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
	startTime := time.Now()

	// Try to load from cache first
	if cached, err := loadFromCache(path); err == nil && cached {
		duration := time.Since(startTime)
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ⚡ Loaded from cache in %v (fast!)}}::green",
			time.Now().Format(time.Stamp), duration))
		return nil
	}

	// Cache miss or invalid, parse from source
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Parsing USB IDs from source...}}::yellow",
		time.Now().Format(time.Stamp)))

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := ParseUsbIDs(file); err != nil {
		return err
	}

	// Save to cache for next time
	if err := saveToCache(path); err != nil {
		// Non-fatal error, just log it
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: failed to save cache: %s}}::yellow",
			time.Now().Format(time.Stamp), err.Error()))
	} else {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Cache saved for faster next load}}::cyan",
			time.Now().Format(time.Stamp)))
	}

	duration := time.Since(startTime)
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Total load time: %v}}::cyan",
		time.Now().Format(time.Stamp), duration))

	return nil
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

// getCachePath returns the cache file path for a given USB IDs file
func getCachePath(sourcePath string) string {
	return sourcePath + ".cache"
}

// getFileHash calculates MD5 hash of a file (first 1MB only for performance)
func getFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	// Only hash first 1MB for performance
	_, err = io.CopyN(hash, file, 1024*1024)
	if err != nil && err != io.EOF {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// loadFromCache attempts to load cached USB IDs data
func loadFromCache(sourcePath string) (bool, error) {
	cachePath := getCachePath(sourcePath)

	// Check if cache file exists
	cacheInfo, err := os.Stat(cachePath)
	if err != nil {
		return false, err
	}

	// Check if source file exists and get its info
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return false, err
	}

	// If source is newer than cache, invalidate cache
	if sourceInfo.ModTime().After(cacheInfo.ModTime()) {
		return false, fmt.Errorf("cache outdated: source modified at %v, cache at %v",
			sourceInfo.ModTime(), cacheInfo.ModTime())
	}

	// Load cache file
	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return false, err
	}
	defer cacheFile.Close()

	// Decode cache data
	var cache CacheData
	decoder := gob.NewDecoder(cacheFile)
	if err := decoder.Decode(&cache); err != nil {
		return false, fmt.Errorf("failed to decode cache: %w", err)
	}

	// Verify source file hash matches
	currentHash, err := getFileHash(sourcePath)
	if err != nil {
		return false, err
	}

	if currentHash != cache.SourceHash {
		return false, fmt.Errorf("cache hash mismatch: expected %s, got %s", cache.SourceHash, currentHash)
	}

	// Cache is valid, load data into global variables
	vendors = cache.Vendors
	Version = cache.Version
	Date = cache.Date

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] usb.ids loaded from cache: %s, Version: %s, Date: %s}}::green",
		time.Now().Format(time.Stamp), cachePath, Version, Date))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] usb.ids %d vendors loaded}}::green", time.Now().Format(time.Stamp),
		len(vendors)))

	return true, nil
}

// saveToCache saves current USB IDs data to cache
func saveToCache(sourcePath string) error {
	cachePath := getCachePath(sourcePath)

	// Get source file info
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// Calculate source file hash
	sourceHash, err := getFileHash(sourcePath)
	if err != nil {
		return err
	}

	// Create cache data
	cache := CacheData{
		Vendors:     vendors,
		Version:     Version,
		Date:        Date,
		SourceHash:  sourceHash,
		CachedAt:    time.Now(),
		SourceMTime: sourceInfo.ModTime(),
	}

	// Create cache file
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer cacheFile.Close()

	// Encode cache data
	encoder := gob.NewEncoder(cacheFile)
	if err := encoder.Encode(&cache); err != nil {
		return fmt.Errorf("failed to encode cache: %w", err)
	}

	return nil
}

// ClearCache removes cache file for a given USB IDs file
func ClearCache(sourcePath string) error {
	cachePath := getCachePath(sourcePath)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cache cleared: %s}}::green", time.Now().Format(time.Stamp), cachePath))
	return nil
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
	// Official USB IDs source
	source := "http://www.linux-usb.org/usb.ids"

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Updating USB IDs database...}}::cyan", time.Now().Format(time.Stamp)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Downloading from: %s}}::yellow", time.Now().Format(time.Stamp), source))

	err := downloadUSBIDs(source, targetPath)
	if err != nil {
		return fmt.Errorf("failed to download USB IDs: %w", err)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ USB IDs database successfully updated to: %s}}::green", time.Now().Format(time.Stamp), targetPath))

	// Try to load and display version info
	if loadErr := LoadFromFile(targetPath); loadErr == nil {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Database version: %s, Date: %s}}::green", time.Now().Format(time.Stamp), Version, Date))
	}

	return nil
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

	// Create progress bar for download
	bar := progressbar.DefaultBytes(
		contentLength,
		"Downloading USB IDs",
	)

	// Download to temp file with progress bar
	_, err = io.Copy(io.MultiWriter(tmpFile, bar), resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	fmt.Println() // New line after progress bar

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
