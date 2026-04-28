package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONRPCRequestMarshal(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", got["jsonrpc"])
	}
	if got["id"] != float64(1) {
		t.Errorf("id = %v, want 1", got["id"])
	}
	if got["method"] != "initialize" {
		t.Errorf("method = %v, want initialize", got["method"])
	}
}

func TestJSONRPCResponseUnmarshal(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != 1 {
		t.Errorf("id = %d, want 1", resp.ID)
	}
	if resp.Result == nil {
		t.Error("result is nil")
	}
	if resp.Error != nil {
		t.Errorf("error = %v, want nil", resp.Error)
	}
}

func TestJSONRPCErrorUnmarshal(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`)
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("error is nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("code = %d, want -32601", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("message = %q, want Method not found", resp.Error.Message)
	}
	if resp.Error.Error() != "JSON-RPC error -32601: Method not found" {
		t.Errorf("Error() = %q", resp.Error.Error())
	}
}

func TestServerConfigRequestTimeout(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
		want   int
	}{
		{"default zero", ServerConfig{}, 30},
		{"explicit", ServerConfig{Timeout: 60}, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.RequestTimeout(); got != tt.want {
				t.Errorf("RequestTimeout() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMcpToolDefMarshal(t *testing.T) {
	tool := McpToolDef{
		Name:        "query",
		Description: "Run a query",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got McpToolDef
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != "query" {
		t.Errorf("name = %q, want query", got.Name)
	}
}

func TestCallToolResultMarshal(t *testing.T) {
	result := CallToolResult{
		Content: []ToolContent{
			{Type: "text", Text: "hello"},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CallToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(got.Content) != 1 {
		t.Fatalf("content len = %d, want 1", len(got.Content))
	}
	if got.Content[0].Text != "hello" {
		t.Errorf("text = %q, want hello", got.Content[0].Text)
	}
}

func TestNotificationMarshal(t *testing.T) {
	n := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		Params:  json.RawMessage(`{}`),
	}
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), `"id"`) {
		t.Error("notification should not contain id field")
	}
	if !strings.Contains(string(data), `"method"`) {
		t.Error("expected method field")
	}
}

func TestJSONRPCRequestOmitID(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notify",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), `"id"`) {
		t.Error("expected id to be omitted when zero")
	}
}

func TestJSONRPCErrorWithData(t *testing.T) {
	errObj := JSONRPCError{Code: -1, Message: "oops", Data: map[string]interface{}{"detail": "xyz"}}
	got := errObj.Error()
	want := "JSON-RPC error -1: oops"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	data, _ := json.Marshal(errObj)
	if !strings.Contains(string(data), `"detail"`) {
		t.Error("expected data to be marshaled")
	}
}

func TestIDGenerator(t *testing.T) {
	var g idGenerator
	v1 := g.Next()
	v2 := g.Next()
	if v1 != 1 {
		t.Errorf("first id = %d, want 1", v1)
	}
	if v2 != 2 {
		t.Errorf("second id = %d, want 2", v2)
	}
}

func TestMcpResourceMarshal(t *testing.T) {
	r := McpResource{URI: "file:///x", Name: "x", MimeType: "text/plain"}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"uri"`) {
		t.Error("expected uri field")
	}
	if !strings.Contains(string(data), `"mimeType"`) {
		t.Error("expected mimeType field")
	}
}

func TestReadResourceResultMarshal(t *testing.T) {
	result := ReadResourceResult{
		Contents: []ResourceContent{
			{URI: "file:///x", Text: "hello"},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ReadResourceResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Contents) != 1 {
		t.Fatalf("len = %d, want 1", len(got.Contents))
	}
	if got.Contents[0].Text != "hello" {
		t.Errorf("text = %q, want hello", got.Contents[0].Text)
	}
}
