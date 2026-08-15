package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccProjectResource runs the full lifecycle against a real Coolify
// instance: create, import, update, destroy.
//
//	COOLIFY_ENDPOINT=... COOLIFY_TOKEN=... TF_ACC=1 \
//	  go test ./internal/provider/ -v -run TestAccProjectResource
func TestAccProjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read.
			{
				Config: testAccProjectResourceConfig("tf-acc-project", "created by acceptance tests"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_project.test",
						tfjsonpath.New("name"), knownvalue.StringExact("tf-acc-project")),
					statecheck.ExpectKnownValue("coolify_project.test",
						tfjsonpath.New("description"), knownvalue.StringExact("created by acceptance tests")),
					statecheck.ExpectKnownValue("coolify_project.test",
						tfjsonpath.New("uuid"), knownvalue.NotNull()),
				},
			},
			// ImportState round-trips the same attributes.
			{
				ResourceName:                         "coolify_project.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
			// Update in place (no replacement expected).
			{
				Config: testAccProjectResourceConfig("tf-acc-project-renamed", "updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_project.test",
						tfjsonpath.New("name"), knownvalue.StringExact("tf-acc-project-renamed")),
				},
			},
			// Destroy is exercised automatically as the final step.
		},
	})
}

func testAccProjectResourceConfig(name, description string) string {
	return `
resource "coolify_project" "test" {
  name        = "` + name + `"
  description = "` + description + `"
}
`
}
