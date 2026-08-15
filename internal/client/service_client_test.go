package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateServiceOneClick(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"svc1","domains":["https://x.example.com"]}`))
			return
		}
		_, _ = w.Write([]byte(`{"uuid":"svc1","name":"plausible","service_type":"plausible"}`))
	}))

	typ := "plausible"
	svc, err := c.CreateService(context.Background(), ServiceCreateRequest{Type: &typ})
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}
	if svc.Type != "plausible" {
		t.Errorf("service_type = %q", svc.Type)
	}
	if body["type"] != "plausible" {
		t.Errorf("body type = %v", body["type"])
	}
}

func TestServiceDeleteWaitsForTeardown(t *testing.T) {
	var gets int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gets++
			if gets < 2 {
				// Still tearing down on the first poll.
				_, _ = w.Write([]byte(`{"uuid":"svc1","status":"exited"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Resource not found."}`))
			return
		}
		_, _ = w.Write([]byte(`{"message":"queued"}`))
	}))

	if err := c.DeleteService(context.Background(), "svc1", nil, nil, nil, nil); err != nil {
		t.Fatalf("DeleteService: %v", err)
	}
	if gets < 2 {
		t.Errorf("expected at least 2 polls, got %d", gets)
	}
}
