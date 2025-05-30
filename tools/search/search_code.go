package search

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/localrivet/gomcp/server"
)

// Go structs for tool arguments
type SearchCodeArgs struct {
	Path          string  `json:"path" description:"The directory path to search within." required:"true"`
	Pattern       string  `json:"pattern" description:"The text or regex pattern to search for." required:"true"`
	FilePattern   *string `json:"filePattern,omitempty" description:"Optional glob pattern to filter files (e.g., '*.go')."`
	IgnoreCase    *bool   `json:"ignoreCase,omitempty" description:"Perform case-insensitive search."`
	MaxResults    *int    `json:"maxResults,omitempty" description:"Maximum number of results to return."`
	IncludeHidden *bool   `json:"includeHidden,omitempty" description:"Include hidden files and directories in the search."`
	ContextLines  *int    `json:"contextLines,omitempty" description:"Number of context lines to show around matches."`
	TimeoutMs     *int    `json:"timeoutMs,omitempty" description:"Optional timeout in milliseconds for the search."`
}

// SearchMatch represents a single search match
type SearchMatch struct {
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Column  int      `json:"column"`
	Content string   `json:"content"`
	Context []string `json:"context,omitempty"`
}

// SearchStats contains performance statistics
type SearchStats struct {
	Duration     time.Duration `json:"duration"`
	FilesScanned int           `json:"files_scanned"`
	BytesScanned int64         `json:"bytes_scanned"`
	MatchesFound int           `json:"matches_found"`
}

// SearchResults contains all search results and metadata
type SearchResults struct {
	Matches []SearchMatch `json:"matches"`
	Stats   SearchStats   `json:"stats"`
}

// Count returns the number of matches found
func (r *SearchResults) Count() int {
	return len(r.Matches)
}

// HasMatches returns true if any matches were found
func (r *SearchResults) HasMatches() bool {
	return len(r.Matches) > 0
}

