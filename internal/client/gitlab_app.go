package client

import (
	"context"
	"strconv"
)

// GitlabApp is a GitLab source used to deploy private repositories
// (docs: applications/ci-cd/gitlab). Addressed by numeric id like GithubApp.
type GitlabApp struct {
	ID           int64  `json:"id"`
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	APIURL       string `json:"api_url"`
	HTMLURL      string `json:"html_url"`
	CustomUser   string `json:"custom_user"`
	CustomPort   int64  `json:"custom_port"`
	GroupName    string `json:"group_name"`
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	IsSystemWide bool   `json:"is_system_wide"`
}

// GitlabAppRequest is the create/update body. Secrets are never echoed back.
type GitlabAppRequest struct {
	Name         *string `json:"name,omitempty"`
	HTMLURL      *string `json:"html_url,omitempty"`
	APIURL       *string `json:"api_url,omitempty"`
	CustomUser   *string `json:"custom_user,omitempty"`
	CustomPort   *int64  `json:"custom_port,omitempty"`
	GroupName    *string `json:"group_name,omitempty"`
	ClientID     *string `json:"client_id,omitempty"`
	ClientSecret *string `json:"client_secret,omitempty"`
	WebhookToken *string `json:"webhook_token,omitempty"`
	RedirectURI  *string `json:"redirect_uri,omitempty"`
	IsSystemWide *bool   `json:"is_system_wide,omitempty"`
}

// ListGitlabApps returns every GitLab app of the token's team.
func (c *Client) ListGitlabApps(ctx context.Context) ([]GitlabApp, error) {
	var out []GitlabApp
	if err := c.get(ctx, "/gitlab-apps", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetGitlabApp returns one app by id, via the list endpoint.
func (c *Client) GetGitlabApp(ctx context.Context, id int64) (*GitlabApp, error) {
	apps, err := c.ListGitlabApps(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range apps {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, &Error{Method: "GET", Path: "/gitlab-apps", StatusCode: 404, Message: "GitLab App not found."}
}

// CreateGitlabApp registers a GitLab source; the API returns the full object.
func (c *Client) CreateGitlabApp(ctx context.Context, req GitlabAppRequest) (*GitlabApp, error) {
	var out GitlabApp
	if err := c.post(ctx, "/gitlab-apps", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateGitlabApp updates an app by id and returns it refreshed.
func (c *Client) UpdateGitlabApp(ctx context.Context, id int64, req GitlabAppRequest) (*GitlabApp, error) {
	if err := c.patch(ctx, "/gitlab-apps/"+strconv.FormatInt(id, 10), req, nil); err != nil {
		return nil, err
	}
	return c.GetGitlabApp(ctx, id)
}

// DeleteGitlabApp removes an app by id.
func (c *Client) DeleteGitlabApp(ctx context.Context, id int64) error {
	return c.delete(ctx, "/gitlab-apps/"+strconv.FormatInt(id, 10))
}
