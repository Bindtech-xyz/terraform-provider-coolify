package client

import (
	"context"
	"fmt"
	"net/url"
)

// VolumeBackupSchedule is a backup schedule attached to a persistent storage
// of an application, service or database (PUT
// /{parent}/{uuid}/storages/{storage_uuid}/backups).
type VolumeBackupSchedule struct {
	UUID               string `json:"uuid"`
	Frequency          string `json:"frequency"`
	Enabled            bool   `json:"enabled"`
	SaveS3             bool   `json:"save_s3"`
	DisableLocalBackup bool   `json:"disable_local_backup"`
	StopDuringBackup   bool   `json:"stop_during_backup"`
	Timeout            int64  `json:"timeout"`

	RetentionAmountLocally int64 `json:"retention_amount_locally"`
	RetentionDaysLocally   int64 `json:"retention_days_locally"`
	RetentionAmountS3      int64 `json:"retention_amount_s3"`
	RetentionDaysS3        int64 `json:"retention_days_s3"`
}

// VolumeBackupScheduleRequest is the upsert body.
type VolumeBackupScheduleRequest struct {
	Frequency          *string `json:"frequency,omitempty"`
	Enabled            *bool   `json:"enabled,omitempty"`
	SaveS3             *bool   `json:"save_s3,omitempty"`
	DisableLocalBackup *bool   `json:"disable_local_backup,omitempty"`
	StopDuringBackup   *bool   `json:"stop_during_backup,omitempty"`
	S3StorageUUID      *string `json:"s3_storage_uuid,omitempty"`
	Timeout            *int64  `json:"timeout,omitempty"`

	RetentionAmountLocally *int64 `json:"retention_amount_locally,omitempty"`
	RetentionDaysLocally   *int64 `json:"retention_days_locally,omitempty"`
	RetentionAmountS3      *int64 `json:"retention_amount_s3,omitempty"`
	RetentionDaysS3        *int64 `json:"retention_days_s3,omitempty"`
}

func volumeBackupPath(parent EnvVarParent, parentUUID, storageUUID string) (string, error) {
	base, ok := envVarParentPaths[parent]
	if !ok {
		return "", fmt.Errorf("unknown volume backup parent %q", parent)
	}
	return base + url.PathEscape(parentUUID) + "/storages/" + url.PathEscape(storageUUID) + "/backups", nil
}

// UpsertVolumeBackupSchedule creates or updates the schedule of a storage.
// The endpoint is a PUT: one schedule per storage.
func (c *Client) UpsertVolumeBackupSchedule(ctx context.Context, parent EnvVarParent, parentUUID, storageUUID string, req VolumeBackupScheduleRequest) (*VolumeBackupSchedule, error) {
	path, err := volumeBackupPath(parent, parentUUID, storageUUID)
	if err != nil {
		return nil, err
	}
	var out VolumeBackupSchedule
	if err := c.do(ctx, "PUT", path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteVolumeBackupSchedule removes the schedule of a storage.
func (c *Client) DeleteVolumeBackupSchedule(ctx context.Context, parent EnvVarParent, parentUUID, storageUUID string) error {
	path, err := volumeBackupPath(parent, parentUUID, storageUUID)
	if err != nil {
		return err
	}
	return c.delete(ctx, path)
}

// RunVolumeBackup triggers an immediate backup of the storage.
func (c *Client) RunVolumeBackup(ctx context.Context, parent EnvVarParent, parentUUID, storageUUID string) error {
	path, err := volumeBackupPath(parent, parentUUID, storageUUID)
	if err != nil {
		return err
	}
	return c.post(ctx, path+"/run", nil, nil)
}
