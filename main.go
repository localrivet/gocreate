package main

import (
	"log"
	"log/slog"
	"os"

	"gocreate/tools/config"
	"gocreate/tools/edit"
	"gocreate/tools/filesystem"
	"gocreate/tools/process"
	"gocreate/tools/search"
	"gocreate/tools/terminal"

	"github.com/localrivet/gomcp/server"
)

func main() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create a new server
	s := server.NewServer("GoCreate",
		server.WithLogger(logger),
	).AsStdio()

	// Register tools using the API
	// Configuration tools
	s.Tool("get_config", "Get the complete server configuration as JSON.",
		config.HandleGetConfig)

	s.Tool("set_config_value", "Set a specific configuration value by key.",
		config.HandleSetConfigValue)

	// Filesystem tools
	s.Tool("read_file", "Read the contents of a file. Supports optional start_line and end_line parameters for paging.",
		filesystem.HandleReadFile)

	s.Tool("read_multiple_files", "Read the contents of multiple files simultaneously.",
		filesystem.HandleReadMultipleFiles)

	s.Tool("write_file", "Completely replace file contents.",
		filesystem.HandleWriteFile)

	s.Tool("create_directory", "Create a new directory or ensure a directory exists.",
		filesystem.HandleCreateDirectory)

	s.Tool("list_directory", "Get a detailed listing of all files and directories in a specified path.",
		filesystem.HandleListDirectory)

	s.Tool("move_file", "Move or rename files and directories.",
		filesystem.HandleMoveFile)

	s.Tool("search_files", "Finds files by name using a case-insensitive substring matching.",
		filesystem.HandleSearchFiles)

	s.Tool("get_file_info", "Retrieve detailed metadata about a file or directory.",
		filesystem.HandleGetFileInfo)

	s.Tool("search_code", "Search for text/code patterns within file contents using ripgrep.",
		search.HandleSearchCode)

	s.Tool("edit_block", "Apply surgical text replacements to files.",
		edit.HandleEditBlock)

	s.Tool("precise_edit", "Precisely edit file content based on start and end line numbers.",
		edit.HandlePreciseEdit)

	// Terminal tools
	s.Tool("execute_command", "Execute a terminal command with timeout.",
		terminal.HandleExecuteCommand)

	s.Tool("read_output", "Read new output from a running terminal session.",
		terminal.HandleReadOutput)

	s.Tool("force_terminate", "Force terminate a running terminal session.",
		terminal.HandleForceTerminate)

	s.Tool("list_sessions", "List all active terminal sessions.",
		terminal.HandleListSessions)

	s.Tool("execute_in_terminal", "Execute a command in the terminal (client-side execution).",
		terminal.HandleExecuteInTerminal)

	// Process tools
	s.Tool("list_processes", "List all running processes.",
		process.HandleListProcesses)

	s.Tool("kill_process", "Terminate a running process by PID.",
		process.HandleKillProcess)

	// Start the server
	logger.Info("Starting GoCreate MCP server...")
	if err := s.Run(); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
	logger.Info("Server shutdown complete.")
}
