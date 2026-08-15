package client

import (
	"context"
	"fmt"
	"net/url"
)

// CloudInitScript is a reusable cloud-init YAML document applied to newly
// provisioned cloud servers.
type CloudInitScript struct {
	ID     int64  `json:"id"`
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Script string `json:"script"`
}

// CloudInitScriptRequest is the create/update body. Script must be valid
// cloud-init YAML (starting with #cloud-config).
type CloudInitScriptRequest struct {
	Name   *string `json:"name,omitempty"`
	Script *string `json:"script,omitempty"`
}

// ListCloudInitScripts returns every stored script.
func (c *Client) ListCloudInitScripts(ctx context.Context) ([]CloudInitScript, error) {
	var out []CloudInitScript
	if err := c.get(ctx, "/cloud-init-scripts", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCloudInitScript fetches one script by UUID.
func (c *Client) GetCloudInitScript(ctx context.Context, uuid string) (*CloudInitScript, error) {
	var out CloudInitScript
	if err := c.get(ctx, "/cloud-init-scripts/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCloudInitScript stores a script and returns it refreshed.
func (c *Client) CreateCloudInitScript(ctx context.Context, req CloudInitScriptRequest) (*CloudInitScript, error) {
	var created uuidResponse
	if err := c.post(ctx, "/cloud-init-scripts", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /cloud-init-scripts: API returned no uuid")
	}
	return c.GetCloudInitScript(ctx, created.UUID)
}

// UpdateCloudInitScript updates a script and returns it refreshed.
func (c *Client) UpdateCloudInitScript(ctx context.Context, uuid string, req CloudInitScriptRequest) (*CloudInitScript, error) {
	if err := c.patch(ctx, "/cloud-init-scripts/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetCloudInitScript(ctx, uuid)
}

// DeleteCloudInitScript removes a script.
func (c *Client) DeleteCloudInitScript(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/cloud-init-scripts/"+url.PathEscape(uuid))
}
