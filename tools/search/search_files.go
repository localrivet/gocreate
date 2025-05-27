package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/localrivet/gomcp/server"
)

// SearchFilesArgs defines the arguments for the search_files tool.
type SearchFilesArgs struct {
	Path        string `json:"path" description:"The path of the directory to search in." required:"true"`
	Regex       string `json:"regex" description:"The regular expression pattern to search for." required:"true"`
	FilePattern string `json:"file_pattern,omitempty" description:"Glob pattern to filter files (e.g., '*.ts'). If not provided, searches all files."`
}

// SearchResult represents a single match found during search.
type SearchResult struct {
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Match    string `json:"match"`
	LineText string `json:"line_text"`
	Context  string `json:"context"` // Surrounding lines for context
}

// HandleSearchFilesNew implements the search_files tool using the new API
func HandleSearchFilesNew(ctx *server.Context, args SearchFilesArgs) (string, error) {
	ctx.Logger.Info("Handling search_files tool call")

	// Compile the regex
	re, err := regexp.Compile(args.Regex)
	if err != nil {
		ctx.Logger.Info("Error compiling regex", "regex", args.Regex, "error", err)
		return "Error compiling regex: " + err.Error(), err
	}

	var results []SearchResult

	// Walk the directory
	err = filepath.Walk(args.Path, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			ctx.Logger.Info("Error walking path", "path", filePath, "error", walkErr)
			return walkErr // Continue walking other paths
		}

		if !info.IsDir() {
			// Apply file pattern filter if provided
			if args.FilePattern != "" {
				matched, _ := filepath.Match(args.FilePattern, info.Name())
				if !matched {
					return nil // Skip this file
				}
			}

			// Read the file content
			contentBytes, readErr := os.ReadFile(filePath)
			if readErr != nil {
				ctx.Logger.Info("Error reading file for search", "filePath", filePath, "error", readErr)
				return nil // Skip this file, continue walking
			}
			content := string(contentBytes)
			lines := strings.Split(content, "\n")

			// Search for matches line by line
			for i, line := range lines {
				matches := re.FindAllStringIndex(line, -1)
				for _, matchIndex := range matches {
					start, end := matchIndex[0], matchIndex[1]
					matchText := line[start:end]

					// Get context (e.g., 2 lines before and after)
					contextLines := []string{}
					contextStart := max(0, i-2)
					contextEnd := min(len(lines)-1, i+2)
					for j := contextStart; j <= contextEnd; j++ {
						contextLines = append(contextLines, fmt.Sprintf("%d | %s", j+1, lines[j]))
					}
					contextText := strings.Join(contextLines, "\n")

					results = append(results, SearchResult{
						FilePath: filePath,
						Line:     i + 1,     // 1-based line number
						Column:   start + 1, // 1-based column number
						Match:    matchText,
						LineText: line,
						Context:  contextText,
					})
				}
			}
		}
		return nil
	})

	if err != nil {
		ctx.Logger.Info("Error during directory walk for search", "error", err)
		return "Error during file search: " + err.Error(), err
	}

	// Marshal the results into JSON
	resultsJson, marshalErr := json.MarshalIndent(results, "", "  ")
	if marshalErr != nil {
		ctx.Logger.Info("Error marshalling search results", "error", marshalErr)
		return "Error generating search results output", marshalErr
	}

	return string(resultsJson), nil
}

// Helper function to find the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Helper function to find the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Legacy handler - keeping for backward compatibility but not used with new API
func HandleSearchFiles(ctx context.Context, progressToken interface{}, arguments any) ([]interface{}, bool) {
	// Legacy implementation - not used with new API
	return []interface{}{"Legacy handler not implemented"}, true
}
