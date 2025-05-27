package filesystem

import (
	"encoding/json"
	"os"

	"github.com/localrivet/gomcp/server"
)

// ReadMultipleFilesArgs defines the arguments for the read_multiple_files tool.
type ReadMultipleFilesArgs struct {
	Paths []string `json:"paths" description:"An array of file paths to read." required:"true"`
}

// HandleReadMultipleFiles implements the read_multiple_files tool using the new API
func HandleReadMultipleFiles(ctx *server.Context, args ReadMultipleFilesArgs) (string, error) {
	ctx.Logger.Info("Handling read_multiple_files tool call")

	results := make(map[string]string)

	for _, path := range args.Paths {
		content, err := os.ReadFile(path)
		if err != nil {
			ctx.Logger.Info("Error reading file", "path", path, "error", err)
			results[path] = "Error reading file: " + err.Error()
		} else {
			results[path] = string(content)
		}
	}

	// Marshal the results into JSON
	resultsJson, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		ctx.Logger.Info("Error marshalling results for read_multiple_files", "error", err)
		return "Error generating results output", err
	}

	return string(resultsJson), nil
}
