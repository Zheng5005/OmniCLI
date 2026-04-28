// mock_server is a simple stdio MCP server for integration and E2E testing.
// It reads JSON-RPC requests from stdin and writes responses to stdout.
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
					"name":    "mock-server",
					"version": "1.0.0",
				},
			}
		case "tools/list":
			result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "echo",
						"description": "Echoes the input back",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"msg": map[string]interface{}{"type": "string"},
							},
						},
					},
					{
						"name":        "add",
						"description": "Adds two numbers",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"a": map[string]interface{}{"type": "number"},
								"b": map[string]interface{}{"type": "number"},
							},
						},
					},
				},
			}
		case "tools/call":
			params, _ := req["params"].(map[string]interface{})
			name := ""
			if params != nil {
				if n, ok := params["name"].(string); ok {
					name = n
				}
			}
			var text string
			switch name {
			case "echo":
				args, _ := params["arguments"].(map[string]interface{})
				msg, _ := args["msg"].(string)
				text = msg
			case "add":
				args, _ := params["arguments"].(map[string]interface{})
				a, _ := args["a"].(float64)
				b, _ := args["b"].(float64)
				text = fmt.Sprintf("%.0f", a+b)
			default:
				text = "unknown tool"
			}
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": text},
				},
			}
		case "resources/list":
			result = map[string]interface{}{
				"resources": []map[string]interface{}{
					{"uri": "mock://readme", "name": "readme", "description": "Readme file"},
				},
			}
		case "resources/read":
			params, _ := req["params"].(map[string]interface{})
			uri := ""
			if params != nil {
				if u, ok := params["uri"].(string); ok {
					uri = u
				}
			}
			var content string
			switch uri {
			case "mock://readme":
				content = "Hello from mock server"
			default:
				content = "unknown resource"
			}
			result = map[string]interface{}{
				"contents": []map[string]interface{}{
					{"uri": uri, "text": content},
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
