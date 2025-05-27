package filesystem

import (
	"encoding/json"
	"os"
	"time"

	"github.com/localrivet/gomcp/server"
)

// GetFileInfoArgs defines the arguments for the get_file_info tool.
type GetFileInfoArgs struct {
	Path string `json:"path" description:"The path of the file or directory to get information for." required:"true"`
}

// HandleGetFileInfo implements the get_file_info tool using the new API
func HandleGetFileInfo(ctx *server.Context, args GetFileInfoArgs) (string, error) {
	ctx.Logger.Info("Handling get_file_info tool call")

	fileInfo, err := os.Stat(args.Path)
	if err != nil {
		ctx.Logger.Info("Error getting file info", "path", args.Path, "error", err)
		return "Error getting file info", err
	}

	// Format the file info
	info := map[string]interface{}{
		"name":     fileInfo.Name(),
		"size":     fileInfo.Size(),
		"is_dir":   fileInfo.IsDir(),
		"mode":     fileInfo.Mode().String(),
		"mod_time": fileInfo.ModTime().Format(time.RFC3339),
	}

	// Marshal the file info into JSON
	infoJson, marshalErr := json.MarshalIndent(info, "", "  ")
	if marshalErr != nil {
		ctx.Logger.Info("Error marshalling file info", "error", marshalErr)
		return "Error generating file info output", marshalErr
	}

	return string(infoJson), nil
}
