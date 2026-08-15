package client

import (
	"context"
	"net/http"
	"testing"
)

func TestCreatePrivateKeyFollowsUpWithRead(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/security/keys":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"key1"}`))
		case "GET /api/v1/security/keys/key1":
			_, _ = w.Write([]byte(`{"uuid":"key1","name":"deploy","fingerprint":"SHA256:abc"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\n..."
	k, err := c.CreatePrivateKey(context.Background(), PrivateKeyRequest{PrivateKey: &pem})
	if err != nil {
		t.Fatalf("CreatePrivateKey: %v", err)
	}
	if k.Fingerprint != "SHA256:abc" {
		t.Errorf("fingerprint = %q", k.Fingerprint)
	}
}

func TestUpdatePrivateKeyUsesUUIDPath(t *testing.T) {
	var patched string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patched = r.URL.Path
		}
		_, _ = w.Write([]byte(`{"uuid":"key1","name":"renamed"}`))
	}))

	name := "renamed"
	if _, err := c.UpdatePrivateKey(context.Background(), "key1", PrivateKeyRequest{Name: &name}); err != nil {
		t.Fatalf("UpdatePrivateKey: %v", err)
	}
	// main-branch routes address keys by uuid (the published OpenAPI lags).
	if want := "/api/v1/security/keys/key1"; patched != want {
		t.Errorf("PATCH path = %q, want %q", patched, want)
	}
}

func TestDuplicateFingerprintSurfaces422(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Private key already exists."}`))
	}))
	pem := "-----BEGIN..."
	_, err := c.CreatePrivateKey(context.Background(), PrivateKeyRequest{PrivateKey: &pem})
	if err == nil || IsNotFound(err) {
		t.Fatalf("expected 422 error, got %v", err)
	}
}
