package client

import (
	"context"
	"net/url"
)

// Deployment mirrors `ApplicationDeploymentQueue` (a queued or running
// deployment).
type Deployment struct {
	ID              int64  `json:"id"`
	DeploymentUUID  string `json:"deployment_uuid"`
	ApplicationID   string `json:"application_id"`
	ApplicationName string `json:"application_name"`
	Status          string `json:"status"`
	Commit          string `json:"commit"`
	IsWebhook       bool   `json:"is_webhook"`
	CreatedAt       string `json:"created_at"`
}

// ListDeployments returns the currently queued/running deployments.
func (c *Client) ListDeployments(ctx context.Context) ([]Deployment, error) {
	var out []Deployment
	if err := c.get(ctx, "/deployments", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListApplicationDeployments returns past deployments of one application
// (paginated by the API; first page by default).
func (c *Client) ListApplicationDeployments(ctx context.Context, applicationUUID string) ([]Deployment, error) {
	var out struct {
		Deployments []Deployment `json:"deployments"`
	}
	path := "/deployments/applications/" + url.PathEscape(applicationUUID)
	// The endpoint may return either a bare array or {deployments: [...]}
	// depending on version; try the wrapped form first, fall back to bare.
	if err := c.get(ctx, path, &out); err != nil {
		var bare []Deployment
		if err2 := c.get(ctx, path, &bare); err2 != nil {
			return nil, err
		}
		return bare, nil
	}
	return out.Deployments, nil
}

// Deploy triggers deployments by resource uuid(s) or tag(s) (comma-separated),
// mirroring POST /deploy. force rebuilds without cache.
func (c *Client) Deploy(ctx context.Context, uuids, tags string, force bool) error {
	q := url.Values{}
	if uuids != "" {
		q.Set("uuid", uuids)
	}
	if tags != "" {
		q.Set("tag", tags)
	}
	if force {
		q.Set("force", "true")
	}
	return c.post(ctx, "/deploy?"+q.Encode(), nil, nil)
}
