package terminal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gocreate/tools/config"

	"github.com/localrivet/gomcp/server"
	"mvdan.cc/sh/syntax"
)

// Go structs for tool arguments
type ExecuteCommandArgs struct {
	Command       string  `json:"command" description:"The command to execute." required:"true"`
	TimeoutMs     *int    `json:"timeout_ms,omitempty" description:"Optional timeout in milliseconds."`
	Shell         *string `json:"shell,omitempty" description:"Optional shell to use (e.g., /bin/bash, powershell.exe, cmd.exe). Defaults to best available shell."`
	UsePowerShell *bool   `json:"use_powershell,omitempty" description:"If true and on Windows, prefer PowerShell over cmd.exe. Ignored on non-Windows systems."`
}

type ReadOutputArgs struct {
	Pid int `json:"pid" description:"The PID of the terminal session to read output from." required:"true"`
}

type ForceTerminateArgs struct {
	Pid int `json:"pid" description:"The PID of the terminal session to force terminate." required:"true"`
}

type ListSessionsArgs struct{}

// detectBestShell determines the best available shell for the current system
func detectBestShell(preferPowerShell bool) string {
	if runtime.GOOS == "windows" {
		// Check for PowerShell if preferred
		if preferPowerShell {
			// Try PowerShell Core (pwsh.exe) first
			if pwshPath, err := exec.LookPath("pwsh.exe"); err == nil {
				return pwshPath
			}
			// Fall back to Windows PowerShell
			if powershellPath, err := exec.LookPath("powershell.exe"); err == nil {
				return powershellPath
			}
		}
		// Fall back to cmd.exe
		return "cmd.exe"
	}

	// For Unix-like systems, try to find the user's preferred shell
	if shellEnv := os.Getenv("SHELL"); shellEnv != "" {
		return shellEnv
	}

	// Check common Unix shells in order of preference
	shells := []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
	for _, shell := range shells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}

	// Final fallback to /bin/sh which should always exist on Unix systems
	return "/bin/sh"
}

// getShellExecuteFlag returns the appropriate flag to execute commands based on shell type
func getShellExecuteFlag(shell string) string {
	shell = filepath.Base(shell)
	switch shell {
	case "powershell.exe", "pwsh.exe":
		return "-Command"
	case "cmd.exe":
		return "/C"
	default:
		return "-c"
	}
}

// isCommandBlockedComplex checks if any command within a potentially complex shell string is blocked using AST parsing.
func isCommandBlockedComplex(ctx *server.Context, commandStr string, blockedCommands []string) (bool, string) {
	if len(blockedCommands) == 0 {
		return false, "" // No commands are blocked
	}

	// Create a map for faster lookup
	blockedSet := make(map[string]struct{}, len(blockedCommands))
	for _, cmd := range blockedCommands {
		blockedSet[cmd] = struct{}{}
	}

	// Parse the command string
	parser := syntax.NewParser()
	reader := strings.NewReader(commandStr)
	file, err := parser.Parse(reader, "")
	if err != nil {
		// If parsing fails, block execution as the command is ambiguous or invalid
		ctx.Logger.Info("Error parsing command string for validation. Blocking execution.", "error", err)
		return true, fmt.Sprintf("invalid syntax: %v", err)
	}

	var firstBlocked string
	blocked := false

	// Walk the AST to find command calls
	syntax.Walk(file, func(node syntax.Node) bool {
		if blocked { // Stop walking if we already found a blocked command
			return false
		}
		if cmd, ok := node.(*syntax.CallExpr); ok {
			if len(cmd.Args) > 0 {
				// Attempt to evaluate the first argument (command name) to a string
				// This handles simple cases, quotes, and potentially some expansions.
				// More complex cases (variables, command substitutions) might require an interpreter.
				// Using WordParts is more direct for simple literals.
				var cmdName string
				if len(cmd.Args[0].Parts) == 1 {
					switch part := cmd.Args[0].Parts[0].(type) {
					case *syntax.Lit:
						cmdName = part.Value
					case *syntax.SglQuoted:
						cmdName = part.Value
					case *syntax.DblQuoted:
						// Only check if it contains simple literals inside
						if len(part.Parts) == 1 {
							if lit, ok := part.Parts[0].(*syntax.Lit); ok {
								cmdName = lit.Value
							}
						}
					}
				}

				if cmdName != "" {
					cmdNameLower := strings.ToLower(cmdName)
					if _, isBlocked := blockedSet[cmdNameLower]; isBlocked {
						ctx.Logger.Info("Command validation failed: Found blocked command", "command", cmdName, "commandStr", commandStr)
						firstBlocked = cmdName // Return the original case name
						blocked = true
						return false // Stop walking
					}
				} else {
					// Log if we encounter a command name we can't easily resolve to a literal
					var sb strings.Builder
					syntax.DebugPrint(&sb, cmd.Args[0])
					ctx.Logger.Info("Warning: Could not resolve command name to simple literal for validation", "debug", sb.String())
				}
			}
		}
		return true // Continue walking
	})

	return blocked, firstBlocked
}

