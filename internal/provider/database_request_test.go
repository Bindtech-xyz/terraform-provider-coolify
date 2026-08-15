package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Placement and instant_deploy are create-only on databases; sending them on
// PATCH would be rejected as not-allowed fields.
func TestDatabaseToRequestPlacementGating(t *testing.T) {
	model := databaseResourceModel{
		Engine:           types.StringValue("postgresql"),
		ProjectUUID:      types.StringValue("proj1"),
		ServerUUID:       types.StringValue("srv1"),
		EnvironmentName:  types.StringValue("production"),
		InstantDeploy:    types.BoolValue(true),
		Name:             types.StringValue("db"),
		PostgresPassword: types.StringValue("secret"),
	}

	create := databaseToRequest(model, true)
	if create.ProjectUUID == nil || create.ServerUUID == nil || create.InstantDeploy == nil {
		t.Error("create request must include placement and instant_deploy")
	}

	update := databaseToRequest(model, false)
	if update.ProjectUUID != nil || update.ServerUUID != nil || update.InstantDeploy != nil {
		t.Error("update request must not include placement or instant_deploy")
	}
	if update.PostgresPassword == nil {
		t.Error("update request must keep engine credentials")
	}
}

// Engine-specific fields must only be sent when configured — Coolify rejects
// e.g. postgres_user on a redis database with a 422.
func TestDatabaseToRequestOmitsForeignEngineFields(t *testing.T) {
	model := databaseResourceModel{
		Engine:        types.StringValue("redis"),
		RedisPassword: types.StringValue("secret"),
	}
	req := databaseToRequest(model, false)
	if req.RedisPassword == nil {
		t.Error("redis_password must be sent")
	}
	if req.PostgresUser != nil || req.MysqlUser != nil || req.MongoInitdbDatabase != nil {
		t.Error("unset foreign-engine fields must stay nil")
	}
}
