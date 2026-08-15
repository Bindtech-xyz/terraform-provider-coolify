package client

import (
	"context"
	"net/http"
	"testing"
)

func TestApplicationDestinationLifecyclePaths(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"uuid":"dest1","is_primary":true},{"uuid":"dest2","is_primary":false}]`))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"message":"Destination attached.","uuid":"dest2"}`))
		case http.MethodDelete:
			_, _ = w.Write([]byte(`{"message":"Destination detached."}`))
		}
	}))
	ctx := context.Background()

	dests, err := c.ListApplicationDestinations(ctx, "app1")
	if err != nil {
		t.Fatalf("ListApplicationDestinations: %v", err)
	}
	if len(dests) != 2 || !dests[0].IsPrimary || dests[1].IsPrimary {
		t.Errorf("dests = %+v, want [primary, non-primary]", dests)
	}
	if err := c.AddApplicationDestination(ctx, "app1", "dest2"); err != nil {
		t.Fatalf("AddApplicationDestination: %v", err)
	}
	if err := c.RemoveApplicationDestination(ctx, "app1", "dest2"); err != nil {
		t.Fatalf("RemoveApplicationDestination: %v", err)
	}

	want := []string{
		"GET /api/v1/applications/app1/destinations",
		"POST /api/v1/applications/app1/destinations",
		"DELETE /api/v1/applications/app1/destinations/dest2",
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
