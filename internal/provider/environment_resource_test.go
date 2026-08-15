package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccEnvironmentResource exercises the project → environment →
// shared-variable chain, the backbone of any multi-app layout.
func TestAccEnvironmentResource(t *testing.T) {
	config := testAccProviderConfig() + `
resource "coolify_project" "test" {
  name = "tf-acc-env-project"
}

resource "coolify_environment" "test" {
  project_uuid = coolify_project.test.uuid
  name         = "staging"
  description  = "created by acceptance tests"
}

resource "coolify_shared_environment_variable" "test" {
  scope        = "environment"
  project_uuid = coolify_project.test.uuid
  environment  = coolify_environment.test.name
  key          = "TF_ACC_SHARED"
  value        = "42"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_environment.test",
						tfjsonpath.New("name"), knownvalue.StringExact("staging")),
					statecheck.ExpectKnownValue("coolify_environment.test",
						tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("coolify_shared_environment_variable.test",
						tfjsonpath.New("key"), knownvalue.StringExact("TF_ACC_SHARED")),
				},
			},
		},
	})
}
