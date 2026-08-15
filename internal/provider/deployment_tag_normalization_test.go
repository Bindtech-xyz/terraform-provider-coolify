package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestDeploymentTriggerLowercasesTag locks in a regression found live:
// Coolify's /deploy?tag= does an exact match against the stored tag name,
// which is always lowercase (coolify_resource_tag and Coolify's own
// tag-creation endpoints normalize it there) — sending a mixed-case tag
// silently deploys nothing ("404: No resources found with this tag").
// trigger() must lowercase/trim before building the request.
func TestDeploymentTriggerLowercasesTag(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"deployments":[]}`))
	}))
	t.Cleanup(srv.Close)
	c, err := client.New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	r := &deploymentResource{client: c}
	m := deploymentResourceModel{
		Tag:               types.StringValue("  Sweep4-Prod  "),
		Force:             types.BoolValue(false),
		WaitForCompletion: types.BoolValue(false),
	}
	resp := &resource.CreateResponse{}
	r.trigger(context.Background(), m, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("trigger: %v", resp.Diagnostics)
	}

	if want := "tag=sweep4-prod"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
}
