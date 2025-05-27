package terminal

import (
	"encoding/json"

	"github.com/localrivet/gomcp/server"
)

// ExecuteInTerminalArgs defines the arguments for the execute_in_terminal tool.
type ExecuteInTerminalArgs struct {
	Command string `json:"command" description:"The command to execute in the terminal." required:"true"`
	Cwd     string `json:"cwd,omitempty" description:"The working directory for the terminal. Defaults to the server's current working directory."`
}

// HandleExecuteInTerminal implements the logic for the execute_in_terminal tool.
func HandleExecuteInTerminal(ctx *server.Context, args ExecuteInTerminalArgs) (string, error) {
	ctx.Logger.Info("Handling execute_in_terminal tool call")

	// This handler signals the client to execute the command in a terminal.
	// The actual execution happens client-side.
	// We return a JSON response with terminal execution instructions.
	response := map[string]interface{}{
		"type":    "terminal",
		"command": args.Command,
		"cwd":     args.Cwd,
		"message": "Terminal execution requested",
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		ctx.Logger.Info("Error marshalling terminal response", "error", err)
		return "", err
	}

	return string(responseJSON), nil
}
