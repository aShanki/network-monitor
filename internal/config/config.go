package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	// InterfaceName is the network interface to capture packets from.
	InterfaceName string `mapstructure:"interface"`
	// ThresholdMbps is the network speed threshold in Mbps.
	ThresholdMbps float64 `mapstructure:"threshold_mbps"`
	// WebhookURL is the Discord webhook URL to send notifications.
	WebhookURL string `mapstructure:"webhook_url"`
	// IntervalSeconds is the time interval in seconds for checking the speed.
	IntervalSeconds int `mapstructure:"interval_seconds"`
	// TopN is the number of top IP addresses to report.
	TopN int `mapstructure:"top_n"`
	// ConfigFile is the path to the configuration file.
	ConfigFile string
}

// LoadConfig reads configuration from file, environment variables, and flags.
func LoadConfig() (*Config, error) {
	var cfg Config

	// --- Defaults ---
	viper.SetDefault("interface", "") // Default: Let pcap find the first available non-loopback interface
	viper.SetDefault("threshold_mbps", 100.0)
	viper.SetDefault("webhook_url", "")
	viper.SetDefault("interval_seconds", 60)
	viper.SetDefault("top_n", 5)

	// --- Flags ---
	pflag.StringVar(&cfg.ConfigFile, "config", "", "Path to config file (e.g., config.yaml)")
	pflag.String("interface", viper.GetString("interface"), "Network interface name")
	pflag.Float64("threshold_mbps", viper.GetFloat64("threshold_mbps"), "Speed threshold in Mbps")
	pflag.String("webhook_url", viper.GetString("webhook_url"), "Discord webhook URL")
	pflag.Int("interval_seconds", viper.GetInt("interval_seconds"), "Monitoring interval in seconds")
	pflag.Int("top_n", viper.GetInt("top_n"), "Number of top talkers to report")

	// Bind flags to Viper keys
	pflag.VisitAll(func(f *pflag.Flag) {
		viper.BindPFlag(f.Name, f)
	})
	pflag.Parse()

	// --- Environment Variables ---
	viper.SetEnvPrefix("NM") // Network Monitor
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // Read in environment variables that match

	// --- Config File ---
	if cfg.ConfigFile != "" {
		viper.SetConfigFile(cfg.ConfigFile)
	} else {
		// Default search paths
		viper.SetConfigName("config")                        // Name of config file (without extension)
		viper.SetConfigType("yaml")                          // REQUIRED if the config file does not have the extension in the name
		viper.AddConfigPath("/etc/network-monitor/")         // Path to look for the config file in
		viper.AddConfigPath("$HOME/.config/network-monitor") // Call multiple times to add many search paths
		viper.AddConfigPath(".")                             // Optionally look for config in the working directory
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if not explicitly specified via flag
			if cfg.ConfigFile != "" {
				return nil, fmt.Errorf("config file specified but not found: %w", err)
			}
		} else {
			// Config file was found but another error was produced
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// --- Unmarshal ---
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// --- Validation (Basic) ---
	if cfg.WebhookURL == "" {
		fmt.Println("Warning: Discord webhook URL is not set. Notifications will not be sent.")
	}
	if cfg.IntervalSeconds <= 0 {
		return nil, fmt.Errorf("interval_seconds must be positive")
	}
	if cfg.TopN <= 0 {
		return nil, fmt.Errorf("top_n must be positive")
	}
	if cfg.ThresholdMbps <= 0 {
		return nil, fmt.Errorf("threshold_mbps must be positive")
	}

	return &cfg, nil
}

// GetIntervalDuration converts the interval seconds to time.Duration.
func (c *Config) GetIntervalDuration() time.Duration {
	return time.Duration(c.IntervalSeconds) * time.Second
}
