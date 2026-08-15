package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

// testAccImportByUUID feeds `terraform import` with the resource's uuid — the
// testing framework defaults to the `id` attribute, which none of this
// provider's resources have.
func testAccImportByUUID(resourceName string) func(*terraform.State) (string, error) {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes["uuid"], nil
	}
}
