package client

import (
	"context"
	"net/http"
	"testing"
)

func TestResourceTagLifecyclePaths(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":1,"uuid":"tag1","name":"prod"}]`))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`[{"id":1,"uuid":"tag1","name":"prod"}]`))
		case http.MethodDelete:
			_, _ = w.Write([]byte(`{"message":"Tag removed."}`))
		}
	}))
	ctx := context.Background()

	if _, err := c.ListResourceTags(ctx, TaggableApplication, "app1"); err != nil {
		t.Fatalf("ListResourceTags: %v", err)
	}
	tags, err := c.AttachResourceTag(ctx, TaggableDatabase, "db1", "prod")
	if err != nil {
		t.Fatalf("AttachResourceTag: %v", err)
	}
	if len(tags) != 1 || tags[0].UUID != "tag1" {
		t.Errorf("tags = %+v, want one entry with uuid tag1", tags)
	}
	if err := c.DetachResourceTag(ctx, TaggableService, "svc1", "tag1"); err != nil {
		t.Fatalf("DetachResourceTag: %v", err)
	}

	want := []string{
		"GET /api/v1/applications/app1/tags",
		"POST /api/v1/databases/db1/tags",
		"DELETE /api/v1/services/svc1/tags/tag1",
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

func TestResourceTagRejectsUnknownType(t *testing.T) {
	c := newTestClient(t, nil)
	if _, err := c.ListResourceTags(context.Background(), "wat", "x"); err == nil {
		t.Fatal("expected error for unknown taggable resource type")
	}
}
