package config

import (
	"encoding/json"
	"os"

	"github.com/localrivet/gomcp/server"
)

// HandleSetConfigValue implements the set_config_value tool using the new API
func HandleSetConfigValue(ctx *server.Context, args SetConfigValueArgs) (string, error) {
	ctx.Logger.Info("Handling set_config_value tool call")

	configPath, err := getConfigPath()
	if err != nil {
		ctx.Logger.Info("Error getting config path", "error", err)
		return "Error getting configuration file path", err
	}

	// Read the current config
	content, err := os.ReadFile(configPath)
	if err != nil {
		// If the file doesn't exist, start with a default empty config
		if os.IsNotExist(err) {
			ctx.Logger.Info("Config file not found, starting with default empty config for set operation", "configPath", configPath)
			content = []byte("{}") // Start with an empty JSON object
		} else {
			ctx.Logger.Info("Error reading config file for set_config_value", "configPath", configPath, "error", err)
			return "Error reading configuration file for update", err
		}
	}

	var config map[string]interface{} // Use a map to handle arbitrary keys
	if err := json.Unmarshal(content, &config); err != nil {
		ctx.Logger.Info("Error unmarshalling config file for set_config_value", "configPath", configPath, "error", err)
		return "Error parsing configuration file for update", err
	}

	// Update the specific key
	config[args.Key] = args.Value

	// Marshal the updated config
	updatedConfigJson, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		ctx.Logger.Info("Error marshalling updated config", "error", err)
		return "Error generating updated config", err
	}

	// Write the updated config back to the file
	if err := os.WriteFile(configPath, updatedConfigJson, 0644); err != nil {
		ctx.Logger.Info("Error writing updated config file", "configPath", configPath, "error", err)
		return "Error writing configuration file", err
	}

	ctx.Logger.Info("Configuration value set successfully", "key", args.Key)
	return "Configuration value set successfully.", nil
}
