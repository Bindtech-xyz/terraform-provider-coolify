package client

import (
	"context"
	"net/http"
	"testing"
)

func TestInstanceSettingsPaths(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	ctx := context.Background()

	if err := c.EnableAPI(ctx); err != nil {
		t.Fatalf("EnableAPI: %v", err)
	}
	if err := c.DisableAPI(ctx); err != nil {
		t.Fatalf("DisableAPI: %v", err)
	}
	if err := c.EnableMCP(ctx); err != nil {
		t.Fatalf("EnableMCP: %v", err)
	}
	if err := c.DisableMCP(ctx); err != nil {
		t.Fatalf("DisableMCP: %v", err)
	}

	want := []string{
		"POST /api/v1/enable",
		"POST /api/v1/disable",
		"POST /api/v1/mcp/enable",
		"POST /api/v1/mcp/disable",
	}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestDeletePreviewPath(t *testing.T) {
	var got string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	if err := c.DeletePreview(context.Background(), "app1", 42); err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if want := "DELETE /api/v1/applications/app1/previews/42"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}
