package client

import (
	"context"
	"fmt"
	"net/url"
)

// EnvVarParent is the kind of resource an environment variable belongs to.
// The three parents expose the same /{parent}/{uuid}/envs contract.
type EnvVarParent string

const (
	EnvVarParentApplication EnvVarParent = "application"
	EnvVarParentService     EnvVarParent = "service"
	EnvVarParentDatabase    EnvVarParent = "database"
)

var envVarParentPaths = map[EnvVarParent]string{
	EnvVarParentApplication: "/applications/",
	EnvVarParentService:     "/services/",
	EnvVarParentDatabase:    "/databases/",
}

// EnvVar mirrors the `EnvironmentVariable` schema.
type EnvVar struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Comment     string `json:"comment"`
	IsPreview   bool   `json:"is_preview"`
	IsLiteral   bool   `json:"is_literal"`
	IsMultiline bool   `json:"is_multiline"`
	IsShownOnce bool   `json:"is_shown_once"`
	IsRuntime   bool   `json:"is_runtime"`
	IsBuildtime bool   `json:"is_buildtime"`
}

// EnvVarRequest is the body for creating and updating a variable. Updates are
// keyed by Key (the API identifies the variable to change by its key).
type EnvVarRequest struct {
	Key         *string `json:"key,omitempty"`
	Value       *string `json:"value,omitempty"`
	Comment     *string `json:"comment,omitempty"`
	IsPreview   *bool   `json:"is_preview,omitempty"`
	IsLiteral   *bool   `json:"is_literal,omitempty"`
	IsMultiline *bool   `json:"is_multiline,omitempty"`
	IsShownOnce *bool   `json:"is_shown_once,omitempty"`
	IsRuntime   *bool   `json:"is_runtime,omitempty"`
	IsBuildtime *bool   `json:"is_buildtime,omitempty"`
}

func envVarBase(parent EnvVarParent, parentUUID string) (string, error) {
	base, ok := envVarParentPaths[parent]
	if !ok {
		return "", fmt.Errorf("unknown env var parent %q", parent)
	}
	return base + url.PathEscape(parentUUID) + "/envs", nil
}

// ListEnvVars returns the environment variables of an application, service or
// database.
func (c *Client) ListEnvVars(ctx context.Context, parent EnvVarParent, parentUUID string) ([]EnvVar, error) {
	base, err := envVarBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	var out []EnvVar
	if err := c.get(ctx, base, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetEnvVar returns a single variable by UUID, via the list endpoint (the API
// has no per-variable GET).
func (c *Client) GetEnvVar(ctx context.Context, parent EnvVarParent, parentUUID, uuid string) (*EnvVar, error) {
	vars, err := c.ListEnvVars(ctx, parent, parentUUID)
	if err != nil {
		return nil, err
	}
	for _, v := range vars {
		if v.UUID == uuid {
			return &v, nil
		}
	}
	return nil, &Error{Method: "GET", Path: string(parent) + " envs", StatusCode: 404, Message: "Environment variable not found."}
}

// CreateEnvVar creates a variable and returns it (create responds {uuid} only).
func (c *Client) CreateEnvVar(ctx context.Context, parent EnvVarParent, parentUUID string, req EnvVarRequest) (*EnvVar, error) {
	base, err := envVarBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	var created uuidResponse
	if err := c.post(ctx, base, req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST %s: API returned no uuid", base)
	}
	return c.GetEnvVar(ctx, parent, parentUUID, created.UUID)
}

// UpdateEnvVar updates a variable. The API matches on req.Key, which must be
// set; the refreshed variable is returned.
func (c *Client) UpdateEnvVar(ctx context.Context, parent EnvVarParent, parentUUID, uuid string, req EnvVarRequest) (*EnvVar, error) {
	base, err := envVarBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	if err := c.patch(ctx, base, req, nil); err != nil {
		return nil, err
	}
	return c.GetEnvVar(ctx, parent, parentUUID, uuid)
}

// DeleteEnvVar removes a variable by UUID.
func (c *Client) DeleteEnvVar(ctx context.Context, parent EnvVarParent, parentUUID, uuid string) error {
	base, err := envVarBase(parent, parentUUID)
	if err != nil {
		return err
	}
	return c.delete(ctx, base+"/"+url.PathEscape(uuid))
}
