package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestNotificationSettingsRoundTrip(t *testing.T) {
	var patched map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notifications/discord" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method == http.MethodPatch {
			_ = json.NewDecoder(r.Body).Decode(&patched)
		}
		_, _ = w.Write([]byte(`{"discord_enabled":true,"deployment_failure_discord_notifications":true}`))
	}))

	settings, err := c.UpdateNotificationSettings(context.Background(), "discord", map[string]any{
		"discord_enabled":                          true,
		"deployment_failure_discord_notifications": true,
	})
	if err != nil {
		t.Fatalf("UpdateNotificationSettings: %v", err)
	}
	if patched["discord_enabled"] != true {
		t.Errorf("patched = %v", patched)
	}
	if settings["discord_enabled"] != true {
		t.Errorf("settings = %v", settings)
	}
}

func TestNotificationSettingsRejectsUnknownChannel(t *testing.T) {
	c := newTestClient(t, nil)
	if _, err := c.GetNotificationSettings(context.Background(), "carrier-pigeon"); err == nil {
		t.Fatal("expected error for unknown channel")
	}
	if _, err := c.UpdateNotificationSettings(context.Background(), "carrier-pigeon", nil); err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

func TestServerSettingSectionsHitTheirEndpoints(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"docker_cleanup_threshold":80}`))
	}))

	ctx := context.Background()
	_, _ = c.GetServerProxy(ctx, "s")
	_ = c.UpdateServerDockerCleanup(ctx, "s", map[string]any{"docker_cleanup_threshold": 80})
	_, _ = c.GetServerSentinel(ctx, "s")
	_ = c.UpdateServerCloudflareTunnel(ctx, "s", map[string]any{"is_cloudflare_tunnel": true})
	_, _ = c.GetServerLogDrains(ctx, "s")

	want := []string{
		"GET /api/v1/servers/s/proxy",
		"PATCH /api/v1/servers/s/docker-cleanup",
		"GET /api/v1/servers/s/sentinel",
		"PATCH /api/v1/servers/s/cloudflare-tunnel",
		"GET /api/v1/servers/s/log-drains",
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("call %d = %q, want %q", i, paths[i], want[i])
		}
	}
}
