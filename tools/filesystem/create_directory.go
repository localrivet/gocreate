package filesystem

import (
	"os"

	"github.com/localrivet/gomcp/server"
)

// CreateDirectoryArgs defines the arguments for the create_directory tool.
type CreateDirectoryArgs struct {
	Path string `json:"path" description:"The path of the directory to create." required:"true"`
}

// HandleCreateDirectory implements the logic for the create_directory tool using the new API.
func HandleCreateDirectory(ctx *server.Context, args CreateDirectoryArgs) (string, error) {
	ctx.Logger.Info("Handling create_directory tool call")

	// Create the directory and any necessary parent directories. 0755 is a common permission for directories.
	if err := os.MkdirAll(args.Path, 0755); err != nil {
		ctx.Logger.Info("Error creating directory", "path", args.Path, "error", err)
		return "Error creating directory", err
	}

	return "Directory created successfully.", nil
}
