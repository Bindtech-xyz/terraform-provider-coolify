package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
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

// TestDatabaseUpdateNeverLeaksForeignEngineFields is the exact regression
// reported live: create a postgresql database, read it back the way Coolify
// actually responds (every other engine's fields simply absent from the
// JSON, decoding to "" on the Go string fields — never present, never a
// configured value), then update a common field like limits_memory. The PATCH
// body must not carry a single key belonging to any engine other than the
// one configured, no matter what stale/empty values ended up in state.
//
// This is a two-layer fix and either alone would have masked the other:
// keepPriorIfHidden used to turn "hidden and never configured" into
// types.StringValue("") instead of null, and databaseToRequest sent every
// engine's fields unconditionally (stringOrNil only nils true nil, not "").
// Together they turned Coolify's normal "field absent" response into a body
// listing every foreign field as "This field is not allowed." on the very
// first update after creation.
func TestDatabaseUpdateNeverLeaksForeignEngineFields(t *testing.T) {
	// What Coolify's GET actually returns for a postgresql-only database:
	// every other engine's field is simply absent from the JSON, so it
	// decodes to Go's zero value, "".
	api := &client.Database{
		UUID:         "db1",
		LimitsMemory: "8g",
		LimitsCPUs:   "6",
		PostgresUser: "postgres",
		PostgresDB:   "app",
	}
	state := databaseToModel(api, databaseResourceModel{Engine: types.StringValue("postgresql")})

	body := databaseToRequest(state, false)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, foreign := range []string{
		"mysql_root_password", "mysql_password", "mysql_user", "mysql_database",
		"mariadb_root_password", "mariadb_password", "mariadb_user", "mariadb_database",
		"mongo_initdb_root_username", "mongo_initdb_root_password", "mongo_initdb_database",
		"redis_password", "keydb_password", "dragonfly_password",
		"clickhouse_admin_user", "clickhouse_admin_password",
	} {
		if _, present := got[foreign]; present {
			t.Errorf("PATCH body must not contain foreign-engine field %q, got %+v", foreign, got)
		}
	}
	if got["postgres_user"] != "postgres" {
		t.Errorf("postgres_user must survive the round trip, got %+v", got)
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

// TestDatabaseToRequestHealthCheckUpdateOnly locks in the same class of bug
// as coolify_application's dockerfile_target_build: Coolify's health check
// fields on a database are accepted by update_by_uuid but absent from every
// create endpoint's allowed-fields list (confirmed against the controller
// source and the add_health_check_to_standalone_databases migration, which
// only wired these columns into update, not create). databaseToRequest must
// send them only when !create; Create() applies a configured value via a
// follow-up UpdateDatabase call instead.
func TestDatabaseToRequestHealthCheckUpdateOnly(t *testing.T) {
	model := databaseResourceModel{
		Engine:                 types.StringValue("postgresql"),
		HealthCheckEnabled:     types.BoolValue(true),
		HealthCheckInterval:    types.Int64Value(15),
		HealthCheckTimeout:     types.Int64Value(5),
		HealthCheckRetries:     types.Int64Value(5),
		HealthCheckStartPeriod: types.Int64Value(5),
	}

	create := databaseToRequest(model, true)
	if create.HealthCheckEnabled != nil || create.HealthCheckInterval != nil ||
		create.HealthCheckTimeout != nil || create.HealthCheckRetries != nil ||
		create.HealthCheckStartPeriod != nil {
		t.Errorf("create request must never include health check fields, got %+v", create)
	}

	update := databaseToRequest(model, false)
	if update.HealthCheckEnabled == nil || !*update.HealthCheckEnabled ||
		update.HealthCheckInterval == nil || *update.HealthCheckInterval != 15 ||
		update.HealthCheckTimeout == nil || *update.HealthCheckTimeout != 5 ||
		update.HealthCheckRetries == nil || *update.HealthCheckRetries != 5 ||
		update.HealthCheckStartPeriod == nil || *update.HealthCheckStartPeriod != 5 {
		t.Errorf("update request must include health check fields, got %+v", update)
	}
}

// TestDatabaseHasConfiguredHealthCheck covers the helper Create() uses to
// decide whether the update-only follow-up PATCH is needed at all.
func TestDatabaseHasConfiguredHealthCheck(t *testing.T) {
	if databaseHasConfiguredHealthCheck(databaseResourceModel{}) {
		t.Error("no health check attributes configured must report false")
	}
	if !databaseHasConfiguredHealthCheck(databaseResourceModel{HealthCheckRetries: types.Int64Value(3)}) {
		t.Error("a single configured health check attribute must report true")
	}
}
