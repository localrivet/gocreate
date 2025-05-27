package filesystem

import (
	"os"

	"github.com/localrivet/gomcp/server"
)

// WriteFileArgs defines the arguments for the write_file tool.
type WriteFileArgs struct {
	Path    string `json:"path" description:"The path of the file to write to." required:"true"`
	Content string `json:"content" description:"The content to write to the file." required:"true"`
}

// HandleWriteFile implements the write_file tool using the new API
func HandleWriteFile(ctx *server.Context, args WriteFileArgs) (string, error) {
	ctx.Logger.Info("Handling write_file tool call")

	// Write the content to the file. 0644 is a common permission for files.
	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		ctx.Logger.Info("Error writing file", "path", args.Path, "error", err)
		return "Error writing file", err
	}

	return "File written successfully.", nil
}
