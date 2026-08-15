package client

import (
	"context"
	"net/url"
)

// Destination is a Docker network on a server that resources deploy into.
type Destination struct {
	ID      int64  `json:"id"`
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Network string `json:"network"`
	// Server is included by the API on create/show (id + uuid only).
	Server *struct {
		UUID string `json:"uuid"`
	} `json:"server,omitempty"`
	ServerUUID string `json:"server_uuid,omitempty"`
}

// DestinationCreateRequest is the body for POST /servers/{server_uuid}/destinations.
// type defaults server-side to standalone (or swarm on swarm servers).
type DestinationCreateRequest struct {
	Name    *string `json:"name,omitempty"`
	Network *string `json:"network,omitempty"`
	Type    *string `json:"type,omitempty"`
}

// DestinationUpdateRequest is the body for PATCH /destinations/{uuid}.
type DestinationUpdateRequest struct {
	Name *string `json:"name,omitempty"`
}

// ServerRef returns the server reference regardless of response shape.
func (d *Destination) ServerRef() string {
	if d.Server != nil && d.Server.UUID != "" {
		return d.Server.UUID
	}
	return d.ServerUUID
}

// ListDestinations returns every destination of the token's team.
func (c *Client) ListDestinations(ctx context.Context) ([]Destination, error) {
	var out []Destination
	if err := c.get(ctx, "/destinations", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListServerDestinations returns the destinations of a single server.
func (c *Client) ListServerDestinations(ctx context.Context, serverUUID string) ([]Destination, error) {
	var out []Destination
	if err := c.get(ctx, "/servers/"+url.PathEscape(serverUUID)+"/destinations", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDestination fetches one destination by UUID.
func (c *Client) GetDestination(ctx context.Context, uuid string) (*Destination, error) {
	var out Destination
	if err := c.get(ctx, "/destinations/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateDestination creates a Docker network destination on a server. Unlike
// most create endpoints this one returns the full object (HTTP 201).
func (c *Client) CreateDestination(ctx context.Context, serverUUID string, req DestinationCreateRequest) (*Destination, error) {
	var out Destination
	if err := c.post(ctx, "/servers/"+url.PathEscape(serverUUID)+"/destinations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDestination renames a destination.
func (c *Client) UpdateDestination(ctx context.Context, uuid string, req DestinationUpdateRequest) (*Destination, error) {
	if err := c.patch(ctx, "/destinations/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetDestination(ctx, uuid)
}

// DeleteDestination removes a destination; the API rejects deletion while
// resources still use the network.
func (c *Client) DeleteDestination(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/destinations/"+url.PathEscape(uuid))
}
