package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath" // Keep for potential DefaultShell logic later
	"sync"

	"github.com/localrivet/gomcp/server"
)

// Path to the configuration file relative to the server executable
const configDir = "config"
const configFileName = "config.json"

// Configuration struct to match config.json
type ServerConfig struct {
	BlockedCommands    []string `json:"blockedCommands"`
	DefaultShell       *string  `json:"defaultShell,omitempty"`       // Pointer to distinguish between empty string and not set
	AllowedDirectories []string `json:"allowedDirectories,omitempty"` // Use omitempty; nil slice means not set, empty slice means allow all
	TelemetryEnabled   *bool    `json:"telemetryEnabled,omitempty"`   // Pointer for explicit true/false/not set
}

var currentConfig *ServerConfig
var loadConfigOnce sync.Once
var loadConfigErr error

// For testing purposes
var testConfigDir string

// loadConfig loads the configuration from file or creates default. Used internally.
func loadConfig(ctx *server.Context) (*ServerConfig, error) {
	loadConfigOnce.Do(func() {
		configPath, err := getConfigPath()
		if err != nil {
			loadConfigErr = fmt.Errorf("failed to get config path: %w", err)
			return
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				ctx.Logger.Info("Config file not found at %s, creating default for internal use", "configPath", configPath)
				cfg := ServerConfig{
					BlockedCommands: defaultBlockedCommands, // Use var from get_config.go
				}
				// Attempt to write default file, but proceed even if write fails
				configJson, marshalErr := json.MarshalIndent(cfg, "", "  ")
				if marshalErr == nil {
					_ = os.MkdirAll(filepath.Dir(configPath), 0755) // Ignore error
					_ = os.WriteFile(configPath, configJson, 0644)  // Ignore error
				} else {
					ctx.Logger.Info("Error marshalling default config for write", "error", marshalErr)
				}
				currentConfig = &cfg // Use in-memory default
			} else {
				loadConfigErr = fmt.Errorf("error reading config file %s: %w", configPath, err)
				return
			}
		} else {
			var cfg ServerConfig
			if err := json.Unmarshal(content, &cfg); err != nil {
				loadConfigErr = fmt.Errorf("error unmarshalling config file %s: %w", configPath, err)
				return
			}
			// Ensure BlockedCommands is not nil if file exists but key is missing
			if cfg.BlockedCommands == nil {
				cfg.BlockedCommands = []string{} // Initialize to empty slice
			}
			currentConfig = &cfg
		}
	})
	return currentConfig, loadConfigErr
}

// GetCurrentConfig provides access to the loaded configuration.
func GetCurrentConfig(ctx *server.Context) (*ServerConfig, error) {
	return loadConfig(ctx)
}

// getConfigPath returns the absolute path to the configuration file.
func getConfigPath() (string, error) {
	if testConfigDir != "" {
		return filepath.Join(testConfigDir, configFileName), nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, configDir, configFileName), nil
}
