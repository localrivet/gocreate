package filesystem

import (
	"os"

	"github.com/localrivet/gomcp/server"
)

// MoveFileArgs defines the arguments for the move_file tool.
type MoveFileArgs struct {
	Source      string `json:"source" description:"The source path of the file or directory." required:"true"`
	Destination string `json:"destination" description:"The destination path for the file or directory." required:"true"`
}

// HandleMoveFile implements the move_file tool using the new API
func HandleMoveFile(ctx *server.Context, args MoveFileArgs) (string, error) {
	ctx.Logger.Info("Handling move_file tool call")

	// Perform the move/rename operation
	if err := os.Rename(args.Source, args.Destination); err != nil {
		ctx.Logger.Info("Error moving/renaming file", "source", args.Source, "destination", args.Destination, "error", err)
		return "Error moving/renaming file", err
	}

	return "File moved/renamed successfully.", nil
}
