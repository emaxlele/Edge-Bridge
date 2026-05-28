package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"edgebridge-mcp/security"
)

// ══════════════════════════════════════
//  read_file
// ══════════════════════════════════════

// ReadFileTool reads the contents of a local file.
type ReadFileTool struct {
	Sandbox   *security.Sandbox
	MaxSizeMB int
}

func (t *ReadFileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "read_file",
		Description: "Read the contents of a file from the local filesystem",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]string{
					"type":        "string",
					"description": "Absolute or relative path of the file to read",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ReadFileTool) Execute(args map[string]interface{}) ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		return ErrorResult("missing required argument: path")
	}
	if err := t.Sandbox.ValidatePath(path); err != nil {
		return ErrorResult(err.Error())
	}

	info, err := os.Stat(path)
	if err != nil {
		return ErrorResult("cannot stat file: " + err.Error())
	}
	maxBytes := int64(t.MaxSizeMB) * 1024 * 1024
	if info.Size() > maxBytes {
		return ErrorResult(fmt.Sprintf("file is too large (%d bytes, max %d MB)", info.Size(), t.MaxSizeMB))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ErrorResult("cannot read file: " + err.Error())
	}
	return TextResult(string(data))
}

// ══════════════════════════════════════
//  write_file
// ══════════════════════════════════════

// WriteFileTool writes content to a local file.
type WriteFileTool struct {
	Sandbox *security.Sandbox
}

func (t *WriteFileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "write_file",
		Description: "Write content to a file (creates parent directories if needed)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]string{
					"type":        "string",
					"description": "Absolute or relative path of the file to write",
				},
				"content": map[string]string{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

func (t *WriteFileTool) Execute(args map[string]interface{}) ToolResult {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return ErrorResult("missing required argument: path")
	}
	if err := t.Sandbox.ValidatePath(path); err != nil {
		return ErrorResult(err.Error())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return ErrorResult("cannot create directories: " + err.Error())
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return ErrorResult("cannot write file: " + err.Error())
	}
	return TextResult(fmt.Sprintf("file written: %s (%d bytes)", path, len(content)))
}

// ══════════════════════════════════════
//  list_directory
// ══════════════════════════════════════

// ListDirectoryTool lists entries in a local directory.
type ListDirectoryTool struct {
	Sandbox *security.Sandbox
}

func (t *ListDirectoryTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "list_directory",
		Description: "List files and sub-directories in a given path",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]string{
					"type":        "string",
					"description": "Absolute or relative path of the directory to list",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ListDirectoryTool) Execute(args map[string]interface{}) ToolResult {
	path, _ := args["path"].(string)
	if path == "" {
		return ErrorResult("missing required argument: path")
	}
	if err := t.Sandbox.ValidatePath(path); err != nil {
		return ErrorResult(err.Error())
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ErrorResult("cannot read directory: " + err.Error())
	}

	var lines []string
	for _, e := range entries {
		if e.IsDir() {
			lines = append(lines, fmt.Sprintf("[DIR]  %s", e.Name()))
		} else {
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			lines = append(lines, fmt.Sprintf("[FILE] %s (%d bytes)", e.Name(), size))
		}
	}
	if len(lines) == 0 {
		return TextResult("(empty directory)")
	}
	return TextResult(strings.Join(lines, "\n"))
}
