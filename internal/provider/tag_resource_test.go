package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTagResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "coolify_tag" "test" { name = "tf-acc-tag" }`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_tag.test",
						tfjsonpath.New("name"), knownvalue.StringExact("tf-acc-tag")),
					statecheck.ExpectKnownValue("coolify_tag.test",
						tfjsonpath.New("uuid"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:                         "coolify_tag.test",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccImportByUUID("coolify_tag.test"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
			},
			{
				Config: `resource "coolify_tag" "test" { name = "tf-acc-tag-renamed" }`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_tag.test",
						tfjsonpath.New("name"), knownvalue.StringExact("tf-acc-tag-renamed")),
				},
			},
		},
	})
}
