package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateStorageReturnsFullObject(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"uuid":"st1","storage_type":"persistent","name":"uploads","mount_path":"/app/uploads"}`))
	}))
	typ, mount := "persistent", "/app/uploads"
	s, err := c.CreateStorage(context.Background(), EnvVarParentApplication, "app1", StorageRequest{Type: &typ, MountPath: &mount})
	if err != nil {
		t.Fatalf("CreateStorage: %v", err)
	}
	if s.UUID != "st1" || s.Type != "persistent" {
		t.Errorf("storage = %+v", s)
	}
}

func TestUpdateStorageSendsUUIDInBody(t *testing.T) {
	var body map[string]any
	var patchPath string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			patchPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.Write([]byte(`{}`))
		default:
			w.Write([]byte(`[{"uuid":"st1","storage_type":"file","mount_path":"/etc/cfg"}]`))
		}
	}))

	typ := "file"
	if _, err := c.UpdateStorage(context.Background(), EnvVarParentService, "svc1", "st1", StorageRequest{Type: &typ}); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}
	// The PATCH endpoint has no storage segment: the uuid rides in the body.
	if want := "/api/v1/services/svc1/storages"; patchPath != want {
		t.Errorf("PATCH path = %q, want %q", patchPath, want)
	}
	if body["uuid"] != "st1" {
		t.Errorf("body uuid = %v, want st1", body["uuid"])
	}
}

func TestStorageParentPaths(t *testing.T) {
	for parent, want := range map[EnvVarParent]string{
		EnvVarParentApplication: "/applications/x/storages",
		EnvVarParentService:     "/services/x/storages",
		EnvVarParentDatabase:    "/databases/x/storages",
	} {
		got, err := storageBase(parent, "x")
		if err != nil {
			t.Fatalf("storageBase(%s): %v", parent, err)
		}
		if got != want {
			t.Errorf("storageBase(%s) = %q, want %q", parent, got, want)
		}
	}
	if _, err := storageBase("nope", "x"); err == nil {
		t.Error("expected error for unknown parent")
	}
}
