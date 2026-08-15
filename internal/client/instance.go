package client

import "context"

// Version returns the Coolify instance version string (GET /version). The
// provider calls it during Configure to fail fast on a bad endpoint or token.
func (c *Client) Version(ctx context.Context) (string, error) {
	var out string
	if err := c.get(ctx, "/version", &out); err != nil {
		return "", err
	}
	return out, nil
}
