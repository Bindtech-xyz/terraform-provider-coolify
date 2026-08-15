package client

import (
	"context"
	"net/http"
	"testing"
)

func TestGetTagFiltersList(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"uuid":"t1","name":"frontend"},{"uuid":"t2","name":"backend"}]`))
	}))
	tag, err := c.GetTag(context.Background(), "t2")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if tag.Name != "backend" {
		t.Errorf("name = %q", tag.Name)
	}
	if _, err := c.GetTag(context.Background(), "missing"); !IsNotFound(err) {
		t.Errorf("missing tag: IsNotFound = false (%v)", err)
	}
}

func TestS3StorageValidatePath(t *testing.T) {
	var gotPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	if err := c.ValidateS3Storage(context.Background(), "s3x"); err != nil {
		t.Fatalf("ValidateS3Storage: %v", err)
	}
	if want := "/api/v1/s3-storages/s3x/validate"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestCreateDestinationReturnsFullObject(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"uuid":"d1","name":"worker-isolated","network":"isolated","server":{"uuid":"srv1"}}`))
	}))
	network := "isolated"
	dest, err := c.CreateDestination(context.Background(), "srv1", DestinationCreateRequest{Network: &network})
	if err != nil {
		t.Fatalf("CreateDestination: %v", err)
	}
	if dest.UUID != "d1" || dest.ServerRef() != "srv1" {
		t.Errorf("dest = %+v", dest)
	}
}

func TestDuplicateDestinationSurfaces409(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"A destination with this network already exists on the server."}`))
	}))
	network := "isolated"
	if _, err := c.CreateDestination(context.Background(), "srv1", DestinationCreateRequest{Network: &network}); err == nil {
		t.Fatal("expected 409 error")
	}
}
