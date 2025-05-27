package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/localrivet/gomcp/server"
)

// Define default blocked commands (more comprehensive list)
var defaultBlockedCommands = []string{
	// File Protection
	"rm",
	// Disk/Partition
	"mkfs", "format", "mount", "umount", "fdisk", "dd", "parted", "diskpart",
	// System Admin/User
	"sudo", "su", "passwd", "adduser", "useradd", "usermod", "groupadd", "chsh", "visudo",
	// System Control
	"shutdown", "reboot", "halt", "poweroff", "init",
	// Network/Security
	"iptables", "firewall", "netsh",
	// Windows Specific
	"sfc", "bcdedit", "reg", "net", "sc", "runas", "cipher", "takeown",
}

// GetConfigArgs defines the arguments for the get_config tool.
type GetConfigArgs struct{}

// SetConfigValueArgs defines the arguments for the set_config_value tool.
type SetConfigValueArgs struct {
	Key   string      `json:"key" description:"The configuration key to set." required:"true"`
	Value interface{} `json:"value" description:"The value to set for the key." required:"true"`
}

// HandleGetConfig implements the logic for the get_config tool using the new API.
func HandleGetConfig(ctx *server.Context, args GetConfigArgs) (string, error) {
	ctx.Logger.Info("Handling get_config tool call")

	configPath, err := getConfigPath()
	if err != nil {
		ctx.Logger.Info("Error getting config path", "error", err)
		return "Error getting configuration file path", err
	}

	// Read the config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		// If the file doesn't exist, create it with default BlockedCommands and return that
		if os.IsNotExist(err) {
			ctx.Logger.Info("Config file not found at %s, creating with default BlockedCommands", "configPath", configPath)
			// Only set BlockedCommands in the default config written to file
			defaultConfig := ServerConfig{
				BlockedCommands: defaultBlockedCommands,
				// DefaultShell, AllowedDirectories, TelemetryEnabled remain nil/zero
			}
			configJson, marshalErr := json.MarshalIndent(defaultConfig, "", "  ")
			if marshalErr != nil {
				ctx.Logger.Info("Error marshalling default config", "error", marshalErr)
				return "Error generating default config output", marshalErr
			}

			// Create the config directory if it doesn't exist
			configDirPath := filepath.Join(filepath.Dir(configPath))
			if err := os.MkdirAll(configDirPath, 0755); err != nil {
				ctx.Logger.Info("Error creating config directory", "configDirPath", configDirPath, "error", err)
				return "Error creating configuration directory", err
			}

			if writeErr := os.WriteFile(configPath, configJson, 0644); writeErr != nil {
				ctx.Logger.Info("Error writing default config file", "configPath", configPath, "error", writeErr)
				return "Error writing default configuration file", writeErr
			}

			return string(configJson), nil
		}
		ctx.Logger.Info("Error reading config file", "configPath", configPath, "error", err)
		return "Error reading configuration file", err
	}

	var config ServerConfig
	if err := json.Unmarshal(content, &config); err != nil {
		ctx.Logger.Info("Error unmarshalling config file", "configPath", configPath, "error", err)
		return "Error parsing configuration file", err
	}

	configJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		ctx.Logger.Info("Error marshalling config", "error", err)
		return "Error generating config output", err
	}

	return string(configJson), nil
}
