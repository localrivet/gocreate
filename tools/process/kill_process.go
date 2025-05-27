package process

import (
	"fmt"
	"os"
	"runtime"

	"github.com/localrivet/gomcp/server"
)

// HandleKillProcess implements the kill_process tool using the new API
func HandleKillProcess(ctx *server.Context, args KillProcessArgs) (string, error) {
	ctx.Logger.Info("Handling kill_process tool call")

	// TODO: Implement kill_process logic for Windows using taskkill
	if runtime.GOOS == "windows" {
		return "kill_process not fully implemented for Windows", nil
	}

	// Find the process by PID
	process, err := os.FindProcess(args.Pid)
	if err != nil {
		ctx.Logger.Info("Error finding process", "pid", args.Pid, "error", err)
		return fmt.Sprintf("Error finding process with PID %d: %v", args.Pid, err), err
	}

	// Send a termination signal (SIGTERM)
	if err := process.Signal(os.Interrupt); err != nil {
		ctx.Logger.Info("Error sending signal to process", "pid", args.Pid, "error", err)
		return fmt.Sprintf("Error sending termination signal to process with PID %d: %v", args.Pid, err), err
	}

	return fmt.Sprintf("Termination signal sent to process with PID %d.", args.Pid), nil
}
