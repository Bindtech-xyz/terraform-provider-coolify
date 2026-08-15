package client

import (
	"context"
	"net/url"
)

// ApplicationDestination is one entry of GET /applications/{uuid}/destinations
// — the primary destination (IsPrimary true, configured via the application's
// own destination_uuid, not managed here) plus any additional ones.
type ApplicationDestination struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Network    string `json:"network"`
	ServerUUID string `json:"server_uuid"`
	IsPrimary  bool   `json:"is_primary"`
}

// ListApplicationDestinations returns every destination (primary + additional)
// attached to an application.
func (c *Client) ListApplicationDestinations(ctx context.Context, applicationUUID string) ([]ApplicationDestination, error) {
	var out []ApplicationDestination
	if err := c.get(ctx, "/applications/"+url.PathEscape(applicationUUID)+"/destinations", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddApplicationDestination attaches an additional standalone destination to
// an application (multi-destination deployment). Coolify rejects this if the
// destination shares a server with the primary destination or another
// already-attached one, or if it's the primary destination itself.
func (c *Client) AddApplicationDestination(ctx context.Context, applicationUUID, destinationUUID string) error {
	body := struct {
		DestinationUUID string `json:"destination_uuid"`
	}{DestinationUUID: destinationUUID}
	return c.post(ctx, "/applications/"+url.PathEscape(applicationUUID)+"/destinations", body, nil)
}

// RemoveApplicationDestination detaches an additional destination. Coolify
// refuses to detach the primary destination (use the application resource's
// own destination_uuid / RequiresReplace for that).
func (c *Client) RemoveApplicationDestination(ctx context.Context, applicationUUID, destinationUUID string) error {
	return c.delete(ctx, "/applications/"+url.PathEscape(applicationUUID)+"/destinations/"+url.PathEscape(destinationUUID))
}
