package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SharedEnvScope identifies where a shared (team/project/environment/server)
// variable lives. Each scope has its own path prefix; the payload contract is
// identical.
type SharedEnvScope struct {
	// Kind is one of team, project, environment, server.
	Kind string
	// ProjectUUID is required for project and environment scopes.
	ProjectUUID string
	// Environment (name or UUID) is required for the environment scope.
	Environment string
	// ServerUUID is required for the server scope.
	ServerUUID string
}

func (s SharedEnvScope) base() (string, error) {
	switch s.Kind {
	case "team":
		return "/team/envs", nil
	case "project":
		if s.ProjectUUID == "" {
			return "", fmt.Errorf("project scope requires a project UUID")
		}
		return "/projects/" + url.PathEscape(s.ProjectUUID) + "/envs", nil
	case "environment":
		if s.ProjectUUID == "" || s.Environment == "" {
			return "", fmt.Errorf("environment scope requires a project UUID and an environment")
		}
		return "/projects/" + url.PathEscape(s.ProjectUUID) + "/environments/" + url.PathEscape(s.Environment) + "/envs", nil
	case "server":
		if s.ServerUUID == "" {
			return "", fmt.Errorf("server scope requires a server UUID")
		}
		return "/servers/" + url.PathEscape(s.ServerUUID) + "/envs", nil
	default:
		return "", fmt.Errorf("unknown shared env scope %q", s.Kind)
	}
}

// SharedEnvVar is a team/project/environment/server-scoped variable. Unlike
// resource variables it is addressed by numeric id.
type SharedEnvVar struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Comment     string `json:"comment"`
	IsLiteral   bool   `json:"is_literal"`
	IsMultiline bool   `json:"is_multiline"`
	IsShownOnce bool   `json:"is_shown_once"`
}

// SharedEnvVarRequest is the create/update body for shared variables.
type SharedEnvVarRequest struct {
	Key         *string `json:"key,omitempty"`
	Value       *string `json:"value,omitempty"`
	Comment     *string `json:"comment,omitempty"`
	IsLiteral   *bool   `json:"is_literal,omitempty"`
	IsMultiline *bool   `json:"is_multiline,omitempty"`
	IsShownOnce *bool   `json:"is_shown_once,omitempty"`
}

// ListSharedEnvVars returns every variable of the scope.
func (c *Client) ListSharedEnvVars(ctx context.Context, scope SharedEnvScope) ([]SharedEnvVar, error) {
	base, err := scope.base()
	if err != nil {
		return nil, err
	}
	var out []SharedEnvVar
	if err := c.get(ctx, base, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetSharedEnvVar returns one variable by id, via the list endpoint.
func (c *Client) GetSharedEnvVar(ctx context.Context, scope SharedEnvScope, id int64) (*SharedEnvVar, error) {
	vars, err := c.ListSharedEnvVars(ctx, scope)
	if err != nil {
		return nil, err
	}
	for _, v := range vars {
		if v.ID == id {
			return &v, nil
		}
	}
	return nil, &Error{Method: "GET", Path: scope.Kind + " envs", StatusCode: 404, Message: "Environment variable not found."}
}

// CreateSharedEnvVar creates a variable (the API responds {id}).
func (c *Client) CreateSharedEnvVar(ctx context.Context, scope SharedEnvScope, req SharedEnvVarRequest) (*SharedEnvVar, error) {
	base, err := scope.base()
	if err != nil {
		return nil, err
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := c.post(ctx, base, req, &created); err != nil {
		return nil, err
	}
	if created.ID == 0 {
		return nil, fmt.Errorf("POST %s: API returned no id", base)
	}
	return c.GetSharedEnvVar(ctx, scope, created.ID)
}

// UpdateSharedEnvVar updates a variable by id and returns it refreshed.
func (c *Client) UpdateSharedEnvVar(ctx context.Context, scope SharedEnvScope, id int64, req SharedEnvVarRequest) (*SharedEnvVar, error) {
	base, err := scope.base()
	if err != nil {
		return nil, err
	}
	if err := c.patch(ctx, base+"/"+strconv.FormatInt(id, 10), req, nil); err != nil {
		return nil, err
	}
	return c.GetSharedEnvVar(ctx, scope, id)
}

// DeleteSharedEnvVar removes a variable by id.
func (c *Client) DeleteSharedEnvVar(ctx context.Context, scope SharedEnvScope, id int64) error {
	base, err := scope.base()
	if err != nil {
		return err
	}
	return c.delete(ctx, base+"/"+strconv.FormatInt(id, 10))
}
