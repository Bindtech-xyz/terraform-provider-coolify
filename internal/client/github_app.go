package client

import (
	"context"
	"strconv"
)

// GithubApp is a GitHub App source used to deploy private repositories
// (docs: applications/ci-cd/github). Unlike most objects it is addressed by
// numeric id, not uuid.
type GithubApp struct {
	ID             int64  `json:"id"`
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Organization   string `json:"organization"`
	APIURL         string `json:"api_url"`
	HTMLURL        string `json:"html_url"`
	CustomUser     string `json:"custom_user"`
	CustomPort     int64  `json:"custom_port"`
	AppID          int64  `json:"app_id"`
	InstallationID int64  `json:"installation_id"`
	ClientID       string `json:"client_id"`
	IsSystemWide   bool   `json:"is_system_wide"`
}

// GithubAppRequest is the create/update body. ClientSecret, WebhookSecret and
// PrivateKeyUUID are required on create and never echoed back by the API.
type GithubAppRequest struct {
	Name           *string `json:"name,omitempty"`
	Organization   *string `json:"organization,omitempty"`
	APIURL         *string `json:"api_url,omitempty"`
	HTMLURL        *string `json:"html_url,omitempty"`
	CustomUser     *string `json:"custom_user,omitempty"`
	CustomPort     *int64  `json:"custom_port,omitempty"`
	AppID          *int64  `json:"app_id,omitempty"`
	InstallationID *int64  `json:"installation_id,omitempty"`
	ClientID       *string `json:"client_id,omitempty"`
	ClientSecret   *string `json:"client_secret,omitempty"`
	WebhookSecret  *string `json:"webhook_secret,omitempty"`
	PrivateKeyUUID *string `json:"private_key_uuid,omitempty"`
	IsSystemWide   *bool   `json:"is_system_wide,omitempty"`
}

// ListGithubApps returns every GitHub App of the token's team.
func (c *Client) ListGithubApps(ctx context.Context) ([]GithubApp, error) {
	var out []GithubApp
	if err := c.get(ctx, "/github-apps", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetGithubApp returns one app by id, via the list endpoint (the API has no
// single-app GET).
func (c *Client) GetGithubApp(ctx context.Context, id int64) (*GithubApp, error) {
	apps, err := c.ListGithubApps(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range apps {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, &Error{Method: "GET", Path: "/github-apps", StatusCode: 404, Message: "GitHub App not found."}
}

// CreateGithubApp registers an existing GitHub App with Coolify; the API
// returns the full object.
func (c *Client) CreateGithubApp(ctx context.Context, req GithubAppRequest) (*GithubApp, error) {
	var out GithubApp
	if err := c.post(ctx, "/github-apps", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateGithubApp updates an app by id and returns it refreshed.
func (c *Client) UpdateGithubApp(ctx context.Context, id int64, req GithubAppRequest) (*GithubApp, error) {
	if err := c.patch(ctx, "/github-apps/"+strconv.FormatInt(id, 10), req, nil); err != nil {
		return nil, err
	}
	return c.GetGithubApp(ctx, id)
}

// DeleteGithubApp removes an app by id.
func (c *Client) DeleteGithubApp(ctx context.Context, id int64) error {
	return c.delete(ctx, "/github-apps/"+strconv.FormatInt(id, 10))
}
