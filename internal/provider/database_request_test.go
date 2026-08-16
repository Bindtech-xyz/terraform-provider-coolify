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

// TestDatabaseToRequestSendsPostgresInitdbFields confirms
// postgres_initdb_args/postgres_host_auth_method are wired through to the
// request — added late (already present in the client since these are
// accepted by Coolify on both create and update, but never exposed on the
// resource schema until requested live: postgres_initdb_args is the only
// field that fixes a PostgreSQL cluster's locale/collation, which only takes
// effect on the very first start, so it's the one field a destroy/recreate
// needs to reproduce an identical database).
func TestDatabaseToRequestSendsPostgresInitdbFields(t *testing.T) {
	model := databaseResourceModel{
		Engine:                 types.StringValue("postgresql"),
		PostgresInitdbArgs:     types.StringValue("--data-checksums --locale=C"),
		PostgresHostAuthMethod: types.StringValue("scram-sha-256"),
	}
	req := databaseToRequest(model, false)
	if req.PostgresInitdbArgs == nil || *req.PostgresInitdbArgs != "--data-checksums --locale=C" {
		t.Error("postgres_initdb_args must be sent")
	}
	if req.PostgresHostAuthMethod == nil || *req.PostgresHostAuthMethod != "scram-sha-256" {
		t.Error("postgres_host_auth_method must be sent")
	}
}

// TestDatabaseToRequestSendsExtendedLimits confirms the five limits_* fields
// beyond limits_memory/limits_cpus (limits_memory_swap,
// limits_memory_swappiness, limits_memory_reservation, limits_cpuset,
// limits_cpu_shares) are sent unconditionally on both create and update —
// confirmed against the controller source, no whitelist mismatch like
// dockerfile/dockerfile_target_build has.
func TestDatabaseToRequestSendsExtendedLimits(t *testing.T) {
	model := databaseResourceModel{
		Engine:                  types.StringValue("postgresql"),
		LimitsMemorySwap:        types.StringValue("1g"),
		LimitsMemorySwappiness:  types.Int64Value(60),
		LimitsMemoryReservation: types.StringValue("256m"),
		LimitsCPUSet:            types.StringValue("0,1"),
		LimitsCPUShares:         types.Int64Value(1024),
	}

	for _, create := range []bool{true, false} {
		req := databaseToRequest(model, create)
		if req.LimitsMemorySwap == nil || *req.LimitsMemorySwap != "1g" ||
			req.LimitsMemorySwappiness == nil || *req.LimitsMemorySwappiness != 60 ||
			req.LimitsMemoryReservation == nil || *req.LimitsMemoryReservation != "256m" ||
			req.LimitsCPUSet == nil || *req.LimitsCPUSet != "0,1" ||
			req.LimitsCPUShares == nil || *req.LimitsCPUShares != 1024 {
			t.Errorf("extended limits must be sent regardless of create=%v, got %+v", create, req)
		}
	}
}
