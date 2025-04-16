package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to reset Viper and pflag state between tests
func resetViper() {
	viper.Reset()
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError) // Reset pflag
}

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err, "Failed to write temp config file")
	return tmpFile
}

// TestLoadConfigDefaults tests loading configuration with default values.
func TestLoadConfigDefaults(t *testing.T) {
	resetViper()
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "", cfg.InterfaceName) // Default interface
	assert.Equal(t, 100.0, cfg.ThresholdMbps)
	assert.Equal(t, "", cfg.WebhookURL)
	assert.Equal(t, 60, cfg.IntervalSeconds)
	assert.Equal(t, 5, cfg.TopN)
	assert.Equal(t, time.Duration(60)*time.Second, cfg.GetIntervalDuration())
}

// TestLoadConfigFromFile tests loading configuration primarily from a file.
func TestLoadConfigFromFile(t *testing.T) {
	resetViper()
	configFileContent := `
interface: "eth_test"
threshold_mbps: 55.5
webhook_url: "http://test.hook"
interval_seconds: 30
top_n: 3
`
	configFile := createTempConfigFile(t, configFileContent)

	// Set the config flag
	pflag.Set("config", configFile)

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, configFile, cfg.ConfigFile)
	assert.Equal(t, "eth_test", cfg.InterfaceName)
	assert.Equal(t, 55.5, cfg.ThresholdMbps)
	assert.Equal(t, "http://test.hook", cfg.WebhookURL)
	assert.Equal(t, 30, cfg.IntervalSeconds)
	assert.Equal(t, 3, cfg.TopN)
	assert.Equal(t, time.Duration(30)*time.Second, cfg.GetIntervalDuration())
}

// TestLoadConfigEnvVars tests loading configuration primarily from environment variables.
func TestLoadConfigEnvVars(t *testing.T) {
	resetViper()

	t.Setenv("NM_INTERFACE", "env_iface")
	t.Setenv("NM_THRESHOLD_MBPS", "123.4")
	t.Setenv("NM_WEBHOOK_URL", "http://env.hook")
	t.Setenv("NM_INTERVAL_SECONDS", "15")
	t.Setenv("NM_TOP_N", "10")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "env_iface", cfg.InterfaceName)
	assert.Equal(t, 123.4, cfg.ThresholdMbps)
	assert.Equal(t, "http://env.hook", cfg.WebhookURL)
	assert.Equal(t, 15, cfg.IntervalSeconds)
	assert.Equal(t, 10, cfg.TopN)
}

// TestLoadConfigFlags tests loading configuration primarily from command-line flags.
func TestLoadConfigFlags(t *testing.T) {
	resetViper()

	// Simulate setting flags (we don't actually parse os.Args, Viper does it internally)
	pflag.String("interface", "", "")
	pflag.Float64("threshold_mbps", 0, "")
	pflag.String("webhook_url", "", "")
	pflag.Int("interval_seconds", 0, "")
	pflag.Int("top_n", 0, "")

	pflag.Set("interface", "flag_iface")
	pflag.Set("threshold_mbps", "99.9")
	pflag.Set("webhook_url", "http://flag.hook")
	pflag.Set("interval_seconds", "5")
	pflag.Set("top_n", "2")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "flag_iface", cfg.InterfaceName)
	assert.Equal(t, 99.9, cfg.ThresholdMbps)
	assert.Equal(t, "http://flag.hook", cfg.WebhookURL)
	assert.Equal(t, 5, cfg.IntervalSeconds)
	assert.Equal(t, 2, cfg.TopN)
}

// TestLoadConfigPrecedence tests the precedence order (Flag > Env > File > Default).
func TestLoadConfigPrecedence(t *testing.T) {
	resetViper()

	// 1. Default values are set internally

	// 2. File values
	configFileContent := `
interface: "file_iface"
threshold_mbps: 50.0
webhook_url: "http://file.hook"
interval_seconds: 600
top_n: 50
`
	configFile := createTempConfigFile(t, configFileContent)
	pflag.Set("config", configFile) // Point to the config file

	// 3. Environment variables (should override file)
	t.Setenv("NM_INTERFACE", "env_iface")
	t.Setenv("NM_THRESHOLD_MBPS", "123.4")
	// Don't set webhook_url or interval_seconds env, file value should persist
	t.Setenv("NM_TOP_N", "10")

	// 4. Flags (should override env and file)
	pflag.String("interface", "", "")
	pflag.Float64("threshold_mbps", 0, "")
	pflag.String("webhook_url", "", "")
	pflag.Int("interval_seconds", 0, "")
	pflag.Int("top_n", 0, "")

	pflag.Set("interface", "flag_iface")
	// Don't set threshold_mbps flag, env value should persist
	pflag.Set("webhook_url", "http://flag.hook") // Should override file value
	// Don't set interval_seconds flag, file value should persist
	// Don't set top_n flag, env value should persist

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "flag_iface", cfg.InterfaceName)    // Flag wins
	assert.Equal(t, 123.4, cfg.ThresholdMbps)           // Env wins
	assert.Equal(t, "http://flag.hook", cfg.WebhookURL) // Flag wins
	assert.Equal(t, 600, cfg.IntervalSeconds)           // File wins
	assert.Equal(t, 10, cfg.TopN)                       // Env wins
}

// TestLoadConfigValidation tests the validation rules in LoadConfig.
func TestLoadConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		envVars     map[string]string
		flags       map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid default config",
			expectError: false,
		},
		{
			name:        "Invalid interval_seconds (zero)",
			envVars:     map[string]string{"NM_INTERVAL_SECONDS": "0"},
			expectError: true,
			errorMsg:    "interval_seconds must be positive",
		},
		{
			name:        "Invalid interval_seconds (negative)",
			flags:       map[string]string{"interval_seconds": "-10"},
			expectError: true,
			errorMsg:    "interval_seconds must be positive",
		},
		{
			name:        "Invalid top_n (zero)",
			envVars:     map[string]string{"NM_TOP_N": "0"},
			expectError: true,
			errorMsg:    "top_n must be positive",
		},
		{
			name:        "Invalid threshold_mbps (zero)",
			flags:       map[string]string{"threshold_mbps": "0.0"},
			expectError: true,
			errorMsg:    "threshold_mbps must be positive",
		},
		{
			name:        "Invalid threshold_mbps (negative)",
			envVars:     map[string]string{"NM_THRESHOLD_MBPS": "-50.5"},
			expectError: true,
			errorMsg:    "threshold_mbps must be positive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetViper()

			// Set environment variables
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			// Define and set flags
			pflag.String("interface", "", "")
			pflag.Float64("threshold_mbps", 0, "")
			pflag.String("webhook_url", "", "")
			pflag.Int("interval_seconds", 0, "")
			pflag.Int("top_n", 0, "")
			for k, v := range tc.flags {
				err := pflag.Set(k, v)
				require.NoError(t, err, "Failed to set flag %s=%s", k, v)
			}

			_, err := LoadConfig()

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLoadConfigFileNotFound tests behavior when a specified config file doesn't exist.
func TestLoadConfigFileNotFound(t *testing.T) {
	resetViper()

	// Set the config flag to a non-existent file
	pflag.Set("config", "non_existent_config.yaml")

	_, err := LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config file specified but not found")
}

// TestLoadConfigBadFileFormat tests behavior with an invalid config file format.
func TestLoadConfigBadFileFormat(t *testing.T) {
	resetViper()
	configFileContent := `this: is: not: valid: yaml`
	configFile := createTempConfigFile(t, configFileContent)

	// Set the config flag
	pflag.Set("config", configFile)

	_, err := LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}
