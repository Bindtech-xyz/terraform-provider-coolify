package client

import (
	"context"
	"strconv"
)

// DeletePreview removes a PR preview deployment. Coolify has no create or
// read endpoint for previews — they're created automatically from GitHub
// App webhook events on pull requests, never via a direct API call — so
// this is the only lifecycle operation the API actually exposes.
func (c *Client) DeletePreview(ctx context.Context, applicationUUID string, pullRequestID int64) error {
	return c.delete(ctx, "/applications/"+applicationUUID+"/previews/"+strconv.FormatInt(pullRequestID, 10))
}
