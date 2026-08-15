package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/d3nailabs/terraform-provider-coolify/internal/client"
)

// TestAccApplicationResource deploys a real container end to end: create,
// wait for Coolify to report it running, then destroy. Uses docker-image
// mode (nginx:alpine) rather than a git source — no clone, no build, no
// external repository to go stale; it still exercises the full application
// lifecycle, including DeleteApplication's async-delete polling against a
// container that is actually running (not just created and immediately
// torn down), which is the scenario that mattered most for that code path.
//
// This test needs COOLIFY_ACC_SERVER_UUID: an existing, usable server with
// exactly one destination (so destination_uuid can stay unset) that this
// process is allowed to deploy throwaway containers on and delete. It is
// skipped, not failed, when unset — most environments running the rest of
// the acceptance suite have no such server available.
func TestAccApplicationResource(t *testing.T) {
	serverUUID := os.Getenv("COOLIFY_ACC_SERVER_UUID")
	if serverUUID == "" {
		t.Skip("COOLIFY_ACC_SERVER_UUID not set — skipping (needs a real deployable server)")
	}

	config := testAccProviderConfig() + fmt.Sprintf(`
resource "coolify_project" "test" {
  name = "tf-acc-app-project"
}

resource "coolify_application" "test" {
  project_uuid      = coolify_project.test.uuid
  environment_name  = "production"
  server_uuid       = %q

  name = "tf-acc-app-nginx"

  docker_registry_image_name = "nginx"
  docker_registry_image_tag  = "alpine"
  ports_exposes              = "80"

  autogenerate_domain = false
  instant_deploy      = true
}
`, serverUUID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("coolify_application.test",
						tfjsonpath.New("uuid"), knownvalue.NotNull()),
					// build_pack/git_branch/git_commit_sha/base_directory/limits_*
					// are Coolify-assigned defaults for a docker-image app the
					// config never set — asserting they resolve to a known
					// (non-null, non-error) value is the regression coverage for
					// the Optional-without-Computed crash this test uncovered.
					statecheck.ExpectKnownValue("coolify_application.test",
						tfjsonpath.New("build_pack"), knownvalue.StringExact("dockerimage")),
					statecheck.ExpectKnownValue("coolify_application.test",
						tfjsonpath.New("limits_cpus"), knownvalue.StringExact("0")),
				},
				Check: testAccWaitForApplicationRunning("coolify_application.test", 2*time.Minute),
			},
		},
	})
}

// testAccWaitForApplicationRunning polls the application's status after
// apply until it reports "running:*" — Create only queues the deployment and
// returns immediately, so the container is not necessarily up yet by the
// time the TestStep's state checks run. Builds its own client straight from
// the same environment variables the provider itself reads (rather than
// reaching into the provider under test), since a resource.TestCheckFunc has
// no access to the provider instance's internal state.
func testAccWaitForApplicationRunning(resourceName string, timeout time.Duration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		uuid := rs.Primary.Attributes["uuid"]

		var opts []client.Option
		if id, secret := os.Getenv("CF_ACCESS_CLIENT_ID"), os.Getenv("CF_ACCESS_CLIENT_SECRET"); id != "" && secret != "" {
			opts = append(opts, client.WithExtraHeaders(map[string]string{
				"CF-Access-Client-Id":     id,
				"CF-Access-Client-Secret": secret,
			}))
		}
		apiClient, err := client.New(os.Getenv("COOLIFY_ENDPOINT"), os.Getenv("COOLIFY_TOKEN"), opts...)
		if err != nil {
			return fmt.Errorf("building status-polling client: %w", err)
		}

		deadline := time.Now().Add(timeout)
		for {
			app, err := apiClient.GetApplication(context.Background(), uuid)
			if err != nil {
				return fmt.Errorf("polling application %s status: %w", uuid, err)
			}
			if strings.HasPrefix(app.Status, "running:") {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("application %s did not reach running within %s (last status: %s)", uuid, timeout, app.Status)
			}
			time.Sleep(3 * time.Second)
		}
	}
}
