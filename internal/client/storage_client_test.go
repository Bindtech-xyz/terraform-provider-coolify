package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// The response never carries a type/storage_type field — Coolify has no
// discriminator column, "persistent" and "file" are two entirely separate
// Eloquent models (LocalPersistentVolume, LocalFileVolume) — so the mock
// response bodies below match what the real API actually sends, not a
// guessed shape. name is real only for persistent volumes, and comes back
// prefixed with the parent's UUID server-side.
func TestCreateStorageReturnsFullObject(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"uuid":"st1","name":"app1-uploads","mount_path":"/app/uploads"}`))
	}))
	typ, mount := "persistent", "/app/uploads"
	s, err := c.CreateStorage(context.Background(), EnvVarParentApplication, "app1", StorageRequest{Type: &typ, MountPath: &mount})
	if err != nil {
		t.Fatalf("CreateStorage: %v", err)
	}
	if s.UUID != "st1" || s.Name != "app1-uploads" {
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
			w.Write([]byte(`{"persistent_storages":[{"uuid":"st1","mount_path":"/etc/cfg"}],"file_storages":[]}`))
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

// TestListStoragesMergesPersistentAndFile locks in a regression found by a
// real deployment: the list endpoint does not return a flat array (Coolify
// has no shared model for the two storage kinds) — it returns
// {"persistent_storages": [...], "file_storages": [...]}. A bare-array
// unmarshal fails outright ("cannot unmarshal object into Go value of type
// []client.Storage") on every real Coolify instance.
func TestListStoragesMergesPersistentAndFile(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"persistent_storages": [{"uuid":"p1","name":"app1-data","mount_path":"/data"}],
			"file_storages": [{"uuid":"f1","mount_path":"/etc/app.conf","fs_path":"./app.conf"}]
		}`))
	}))
	storages, err := c.ListStorages(context.Background(), EnvVarParentApplication, "app1")
	if err != nil {
		t.Fatalf("ListStorages: %v", err)
	}
	if len(storages) != 2 {
		t.Fatalf("len = %d, want 2", len(storages))
	}
	if storages[0].UUID != "p1" || storages[1].UUID != "f1" {
		t.Errorf("storages = %+v", storages)
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
