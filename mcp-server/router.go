package main

import (
	"encoding/json"
	"log"

	"edgebridge-mcp/security"
	"edgebridge-mcp/tools"
)

// ══════════════════════════════════════
//  Tool handler interface
// ══════════════════════════════════════

// ToolHandler is implemented by every MCP tool.
type ToolHandler interface {
	Definition() tools.ToolDefinition
	Execute(args map[string]interface{}) tools.ToolResult
}

// ══════════════════════════════════════
//  Router
// ══════════════════════════════════════

// Router dispatches incoming JSON-RPC requests to the correct handler.
type Router struct {
	handlers map[string]ToolHandler
}

// NewRouter creates a Router and registers all built-in tools.
func NewRouter(cfg *Config) *Router {
	sb := security.NewSandbox(cfg.AllowedPaths, cfg.BlockedCommands, cfg.MaxFileSizeMB)

	r := &Router{handlers: make(map[string]ToolHandler)}

	// Filesystem tools
	r.register(&tools.ReadFileTool{Sandbox: sb, MaxSizeMB: cfg.MaxFileSizeMB})
	r.register(&tools.WriteFileTool{Sandbox: sb})
	r.register(&tools.ListDirectoryTool{Sandbox: sb})

	// Shell tool
	r.register(&tools.RunCommandTool{Sandbox: sb, TimeoutSeconds: cfg.CommandTimeoutSeconds})

	// Search tool
	r.register(&tools.SearchFilesTool{Sandbox: sb, MaxResults: cfg.SearchMaxResults})

	log.Printf("[router] registered %d tools", len(r.handlers))
	return r
}

func (r *Router) register(h ToolHandler) {
	name := h.Definition().Name
	r.handlers[name] = h
	log.Printf("[router] + %s", name)
}

// Handle processes a single JSON-RPC request and returns a response.
func (r *Router) Handle(req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {

	// ── MCP initialize ──────────────────────────────────────
	case "initialize":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{"tools": map[string]bool{"listChanged": true}},
				"serverInfo":      map[string]string{"name": "edgebridge-mcp", "version": "1.0.0"},
			},
		}

	// ── List available tools ────────────────────────────────
	case "tools/list":
		defs := make([]tools.ToolDefinition, 0, len(r.handlers))
		for _, h := range r.handlers {
			defs = append(defs, h.Definition())
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"tools": defs},
		}

	// ── Call a tool ─────────────────────────────────────────
	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32602, Message: "invalid params: " + err.Error()},
			}
		}
		handler, ok := r.handlers[params.Name]
		if !ok {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32602, Message: "unknown tool: " + params.Name},
			}
		}
		log.Printf("[router] tools/call %s %v", params.Name, params.Arguments)
		result := handler.Execute(params.Arguments)
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	// ── Unknown method ──────────────────────────────────────
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: "method not found: " + req.Method},
		}
	}
}
