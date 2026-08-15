package client

import (
	"context"
	"net/url"
)

// Server sub-settings live on singleton endpoints under /servers/{uuid}/…
// (proxy, docker-cleanup, log-drains, sentinel, cloudflare-tunnel). Each has
// its own field set; maps keep the transport faithful to the API contract and
// let the resource layer own the typed schema.

func (c *Client) getServerSetting(ctx context.Context, serverUUID, section string) (map[string]any, error) {
	var out map[string]any
	if err := c.get(ctx, "/servers/"+url.PathEscape(serverUUID)+"/"+section, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) patchServerSetting(ctx context.Context, serverUUID, section string, body map[string]any) error {
	return c.patch(ctx, "/servers/"+url.PathEscape(serverUUID)+"/"+section, body, nil)
}

// GetServerProxy returns the proxy settings of a server.
func (c *Client) GetServerProxy(ctx context.Context, serverUUID string) (map[string]any, error) {
	return c.getServerSetting(ctx, serverUUID, "proxy")
}

// UpdateServerProxy patches the proxy settings
// (redirect_enabled, redirect_url, generate_exact_labels, proxy_type).
func (c *Client) UpdateServerProxy(ctx context.Context, serverUUID string, body map[string]any) error {
	return c.patchServerSetting(ctx, serverUUID, "proxy", body)
}

// GetServerDockerCleanup returns the automated-cleanup settings.
func (c *Client) GetServerDockerCleanup(ctx context.Context, serverUUID string) (map[string]any, error) {
	return c.getServerSetting(ctx, serverUUID, "docker-cleanup")
}

// UpdateServerDockerCleanup patches the automated-cleanup settings
// (docker_cleanup_frequency, docker_cleanup_threshold, force_docker_cleanup,
// delete_unused_volumes, delete_unused_networks, ...).
func (c *Client) UpdateServerDockerCleanup(ctx context.Context, serverUUID string, body map[string]any) error {
	return c.patchServerSetting(ctx, serverUUID, "docker-cleanup", body)
}

// GetServerLogDrains returns the log-drain settings.
func (c *Client) GetServerLogDrains(ctx context.Context, serverUUID string) (map[string]any, error) {
	return c.getServerSetting(ctx, serverUUID, "log-drains")
}

// UpdateServerLogDrains patches the log-drain settings (New Relic, Axiom,
// custom FluentBit config).
func (c *Client) UpdateServerLogDrains(ctx context.Context, serverUUID string, body map[string]any) error {
	return c.patchServerSetting(ctx, serverUUID, "log-drains", body)
}

// GetServerSentinel returns the Sentinel (monitoring agent) settings.
func (c *Client) GetServerSentinel(ctx context.Context, serverUUID string) (map[string]any, error) {
	return c.getServerSetting(ctx, serverUUID, "sentinel")
}

// UpdateServerSentinel patches the Sentinel settings.
func (c *Client) UpdateServerSentinel(ctx context.Context, serverUUID string, body map[string]any) error {
	return c.patchServerSetting(ctx, serverUUID, "sentinel", body)
}

// GetServerCloudflareTunnel returns the Cloudflare Tunnel settings.
func (c *Client) GetServerCloudflareTunnel(ctx context.Context, serverUUID string) (map[string]any, error) {
	return c.getServerSetting(ctx, serverUUID, "cloudflare-tunnel")
}

// UpdateServerCloudflareTunnel patches the Cloudflare Tunnel settings
// (is_cloudflare_tunnel).
func (c *Client) UpdateServerCloudflareTunnel(ctx context.Context, serverUUID string, body map[string]any) error {
	return c.patchServerSetting(ctx, serverUUID, "cloudflare-tunnel", body)
}
