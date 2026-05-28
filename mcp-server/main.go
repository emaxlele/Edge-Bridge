package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// ── Setup logging to bridge.log (stdout is reserved for Native Messaging) ──
	exePath, _ := os.Executable()
	logPath := filepath.Join(filepath.Dir(exePath), "bridge.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("════════════════════════════════════════")
	log.Println("  edgebridge-mcp starting")
	log.Println("════════════════════════════════════════")

	// ── Load config ─────────────────────────────────────────
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("[main] config error: %v — using defaults", err)
		cfg = &Config{
			AllowedPaths:          []string{},
			BlockedCommands:       []string{},
			MaxFileSizeMB:         10,
			CommandTimeoutSeconds: 30,
			SearchMaxResults:      50,
		}
	}
	log.Printf("[main] config loaded: %d allowed paths, %d blocked commands",
		len(cfg.AllowedPaths), len(cfg.BlockedCommands))

	// ── Create router ───────────────────────────────────────
	router := NewRouter(cfg)

	// ── Main message loop ───────────────────────────────────
	log.Println("[main] entering message loop")
	for {
		msg, err := readNativeMessage()
		if err != nil {
			if err == io.EOF {
				log.Println("[main] stdin EOF — browser disconnected, exiting")
				break
			}
			log.Printf("[main] read error: %v", err)
			_ = writeNativeResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      0,
				Error:   &RPCError{Code: -32700, Message: "parse error: " + err.Error()},
			})
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			log.Printf("[main] JSON unmarshal error: %v", err)
			_ = writeNativeResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      0,
				Error:   &RPCError{Code: -32700, Message: "invalid JSON: " + err.Error()},
			})
			continue
		}

		log.Printf("[main] <- %s (id=%d)", req.Method, req.ID)

		resp := router.Handle(req)

		if err := writeNativeResponse(resp); err != nil {
			log.Printf("[main] write error: %v", err)
			break
		}
		log.Printf("[main] -> response sent (id=%d)", resp.ID)
	}

	log.Println("[main] exiting")
}
