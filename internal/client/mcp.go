package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// MCPSessionHeader carries the MCP session id, set by the server on the
// initialize response and echoed on every later request.
const MCPSessionHeader = "Mcp-Session-Id"

// MCPEndpoint is the Ownkube MCP server's URL.
func (c *Client) MCPEndpoint() string { return c.apiURL + "/api/mcp" }

// MCP sends one JSON-RPC message to the Ownkube MCP server with okctl's
// credentials and returns the response as-is: JSON, an event stream, or 202
// with no body for a notification. method is POST for a message, or DELETE
// (with a nil body) to end the session. The caller closes the body.
func (c *Client) MCP(ctx context.Context, method string, body []byte, sessionID string) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.MCPEndpoint(), reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set(MCPSessionHeader, sessionID)
	}
	if err := c.edit(ctx, req); err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MCP request failed: %w", err)
	}
	return resp, nil
}
