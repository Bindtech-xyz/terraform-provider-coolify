package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestS3StorageValidateAndRefreshCallsValidate locks in a regression
// reported live: POST/PATCH /s3-storages never call
// S3Storage::testConnection() themselves (confirmed against the controller
// source — store() takes is_usable straight from the request body,
// defaulting to false), so a storage created by this provider stayed
// permanently is_usable = false, and any coolify_volume_backup/
// coolify_database_backup referencing it 422ed with "Select a usable S3
// storage owned by your team." validateAndRefresh must call
// POST /{uuid}/validate (the only path that ever sets is_usable = true) and
// re-read the storage so state reflects the real, just-tested value instead
// of the create response's default false.
func TestS3StorageValidateAndRefreshCallsValidate(t *testing.T) {
	var validateCalled bool
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/s3-storages/s3x/validate":
			validateCalled = true
			_, _ = w.Write([]byte(`{"valid":true,"message":"S3 storage connection is valid."}`))
		case r.URL.Path == "/api/v1/s3-storages/s3x":
			_, _ = w.Write([]byte(`{"uuid":"s3x","name":"backups","is_usable":true}`))
		}
	}))

	r := &s3StorageResource{client: c}
	var diags diag.Diagnostics
	storage := r.validateAndRefresh(context.Background(), &client.S3Storage{UUID: "s3x", IsUsable: false}, &diags)

	if diags.HasError() {
		t.Fatalf("validateAndRefresh: %v", diags)
	}
	if !validateCalled {
		t.Fatal("validateAndRefresh must call POST /s3-storages/{uuid}/validate")
	}
	if !storage.IsUsable {
		t.Errorf("is_usable = %v, want true after a successful validate+refresh", storage.IsUsable)
	}
}

// TestS3StorageValidateAndRefreshWarnsOnFailedConnection covers a genuine
// connection failure (bad credentials, unreachable endpoint): the storage
// still exists and must not be left untracked in state, so this is a
// warning, not an error — but the caller should be told why is_usable is
// false instead of finding out only when coolify_volume_backup 422s later.
func TestS3StorageValidateAndRefreshWarnsOnFailedConnection(t *testing.T) {
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/s3-storages/s3x/validate":
			_, _ = w.Write([]byte(`{"valid":false,"message":"Could not connect: Access Denied"}`))
		case r.URL.Path == "/api/v1/s3-storages/s3x":
			_, _ = w.Write([]byte(`{"uuid":"s3x","name":"backups","is_usable":false}`))
		}
	}))

	r := &s3StorageResource{client: c}
	var diags diag.Diagnostics
	storage := r.validateAndRefresh(context.Background(), &client.S3Storage{UUID: "s3x"}, &diags)

	if diags.HasError() {
		t.Fatalf("a failed connection test must be a warning, not an error: %v", diags)
	}
	if diags.WarningsCount() == 0 {
		t.Error("expected a warning naming the connection failure")
	}
	if storage.IsUsable {
		t.Error("is_usable must stay false when the connection test failed")
	}
}

func newProviderTestClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := client.New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	return c
}

func TestAccS3StorageResource(t *testing.T) {
	config := testAccProviderConfig() + `
resource "coolify_s3_storage" "test" {
  name       = "tf-acc-s3"
  endpoint   = "https://s3.eu-west-1.amazonaws.com"
  bucket     = "tf-acc-bucket"
  region     = "eu-west-1"
  access_key = "AKIAIOSFODNN7EXAMPLE"
  secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_s3_storage.test",
						tfjsonpath.New("bucket"), knownvalue.StringExact("tf-acc-bucket")),
					statecheck.ExpectKnownValue("coolify_s3_storage.test",
						tfjsonpath.New("uuid"), knownvalue.NotNull()),
				},
			},
		},
	})
}
