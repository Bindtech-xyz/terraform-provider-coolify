package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// DatabaseEngine selects the Coolify create endpoint for a standalone database.
type DatabaseEngine string

const (
	DatabasePostgreSQL DatabaseEngine = "postgresql"
	DatabaseMySQL      DatabaseEngine = "mysql"
	DatabaseMariaDB    DatabaseEngine = "mariadb"
	DatabaseMongoDB    DatabaseEngine = "mongodb"
	DatabaseRedis      DatabaseEngine = "redis"
	DatabaseKeyDB      DatabaseEngine = "keydb"
	DatabaseDragonfly  DatabaseEngine = "dragonfly"
	DatabaseClickhouse DatabaseEngine = "clickhouse"
)

// DatabaseEngines lists every supported engine (order matters for docs).
var DatabaseEngines = []DatabaseEngine{
	DatabasePostgreSQL, DatabaseMySQL, DatabaseMariaDB, DatabaseMongoDB,
	DatabaseRedis, DatabaseKeyDB, DatabaseDragonfly, DatabaseClickhouse,
}

// Database mirrors the standalone database object returned by GET
// /databases/{uuid}. Engine-specific credentials are all present, with only the
// relevant ones populated.
type Database struct {
	ID            int64  `json:"id"`
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Image         string `json:"image"`
	IsPublic      bool   `json:"is_public"`
	PublicPort    *int64 `json:"public_port"`
	InternalDBURL string `json:"internal_db_url"`
	ExternalDBURL string `json:"external_db_url"`
	Status        string `json:"status"`
	LimitsMemory  string `json:"limits_memory"`
	LimitsCPUs    string `json:"limits_cpus"`

	// PostgreSQL.
	PostgresUser           string `json:"postgres_user"`
	PostgresPassword       string `json:"postgres_password"`
	PostgresDB             string `json:"postgres_db"`
	PostgresInitdbArgs     string `json:"postgres_initdb_args"`
	PostgresHostAuthMethod string `json:"postgres_host_auth_method"`
	PostgresConf           string `json:"postgres_conf"`

	// MySQL.
	MysqlRootPassword string `json:"mysql_root_password"`
	MysqlPassword     string `json:"mysql_password"`
	MysqlUser         string `json:"mysql_user"`
	MysqlDatabase     string `json:"mysql_database"`
	MysqlConf         string `json:"mysql_conf"`

	// MariaDB.
	MariadbRootPassword string `json:"mariadb_root_password"`
	MariadbPassword     string `json:"mariadb_password"`
	MariadbUser         string `json:"mariadb_user"`
	MariadbDatabase     string `json:"mariadb_database"`
	MariadbConf         string `json:"mariadb_conf"`

	// MongoDB.
	MongoInitdbRootUsername string `json:"mongo_initdb_root_username"`
	MongoInitdbRootPassword string `json:"mongo_initdb_root_password"`
	MongoInitdbDatabase     string `json:"mongo_initdb_database"`
	MongoConf               string `json:"mongo_conf"`

	// Redis / KeyDB / Dragonfly.
	RedisPassword     string `json:"redis_password"`
	RedisConf         string `json:"redis_conf"`
	KeydbPassword     string `json:"keydb_password"`
	KeydbConf         string `json:"keydb_conf"`
	DragonflyPassword string `json:"dragonfly_password"`

	// Clickhouse.
	ClickhouseAdminUser     string `json:"clickhouse_admin_user"`
	ClickhouseAdminPassword string `json:"clickhouse_admin_password"`
}

