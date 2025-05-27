package filesystem

import (
	"encoding/json"
	"os"

	"github.com/localrivet/gomcp/server"
)

// ListDirectoryArgs defines the arguments for the list_directory tool.
type ListDirectoryArgs struct {
	Path string `json:"path" description:"The path of the directory to list." required:"true"`
}

// HandleListDirectory implements the list_directory tool using the new API
func HandleListDirectory(ctx *server.Context, args ListDirectoryArgs) (string, error) {
	ctx.Logger.Info("Handling list_directory tool call")

	files, err := os.ReadDir(args.Path)
	if err != nil {
		ctx.Logger.Info("Error reading directory", "path", args.Path, "error", err)
		return "Error reading directory", err
	}

	var fileList []string
	for _, file := range files {
		fileType := "[FILE]"
		if file.IsDir() {
			fileType = "[DIR]"
		}
		fileList = append(fileList, fileType+" "+file.Name())
	}

	// Marshal the file list into JSON
	fileListJson, err := json.MarshalIndent(fileList, "", "  ")
	if err != nil {
		ctx.Logger.Info("Error marshalling file list for list_directory", "error", err)
		return "Error generating file list output", err
	}

	return string(fileListJson), nil
}
