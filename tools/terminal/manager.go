package terminal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/localrivet/gomcp/server"
)

// TerminalSession holds information about a running command process.
type TerminalSession struct {
	PID       int
	Cmd       *exec.Cmd
	StartTime time.Time
	Stdout    bytes.Buffer // Buffer to capture stdout
	Stderr    bytes.Buffer // Buffer to capture stderr
	Done      chan error   // Channel to signal completion
	// TODO: Consider adding command string, shell used, etc. if needed for list_sessions
}

// TerminalManager manages active terminal sessions.
type TerminalManager struct {
	mu       sync.Mutex // Mutex to protect concurrent access to sessions map
	sessions map[int]*TerminalSession
	// TODO: Consider adding completed sessions tracking if needed
}

// Global instance of the TerminalManager
var globalTerminalManager *TerminalManager
var once sync.Once

// GetManager returns the singleton instance of the TerminalManager.
func GetManager() *TerminalManager {
	once.Do(func() {
		globalTerminalManager = &TerminalManager{
			sessions: make(map[int]*TerminalSession),
		}
		// TODO: Add any background cleanup routines if needed (e.g., for old completed sessions)
	})
	return globalTerminalManager
}

// AddSession adds a new session to the manager.
func (tm *TerminalManager) AddSession(pid int, session *TerminalSession) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.sessions[pid] = session
}

// GetSession retrieves a session by PID.
func (tm *TerminalManager) GetSession(pid int) (*TerminalSession, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	session, exists := tm.sessions[pid]
	return session, exists
}

// RemoveSession removes a session by PID.
func (tm *TerminalManager) RemoveSession(pid int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.sessions, pid)
	// TODO: Potentially add session to a completed list here
}

// StartCommand starts a command asynchronously and manages its session.
// Returns PID and error (nil if start was successful).
func (tm *TerminalManager) StartCommand(ctx *server.Context, commandStr string, shell string, executeFlag string) (int, error) {
	cmd := exec.Command(shell, executeFlag, commandStr)

	session := &TerminalSession{
		Cmd:       cmd,
		StartTime: time.Now(),
		Done:      make(chan error, 1), // Buffered channel
	}

	// Assign buffers for stdout and stderr capture
	cmd.Stdout = &session.Stdout
	cmd.Stderr = &session.Stderr

	// Start the command asynchronously
	err := cmd.Start()
	if err != nil {
		return -1, err // Failed to start
	}

	session.PID = cmd.Process.Pid
	tm.AddSession(session.PID, session)

	ctx.Logger.Info("Started command", "pid", session.PID, "command", commandStr)

	// Start a goroutine to wait for the command to finish
	go func() {
		err := cmd.Wait()
		session.Done <- err // Send completion error (or nil) to the channel
		close(session.Done) // Close channel to signal completion fully

		ctx.Logger.Info("Command finished", "pid", session.PID, "error", err)

		// Clean up the session from the active map
		// TODO: Consider moving completed session info elsewhere before removing
		tm.RemoveSession(session.PID)
	}()

	return session.PID, nil // Return PID and nil error indicating successful start
}

// ReadNewOutput retrieves any output captured since the last call for a given PID.
// It clears the internal buffer after reading.
func (tm *TerminalManager) ReadNewOutput(pid int) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	session, exists := tm.sessions[pid]
	if !exists {
		// TODO: Check completed sessions here?
		return "", fmt.Errorf("session with PID %d not found or already completed", pid)
	}

	// Read directly from the session's buffers
	// Note: This might not be perfectly synchronized if the process writes rapidly
	// between reads, but it captures what's available.
	// A more robust solution might involve dedicated goroutines reading streams.

	stdoutBytes := session.Stdout.Bytes()
	stderrBytes := session.Stderr.Bytes()

	// Reset buffers after reading
	session.Stdout.Reset()
	session.Stderr.Reset()

	output := string(stdoutBytes) + string(stderrBytes)

	return output, nil
}

// TerminateSession attempts to terminate the process associated with the given PID.
// It first tries SIGINT, then SIGKILL if necessary.
func (tm *TerminalManager) TerminateSession(ctx *server.Context, pid int) error {
	tm.mu.Lock()
	// Unlock happens within the function to allow process killing which might block briefly

	session, exists := tm.sessions[pid]
	if !exists {
		tm.mu.Unlock()
		// TODO: Check completed sessions? Maybe return a specific "already completed" error?
		return fmt.Errorf("session with PID %d not found or already completed", pid)
	}

	// Get the process
	process := session.Cmd.Process
	if process == nil {
		// Should not happen if session exists, but check anyway
		tm.mu.Unlock()
		// Remove the session as it's invalid
		delete(tm.sessions, pid)
		return fmt.Errorf("process not found for session PID %d", pid)
	}
	tm.mu.Unlock() // Unlock before potentially blocking kill operations

	ctx.Logger.Info("Attempting to terminate process", "pid", pid)

	// Try SIGINT first (graceful shutdown) - platform dependent
	err := process.Signal(os.Interrupt)
	if err != nil {
		ctx.Logger.Info("Failed to send SIGINT", "pid", pid, "error", err)
		// If SIGINT fails or isn't supported, try SIGKILL (forceful)
		err = process.Kill()
		if err != nil {
			ctx.Logger.Info("Failed to send SIGKILL", "pid", pid, "error", err)
			// Even if kill fails, the process might have exited already.
			// The background goroutine in StartCommand should handle cleanup.
			return fmt.Errorf("failed to terminate process PID %d: %w", pid, err)
		}
	}

	// SIGINT sent successfully - process should terminate gracefully
	// The background goroutine in StartCommand will handle cleanup when cmd.Wait() returns
	// Note: The session is removed from the map by the goroutine in StartCommand when cmd.Wait() returns.
	// We don't remove it here directly.
	ctx.Logger.Info("Termination signal sent", "pid", pid)
	return nil // Signal sent successfully (doesn't guarantee process exited immediately)
}

// ActiveSessionInfo provides basic info about a running session.
type ActiveSessionInfo struct {
	PID       int    `json:"pid"`
	StartTime string `json:"startTime"`
	RuntimeMs int64  `json:"runtimeMs"`
	// TODO: Add Command string if stored in TerminalSession
}

// ListActiveSessions returns information about currently running sessions.
func (tm *TerminalManager) ListActiveSessions() []ActiveSessionInfo {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	active := make([]ActiveSessionInfo, 0, len(tm.sessions))

	for pid, session := range tm.sessions {
		active = append(active, ActiveSessionInfo{
			PID:       pid,
			StartTime: session.StartTime.Format(time.RFC3339),
			RuntimeMs: now.Sub(session.StartTime).Milliseconds(),
		})
	}
	return active
}
