package client

import (
	"context"
	"net/http"
	"net/url"
)

// Hand-written wrappers for the SSH access endpoints. Replace with generated
// calls once api/openapi.json carries them.

// SSHKey is a public key registered to the calling user.
type SSHKey struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Algorithm   string  `json:"algorithm"`
	Fingerprint string  `json:"fingerprint"`
	Source      string  `json:"source"`
	CreatedAt   string  `json:"createdAt"`
	LastUsedAt  *string `json:"lastUsedAt"`
}

// SSHInstance is one running copy of a deployment, addressable as +Index.
type SSHInstance struct {
	Index int  `json:"index"`
	Ready bool `json:"ready"`
}

// SSHTarget is everything needed to dial a deployment over SSH.
type SSHTarget struct {
	DeploymentID string        `json:"deploymentId"`
	Name         string        `json:"name"`
	Kind         string        `json:"kind"`
	Region       string        `json:"region"`
	SSHHost      string        `json:"sshHost"`
	SSHPort      int           `json:"sshPort"`
	Username     string        `json:"username"`
	Instances    []SSHInstance `json:"instances"`
	HostKeys     []string      `json:"hostKeys"`
}

// ConnectRegion is one region's SSH and browser-shell front door.
type ConnectRegion struct {
	Region     string   `json:"region"`
	Label      string   `json:"label"`
	SSHHost    string   `json:"sshHost"`
	SSHPort    int      `json:"sshPort"`
	ConnectURL string   `json:"connectUrl"`
	HostKeys   []string `json:"hostKeys"`
}

// ListSSHKeys calls GET /v1/ssh-keys.
func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	var out struct {
		Keys []SSHKey `json:"keys"`
	}
	if _, err := c.doJSON(ctx, http.MethodGet, "/v1/ssh-keys", nil, &out); err != nil {
		return nil, err
	}
	return out.Keys, nil
}

// AddSSHKey calls POST /v1/ssh-keys. created is false when the key was already
// registered to this user (the endpoint is idempotent for the same user).
func (c *Client) AddSSHKey(ctx context.Context, name, publicKey string) (key *SSHKey, created bool, err error) {
	body := map[string]string{"name": name, "publicKey": publicKey}
	var out struct {
		SSHKey
		Created *bool `json:"created"`
	}
	status, err := c.doJSON(ctx, http.MethodPost, "/v1/ssh-keys", body, &out)
	if err != nil {
		return nil, false, err
	}
	// The API answers 200 with an explicit created flag; 201 is the fallback.
	created = status == http.StatusCreated
	if out.Created != nil {
		created = *out.Created
	}
	return &out.SSHKey, created, nil
}

// RemoveSSHKey calls DELETE /v1/ssh-keys/{id}.
func (c *Client) RemoveSSHKey(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/v1/ssh-keys/"+url.PathEscape(id), nil, nil)
	return err
}

// ImportGitHubSSHKeys calls POST /v1/ssh-keys/github. An empty username uses
// the GitHub account linked to the caller.
func (c *Client) ImportGitHubSSHKeys(ctx context.Context, username string) ([]SSHKey, error) {
	body := map[string]any{"username": nil}
	if username != "" {
		body["username"] = username
	}
	var out struct {
		Keys []SSHKey `json:"keys"`
	}
	if _, err := c.doJSON(ctx, http.MethodPost, "/v1/ssh-keys/github", body, &out); err != nil {
		return nil, err
	}
	return out.Keys, nil
}

// GetSSHTarget calls GET /v1/deployments/{id}/ssh-target.
func (c *Client) GetSSHTarget(ctx context.Context, deploymentID string) (*SSHTarget, error) {
	var out SSHTarget
	path := "/v1/deployments/" + url.PathEscape(deploymentID) + "/ssh-target"
	if _, err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListConnectRegions calls GET /v1/connect/regions.
func (c *Client) ListConnectRegions(ctx context.Context) ([]ConnectRegion, error) {
	var out struct {
		Regions []ConnectRegion `json:"regions"`
	}
	if _, err := c.doJSON(ctx, http.MethodGet, "/v1/connect/regions", nil, &out); err != nil {
		return nil, err
	}
	return out.Regions, nil
}
