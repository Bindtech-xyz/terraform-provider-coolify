package client

import (
	"context"
	"fmt"
	"slices"
)

// NotificationChannels lists the team notification channels exposed by the API
// (GET/PATCH /notifications/{channel}).
var NotificationChannels = []string{"email", "discord", "slack", "telegram", "pushover", "webhook"}

// GetNotificationSettings returns the raw settings of a channel. The field set
// is channel-specific (Laravel model fillables), so a map keeps the client
// honest about what the API accepts.
func (c *Client) GetNotificationSettings(ctx context.Context, channel string) (map[string]any, error) {
	if !slices.Contains(NotificationChannels, channel) {
		return nil, fmt.Errorf("unknown notification channel %q", channel)
	}
	var out map[string]any
	if err := c.get(ctx, "/notifications/"+channel, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateNotificationSettings patches a channel's settings and returns them
// refreshed. Unknown fields are rejected by the API with a 422.
func (c *Client) UpdateNotificationSettings(ctx context.Context, channel string, settings map[string]any) (map[string]any, error) {
	if !slices.Contains(NotificationChannels, channel) {
		return nil, fmt.Errorf("unknown notification channel %q", channel)
	}
	if err := c.patch(ctx, "/notifications/"+channel, settings, nil); err != nil {
		return nil, err
	}
	return c.GetNotificationSettings(ctx, channel)
}
