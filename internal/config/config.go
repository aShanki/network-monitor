package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	InterfaceName string `mapstructure:"interface"`

	ThresholdMbps float64 `mapstructure:"threshold_mbps"`

	WebhookURL string `mapstructure:"webhook_url"`

	IntervalSeconds int `mapstructure:"interval_seconds"`

	TopN int `mapstructure:"top_n"`

	// Prometheus metrics configuration
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	MetricsPort    string `mapstructure:"metrics_port"`

	ConfigFile string
}

func LoadConfig() (*Config, error) {
	var cfg Config

	viper.SetDefault("interface", "")
	viper.SetDefault("threshold_mbps", 100.0)
	viper.SetDefault("webhook_url", "")
	viper.SetDefault("interval_seconds", 60)
	viper.SetDefault("top_n", 5)
	// Default Prometheus settings
	viper.SetDefault("metrics_enabled", true)
	viper.SetDefault("metrics_port", "9090")

	pflag.StringVar(&cfg.ConfigFile, "config", "", "Path to config file (e.g., config.yaml)")
	pflag.String("interface", viper.GetString("interface"), "Network interface name")
	pflag.Float64("threshold_mbps", viper.GetFloat64("threshold_mbps"), "Speed threshold in Mbps")
	pflag.String("webhook_url", viper.GetString("webhook_url"), "Discord webhook URL")
	pflag.Int("interval_seconds", viper.GetInt("interval_seconds"), "Monitoring interval in seconds")
	pflag.Int("top_n", viper.GetInt("top_n"), "Number of top talkers to report")
	// Add Prometheus flags
	pflag.Bool("metrics_enabled", viper.GetBool("metrics_enabled"), "Enable Prometheus metrics endpoint")
	pflag.String("metrics_port", viper.GetString("metrics_port"), "Port for Prometheus metrics endpoint")

	pflag.VisitAll(func(f *pflag.Flag) {
		viper.BindPFlag(f.Name, f)
	})
	pflag.Parse()

	viper.SetEnvPrefix("NM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if cfg.ConfigFile != "" {
		viper.SetConfigFile(cfg.ConfigFile)
	} else {

		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/network-monitor/")
		viper.AddConfigPath("$HOME/.config/network-monitor")
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			if cfg.ConfigFile != "" {
				return nil, fmt.Errorf("config file specified but not found: %w", err)
			}
		} else {

			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

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

func (c *Config) GetIntervalDuration() time.Duration {
	return time.Duration(c.IntervalSeconds) * time.Second
}
