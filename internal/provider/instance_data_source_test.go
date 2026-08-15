package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccInstanceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "coolify_instance" "this" {}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.coolify_instance.this",
						tfjsonpath.New("healthy"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("data.coolify_instance.this",
						tfjsonpath.New("version"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccServiceTemplatesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "coolify_service_templates" "all" {}`,
				ConfigStateChecks: []statecheck.StateCheck{
					// Assert on gitea, one of the oldest entries of the live
					// catalog (asserting on the full list would chase a moving
					// target — the feed gains and loses entries with releases).
					statecheck.ExpectKnownValue("data.coolify_service_templates.all",
						tfjsonpath.New("templates").AtMapKey("gitea").AtMapKey("category"),
						knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccTeamDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "coolify_team" "current" {}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.coolify_team.current",
						tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
		},
	})
}
