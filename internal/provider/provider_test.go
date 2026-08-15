package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories instantiates the provider in-process for
// acceptance tests, so terraform-plugin-testing drives the real CRUD code paths.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"coolify": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck fails fast with a readable message when the environment is
// not set up for acceptance testing. TF_ACC itself is checked by the framework.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("COOLIFY_ENDPOINT") == "" {
		t.Fatal("COOLIFY_ENDPOINT must be set for acceptance tests (URL of a disposable Coolify instance)")
	}
	if os.Getenv("COOLIFY_TOKEN") == "" {
		t.Fatal("COOLIFY_TOKEN must be set for acceptance tests")
	}
}
