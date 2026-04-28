//go:build ignore

// mcp_echo is a test helper that reads JSON-RPC requests from stdin and
// writes correlated responses to stdout.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		method, _ := req["method"].(string)
		id, _ := req["id"].(float64)

		var result interface{}
		switch method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"serverInfo": map[string]interface{}{
					"name":    "test-server",
					"version": "1.0.0",
				},
			}
		case "tools/list":
			result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "echo",
						"description": "Echo input",
						"inputSchema": map[string]interface{}{"type": "object"},
					},
				},
			}
		case "tools/call":
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "done"},
				},
			}
		case "resources/list":
			result = map[string]interface{}{
				"resources": []map[string]interface{}{
					{"uri": "file:///test.txt", "name": "test"},
				},
			}
		case "resources/read":
			result = map[string]interface{}{
				"contents": []map[string]interface{}{
					{"uri": "file:///test.txt", "text": "hello world"},
				},
			}
		default:
			result = map[string]interface{}{"ok": true}
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		}

		data, _ := json.Marshal(resp)
		fmt.Println(string(data))
	}
}
