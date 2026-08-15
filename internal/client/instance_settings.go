package client

import "context"

// EnableAPI, DisableAPI, EnableMCP and DisableMCP require a root-team (team
// 0) token — Coolify returns 403 otherwise. There is no GET to read current
// state back; these are fire-and-forget POSTs, mirrored by the provider's
// coolify_api_settings resource accordingly (no Read drift detection
// possible, matching coolify_volume_backup's own documented constraint).

func (c *Client) EnableAPI(ctx context.Context) error  { return c.post(ctx, "/enable", nil, nil) }
func (c *Client) DisableAPI(ctx context.Context) error { return c.post(ctx, "/disable", nil, nil) }

func (c *Client) EnableMCP(ctx context.Context) error {
	return c.post(ctx, "/mcp/enable", nil, nil)
}

func (c *Client) DisableMCP(ctx context.Context) error {
	return c.post(ctx, "/mcp/disable", nil, nil)
}
