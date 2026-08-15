package client

import (
	"context"
	"fmt"
	"net/url"
)

// Environment mirrors the `Environment` schema. Environments belong to a
// project and are addressed as /projects/{project}/environments/{name_or_uuid}.
type Environment struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ProjectID   int64  `json:"project_id"`
}

// EnvironmentRequest is the body for create and update.
type EnvironmentRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func environmentPath(projectUUID, nameOrUUID string) string {
	return "/projects/" + url.PathEscape(projectUUID) + "/environments/" + url.PathEscape(nameOrUUID)
}

// ListEnvironments returns every environment of a project.
func (c *Client) ListEnvironments(ctx context.Context, projectUUID string) ([]Environment, error) {
	var out []Environment
	if err := c.get(ctx, "/projects/"+url.PathEscape(projectUUID)+"/environments", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetEnvironment fetches one environment by name or UUID.
func (c *Client) GetEnvironment(ctx context.Context, projectUUID, nameOrUUID string) (*Environment, error) {
	var out Environment
	// GET /projects/{uuid}/{environment_name_or_uuid} is the details endpoint.
	if err := c.get(ctx, "/projects/"+url.PathEscape(projectUUID)+"/"+url.PathEscape(nameOrUUID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateEnvironment creates an environment in a project. A duplicate name
// yields HTTP 409.
func (c *Client) CreateEnvironment(ctx context.Context, projectUUID string, req EnvironmentRequest) (*Environment, error) {
	var created uuidResponse
	if err := c.post(ctx, "/projects/"+url.PathEscape(projectUUID)+"/environments", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST environments: API returned no uuid")
	}
	return c.GetEnvironment(ctx, projectUUID, created.UUID)
}

// UpdateEnvironment renames/redescribes an environment.
func (c *Client) UpdateEnvironment(ctx context.Context, projectUUID, nameOrUUID string, req EnvironmentRequest) (*Environment, error) {
	if err := c.patch(ctx, environmentPath(projectUUID, nameOrUUID), req, nil); err != nil {
		return nil, err
	}
	target := nameOrUUID
	if req.Name != nil {
		target = *req.Name
	}
	return c.GetEnvironment(ctx, projectUUID, target)
}

// DeleteEnvironment removes an empty environment; the API rejects deletion when
// resources still live in it.
func (c *Client) DeleteEnvironment(ctx context.Context, projectUUID, nameOrUUID string) error {
	return c.delete(ctx, environmentPath(projectUUID, nameOrUUID))
}
