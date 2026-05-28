# EdgeBridge — MCP Server (Go)

A local MCP (Model Context Protocol) server that communicates with a
Chrome/Edge extension via **Native Messaging** (stdin/stdout, 4-byte
length-prefixed JSON frames).

## Quick Start

```bash
cd mcp-server

# Initialise Go modules (first time only)
go mod tidy

# Build
go build -o bridge.exe .
```

The resulting `bridge.exe` is registered as a Native Messaging host for
the Edge extension.

## Project Structure

```
mcp-server/
├── main.go              <- entry point + message loop
├── config.go            <- config.json loader
├── config.json          <- runtime configuration
├── protocol.go          <- JSON-RPC 2.0 types + Native Messaging I/O
├── router.go            <- request dispatcher + tool registry
├── security/
│   └── sandbox.go       <- path & command validation
├── tools/
│   ├── types.go         <- ToolDefinition, ToolResult, ContentBlock
│   ├── filesystem.go    <- read_file, write_file, list_directory
│   ├── shell.go         <- run_command
│   └── search.go        <- search_files
├── go.mod
└── README.md
```

## Configuration (`config.json`)

The config file must sit next to `bridge.exe`.

| Key                       | Type       | Description                                     |
| ------------------------- | ---------- | ----------------------------------------------- |
| `allowed_paths`           | `string[]` | Directories the tools are allowed to access.    |
|                           |            | Supports `%ENV_VAR%` expansion.                 |
| `blocked_commands`        | `string[]` | Sub-strings that cause `run_command` to reject. |
| `max_file_size_mb`        | `int`      | Max file size `read_file` will accept.          |
| `command_timeout_seconds` | `int`      | Timeout for `run_command` execution.            |
| `search_max_results`      | `int`      | Max number of file hits `search_files` returns. |

## Available Tools

### `read_file`

Read the content of a file.

```json
{
	"name": "read_file",
	"arguments": { "path": "C:\\Users\\me\\Projects\\app.js" }
}
```

### `write_file`

Write content to a file (creates parent directories).

```json
{
	"name": "write_file",
	"arguments": { "path": "C:\\Users\\me\\out.txt", "content": "hello" }
}
```

### `list_directory`

List entries in a directory.

```json
{ "name": "list_directory", "arguments": { "path": "C:\\Users\\me\\Projects" } }
```

### `run_command`

Run a shell command via `cmd /c`.

```json
{
	"name": "run_command",
	"arguments": { "command": "dir /b", "cwd": "C:\\Users\\me\\Projects" }
}
```

### `search_files`

Search for text in files (case-insensitive, supports common text file
extensions, skips `.git` and `node_modules`).

```json
{
	"name": "search_files",
	"arguments": { "path": "C:\\Users\\me\\Projects", "pattern": "TODO" }
}
```

## Security

-   **Path sandbox**: all file operations validate that the target path is
    under one of the `allowed_paths`. Path traversal (`..`) is blocked.
-   **Command blocklist**: `run_command` rejects commands containing any
    of the `blocked_commands` sub-strings.
-   **File size limits**: `read_file` refuses files larger than
    `max_file_size_mb`. `search_files` skips files over 1 MB.
-   **Timeout**: `run_command` kills processes that exceed
    `command_timeout_seconds`.

## Logging

All debug output goes to `bridge.log` next to the executable (stdout is
reserved for Native Messaging).

## Next Steps

This server is **Step 1** of the EdgeBridge project. Upcoming
steps:

-   **Step 2** — Edge extension `background.js` with Native Messaging client
-   **Step 3** — Floating MCP panel UI
-   **Step 4** — Prompt enricher with `@mcp` prefix + anti-loop guard
-   **Step 5** — `install.ps1` + `launch.ps1`
-   **Step 6** — Full project `.zip` package
