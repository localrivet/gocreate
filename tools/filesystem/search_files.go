package filesystem

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/localrivet/gomcp/server"
)

// SearchFilesArgs defines the arguments for the search_files tool.
type SearchFilesArgs struct {
	Path      string `json:"path" description:"The directory path to search in." required:"true"`
	Pattern   string `json:"pattern" description:"The case-insensitive substring pattern to search for in file names." required:"true"`
	TimeoutMs *int   `json:"timeoutMs,omitempty" description:"Optional timeout in milliseconds for the search."`
}

// HandleSearchFiles implements the search_files tool using the new API
func HandleSearchFiles(ctx *server.Context, args SearchFilesArgs) (string, error) {
	ctx.Logger.Info("Handling search_files tool call")

	var foundFiles []string

	// Set up context with timeout
	searchCtx := context.Background()
	if args.TimeoutMs != nil && *args.TimeoutMs > 0 {
		var cancel context.CancelFunc
		searchCtx, cancel = context.WithTimeout(context.Background(), time.Duration(*args.TimeoutMs)*time.Millisecond)
		defer cancel()
	}

	// Walk the directory tree
	err := filepath.WalkDir(args.Path, func(path string, d os.DirEntry, err error) error {
		// Check for context cancellation
		select {
		case <-searchCtx.Done():
			ctx.Logger.Info("Search timed out or cancelled")
			return searchCtx.Err()
		default:
			// Continue
		}

		if err != nil {
			ctx.Logger.Info("Error accessing path", "path", path, "error", err)
			return nil // Don't stop the walk for individual errors
		}

		// Skip directories themselves, we only care about files matching the pattern
		if d.IsDir() {
			return nil
		}

		// Perform case-insensitive substring match on the file name
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(args.Pattern)) {
			foundFiles = append(foundFiles, path)
		}

		return nil
	})

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		ctx.Logger.Info("Error during directory walk for search_files", "error", err)
		return "Error during file search", err
	}

	// Marshal the found files list into JSON
	foundFilesJson, marshalErr := json.MarshalIndent(foundFiles, "", "  ")
	if marshalErr != nil {
		ctx.Logger.Info("Error marshalling found files for search_files", "error", marshalErr)
		return "Error generating search results output", marshalErr
	}

	// If the context was cancelled due to timeout, indicate an error
	if searchCtx.Err() != nil {
		return "Search timed out.", nil
	}

	return string(foundFilesJson), nil
}
