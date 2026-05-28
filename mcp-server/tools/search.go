package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"edgebridge-mcp/security"
)

// ══════════════════════════════════════
//  search_files
// ══════════════════════════════════════

// SearchFilesTool searches for a text pattern in files.
type SearchFilesTool struct {
	Sandbox    *security.Sandbox
	MaxResults int
}

func (t *SearchFilesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "search_files",
		Description: "Search for a text pattern within files in a directory tree (case-insensitive)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]string{
					"type":        "string",
					"description": "Root directory to search in",
				},
				"pattern": map[string]string{
					"type":        "string",
					"description": "Text pattern to search for",
				},
			},
			"required": []string{"path", "pattern"},
		},
	}
}

// Text file extensions that we search inside.
var searchableExtensions = map[string]bool{
	".txt": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".go": true, ".py": true, ".md": true, ".json": true, ".css": true,
	".html": true, ".xml": true, ".yaml": true, ".yml": true, ".toml": true,
	".cfg": true, ".ini": true, ".sh": true, ".bat": true, ".ps1": true,
	".sql": true, ".env": true, ".log": true,
}

// Directories to skip during search.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "__pycache__": true,
	".idea": true, ".vscode": true,
}

const maxSearchFileSize = 1 * 1024 * 1024 // 1 MB

func (t *SearchFilesTool) Execute(args map[string]interface{}) ToolResult {
	searchPath, _ := args["path"].(string)
	pattern, _ := args["pattern"].(string)

	if searchPath == "" || pattern == "" {
		return ErrorResult("missing required arguments: path and pattern")
	}
	if err := t.Sandbox.ValidatePath(searchPath); err != nil {
		return ErrorResult(err.Error())
	}

	lowerPattern := strings.ToLower(pattern)
	var matches []string
	hitCount := 0

	filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if hitCount >= t.MaxResults {
			return filepath.SkipAll
		}

		// Skip excluded directories.
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Only search known text extensions.
		ext := strings.ToLower(filepath.Ext(path))
		if !searchableExtensions[ext] {
			return nil
		}
		// Skip large files.
		if info.Size() > maxSearchFileSize {
			return nil
		}

		// Scan file line by line.
		lineNums := searchInFile(path, lowerPattern)
		if len(lineNums) > 0 {
			relPath, _ := filepath.Rel(searchPath, path)
			if relPath == "" {
				relPath = path
			}
			matches = append(matches, fmt.Sprintf("%s  (lines: %v)", relPath, lineNums))
			hitCount++
		}
		return nil
	})

	if len(matches) == 0 {
		return TextResult("no matches found for: " + pattern)
	}
	header := fmt.Sprintf("found in %d file(s):\n", len(matches))
	return TextResult(header + strings.Join(matches, "\n"))
}

// searchInFile returns a slice of 1-based line numbers that contain the
// pattern (already lowered).
func searchInFile(path, lowerPattern string) []int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var hits []int
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if strings.Contains(strings.ToLower(scanner.Text()), lowerPattern) {
			hits = append(hits, lineNo)
			if len(hits) >= 20 { // cap per file to avoid huge output
				break
			}
		}
	}
	return hits
}