// DatabaseRequest is the union body for the eight create endpoints and for
// PATCH /databases/{uuid}. Only fields valid for the target engine may be set —
// Coolify rejects extras with a 422 "This field is not allowed".
type DatabaseRequest struct {
	// Placement (create only).
	ProjectUUID     *string `json:"project_uuid,omitempty"`
	EnvironmentName *string `json:"environment_name,omitempty"`
	EnvironmentUUID *string `json:"environment_uuid,omitempty"`
	ServerUUID      *string `json:"server_uuid,omitempty"`
	DestinationUUID *string `json:"destination_uuid,omitempty"`

	// Common.
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Image         *string `json:"image,omitempty"`
	IsPublic      *bool   `json:"is_public,omitempty"`
	PublicPort    *int64  `json:"public_port,omitempty"`
	InstantDeploy *bool   `json:"instant_deploy,omitempty"`
	LimitsMemory  *string `json:"limits_memory,omitempty"`
	LimitsCPUs    *string `json:"limits_cpus,omitempty"`

	// PostgreSQL.
	PostgresUser           *string `json:"postgres_user,omitempty"`
	PostgresPassword       *string `json:"postgres_password,omitempty"`
	PostgresDB             *string `json:"postgres_db,omitempty"`
	PostgresInitdbArgs     *string `json:"postgres_initdb_args,omitempty"`
	PostgresHostAuthMethod *string `json:"postgres_host_auth_method,omitempty"`
	PostgresConf           *string `json:"postgres_conf,omitempty"` // base64

	// MySQL.
	MysqlRootPassword *string `json:"mysql_root_password,omitempty"`
	MysqlPassword     *string `json:"mysql_password,omitempty"`
	MysqlUser         *string `json:"mysql_user,omitempty"`
	MysqlDatabase     *string `json:"mysql_database,omitempty"`
	MysqlConf         *string `json:"mysql_conf,omitempty"` // base64

	// MariaDB.
	MariadbRootPassword *string `json:"mariadb_root_password,omitempty"`
	MariadbPassword     *string `json:"mariadb_password,omitempty"`
	MariadbUser         *string `json:"mariadb_user,omitempty"`
	MariadbDatabase     *string `json:"mariadb_database,omitempty"`
	MariadbConf         *string `json:"mariadb_conf,omitempty"` // base64

	// MongoDB.
	MongoInitdbRootUsername *string `json:"mongo_initdb_root_username,omitempty"`
	MongoInitdbRootPassword *string `json:"mongo_initdb_root_password,omitempty"`
	MongoInitdbDatabase     *string `json:"mongo_initdb_database,omitempty"`
	MongoConf               *string `json:"mongo_conf,omitempty"` // base64

	// Redis / KeyDB / Dragonfly.
	RedisPassword     *string `json:"redis_password,omitempty"`
	RedisConf         *string `json:"redis_conf,omitempty"` // base64
	KeydbPassword     *string `json:"keydb_password,omitempty"`
	KeydbConf         *string `json:"keydb_conf,omitempty"` // base64
	DragonflyPassword *string `json:"dragonfly_password,omitempty"`

	// Clickhouse.
	ClickhouseAdminUser     *string `json:"clickhouse_admin_user,omitempty"`
	ClickhouseAdminPassword *string `json:"clickhouse_admin_password,omitempty"`
}

// ListDatabases returns every standalone database of the token's team.
func (c *Client) ListDatabases(ctx context.Context) ([]Database, error) {
	var out []Database
	if err := c.get(ctx, "/databases", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDatabase fetches one database by UUID.
func (c *Client) GetDatabase(ctx context.Context, uuid string) (*Database, error) {
	var out Database
	if err := c.get(ctx, "/databases/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateDatabase creates a standalone database of the given engine. The API
// responds with {uuid, internal_db_url[, external_db_url]}; the full object is
// fetched before returning.
func (c *Client) CreateDatabase(ctx context.Context, engine DatabaseEngine, req DatabaseRequest) (*Database, error) {
	var created uuidResponse
	path := "/databases/" + string(engine)
	if err := c.post(ctx, path, req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST %s: API returned no uuid", path)
	}
	return c.GetDatabase(ctx, created.UUID)
}

// UpdateDatabase applies a partial update and returns the refreshed object.
func (c *Client) UpdateDatabase(ctx context.Context, uuid string, req DatabaseRequest) (*Database, error) {
	if err := c.patch(ctx, "/databases/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetDatabase(ctx, uuid)
}

// DeleteDatabase removes a database and waits for the asynchronous teardown to
// finish. Nil flags keep the API defaults (all true).
func (c *Client) DeleteDatabase(ctx context.Context, uuid string, deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks *bool) error {
	if err := c.deleteWithQuery(ctx, "/databases/"+url.PathEscape(uuid),
		deletionQuery(deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks)); err != nil {
		return err
	}
	return c.waitForDeletion(ctx, 2*time.Minute, func(ctx context.Context) error {
		_, err := c.GetDatabase(ctx, uuid)
		return err
	})
}

// StartDatabase starts the database container.
func (c *Client) StartDatabase(ctx context.Context, uuid string) error {
	return c.post(ctx, "/databases/"+url.PathEscape(uuid)+"/start", nil, nil)
}

// StopDatabase stops the database container.
func (c *Client) StopDatabase(ctx context.Context, uuid string) error {
	return c.post(ctx, "/databases/"+url.PathEscape(uuid)+"/stop", nil, nil)
}

// RestartDatabase restarts the database container.
func (c *Client) RestartDatabase(ctx context.Context, uuid string) error {
	return c.post(ctx, "/databases/"+url.PathEscape(uuid)+"/restart", nil, nil)
}
