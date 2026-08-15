package client

import (
	"context"
	"fmt"
	"net/url"
)

// PrivateKey mirrors the `PrivateKey` schema. The key material is returned by
// the API only when the token has the `read:sensitive` ability.
type PrivateKey struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PrivateKey  string `json:"private_key"`
	Fingerprint string `json:"fingerprint"`
}

// PrivateKeyRequest is the body for POST /security/keys and
// PATCH /security/keys/{uuid}. private_key accepts raw PEM or base64.
type PrivateKeyRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	PrivateKey  *string `json:"private_key,omitempty"`
}

// ListPrivateKeys returns every private key of the token's team.
func (c *Client) ListPrivateKeys(ctx context.Context) ([]PrivateKey, error) {
	var out []PrivateKey
	if err := c.get(ctx, "/security/keys", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPrivateKey fetches a single key by UUID.
func (c *Client) GetPrivateKey(ctx context.Context, uuid string) (*PrivateKey, error) {
	var out PrivateKey
	if err := c.get(ctx, "/security/keys/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreatePrivateKey registers a key. Coolify rejects keys whose fingerprint
// already exists on the instance (422).
func (c *Client) CreatePrivateKey(ctx context.Context, req PrivateKeyRequest) (*PrivateKey, error) {
	var created uuidResponse
	if err := c.post(ctx, "/security/keys", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /security/keys: API returned no uuid")
	}
	return c.GetPrivateKey(ctx, created.UUID)
}

// UpdatePrivateKey applies a partial update and returns the refreshed key.
func (c *Client) UpdatePrivateKey(ctx context.Context, uuid string, req PrivateKeyRequest) (*PrivateKey, error) {
	if err := c.patch(ctx, "/security/keys/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetPrivateKey(ctx, uuid)
}

// DeletePrivateKey removes a key. Keys still referenced by a server are
// rejected by the API.
func (c *Client) DeletePrivateKey(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/security/keys/"+url.PathEscape(uuid))
}
