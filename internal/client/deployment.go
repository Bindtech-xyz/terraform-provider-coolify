package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
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

// DeployResult is one entry of POST /deploy's response: one per resource
// matched by uuid(s) or tag(s). DeploymentUUID is empty when the resource
// could not be queued (see Message for why).
type DeployResult struct {
	Message        string `json:"message"`
	ResourceUUID   string `json:"resource_uuid"`
	DeploymentUUID string `json:"deployment_uuid"`
}

// Deploy triggers deployments by resource uuid(s) or tag(s) (comma-separated,
// mutually exclusive), mirroring POST /deploy. force rebuilds without cache.
//
// The by-uuid and by-tag code paths return genuinely different response
// envelopes (confirmed live, and in DeployController::by_uuids vs
// ::by_tags): by-uuid wraps each result in "deployments" with a per-entry
// "message"; by-tag wraps the same {resource_uuid, deployment_uuid} pairs
// in "details" instead, with only an aggregate top-level "message" array
// that doesn't line up 1:1 with results. Decoding only the "deployments"
// shape silently produced an empty result set for every by-tag deploy — no
// error, just nothing to show for it. Both are decoded into the same
// DeployResult; "details" entries simply have an empty Message (Coolify
// doesn't give us a per-resource one on that path).
func (c *Client) Deploy(ctx context.Context, uuids, tags string, force bool) ([]DeployResult, error) {
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
	var out struct {
		Deployments []DeployResult `json:"deployments"`
		Details     []DeployResult `json:"details"`
	}
	if err := c.post(ctx, "/deploy?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	if len(out.Deployments) > 0 {
		return out.Deployments, nil
	}
	return out.Details, nil
}

// GetDeployment reads one deployment's current status by its deployment
// UUID (not the resource UUID) — used to poll a triggered deployment to
// completion.
func (c *Client) GetDeployment(ctx context.Context, deploymentUUID string) (*Deployment, error) {
	var out Deployment
	if err := c.get(ctx, "/deployments/"+url.PathEscape(deploymentUUID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// WaitForDeploymentCompletion polls a deployment (fixed 3s interval — no
// backoff needed, deployments run for minutes not the sub-second timescale
// waitForDeletion's backoff is tuned for) until it reaches a terminal
// status. Returns the final Deployment even on failure/timeout, so the
// caller can surface its status in the error.
func (c *Client) WaitForDeploymentCompletion(ctx context.Context, deploymentUUID string, deadline time.Duration) (*Deployment, error) {
	waitCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	for {
		d, err := c.GetDeployment(waitCtx, deploymentUUID)
		if err != nil {
			return nil, err
		}
		switch d.Status {
		case "finished":
			return d, nil
		case "failed":
			return d, fmt.Errorf("deployment %s failed", deploymentUUID)
		}

		select {
		case <-waitCtx.Done():
			return d, fmt.Errorf("deployment %s still %q after %s", deploymentUUID, d.Status, deadline)
		case <-time.After(3 * time.Second):
		}
	}
}