// New API handlers that return strings instead of protocol.Content

// HandleExecuteCommand implements the execute_command tool using the new API
func HandleExecuteCommand(ctx *server.Context, args ExecuteCommandArgs) (string, error) {
	ctx.Logger.Info("Handling execute_command tool call")

	// Determine shell preference
	usePowerShell := false
	if args.UsePowerShell != nil {
		usePowerShell = *args.UsePowerShell
	}

	// Get shell path
	shellPath := ""
	if args.Shell != nil && *args.Shell != "" {
		shellPath = *args.Shell
	} else {
		shellPath = detectBestShell(usePowerShell)
	}

	// --- Command Validation ---
	cfg, err := config.GetCurrentConfig(ctx) // Get loaded config
	if err != nil {
		ctx.Logger.Info("Error loading config for command validation", "error", err)
		return "Error loading configuration for validation", err
	}

	// Use the complex validation function
	blocked, blockedCmdName := isCommandBlockedComplex(ctx, args.Command, cfg.BlockedCommands)
	if blocked {
		errMsg := fmt.Sprintf("Command execution blocked: Command '%s' is blocked or syntax is invalid/unsupported for validation.", blockedCmdName)
		ctx.Logger.Info("Command blocked", "error", errMsg)
		return errMsg, nil
	}
	// --- End Command Validation ---

	// Get the appropriate execute flag for the shell
	executeFlag := getShellExecuteFlag(shellPath)

	// Get the terminal manager instance
	tm := GetManager()

	// Start the command asynchronously using the manager
	pid, startErr := tm.StartCommand(ctx, args.Command, shellPath, executeFlag)

	// Check for errors during start
	if startErr != nil {
		ctx.Logger.Info("Error starting command", "command", args.Command, "shell", shellPath, "error", startErr)
		// Attempt to provide more specific error message if possible
		errMsg := fmt.Sprintf("Error starting command: %v", startErr)
		// Check if it's a 'command not found' type error
		if exitErr, ok := startErr.(*exec.ExitError); ok {
			errMsg = fmt.Sprintf("Error starting command: %v. Stderr: %s", startErr, string(exitErr.Stderr))
		} else if pathErr, ok := startErr.(*exec.Error); ok {
			if pathErr.Err == exec.ErrNotFound {
				errMsg = fmt.Sprintf("Error starting command: Shell or command not found: %v", startErr)
			}
		}
		return errMsg, startErr
	}

	// Return PID indicating successful start
	ctx.Logger.Info("Command started successfully in background", "pid", pid, "shell", shellPath, "command", args.Command)
	resultText := fmt.Sprintf("Command started in background with PID: %d", pid)
	return resultText, nil
}

// HandleReadOutput implements the read_output tool using the new API
func HandleReadOutput(ctx *server.Context, args ReadOutputArgs) (string, error) {
	ctx.Logger.Info("Handling read_output tool call")

	// Get the terminal manager instance
	tm := GetManager()

	// Read new output from the manager
	output, err := tm.ReadNewOutput(args.Pid)
	if err != nil {
		ctx.Logger.Info("Error reading output", "pid", args.Pid, "error", err)
		return err.Error(), err
	}

	ctx.Logger.Info("Read output", "pid", args.Pid, "bytes", len(output))
	return output, nil
}

// HandleForceTerminate implements the force_terminate tool using the new API
func HandleForceTerminate(ctx *server.Context, args ForceTerminateArgs) (string, error) {
	ctx.Logger.Info("Handling force_terminate tool call")

	// Get the terminal manager instance
	tm := GetManager()

	// Terminate the session using the manager
	err := tm.TerminateSession(ctx, args.Pid)
	if err != nil {
		ctx.Logger.Info("Error terminating process", "pid", args.Pid, "error", err)
		return err.Error(), err
	}

	ctx.Logger.Info("Termination signal sent", "pid", args.Pid)
	resultText := fmt.Sprintf("Termination signal sent to PID %d.", args.Pid)
	return resultText, nil
}

// HandleListSessions implements the list_sessions tool using the new API
func HandleListSessions(ctx *server.Context, args ListSessionsArgs) (string, error) {
	ctx.Logger.Info("Handling list_sessions tool call")

	// Get the terminal manager instance
	tm := GetManager()

	// Get the list of active sessions
	activeSessions := tm.ListActiveSessions()

	// Marshal the result to JSON
	resultJson, err := json.MarshalIndent(activeSessions, "", "  ")
	if err != nil {
		ctx.Logger.Info("Error marshalling active sessions", "error", err)
		return "Error formatting session list", err
	}

	ctx.Logger.Info("Found active sessions", "count", len(activeSessions))
	return string(resultJson), nil
}
