package client

import (
	"context"
	"net/http"
	"testing"
)

func TestDatabaseBackupLifecyclePaths(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"bk1","message":"Backup configuration created successfully."}`))
		case http.MethodPatch, http.MethodDelete:
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		default:
			_, _ = w.Write([]byte(`[{"uuid":"bk1","frequency":"@daily","enabled":true,"save_s3":true}]`))
		}
	}))

	ctx := context.Background()
	freq := "@daily"
	backup, err := c.CreateDatabaseBackup(ctx, "db1", DatabaseBackupRequest{Frequency: &freq})
	if err != nil {
		t.Fatalf("CreateDatabaseBackup: %v", err)
	}
	if backup.Frequency != "@daily" || !backup.SaveS3 {
		t.Errorf("backup = %+v", backup)
	}
	if _, err := c.UpdateDatabaseBackup(ctx, "db1", "bk1", DatabaseBackupRequest{Frequency: &freq}); err != nil {
		t.Fatalf("UpdateDatabaseBackup: %v", err)
	}
	if err := c.DeleteDatabaseBackup(ctx, "db1", "bk1"); err != nil {
		t.Fatalf("DeleteDatabaseBackup: %v", err)
	}

	wants := map[int]string{
		0: "POST /api/v1/databases/db1/backups",
		2: "PATCH /api/v1/databases/db1/backups/bk1",
	}
	for i, want := range wants {
		if paths[i] != want {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], want)
		}
	}
	if last := paths[len(paths)-1]; last != "DELETE /api/v1/databases/db1/backups/bk1" {
		t.Errorf("delete path = %q", last)
	}
}

func TestVolumeBackupUpsertUsesPUT(t *testing.T) {
	var got string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		_, _ = w.Write([]byte(`{"uuid":"vb1","frequency":"@daily","enabled":true}`))
	}))

	freq := "@daily"
	schedule, err := c.UpsertVolumeBackupSchedule(context.Background(), EnvVarParentApplication, "app1", "st1",
		VolumeBackupScheduleRequest{Frequency: &freq})
	if err != nil {
		t.Fatalf("UpsertVolumeBackupSchedule: %v", err)
	}
	if want := "PUT /api/v1/applications/app1/storages/st1/backups"; got != want {
		t.Errorf("call = %q, want %q", got, want)
	}
	if schedule.UUID != "vb1" {
		t.Errorf("uuid = %q", schedule.UUID)
	}
}
