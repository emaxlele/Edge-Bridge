package tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"edgebridge-mcp/security"
)

// ══════════════════════════════════════
//  run_command
// ══════════════════════════════════════

// RunCommandTool executes a shell command via cmd /c.
type RunCommandTool struct {
	Sandbox        *security.Sandbox
	TimeoutSeconds int
}

func (t *RunCommandTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        "run_command",
		Description: "Execute a shell command on the local machine (Windows cmd)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "The command to execute",
				},
				"cwd": map[string]string{
					"type":        "string",
					"description": "Working directory (optional)",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *RunCommandTool) Execute(args map[string]interface{}) ToolResult {
	command, _ := args["command"].(string)
	cwd, _ := args["cwd"].(string)

	if command == "" {
		return ErrorResult("missing required argument: command")
	}
	if err := t.Sandbox.ValidateCommand(command); err != nil {
		return ErrorResult(err.Error())
	}

	timeout := time.Duration(t.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cmd", "/c", command)
	if cwd != "" {
		cmd.Dir = cwd
	}

	output, err := cmd.CombinedOutput()
	result := string(output)

	if ctx.Err() == context.DeadlineExceeded {
		return ErrorResult(fmt.Sprintf("command timed out after %d seconds\n%s", t.TimeoutSeconds, result))
	}
	if err != nil {
		return ToolResult{
			Content: []ContentBlock{{Type: "text", Text: result + "\nexit error: " + err.Error()}},
			IsError: true,
		}
	}
	return TextResult(result)
}
