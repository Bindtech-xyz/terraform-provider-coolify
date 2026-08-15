package provider

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"golang.org/x/crypto/ssh"
)

// testAccThrowawayKey generates a fresh ed25519 key per run — Coolify
// validates key material and rejects duplicated fingerprints, so a fixed
// constant would break on the second run against the same instance.
func testAccThrowawayKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating throwaway key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "tf-acc throwaway")
	if err != nil {
		t.Fatalf("marshalling throwaway key: %v", err)
	}
	return string(pem.EncodeToMemory(block))
}

func TestAccPrivateKeyResource(t *testing.T) {
	config := `
resource "coolify_private_key" "test" {
  name        = "tf-acc-key"
  description = "created by acceptance tests"
  private_key = <<-EOT
` + testAccThrowawayKey(t) + `
EOT
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_private_key.test",
						tfjsonpath.New("name"), knownvalue.StringExact("tf-acc-key")),
					statecheck.ExpectKnownValue("coolify_private_key.test",
						tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("coolify_private_key.test",
						tfjsonpath.New("fingerprint"), knownvalue.NotNull()),
				},
			},
		},
	})
}
