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

// testAccProviderConfig prepends an explicit `provider "coolify" {}` block
// carrying edge-auth headers when CF_ACCESS_CLIENT_ID/CF_ACCESS_CLIENT_SECRET
// are set — needed when the target instance sits behind an authenticating
// proxy (Cloudflare Access, ...). endpoint/token are deliberately left out
// here: they still come from COOLIFY_ENDPOINT/COOLIFY_TOKEN, matched by every
// test's testAccPreCheck. Returns "" when neither env var is set, so tests
// against a directly-reachable instance are unaffected — every TestStep
// prepends this, not just the ones added for this scenario.
func testAccProviderConfig() string {
	id := os.Getenv("CF_ACCESS_CLIENT_ID")
	secret := os.Getenv("CF_ACCESS_CLIENT_SECRET")
	if id == "" || secret == "" {
		return ""
	}
	return fmt.Sprintf(`
provider "coolify" {
  headers = {
    "CF-Access-Client-Id"     = %q
    "CF-Access-Client-Secret" = %q
  }
}
`, id, secret)
}
