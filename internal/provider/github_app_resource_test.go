package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestGithubAppUpdateOmitsUnchangedPrivateKeyUUID locks in a regression
// reported live: PATCH /github-apps/{id} validates private_key_uuid with
// Laravel's `uuid` rule (RFC 4122), but Coolify's own private key ids are
// short nanoids, so no value can ever pass — every PATCH that included the
// field 422ed, and since the API never echoes it back as a uuid (only
// private_key_id, resolved separately by resolvePrivateKeyUUID), the field
// always differed from a freshly-imported null state, so *every* apply after
// import hit this even when nothing about the private key changed.
// githubAppUpdateRequest must omit the field from the PATCH body whenever
// it's unchanged from state.
func TestGithubAppUpdateOmitsUnchangedPrivateKeyUUID(t *testing.T) {
	m := githubAppResourceModel{
		ID:             types.Int64Value(42),
		PrivateKeyUUID: types.StringValue("same-uuid"),
	}
	body := githubAppUpdateRequest(m, m)
	if body.PrivateKeyUUID != nil {
		t.Errorf("PATCH body must omit private_key_uuid when unchanged, got %+v", *body.PrivateKeyUUID)
	}
}

// TestGithubAppUpdateSendsChangedPrivateKeyUUID confirms the omission is
// conditional, not blanket — a genuine change is still attempted (and will
// still 422 today until upstream fixes the validator, but the provider must
// not silently swallow an intentional change).
func TestGithubAppUpdateSendsChangedPrivateKeyUUID(t *testing.T) {
	state := githubAppResourceModel{
		ID:             types.Int64Value(42),
		PrivateKeyUUID: types.StringValue("old-uuid"),
	}
	plan := state
	plan.PrivateKeyUUID = types.StringValue("new-uuid")

	body := githubAppUpdateRequest(plan, state)
	if body.PrivateKeyUUID == nil || *body.PrivateKeyUUID != "new-uuid" {
		t.Errorf("PATCH body must send the changed private_key_uuid, got %+v", body.PrivateKeyUUID)
	}
}

// TestResolvePrivateKeyUUID covers both outcomes of the id->uuid lookup
// ImportState/githubAppToModel rely on to keep private_key_uuid accurate
// (the API only ever returns the numeric private_key_id).
func TestResolvePrivateKeyUUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":7,"uuid":"key-uuid-7"},{"id":9,"uuid":"key-uuid-9"}]`))
	}))
	t.Cleanup(srv.Close)
	c, err := client.New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	uuid, err := resolvePrivateKeyUUID(context.Background(), c, 9)
	if err != nil {
		t.Fatalf("resolvePrivateKeyUUID: %v", err)
	}
	if uuid != "key-uuid-9" {
		t.Errorf("uuid = %q, want %q", uuid, "key-uuid-9")
	}

	if _, err := resolvePrivateKeyUUID(context.Background(), c, 404); err == nil {
		t.Error("resolvePrivateKeyUUID should error when no key matches the id")
	}
}
