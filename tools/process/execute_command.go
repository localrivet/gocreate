package process

import (
	"context"
	"os/exec"
	"strings"

	"github.com/localrivet/gomcp/server"
)

// ExecuteCommandArgs defines the arguments for the execute_command tool.
type ExecuteCommandArgs struct {
	Command string `json:"command" description:"The command to execute." required:"true"`
	Cwd     string `json:"cwd,omitempty" description:"The working directory for the command. Defaults to the server's current working directory."`
}

// ListProcessesArgs defines the arguments for the list_processes tool.
type ListProcessesArgs struct{}

// KillProcessArgs defines the arguments for the kill_process tool.
type KillProcessArgs struct {
	Pid int `json:"pid" description:"The process ID to terminate." required:"true"`
}

// HandleExecuteCommandNew implements the execute_command tool using the new API
func HandleExecuteCommandNew(ctx *server.Context, args ExecuteCommandArgs) (string, error) {
	ctx.Logger.Info("Handling execute_command tool call")

	// Basic sanitization: prevent execution of potentially harmful commands
	blockedCommands := []string{"rm ", "format ", "mount ", "umount ", "mkfs ", "fdisk ", "dd ", "sudo ", "su ", "passwd ", "adduser ", "useradd ", "usermod ", "groupadd "}
	commandLower := strings.ToLower(args.Command)
	for _, blocked := range blockedCommands {
		if strings.Contains(commandLower, blocked) {
			ctx.Logger.Info("Blocked command execution attempt", "command", args.Command)
			return "Error: Execution of this command is blocked for security reasons.", nil
		}
	}

	// Execute the command
	cmd := exec.CommandContext(context.Background(), "/bin/sh", "-c", args.Command)
	if args.Cwd != "" {
		cmd.Dir = args.Cwd
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Logger.Info("Error executing command", "command", args.Command, "error", err, "output", string(output))
		return "Error executing command:\n" + string(output), err
	}

	ctx.Logger.Info("Command executed successfully", "command", args.Command, "output", string(output))
	return string(output), nil
}

// Legacy handler - keeping for backward compatibility but not used with new API
func HandleExecuteCommand(ctx context.Context, progressToken interface{}, arguments any) ([]interface{}, bool) {
	// Legacy implementation - not used with new API
	return []interface{}{"Legacy handler not implemented"}, true
}
