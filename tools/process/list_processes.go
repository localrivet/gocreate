package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/localrivet/gomcp/server"
)

// ProcessInfo represents information about a running process.
type ProcessInfo struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
	// Add other fields like CPU, Memory if parsing is enhanced
}

// HandleListProcesses implements the list_processes tool using the new API
func HandleListProcesses(ctx *server.Context, args ListProcessesArgs) (string, error) {
	ctx.Logger.Info("Handling list_processes tool call")

	// TODO: Implement list_processes logic for Windows using tasklist
	if runtime.GOOS == "windows" {
		return "list_processes not fully implemented for Windows", nil
	}

	// Implementation for Unix-like systems using 'ps aux'
	cmd := exec.CommandContext(context.Background(), "ps", "aux")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		ctx.Logger.Info("Error executing 'ps aux'", "error", err, "stderr", stderr.String())
		return "Error listing processes: " + err.Error() + "\n" + stderr.String(), err
	}

	// Parse the output of 'ps aux'
	lines := strings.Split(stdout.String(), "\n")
	var processes []ProcessInfo
	// Skip header line and empty lines
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 { // Basic check for expected number of fields in ps aux output
			ctx.Logger.Info("Skipping unexpected line format in ps aux output", "line", line)
			continue
		}

		pid := 0
		// Attempt to parse PID (usually the second field)
		_, err := fmt.Sscan(fields[1], &pid)
		if err != nil {
			ctx.Logger.Info("Error parsing PID from line", "line", line, "error", err)
			continue // Skip this line if PID cannot be parsed
		}

		// The command is typically the 11th field onwards
		command := strings.Join(fields[10:], " ")

		processes = append(processes, ProcessInfo{PID: pid, Command: command})
	}

	// Marshal the process list into JSON
	processesJson, marshalErr := json.MarshalIndent(processes, "", "  ")
	if marshalErr != nil {
		ctx.Logger.Info("Error marshalling process list", "error", marshalErr)
		return "Error generating process list output", marshalErr
	}

	return string(processesJson), nil
}
