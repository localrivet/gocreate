package edit

import (
	"fmt"
	"os"
	"strings"

	"github.com/localrivet/gomcp/server"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const maxEditFileSize = 100 * 1024 * 1024 // 100 MB limit

// Go structs for tool arguments
type EditBlockArgs struct {
	FilePath             string `json:"file_path" description:"The path to the file to edit." required:"true"`
	OldString            string `json:"old_string" description:"The exact block of text to find and replace." required:"true"`
	NewString            string `json:"new_string" description:"The new block of text to insert." required:"true"`
	ExpectedReplacements *int   `json:"expected_replacements,omitempty" description:"Optional. If provided, the exact number of replacements expected. Defaults to 1."`
}

// HandleEditBlock implements the edit_block tool using the new API
func HandleEditBlock(ctx *server.Context, args EditBlockArgs) (string, error) {
	ctx.Logger.Info("Handling edit_block tool call")

	// --- File Size Check ---
	fileInfo, err := os.Stat(args.FilePath)
	if err != nil {
		// Handle file not found or other stat errors
		if os.IsNotExist(err) {
			ctx.Logger.Info("File not found", "filePath", args.FilePath)
			return "Error: File not found.", err
		}
		ctx.Logger.Info("Error getting file info", "filePath", args.FilePath, "error", err)
		return "Error accessing file information.", err
	}

	if fileInfo.Size() > maxEditFileSize {
		errorMsg := fmt.Sprintf("Error: File size (%d bytes) exceeds the %d MB limit for this editing tool due to memory constraints. Please use a different tool or method for editing very large files. If this is a source code file, consider splitting it into smaller modules/files if appropriate for the language.", fileInfo.Size(), maxEditFileSize/(1024*1024))
		ctx.Logger.Info(errorMsg)
		return errorMsg, nil
	}
	// --- End File Size Check ---

	// Read the file (now known to be within size limit)
	content, err := os.ReadFile(args.FilePath)
	if err != nil {
		// This error should be less likely now after Stat, but handle anyway
		ctx.Logger.Info("Error reading file", "filePath", args.FilePath, "error", err)
		return "Error reading file for editing", err
	}

	originalContent := string(content)
	var modifiedContent string

	// --- Perform Context-Aware Replacement ---
	replacementsMade := 0
	expected := 1 // Default expectation

	if args.ExpectedReplacements != nil {
		expected = *args.ExpectedReplacements
		if expected <= 0 {
			return "expected_replacements must be positive", nil
		}
		// --- Handle Multiple Replacements (Using strings.Replace for now) ---
		actualOccurrences := strings.Count(originalContent, args.OldString)
		if actualOccurrences < expected {
			msg := fmt.Sprintf("Expected %d replacements, but only found %d occurrences of the old string.", expected, actualOccurrences)
			ctx.Logger.Info(msg, "filePath", args.FilePath)
			return msg, nil
		}
		modifiedContent = strings.Replace(originalContent, args.OldString, args.NewString, expected)
		if modifiedContent == originalContent && expected > 0 && actualOccurrences > 0 {
			msg := fmt.Sprintf("Replacement failed unexpectedly for %d expected replacements despite %d occurrences.", expected, actualOccurrences)
			ctx.Logger.Info(msg, "filePath", args.FilePath)
			return msg, nil
		}
		replacementsMade = expected
	} else {
		// --- Handle Single Replacement (Default) ---
		index := strings.Index(originalContent, args.OldString)

		if index == -1 {
			// Old string not found, generate near-miss diff if possible
			ctx.Logger.Info("Old string block not found in file", "filePath", args.FilePath)

			// --- Generate Diff for Near Miss ---
			dmp := diffmatchpatch.New()
			bestMatchIndex := dmp.MatchMain(originalContent, args.OldString, 0)

			var errorMsg string
			if bestMatchIndex != -1 {
				// Found a potential near miss location
				endIndex := bestMatchIndex + len(args.OldString)
				if endIndex > len(originalContent) {
					endIndex = len(originalContent)
				}
				closestMatchBlock := originalContent[bestMatchIndex:endIndex]

				// Generate diff between expected OldString and the actual block found
				diffs := dmp.DiffMain(args.OldString, closestMatchBlock, false)
				diffText := dmp.DiffPrettyText(diffs)
				diffText = strings.ReplaceAll(diffText, "\\n", "\n")
				diffText = strings.ReplaceAll(diffText, "%", "%%")
				errorMsg = fmt.Sprintf("Failed to apply edit. Found a potential match near character %d with differences:\n---\n%s\n---", bestMatchIndex, diffText)
				ctx.Logger.Info("Near miss found for edit_block", "filePath", args.FilePath)

			} else {
				// Couldn't find a reasonable match, just show the expected block
				ctx.Logger.Info("Near miss check failed to find any likely match for edit_block", "filePath", args.FilePath)
				diffsNotFound := dmp.DiffMain(args.OldString, "", false)
				diffText := dmp.DiffPrettyText(diffsNotFound)
				diffText = strings.ReplaceAll(diffText, "\\n", "\n")
				diffText = strings.ReplaceAll(diffText, "%", "%%")
				errorMsg = fmt.Sprintf("Failed to apply edit. Old string block not found/matched exactly. Expected block looked like:\n---\n%s\n---", diffText)
			}
			return errorMsg, nil

		} else {
			// Old string found, perform the replacement
			modifiedContent = originalContent[:index] + args.NewString + originalContent[index+len(args.OldString):]
			replacementsMade = 1
		}
	}

	// This check is slightly redundant now but kept as a safeguard
	if replacementsMade == 0 && expected > 0 {
		ctx.Logger.Info("Replacement logic failed unexpectedly", "filePath", args.FilePath)
		return "Internal error during replacement.", nil
	}

	// Write the modified content back to the file
	if err := os.WriteFile(args.FilePath, []byte(modifiedContent), 0644); err != nil {
		ctx.Logger.Info("Error writing file after edit_block", "filePath", args.FilePath, "error", err)
		return "Error writing file after editing", err
	}

	return "File edited successfully.", nil
}
