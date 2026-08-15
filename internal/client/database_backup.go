package client

import (
	"context"
	"fmt"
	"net/url"
)

// DatabaseBackup is a scheduled backup configuration on a standalone database
// (docs: databases/backups), optionally shipping to an S3 storage.
type DatabaseBackup struct {
	ID                int64  `json:"id"`
	UUID              string `json:"uuid"`
	Frequency         string `json:"frequency"`
	Enabled           bool   `json:"enabled"`
	SaveS3            bool   `json:"save_s3"`
	S3StorageID       *int64 `json:"s3_storage_id"`
	DumpAll           bool   `json:"dump_all"`
	DatabasesToBackup string `json:"databases_to_backup"`
	Timeout           int64  `json:"timeout"`

	RetentionAmountLocally     int64   `json:"database_backup_retention_amount_locally"`
	RetentionDaysLocally       int64   `json:"database_backup_retention_days_locally"`
	RetentionMaxStorageLocally float64 `json:"database_backup_retention_max_storage_locally"`
	RetentionAmountS3          int64   `json:"database_backup_retention_amount_s3"`
	RetentionDaysS3            int64   `json:"database_backup_retention_days_s3"`
	RetentionMaxStorageS3      float64 `json:"database_backup_retention_max_storage_s3"`
}

// DatabaseBackupRequest is the create/update body.
type DatabaseBackupRequest struct {
	Frequency         *string `json:"frequency,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
	SaveS3            *bool   `json:"save_s3,omitempty"`
	S3StorageUUID     *string `json:"s3_storage_uuid,omitempty"`
	DumpAll           *bool   `json:"dump_all,omitempty"`
	DatabasesToBackup *string `json:"databases_to_backup,omitempty"`
	Timeout           *int64  `json:"timeout,omitempty"`
	BackupNow         *bool   `json:"backup_now,omitempty"`

	RetentionAmountLocally     *int64   `json:"database_backup_retention_amount_locally,omitempty"`
	RetentionDaysLocally       *int64   `json:"database_backup_retention_days_locally,omitempty"`
	RetentionMaxStorageLocally *float64 `json:"database_backup_retention_max_storage_locally,omitempty"`
	RetentionAmountS3          *int64   `json:"database_backup_retention_amount_s3,omitempty"`
	RetentionDaysS3            *int64   `json:"database_backup_retention_days_s3,omitempty"`
	RetentionMaxStorageS3      *float64 `json:"database_backup_retention_max_storage_s3,omitempty"`
}

// ListDatabaseBackups returns the backup configurations of a database.
func (c *Client) ListDatabaseBackups(ctx context.Context, databaseUUID string) ([]DatabaseBackup, error) {
	var out []DatabaseBackup
	if err := c.get(ctx, "/databases/"+url.PathEscape(databaseUUID)+"/backups", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetDatabaseBackup returns one backup configuration, via the list endpoint.
func (c *Client) GetDatabaseBackup(ctx context.Context, databaseUUID, uuid string) (*DatabaseBackup, error) {
	backups, err := c.ListDatabaseBackups(ctx, databaseUUID)
	if err != nil {
		return nil, err
	}
	for _, b := range backups {
		if b.UUID == uuid {
			return &b, nil
		}
	}
	return nil, &Error{Method: "GET", Path: "database backups", StatusCode: 404, Message: "Backup configuration not found."}
}

// CreateDatabaseBackup creates a backup schedule and returns it refreshed.
func (c *Client) CreateDatabaseBackup(ctx context.Context, databaseUUID string, req DatabaseBackupRequest) (*DatabaseBackup, error) {
	var created uuidResponse
	base := "/databases/" + url.PathEscape(databaseUUID) + "/backups"
	if err := c.post(ctx, base, req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST %s: API returned no uuid", base)
	}
	return c.GetDatabaseBackup(ctx, databaseUUID, created.UUID)
}

// UpdateDatabaseBackup updates a backup schedule and returns it refreshed.
func (c *Client) UpdateDatabaseBackup(ctx context.Context, databaseUUID, uuid string, req DatabaseBackupRequest) (*DatabaseBackup, error) {
	if err := c.patch(ctx, "/databases/"+url.PathEscape(databaseUUID)+"/backups/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetDatabaseBackup(ctx, databaseUUID, uuid)
}

// DeleteDatabaseBackup removes a backup schedule (past executions follow the
// API's own cleanup rules).
func (c *Client) DeleteDatabaseBackup(ctx context.Context, databaseUUID, uuid string) error {
	return c.delete(ctx, "/databases/"+url.PathEscape(databaseUUID)+"/backups/"+url.PathEscape(uuid))
}
