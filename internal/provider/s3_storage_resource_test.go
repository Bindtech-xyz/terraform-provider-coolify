package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccS3StorageResource(t *testing.T) {
	config := `
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
