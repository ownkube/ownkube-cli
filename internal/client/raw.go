package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ownkube/okctl/internal/api"
)

// doJSON issues a hand-written request against the CLI API for endpoints the
// generated client does not cover yet. It sends the same headers as the
// generated client, decodes a 2xx body into out (when non-nil), and maps
// error bodies through checkError. It returns the HTTP status so callers can
// tell idempotent outcomes (200 vs 201) apart.
func (c *Client) doJSON(ctx context.Context, method, path string, body, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("encoding request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+"/api/cli"+path, reader)
	if err != nil {
		return 0, fmt.Errorf("building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if err := c.edit(ctx, req); err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		var env api.ErrorResponse
		if json.Unmarshal(raw, &env) == nil && env.Code != "" && env.Code != "ORGANIZATION_REQUIRED" {
			return resp.StatusCode, &APIError{Status: resp.StatusCode, Code: env.Code, Message: env.Error}
		}
		return resp.StatusCode, checkError(resp.StatusCode, nil, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, unexpectedStatus(resp.StatusCode, raw)
		}
	}
	return resp.StatusCode, nil
}

// APIError is a structured error envelope from a hand-written request, so
// callers can branch on Code. Its message matches checkError's format.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d (%s): %s", e.Status, e.Code, e.Message)
}

// IsAPICode reports whether err is an APIError carrying the given code.
func IsAPICode(err error, code string) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == code
}
