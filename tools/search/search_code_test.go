package search

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/localrivet/gomcp/server"
)

func TestSearchCodeBasic(t *testing.T) {
	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"test1.go": `package main
import "fmt"
func main() {
	fmt.Println("Hello, World!")
}`,
		"test2.go": `package test
func TestFunction() {
	// This is a test function
	return
}`,
		"test3.txt": `This is a text file
with some content
that contains the word function`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Test basic search using GoRipGrep API
	results, err := Find("func", tempDir)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Fatal("Expected to find matches for 'func'")
	}

	if results.Count() < 2 {
		t.Fatalf("Expected at least 2 matches, got %d", results.Count())
	}

	// Verify we found matches in Go files
	files := results.Files()
	foundGoFiles := false
	for _, file := range files {
		if strings.HasSuffix(file, ".go") {
			foundGoFiles = true
			break
		}
	}
	if !foundGoFiles {
		t.Fatal("Expected to find matches in .go files")
	}
}

func TestSearchCodeWithOptions(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"test1.go": `package main
func Main() {
	// TODO: implement this
}`,
		"test2.js": `function test() {
	// todo: fix this
}`,
		"test3.py": `def function():
	# TODO: complete
	pass`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Test case-insensitive search with file pattern
	results, err := Find("todo", tempDir,
		WithIgnoreCase(),
		WithFilePattern("*.go"), // Note: simplified pattern for filepath.Match
		WithContextLines(1),
		WithMaxResults(10),
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Fatal("Expected to find matches for 'todo' (case-insensitive)")
	}

	// Should find TODO in Go file
	if results.Count() < 1 {
		t.Fatalf("Expected at least 1 match, got %d", results.Count())
	}

	// Check that context lines are included
	hasContext := false
	for _, match := range results.Matches {
		if len(match.Context) > 0 {
			hasContext = true
			break
		}
	}
	if !hasContext {
		t.Fatal("Expected context lines to be included")
	}
}

func TestSearchCodeRegex(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file with function definitions
	content := `package main

func main() {
	fmt.Println("Hello")
}

func TestFunction(t *testing.T) {
	// test code
}

func (r *Receiver) Method() {
	// method code
}`

	filePath := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test regex pattern for function definitions
	results, err := Find(`func\s+\w+\(`, tempDir)
	if err != nil {
		t.Fatalf("Regex search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Fatal("Expected to find function definitions")
	}

	// Should find at least 2 function definitions (main and TestFunction)
	if results.Count() < 2 {
		t.Fatalf("Expected at least 2 function definitions, got %d", results.Count())
	}
}

func TestSearchCodePerformance(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple test files
	for i := 0; i < 10; i++ {
		content := `package test
func TestFunction() {
	// This is a test function
	return
}`
		filePath := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test with performance options
	start := time.Now()
	results, err := Find("func", tempDir,
		WithWorkers(4),
		WithOptimization(true),
		WithMaxResults(100),
	)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Performance search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Fatal("Expected to find matches")
	}

	// Check performance stats
	if results.Stats.Duration == 0 {
		t.Fatal("Expected performance stats to be recorded")
	}

	if results.Stats.FilesScanned == 0 {
		t.Fatal("Expected files to be scanned")
	}

	// Search should complete reasonably quickly
	if duration > 5*time.Second {
		t.Fatalf("Search took too long: %v", duration)
	}

	t.Logf("Search completed in %v, scanned %d files, %d bytes",
		results.Stats.Duration,
		results.Stats.FilesScanned,
		results.Stats.BytesScanned)
}

func TestSearchCodeTimeout(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	content := "This is test content with the word test in it"
	filePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with very short timeout - the search might complete before timeout
	// or it might timeout, both are acceptable for this test
	results, err := Find("test", tempDir,
		WithTimeout(1*time.Microsecond), // Very short timeout
	)

	// Should either succeed quickly or timeout
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test passed if we get here without hanging or unexpected errors
	if err == nil {
		t.Logf("Search completed quickly before timeout with %d matches", results.Count())
	} else if err == context.DeadlineExceeded {
		t.Logf("Search timed out as expected")
	}
}

func TestHandleSearchCode(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	content := `package main
func main() {
	fmt.Println("Hello, World!")
}`
	filePath := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock server context
	ctx := &server.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Test HandleSearchCode function
	args := SearchCodeArgs{
		Path:    tempDir,
		Pattern: "func",
	}

	result, err := HandleSearchCode(ctx, args)
	if err != nil {
		t.Fatalf("HandleSearchCode failed: %v", err)
	}

	if result == "" {
		t.Fatal("Expected non-empty result")
	}

	// Result should contain file path and line number
	if !strings.Contains(result, "test.go") {
		t.Fatal("Expected result to contain filename")
	}

	if !strings.Contains(result, ":") {
		t.Fatal("Expected result to contain line number separator")
	}
}

func TestHandleSearchCodeWithOptions(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"test1.go": `package main
func main() {
	// TODO: implement
}`,
		"test2.txt": `This file contains TODO items`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create mock server context
	ctx := &server.Context{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Test with various options
	ignoreCase := true
	contextLines := 1
	filePattern := "*.go"
	maxResults := 5

	args := SearchCodeArgs{
		Path:         tempDir,
		Pattern:      "todo",
		IgnoreCase:   &ignoreCase,
		ContextLines: &contextLines,
		FilePattern:  &filePattern,
		MaxResults:   &maxResults,
	}

	result, err := HandleSearchCode(ctx, args)
	if err != nil {
		t.Fatalf("HandleSearchCode with options failed: %v", err)
	}

	if result == "" {
		t.Fatal("Expected non-empty result")
	}

	// Should only find matches in .go files due to file pattern
	if !strings.Contains(result, "test1.go") {
		t.Fatal("Expected to find match in test1.go")
	}

	if strings.Contains(result, "test2.txt") {
		t.Fatal("Should not find match in test2.txt due to file pattern")
	}
}
