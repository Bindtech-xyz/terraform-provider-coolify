package client

import (
	"context"
	"fmt"
	"net/url"
)

// Project mirrors the `Project` schema in the Coolify OpenAPI document.
type Project struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ProjectRequest is the body for both create (POST /projects) and update
// (PATCH /projects/{uuid}). Fields are pointers so an unset field is omitted
// rather than sent as an empty string, which Coolify would treat as a clear.
type ProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// uuidResponse is the shape Coolify returns from most create endpoints: the new
// object's UUID only, with no other attributes. Callers must follow up with a
// read to populate computed values.
type uuidResponse struct {
	UUID string `json:"uuid"`
}

// ListProjects returns every project visible to the token's team.
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out []Project
	if err := c.get(ctx, "/projects", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetProject fetches a single project by UUID.
func (c *Client) GetProject(ctx context.Context, uuid string) (*Project, error) {
	var out Project
	if err := c.get(ctx, "/projects/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateProject creates a project and returns it, re-reading to pick up the
// server-assigned id. POST /projects only returns {"uuid": "..."}.
func (c *Client) CreateProject(ctx context.Context, req ProjectRequest) (*Project, error) {
	var created uuidResponse
	if err := c.post(ctx, "/projects", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /projects: API returned no uuid")
	}
	return c.GetProject(ctx, created.UUID)
}

// UpdateProject applies a partial update and returns the refreshed project.
func (c *Client) UpdateProject(ctx context.Context, uuid string, req ProjectRequest) (*Project, error) {
	if err := c.patch(ctx, "/projects/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetProject(ctx, uuid)
}

// DeleteProject removes a project. Deleting a project with resources still in it
// is rejected by the API with a 4xx.
func (c *Client) DeleteProject(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/projects/"+url.PathEscape(uuid))
}
