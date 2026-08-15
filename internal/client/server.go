package client

import (
	"context"
	"fmt"
	"net/url"
)

// Server mirrors the `Server` schema in the Coolify OpenAPI document. Only the
// fields the provider surfaces are declared; the API returns more.
type Server struct {
	ID          int64          `json:"id"`
	UUID        string         `json:"uuid"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	IP          string         `json:"ip"`
	Port        int64          `json:"port"`
	User        string         `json:"user"`
	ProxyType   string         `json:"proxy_type"`
	Settings    *ServerSetting `json:"settings"`
}

// ServerSetting is the nested `settings` object on a Server. Coolify stores most
// mutable server behaviour here even though it is written through PATCH /servers/{uuid}.
type ServerSetting struct {
	ConcurrentBuilds     int64 `json:"concurrent_builds"`
	ConnectionTimeout    int64 `json:"connection_timeout"`
	DeploymentQueueLimit int64 `json:"deployment_queue_limit"`
	DynamicTimeout       int64 `json:"dynamic_timeout"`
	IsBuildServer        bool  `json:"is_build_server"`
	IsReachable          bool  `json:"is_reachable"`
	IsUsable             bool  `json:"is_usable"`
}

// ServerCreateRequest is the body for POST /servers.
type ServerCreateRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	IP              *string `json:"ip,omitempty"`
	Port            *int64  `json:"port,omitempty"`
	User            *string `json:"user,omitempty"`
	PrivateKeyUUID  *string `json:"private_key_uuid,omitempty"`
	IsBuildServer   *bool   `json:"is_build_server,omitempty"`
	InstantValidate *bool   `json:"instant_validate,omitempty"`
	ProxyType       *string `json:"proxy_type,omitempty"`
}

// ServerUpdateRequest is the body for PATCH /servers/{uuid}. It accepts a
// superset of the create fields.
type ServerUpdateRequest struct {
	ServerCreateRequest
	ConcurrentBuilds     *int64 `json:"concurrent_builds,omitempty"`
	ConnectionTimeout    *int64 `json:"connection_timeout,omitempty"`
	DeploymentQueueLimit *int64 `json:"deployment_queue_limit,omitempty"`
	DynamicTimeout       *int64 `json:"dynamic_timeout,omitempty"`
}

// ListServers returns every server visible to the token's team.
func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	var out []Server
	if err := c.get(ctx, "/servers", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetServer fetches a single server by UUID.
func (c *Client) GetServer(ctx context.Context, uuid string) (*Server, error) {
	var out Server
	if err := c.get(ctx, "/servers/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateServer registers an existing machine with Coolify over SSH and returns
// the refreshed server. POST /servers only returns {"uuid": "..."}.
func (c *Client) CreateServer(ctx context.Context, req ServerCreateRequest) (*Server, error) {
	var created uuidResponse
	if err := c.post(ctx, "/servers", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /servers: API returned no uuid")
	}
	return c.GetServer(ctx, created.UUID)
}

// UpdateServer applies a partial update and returns the refreshed server.
func (c *Client) UpdateServer(ctx context.Context, uuid string, req ServerUpdateRequest) (*Server, error) {
	if err := c.patch(ctx, "/servers/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetServer(ctx, uuid)
}

// DeleteServer removes a server from Coolify. It does not destroy the machine.
func (c *Client) DeleteServer(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/servers/"+url.PathEscape(uuid))
}

// ValidateServer asks Coolify to SSH into the server and install/verify Docker.
func (c *Client) ValidateServer(ctx context.Context, uuid string) error {
	return c.post(ctx, "/servers/"+url.PathEscape(uuid)+"/validate", nil, nil)
}
