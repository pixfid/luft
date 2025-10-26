package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Whitelist   string       `mapstructure:"whitelist" yaml:"whitelist"`
	UsbIds      string       `mapstructure:"usbids" yaml:"usbids"`
	LogPath     string       `mapstructure:"log_path" yaml:"log_path"`
	MassStorage bool         `mapstructure:"mass_storage" yaml:"mass_storage"`
	Untrusted   bool         `mapstructure:"untrusted" yaml:"untrusted"`
	CheckWl     bool         `mapstructure:"check_whitelist" yaml:"check_whitelist"`
	Export      ExportConfig `mapstructure:"export" yaml:"export"`
	RemoteHosts []RemoteHost `mapstructure:"remote_hosts" yaml:"remote_hosts"`
}

// ExportConfig represents export configuration
type ExportConfig struct {
	Format string `mapstructure:"format" yaml:"format"`
	Path   string `mapstructure:"path" yaml:"path"`
}

// RemoteHost represents a remote host configuration
type RemoteHost struct {
	Name        string `mapstructure:"name" yaml:"name"`
	IP          string `mapstructure:"ip" yaml:"ip"`
	Port        string `mapstructure:"port" yaml:"port"`
	User        string `mapstructure:"user" yaml:"user"`
	SSHKey      string `mapstructure:"ssh_key" yaml:"ssh_key"`
	Password    string `mapstructure:"password,omitempty" yaml:"password,omitempty"`
	Timeout     int    `mapstructure:"timeout" yaml:"timeout"`
	InsecureSSH bool   `mapstructure:"insecure_ssh" yaml:"insecure_ssh"`
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		UsbIds:      "/var/lib/usbutils/usb.ids",
		LogPath:     "/var/log/",
		MassStorage: false,
		Untrusted:   false,
		CheckWl:     false,
		Export: ExportConfig{
			Format: "pdf",
			Path:   ".",
		},
		RemoteHosts: []RemoteHost{},
	}
}

// Load loads configuration from file and returns merged config
// Priority: CLI flags > Environment variables > Config file > Defaults
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file details
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Search for config in multiple locations
		v.SetConfigName(".luft")
		v.SetConfigType("yaml")

		// Add config search paths
		homeDir, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(homeDir)
		}
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/luft/")
	}

	// Enable environment variables
	v.SetEnvPrefix("LUFT")
	v.AutomaticEnv()

	// Read config file (optional - don't fail if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found - use defaults
	}

	// Unmarshal into config struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand paths
	if cfg.Whitelist != "" {
		cfg.Whitelist = expandPath(cfg.Whitelist)
	}
	if cfg.UsbIds != "" {
		cfg.UsbIds = expandPath(cfg.UsbIds)
	}
	if cfg.Export.Path != "" {
		cfg.Export.Path = expandPath(cfg.Export.Path)
	}
	for i := range cfg.RemoteHosts {
		if cfg.RemoteHosts[i].SSHKey != "" {
			cfg.RemoteHosts[i].SSHKey = expandPath(cfg.RemoteHosts[i].SSHKey)
		}
	}

	return cfg, nil
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	v.SetDefault("usbids", "/var/lib/usbutils/usb.ids")
	v.SetDefault("log_path", "/var/log/")
	v.SetDefault("mass_storage", false)
	v.SetDefault("untrusted", false)
	v.SetDefault("check_whitelist", false)
	v.SetDefault("export.format", "pdf")
	v.SetDefault("export.path", ".")
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[1:])
		}
	}
	return path
}

// GetRemoteHost returns remote host configuration by name
func (c *Config) GetRemoteHost(name string) (*RemoteHost, error) {
	for _, host := range c.RemoteHosts {
		if host.Name == name {
			return &host, nil
		}
	}
	return nil, fmt.Errorf("remote host '%s' not found in configuration", name)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate export format
	validFormats := map[string]bool{"json": true, "xml": true, "pdf": true}
	if c.Export.Format != "" && !validFormats[c.Export.Format] {
		return fmt.Errorf("invalid export format: %s (must be json, xml, or pdf)", c.Export.Format)
	}

	// Validate remote hosts
	for i, host := range c.RemoteHosts {
		if host.Name == "" {
			return fmt.Errorf("remote host #%d: name is required", i)
		}
		if host.IP == "" {
			return fmt.Errorf("remote host '%s': IP is required", host.Name)
		}
		if host.User == "" {
			return fmt.Errorf("remote host '%s': user is required", host.Name)
		}
		if host.SSHKey == "" && host.Password == "" {
			return fmt.Errorf("remote host '%s': either ssh_key or password is required", host.Name)
		}
		if host.Port == "" {
			host.Port = "22"
		}
		if host.Timeout == 0 {
			host.Timeout = 30
		}
	}

	return nil
}
