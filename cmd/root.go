package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	"github.com/pixfid/luft/config"
	"github.com/spf13/cobra"
)

const (
	version = "v1.0"
	url     = "https://github.com/pixfid/luft"
)

var (
	// Global flags
	cfgFile      string
	configLoaded *config.Config

	// Root context for graceful shutdown
	rootCtx    context.Context
	cancelFunc context.CancelFunc
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "luft",
	Short: "Linux USB Forensic Tool",
	Long: `LUFT - Linux USB Forensic Tool

A forensic tool for analyzing USB device connection history on Linux systems.
Supports local and remote log analysis, USB device whitelisting, and various export formats.`,
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Setup signal handler for all commands
		rootCtx, cancelFunc = setupSignalHandler()

		// Print banner for non-help commands
		if cmd.Name() != "help" && cmd.Name() != "completion" {
			printBanner()
		}

		// Load configuration file
		var err error
		configLoaded, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if cancelFunc != nil {
			cancelFunc()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.luft.yaml)")

	// Set custom version template
	rootCmd.SetVersionTemplate(`{{.Version}}
`)
}

// setupSignalHandler creates a context that will be cancelled on SIGINT or SIGTERM
func setupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		_, _ = cfmt.Println(cfmt.Sprintf("\n{{[%v] Received signal: %v - initiating graceful shutdown...}}::yellow|bold",
			time.Now().Format(time.Stamp), sig))
		cancel()

		// Force exit if second signal received
		sig = <-sigChan
		_, _ = cfmt.Println(cfmt.Sprintf("\n{{[%v] Received second signal: %v - forcing immediate exit}}::red|bold",
			time.Now().Format(time.Stamp), sig))
		os.Exit(1)
	}()

	return ctx, cancel
}

func printBanner() {
	_, _ = cfmt.Println(cfmt.Sprintf(`
{{┬  ┬ ┬┌─┐┌┬┐}}::bgLightRed
{{│  │ │├┤  │ }}::bgLightRed {{Linux Usb Forensic Tool %s}}::lightYellow
{{┴─┘└─┘└   ┴ }}::bgLightRed {{%s}}::lightBlue`, version, url))
	_, _ = cfmt.Println(cfmt.Sprintf("[*] Starting at: %v", time.Now().Format(time.Stamp)))
}
