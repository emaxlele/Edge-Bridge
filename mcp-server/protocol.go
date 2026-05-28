package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ══════════════════════════════════════
//  JSON-RPC 2.0 types
// ══════════════════════════════════════

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ══════════════════════════════════════
//  MCP-specific types
// ══════════════════════════════════════

// ToolCallParams is the "params" payload for a "tools/call" request.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ══════════════════════════════════════
//  Native Messaging I/O
// ══════════════════════════════════════

const maxMessageSize = 1 * 1024 * 1024 // 1 MB safety limit

// readNativeMessage reads a single Chrome Native Messaging frame from
// stdin (4-byte little-endian length prefix + JSON payload).
func readNativeMessage() ([]byte, error) {
	var length uint32
	if err := binary.Read(os.Stdin, binary.LittleEndian, &length); err != nil {
		return nil, err // io.EOF when the browser closes the pipe
	}
	if length == 0 || length > maxMessageSize {
		return nil, fmt.Errorf("invalid message length: %d", length)
	}
	msg := make([]byte, length)
	if _, err := io.ReadFull(os.Stdin, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// writeNativeResponse serialises a JSONRPCResponse and writes it to
// stdout using the Native Messaging wire format.
func writeNativeResponse(resp JSONRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if err := binary.Write(os.Stdout, binary.LittleEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
