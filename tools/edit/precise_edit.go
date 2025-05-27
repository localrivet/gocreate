package edit

import (
	"fmt"
	"os"
	"strings"

	"github.com/localrivet/gomcp/server"
)

// const maxEditFileSize = 100 * 1024 * 1024 // Defined in edit.go

// Go structs for tool arguments - Updated for line-based editing
type PreciseEditArgs struct {
	FilePath   string `json:"file_path" description:"The path to the file to edit." required:"true"`
	StartLine  int    `json:"start_line" description:"The 1-indexed line number where the edit begins (inclusive)." required:"true"`
	EndLine    int    `json:"end_line" description:"The 1-indexed line number where the block to be replaced ends (inclusive). For insertion before start_line, use end_line = start_line - 1." required:"true"`
	NewContent string `json:"new_content" description:"The new content (potentially multi-line) to insert or replace the specified lines with." required:"true"`
}

// HandlePreciseEdit performs line-based editing on a file using the new API
func HandlePreciseEdit(ctx *server.Context, args PreciseEditArgs) (string, error) {
	ctx.Logger.Info("Handling precise_edit tool call (line-based editing, in-memory)")

	// --- Input Validation ---
	if args.StartLine <= 0 {
		msg := "start_line must be positive and 1-indexed"
		ctx.Logger.Info(msg)
		return msg, nil
	}
	// Allow end_line to be start_line - 1 for insertion
	if args.EndLine < args.StartLine-1 {
		msg := "end_line cannot be less than start_line - 1"
		ctx.Logger.Info(msg)
		return msg, nil
	}

	// --- File Size Check ---
	fileInfo, err := os.Stat(args.FilePath)
	fileExists := !os.IsNotExist(err)

	if err != nil && fileExists { // Handle stat errors only if file exists
		ctx.Logger.Info("Error getting file info", "filePath", args.FilePath, "error", err)
		return "Error accessing file information.", err
	}

	// Allow file not found only if inserting at the beginning of a new file
	if !fileExists && !(args.StartLine == 1 && args.EndLine == 0) {
		ctx.Logger.Info("File does not exist and cannot perform edit", "filePath", args.FilePath)
		return "Error: File not found.", nil
	}

	// Check size only if the file exists
	if fileExists && fileInfo.Size() > maxEditFileSize {
		errorMsg := fmt.Sprintf("Error: File size (%d bytes) exceeds the %d MB limit for this editing tool due to memory constraints. Please use a different tool or method for editing very large files. If this is a source code file, consider splitting it into smaller modules/files if appropriate for the language.", fileInfo.Size(), maxEditFileSize/(1024*1024))
		ctx.Logger.Info(errorMsg)
		return errorMsg, nil
	}
	// --- End File Size Check ---

	// --- Read File ---
	var contentBytes []byte
	if fileExists {
		contentBytes, err = os.ReadFile(args.FilePath)
		if err != nil {
			// This error should be less likely now after Stat, but handle anyway
			ctx.Logger.Info("Error reading file for precise_edit", "filePath", args.FilePath, "error", err)
			return "Error reading file for patching", err
		}
	} else {
		// File doesn't exist, but we are inserting at start
		ctx.Logger.Info("File does not exist, creating new file for insertion", "filePath", args.FilePath)
		contentBytes = []byte{} // Start with empty content
	}

	originalContent := string(contentBytes)
	// Detect line endings, default to \n
	lineEnding := "\n"
	if strings.Contains(originalContent, "\r\n") {
		lineEnding = "\r\n"
	}
	lines := strings.Split(originalContent, lineEnding)

	// Handle potential trailing newline splitting issue
	// If the file ends with a newline, Split leaves an empty string at the end.
	if len(originalContent) > 0 && strings.HasSuffix(originalContent, lineEnding) && len(lines) > 0 {
		// Keep the empty string unless the file was *only* the line ending(s)
		if len(lines) == 1 && lines[0] == "" {
			lines = []string{""} // File was empty or just newline(s)
		}
		// Otherwise, the empty string from split is handled correctly by len(lines) below
	} else if originalContent == "" {
		lines = []string{} // If the original file was completely empty
	}

	numLines := len(lines)
	// Correct numLines if the split resulted in [""] for an empty file
	if numLines == 1 && lines[0] == "" {
		numLines = 0
	}

	// --- Line Number Validation ---
	// Allow insertion *after* the last line
	if args.StartLine > numLines+1 {
		msg := fmt.Sprintf("start_line (%d) exceeds the number of lines (%d) + 1", args.StartLine, numLines)
		ctx.Logger.Info(msg)
		return msg, nil
	}
	// EndLine must be within bounds or StartLine-1 for insertion
	if args.EndLine > numLines || args.EndLine < args.StartLine-1 {
		if !(args.EndLine == 0 && args.StartLine == 1 && numLines == 0) { // Allow insert into empty file
			msg := fmt.Sprintf("end_line (%d) is out of bounds [0..%d] or invalid relative to start_line (%d)", args.EndLine, numLines, args.StartLine)
			ctx.Logger.Info(msg)
			return msg, nil
		}
	}

	// --- Construct New Content ---
	var newLines []string

	// 1. Add lines before the start line (adjust index to 0-based)
	startIdx := args.StartLine - 1
	if startIdx > 0 && startIdx <= numLines { // Ensure startIdx is valid
		newLines = append(newLines, lines[0:startIdx]...)
	}

	// 2. Add the new content (split if multi-line)
	if args.NewContent != "" {
		// Use the detected line ending for splitting NewContent as well
		newContentLines := strings.Split(args.NewContent, lineEnding)
		// Handle potential trailing newline in NewContent causing extra empty string
		if len(args.NewContent) > 0 && strings.HasSuffix(args.NewContent, lineEnding) && len(newContentLines) > 0 {
			// If NewContent ends with newline, Split adds an empty string. We usually want that empty string
			// to represent the line break *after* the last line of actual content.
			newLines = append(newLines, newContentLines...)
		} else {
			newLines = append(newLines, newContentLines...)
		}
	}

	// 3. Add lines after the end line (adjust index to 0-based)
	// endIdx is the line number *after* the last line to be replaced/skipped
	endIdx := args.EndLine + 1
	if endIdx <= numLines { // Check if endIdx is within the bounds of original lines
		newLines = append(newLines, lines[endIdx-1:]...) // Add from the line *after* EndLine
	}

	// --- Write File ---
	// Join lines with the original line ending
	finalContent := strings.Join(newLines, lineEnding)
	// Ensure trailing newline if the original had one and the edit didn't remove the last line
	// Or if the original was empty and new content was added.
	if (len(originalContent) > 0 && strings.HasSuffix(originalContent, lineEnding) && endIdx <= numLines) ||
		(len(originalContent) == 0 && len(finalContent) > 0) {
		if !strings.HasSuffix(finalContent, lineEnding) {
			finalContent += lineEnding
		}
	}

	// Get original file info for permissions
	fileMode := os.FileMode(0644) // Default permission
	if fileExists {
		fileInfo, infoErr := os.Stat(args.FilePath)
		if infoErr == nil {
			fileMode = fileInfo.Mode()
		} else { // Log if error is something other than NotExist (already handled)
			ctx.Logger.Info("Warning: Could not get file info, using default permissions", "filePath", args.FilePath, "error", infoErr)
		}
	}

	// Write the patched content back to the original file path (truncates existing)
	if err := os.WriteFile(args.FilePath, []byte(finalContent), fileMode); err != nil {
		ctx.Logger.Info("Error writing patched file", "filePath", args.FilePath, "error", err)
		return "Error writing patched file", err
	}

	ctx.Logger.Info("File edited successfully using precise_edit (in-memory)", "filePath", args.FilePath)
	return "File edited successfully.", nil
}
