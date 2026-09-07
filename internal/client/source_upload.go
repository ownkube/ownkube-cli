package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/ownkube/okctl/internal/api"
)

// This file wraps the `okctl up` local-source flow: presign an upload slot,
// PUT the working-tree tarball straight to object storage, then trigger an
// in-cluster build+deploy of the uploaded source. Follows the same wrapper
// pattern as deployment_write.go (WithResponse → checkError → guard payload),
// except UploadSource, which talks to the pre-signed URL directly (no API key).

// PresignSourceUpload calls POST /v1/source-uploads, minting an upload slot for
// a local-source deploy. The returned UploadUrl is a short-lived pre-signed URL
// the caller PUTs its tarball to; UploadId is echoed back to DeployFromUpload.
func (c *Client) PresignSourceUpload(ctx context.Context) (*api.SourceUploadResponse, error) {
	resp, err := c.inner.PostV1SourceUploadsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromSourceUpload(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UploadSource PUTs the gzipped source tarball to a pre-signed URL. The URL is
// self-authenticating, so this does NOT go through the API client — no API key,
// no org header, just the exact Content-Type the presign was signed with and an
// explicit Content-Length (size) so the object-store validates the body.
func (c *Client) UploadSource(ctx context.Context, uploadURL, contentType string, body io.Reader, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, body)
	if err != nil {
		return fmt.Errorf("building upload request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = size

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("source upload failed (status %d): %s", resp.StatusCode, snippet)
	}
	return nil
}

// DeployFromUpload calls POST /v1/deployments/{id}/up, building and deploying
// source previously uploaded via PresignSourceUpload/UploadSource. note is
// recorded on the queued revision when set. Returns the new revision id.
func (c *Client) DeployFromUpload(ctx context.Context, deploymentID, uploadID string, note *string) (*api.UpResult, error) {
	parsed, err := uuid.Parse(uploadID)
	if err != nil {
		return nil, fmt.Errorf("invalid upload id %q: %w", uploadID, err)
	}
	body := api.UpBody{UploadId: parsed, Note: note}
	resp, err := c.inner.PostV1DeploymentsDeploymentIdUpWithResponse(ctx, deploymentID, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromDeployFromUpload(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromSourceUpload(r *api.PostV1SourceUploadsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromDeployFromUpload(r *api.PostV1DeploymentsDeploymentIdUpResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
