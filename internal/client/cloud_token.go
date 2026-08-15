package client

import (
	"context"
	"fmt"
	"net/url"
)

// CloudProviders lists the VM providers Coolify can provision through.
var CloudProviders = []string{"hetzner", "digitalocean", "vultr"}

// CloudToken is an API token for a cloud provider, used to provision servers.
type CloudToken struct {
	ID           int64  `json:"id"`
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Token        string `json:"token"`
	ServersCount int64  `json:"servers_count"`
}

// CloudTokenRequest is the create/update body.
type CloudTokenRequest struct {
	Name     *string `json:"name,omitempty"`
	Provider *string `json:"provider,omitempty"`
	Token    *string `json:"token,omitempty"`
}

// ListCloudTokens returns every cloud provider token of the team.
func (c *Client) ListCloudTokens(ctx context.Context) ([]CloudToken, error) {
	var out []CloudToken
	if err := c.get(ctx, "/cloud-tokens", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCloudToken fetches one token by UUID.
func (c *Client) GetCloudToken(ctx context.Context, uuid string) (*CloudToken, error) {
	var out CloudToken
	if err := c.get(ctx, "/cloud-tokens/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCloudToken stores a provider token and returns it refreshed.
func (c *Client) CreateCloudToken(ctx context.Context, req CloudTokenRequest) (*CloudToken, error) {
	var created uuidResponse
	if err := c.post(ctx, "/cloud-tokens", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /cloud-tokens: API returned no uuid")
	}
	return c.GetCloudToken(ctx, created.UUID)
}

// UpdateCloudToken updates a token and returns it refreshed.
func (c *Client) UpdateCloudToken(ctx context.Context, uuid string, req CloudTokenRequest) (*CloudToken, error) {
	if err := c.patch(ctx, "/cloud-tokens/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetCloudToken(ctx, uuid)
}

// DeleteCloudToken removes a token. Tokens still referenced by provisioned
// servers are rejected by the API.
func (c *Client) DeleteCloudToken(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/cloud-tokens/"+url.PathEscape(uuid))
}

// ValidateCloudToken asks Coolify to verify the token against the provider API.
func (c *Client) ValidateCloudToken(ctx context.Context, uuid string) error {
	return c.post(ctx, "/cloud-tokens/"+url.PathEscape(uuid)+"/validate", nil, nil)
}