// Files returns a list of unique files that contain matches
func (r *SearchResults) Files() []string {
	fileSet := make(map[string]bool)
	for _, match := range r.Matches {
		fileSet[match.File] = true
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

// SearchConfig holds configuration for the search engine
type SearchConfig struct {
	SearchPath      string
	Pattern         string
	MaxWorkers      int
	BufferSize      int
	MaxResults      int
	UseOptimization bool
	UseGitignore    bool
	IgnoreCase      bool
	FilePattern     string
	ContextLines    int
	IncludeHidden   bool
	Timeout         time.Duration
}

// SearchOption is a functional option for configuring searches
type SearchOption func(*SearchConfig)

// WithIgnoreCase enables case-insensitive search
func WithIgnoreCase() SearchOption {
	return func(c *SearchConfig) {
		c.IgnoreCase = true
	}
}

// WithContextLines sets the number of context lines around matches
func WithContextLines(lines int) SearchOption {
	return func(c *SearchConfig) {
		c.ContextLines = lines
	}
}

// WithFilePattern sets a glob pattern to filter files
func WithFilePattern(pattern string) SearchOption {
	return func(c *SearchConfig) {
		c.FilePattern = pattern
	}
}

// WithMaxResults limits the number of results returned
func WithMaxResults(max int) SearchOption {
	return func(c *SearchConfig) {
		c.MaxResults = max
	}
}

// WithWorkers sets the number of worker goroutines
func WithWorkers(workers int) SearchOption {
	return func(c *SearchConfig) {
		c.MaxWorkers = workers
	}
}

// WithHidden includes hidden files and directories
func WithHidden() SearchOption {
	return func(c *SearchConfig) {
		c.IncludeHidden = true
	}
}

// WithTimeout sets a timeout for the search operation
func WithTimeout(timeout time.Duration) SearchOption {
	return func(c *SearchConfig) {
		c.Timeout = timeout
	}
}

// WithGitignore enables respecting .gitignore files
func WithGitignore(enabled bool) SearchOption {
	return func(c *SearchConfig) {
		c.UseGitignore = enabled
	}
}

// WithBufferSize sets the buffer size for file I/O
func WithBufferSize(size int) SearchOption {
	return func(c *SearchConfig) {
		c.BufferSize = size
	}
}

// WithOptimization enables performance optimizations
func WithOptimization(enabled bool) SearchOption {
	return func(c *SearchConfig) {
		c.UseOptimization = enabled
	}
}

// Find performs a search with the given pattern and options
func Find(pattern, searchPath string, options ...SearchOption) (*SearchResults, error) {
	config := &SearchConfig{
		SearchPath:      searchPath,
		Pattern:         pattern,
		MaxWorkers:      runtime.NumCPU(),
		BufferSize:      64 * 1024,
		MaxResults:      1000,
		UseOptimization: true,
		UseGitignore:    false,
		IgnoreCase:      false,
		ContextLines:    0,
		IncludeHidden:   false,
		Timeout:         0,
	}

	// Apply options
	for _, option := range options {
		option(config)
	}

	engine := NewSearchEngine(*config)
	ctx := context.Background()
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	return engine.Search(ctx, pattern)
}

// SearchEngine provides fast text search functionality
type SearchEngine struct {
	config        SearchConfig
	pattern       *regexp.Regexp
	literalSearch string
}

// NewSearchEngine creates a new search engine with the given configuration
func NewSearchEngine(config SearchConfig) *SearchEngine {
	engine := &SearchEngine{
		config: config,
	}

	// Check if pattern is a simple literal string or regex
	if isLiteralPattern(config.Pattern) {
		// Use literal string search for better performance
		if config.IgnoreCase {
			engine.literalSearch = strings.ToLower(config.Pattern)
		} else {
			engine.literalSearch = config.Pattern
		}
	} else {
		// Compile regex pattern
		pattern := config.Pattern
		if config.IgnoreCase {
			pattern = "(?i)" + pattern
		}

		var err error
		engine.pattern, err = regexp.Compile(pattern)
		if err != nil {
			// Return engine with error state - will be caught in Search
			return engine
		}
	}

	return engine
}

// Search performs the text search operation
func (e *SearchEngine) Search(ctx context.Context, pattern string) (*SearchResults, error) {
	startTime := time.Now()

	// Validate pattern if using regex
	if e.pattern == nil && !isLiteralPattern(pattern) {
		regexPattern := pattern
		if e.config.IgnoreCase {
			regexPattern = "(?i)" + pattern
		}
		var err error
		e.pattern, err = regexp.Compile(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

	results := &SearchResults{
		Matches: make([]SearchMatch, 0),
		Stats: SearchStats{
			FilesScanned: 0,
			BytesScanned: 0,
			MatchesFound: 0,
		},
	}

	matchChan := make(chan SearchMatch, 1000)
	var wg sync.WaitGroup
	var resultCount int64
	var filesScanned int64
	var bytesScanned int64

	// Use worker pool for concurrent file processing
	filePaths := make(chan string, e.config.MaxWorkers*2)

	// Start workers
	for i := 0; i < e.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range filePaths {
				select {
				case <-ctx.Done():
					return
				default:
				}

				matches, fileBytes, err := e.searchFile(ctx, filePath, &resultCount)
				if err != nil {
					continue // Skip files with errors
				}

				atomic.AddInt64(&filesScanned, 1)
				atomic.AddInt64(&bytesScanned, fileBytes)

				for _, match := range matches {
					select {
					case matchChan <- match:
						atomic.AddInt64(&resultCount, 1)
					case <-ctx.Done():
						return
					}

					// Check max results limit
					if e.config.MaxResults > 0 && int(resultCount) >= e.config.MaxResults {
						return
					}
				}
			}
		}()
	}

	// Walk directory and send file paths to workers
	go func() {
		defer close(filePaths)

		_ = filepath.Walk(e.config.SearchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files with errors
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if e.shouldSkipFile(path, info) {
				if info.IsDir() && !e.config.IncludeHidden && strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}

			select {
			case filePaths <- path:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		// Walk completed - errors are handled individually during the walk
	}()

	// Wait for all workers to finish and close match channel
	go func() {
		wg.Wait()
		close(matchChan)
	}()

	// Collect results
	for match := range matchChan {
		results.Matches = append(results.Matches, match)
		if e.config.MaxResults > 0 && len(results.Matches) >= e.config.MaxResults {
			break
		}
	}

	// Sort results by file path and line number
	sort.Slice(results.Matches, func(i, j int) bool {
		if results.Matches[i].File == results.Matches[j].File {
			return results.Matches[i].Line < results.Matches[j].Line
		}
		return results.Matches[i].File < results.Matches[j].File
	})

	// Update statistics
	results.Stats.Duration = time.Since(startTime)
	results.Stats.FilesScanned = int(filesScanned)
	results.Stats.BytesScanned = bytesScanned
	results.Stats.MatchesFound = len(results.Matches)

	return results, nil
}

// searchFile searches for the pattern in a single file
func (e *SearchEngine) searchFile(ctx context.Context, filePath string, resultCount *int64) ([]SearchMatch, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var matches []SearchMatch
	scanner := bufio.NewScanner(file)
	lineNum := 1
	var lines []string
	var bytesRead int64

	// Store lines for context if needed
	if e.config.ContextLines > 0 {
		lines = make([]string, 0)
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return matches, bytesRead, ctx.Err()
		default:
		}

		line := scanner.Text()
		bytesRead += int64(len(line) + 1) // +1 for newline

		if e.config.ContextLines > 0 {
			lines = append(lines, line)
		}

		var matched bool
		var column int

		if e.literalSearch != "" {
			// Literal string search
			searchLine := line
			if e.config.IgnoreCase {
				searchLine = strings.ToLower(line)
			}
			if idx := strings.Index(searchLine, e.literalSearch); idx >= 0 {
				matched = true
				column = idx + 1 // 1-indexed
			}
		} else if e.pattern != nil {
			// Regex search
			if loc := e.pattern.FindStringIndex(line); loc != nil {
				matched = true
				column = loc[0] + 1 // 1-indexed
			}
		}

		if matched {
			// Check if we've hit the max results limit
			if e.config.MaxResults > 0 && *resultCount >= int64(e.config.MaxResults) {
				break
			}

			match := SearchMatch{
				File:    filePath,
				Line:    lineNum,
				Column:  column,
				Content: line,
			}

			// Add context lines if requested
			if e.config.ContextLines > 0 && len(lines) > 0 {
				start := max(0, len(lines)-e.config.ContextLines-1)
				end := min(len(lines)-1, len(lines)-1+e.config.ContextLines)

				for i := start; i <= end; i++ {
					if i != len(lines)-1 { // Don't include the matched line itself
						match.Context = append(match.Context, lines[i])
					}
				}
			}

			matches = append(matches, match)
		}

		lineNum++
	}

	return matches, bytesRead, scanner.Err()
}

// shouldSkipFile determines if a file should be skipped based on various criteria
func (e *SearchEngine) shouldSkipFile(path string, info os.FileInfo) bool {
	// Skip directories
	if info.IsDir() {
		return true
	}

	// Skip hidden files unless explicitly included
	if !e.config.IncludeHidden && strings.HasPrefix(info.Name(), ".") {
		return true
	}

	// Skip binary files (basic heuristic)
	if isBinaryFile(path) {
		return true
	}

	// Check file pattern
	if e.config.FilePattern != "" {
		matched, _ := filepath.Match(e.config.FilePattern, info.Name())
		if !matched {
			return true
		}
	}

	return false
}

// isLiteralPattern checks if a pattern is a simple literal string
func isLiteralPattern(pattern string) bool {
	// Check for regex metacharacters
	metaChars := []string{".", "*", "+", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|", "\\"}
	for _, char := range metaChars {
		if strings.Contains(pattern, char) {
			return false
		}
	}
	return true
}

// isBinaryFile performs a basic check to determine if a file is binary
func isBinaryFile(path string) bool {
	// Check file extension first
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".obj": true, ".o": true, ".a": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	}
	if binaryExts[ext] {
		return true
	}

	// Quick content check - read first 512 bytes and look for null bytes
	file, err := os.Open(path)
	if err != nil {
		return true // Assume binary if we can't read it
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return true
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true
		}
	}

	return false
}

// Helper functions for min/max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HandleSearchCode implements the search_code tool using GoRipGrep API
func HandleSearchCode(ctx *server.Context, args SearchCodeArgs) (string, error) {
	ctx.Logger.Info("Handling search_code tool call with GoRipGrep implementation")

	// Build options from args
	var options []SearchOption

	if args.IgnoreCase != nil && *args.IgnoreCase {
		options = append(options, WithIgnoreCase())
	}

	if args.ContextLines != nil && *args.ContextLines > 0 {
		options = append(options, WithContextLines(*args.ContextLines))
	}

	if args.FilePattern != nil && *args.FilePattern != "" {
		options = append(options, WithFilePattern(*args.FilePattern))
	}

	if args.MaxResults != nil && *args.MaxResults > 0 {
		options = append(options, WithMaxResults(*args.MaxResults))
	}

	if args.IncludeHidden != nil && *args.IncludeHidden {
		options = append(options, WithHidden())
	}

	if args.TimeoutMs != nil && *args.TimeoutMs > 0 {
		timeout := time.Duration(*args.TimeoutMs) * time.Millisecond
		options = append(options, WithTimeout(timeout))
	}

	// Perform search using GoRipGrep API
	results, err := Find(args.Pattern, args.Path, options...)
	if err != nil {
		if err == context.DeadlineExceeded {
			ctx.Logger.Info("Search timed out", "pattern", args.Pattern)
			return "Search timed out.", nil
		}
		ctx.Logger.Info("Error during search", "error", err, "pattern", args.Pattern)
		return "", fmt.Errorf("search failed: %v", err)
	}

	// Format results in ripgrep-like output format
	if !results.HasMatches() {
		ctx.Logger.Info("Search completed with no matches", "pattern", args.Pattern)
		return "", nil
	}

	var output strings.Builder
	for _, match := range results.Matches {
		// Format: filename:line:content
		output.WriteString(fmt.Sprintf("%s:%d:%s\n", match.File, match.Line, match.Content))

		// Add context lines if available
		if len(match.Context) > 0 {
			for _, contextLine := range match.Context {
				output.WriteString(fmt.Sprintf("%s-%s\n", match.File, contextLine))
			}
		}
	}

	ctx.Logger.Info("Search completed successfully",
		"pattern", args.Pattern,
		"matches", results.Count(),
		"files_scanned", results.Stats.FilesScanned,
		"duration", results.Stats.Duration)

	return strings.TrimSuffix(output.String(), "\n"), nil
}
