package search

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"time"

	"github.com/localrivet/gomcp/server"
)

// Go structs for tool arguments
type SearchCodeArgs struct {
	Path          string  `json:"path" description:"The directory path to search within." required:"true"`
	Pattern       string  `json:"pattern" description:"The text or regex pattern to search for." required:"true"`
	FilePattern   *string `json:"filePattern,omitempty" description:"Optional glob pattern to filter files (e.g., '*.go')."`
	IgnoreCase    *bool   `json:"ignoreCase,omitempty" description:"Perform case-insensitive search."`
	MaxResults    *int    `json:"maxResults,omitempty" description:"Maximum number of results to return."`
	IncludeHidden *bool   `json:"includeHidden,omitempty" description:"Include hidden files and directories in the search."`
	ContextLines  *int    `json:"contextLines,omitempty" description:"Number of context lines to show around matches."`
	TimeoutMs     *int    `json:"timeoutMs,omitempty" description:"Optional timeout in milliseconds for the search."`
}

// HandleSearchCode implements the search_code tool using the new API
func HandleSearchCode(ctx *server.Context, args SearchCodeArgs) (string, error) {
	ctx.Logger.Info("Handling search_code tool call")

	// Construct the ripgrep command
	cmdArgs := []string{"rg", "--line-number", args.Pattern, args.Path}

	if args.FilePattern != nil && *args.FilePattern != "" {
		cmdArgs = append(cmdArgs, "--glob", *args.FilePattern)
	}
	if args.IgnoreCase != nil && *args.IgnoreCase {
		cmdArgs = append(cmdArgs, "--ignore-case")
	}
	if args.MaxResults != nil && *args.MaxResults > 0 {
		cmdArgs = append(cmdArgs, "--max-count", strconv.Itoa(*args.MaxResults))
	}
	if args.IncludeHidden != nil && *args.IncludeHidden {
		cmdArgs = append(cmdArgs, "--hidden")
	}
	if args.ContextLines != nil && *args.ContextLines >= 0 {
		cmdArgs = append(cmdArgs, "--context", strconv.Itoa(*args.ContextLines))
	}

	// Set up context with timeout if specified
	searchCtx := context.Background()
	if args.TimeoutMs != nil && *args.TimeoutMs > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(context.Background(), time.Duration(*args.TimeoutMs)*time.Millisecond)
		defer cancel()
	}

	cmd := exec.CommandContext(searchCtx, cmdArgs[0], cmdArgs[1:]...)

	// Set up output capture
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String()
	errorOutput := stderr.String()

	// Check for timeout error
	if searchCtx.Err() == context.DeadlineExceeded {
		ctx.Logger.Info("Search timed out", "pattern", args.Pattern)
		combinedOutput := output + "\n" + errorOutput
		return "Search timed out.\nOutput:\n" + combinedOutput, nil
	}

	// ripgrep returns non-zero exit code if no matches are found (code 1) or if there was an error (code 2).
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means no matches found, which is not an error for the tool itself.
			if exitErr.ExitCode() == 1 {
				ctx.Logger.Info("Search completed with no matches", "pattern", args.Pattern)
				return "", nil
			}
		}
		// Any other error (including rg exit code 2) is treated as a tool error.
		ctx.Logger.Info("Error executing ripgrep command", "cmdArgs", cmdArgs, "error", err, "stdout", output, "stderr", errorOutput)
		combinedOutput := output + "\n" + errorOutput
		return "Error executing search command: " + err.Error() + "\nOutput:\n" + combinedOutput, err
	}

	ctx.Logger.Info("Search completed successfully", "pattern", args.Pattern)
	return output, nil
}
