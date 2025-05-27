package filesystem

import (
	"fmt"
	"os"
	"strings"

	"github.com/localrivet/gomcp/server"
)

// Go structs for tool arguments
type ReadFileArgs struct {
	FilePath  string `json:"file_path" description:"The path to the file to read." required:"true"`
	StartLine *int   `json:"start_line,omitempty" description:"Optional starting line number (1-indexed) for paging."`
	EndLine   *int   `json:"end_line,omitempty" description:"Optional ending line number (1-indexed, inclusive) for paging."`
}

// HandleReadFile implements the read_file tool using the new API
func HandleReadFile(ctx *server.Context, args ReadFileArgs) (string, error) {
	ctx.Logger.Info("Handling read_file tool call")

	// Read the file
	content, err := os.ReadFile(args.FilePath)
	if err != nil {
		ctx.Logger.Info("Error reading file", "file_path", args.FilePath, "error", err)
		return "Error reading file", err
	}

	fileContent := string(content)

	// If no line range specified, return the entire file
	if args.StartLine == nil && args.EndLine == nil {
		return fileContent, nil
	}

	// Handle line-based paging
	lines := strings.Split(fileContent, "\n")
	totalLines := len(lines)

	startLine := 1
	if args.StartLine != nil {
		startLine = *args.StartLine
	}

	endLine := totalLines
	if args.EndLine != nil {
		endLine = *args.EndLine
	}

	// Validate line numbers
	if startLine < 1 {
		startLine = 1
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine > endLine {
		return "Invalid line range: start_line must be <= end_line", nil
	}

	// Extract the requested lines (convert to 0-based indexing)
	selectedLines := lines[startLine-1 : endLine]
	result := strings.Join(selectedLines, "\n")

	// Add line number information
	info := fmt.Sprintf("Lines %d-%d of %d total lines:\n%s", startLine, endLine, totalLines, result)
	return info, nil
}
