package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/usbids"
	"github.com/spf13/cobra"
)

var (
	updateTarget string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update USB IDs database",
	Long: `Download and update the USB IDs database from the official source.

The update command downloads from:
  http://www.linux-usb.org/usb.ids

A progress bar shows download status. After download, the database
is verified by loading it and displaying version information.

Examples:
  # Update to default location (requires sudo for system paths)
  sudo luft update

  # Update to custom location
  luft update --path ~/.local/share/luft/usb.ids

  # Update and use the new database
  luft update --path ~/usb.ids
  luft events --source local --usbids ~/usb.ids`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&updateTarget, "path", "/var/lib/usbutils/usb.ids", "target path for USB IDs file")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] USB IDs Update Mode}}::cyan|bold", time.Now().Format(time.Stamp)))

	// Check if target path is writable
	if !isWritable(updateTarget) {
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Warning: %s is not writable, using alternative...}}::yellow",
			time.Now().Format(time.Stamp), updateTarget))

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine writable location: %w", err)
		}

		updateTarget = filepath.Join(homeDir, ".local", "share", "luft", "usb.ids")
		_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Using alternative path: %s}}::cyan", time.Now().Format(time.Stamp), updateTarget))
	}

	// Update USB IDs
	if err := usbids.UpdateUSBIDs(updateTarget); err != nil {
		return fmt.Errorf("failed to update USB IDs: %w", err)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Update completed successfully!}}::green|bold", time.Now().Format(time.Stamp)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] To use this database: --usbids=%s}}::cyan", time.Now().Format(time.Stamp), updateTarget))

	return nil
}

func isWritable(path string) bool {
	info, err := os.Stat(path)
	if err == nil {
		// File exists, try to open for writing
		testFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return false
		}
		testFile.Close()
		return true
	}

	// File doesn't exist, check if directory is writable
	dir := path
	if info == nil || !info.IsDir() {
		dir = filepath.Dir(path)
	}

	// Try to create a temp file in the directory
	testFile, err := os.CreateTemp(dir, ".luft-write-test-*")
	if err != nil {
		return false
	}
	testFile.Close()
	os.Remove(testFile.Name())
	return true
}
