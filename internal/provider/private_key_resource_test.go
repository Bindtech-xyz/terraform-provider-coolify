package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// A throwaway ed25519 key generated for acceptance testing only — never used
// anywhere.
const testAccThrowawayKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBoNJUUBJhDaVHzLoZVYcVGWk8UmiCiKvDZzZZDdIvHzwAAAJgh8m9uIfJv
bgAAAAtzc2gtZWQyNTUxOQAAACBoNJUUBJhDaVHzLoZVYcVGWk8UmiCiKvDZzZZDdIvHzw
AAAEAxbcM4LnDpv4H2AH8dnLXcSIF6dHW04d5PrCUeXwjBQGg0lRQEmENpUfMuhlVhxUZa
TxSaIKIq8NnNlkN0i8fPAAAAEHRmLWFjY0BleGFtcGxlLjEBAgMEBQ==
-----END OPENSSH PRIVATE KEY-----`

func TestAccPrivateKeyResource(t *testing.T) {
	config := `
resource "coolify_private_key" "test" {
  name        = "tf-acc-key"
  description = "created by acceptance tests"
  private_key = <<-EOT
` + testAccThrowawayKey + `
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
