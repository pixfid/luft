package cmd

import (
	"fmt"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/usbids"
	"github.com/spf13/cobra"
)

var (
	cacheUSBIDsPath string
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage USB IDs cache",
	Long: `Manage the USB IDs database cache.

LUFT automatically caches parsed USB IDs for faster loading (2-3x speedup).
This command allows you to manually clear the cache if needed.

Cache files are stored alongside the USB IDs file with '.cache' extension.
The cache is automatically invalidated when the source file is modified.

Performance:
  - First load (parsing): ~13ms
  - Cached loads: ~5ms (2-3x faster!)

Examples:
  # Clear cache for default USB IDs file
  luft cache clear

  # Clear cache for custom USB IDs file
  luft cache clear --usbids ~/.local/share/luft/usb.ids`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear USB IDs cache",
	Long:  `Clear the cached USB IDs database. Cache will be rebuilt on next use.`,
	RunE:  runCacheClear,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	cacheClearCmd.Flags().StringVar(&cacheUSBIDsPath, "usbids", "/var/lib/usbutils/usb.ids", "USB IDs file path")
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Cache Clear Mode}}::cyan|bold", time.Now().Format(time.Stamp)))

	if err := usbids.ClearCache(cacheUSBIDsPath); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] ✓ Cache cleared successfully!}}::green|bold", time.Now().Format(time.Stamp)))
	_, _ = cfmt.Println(cfmt.Sprintf("{{[%v] Next load will parse from source and rebuild cache}}::cyan", time.Now().Format(time.Stamp)))

	return nil
}
