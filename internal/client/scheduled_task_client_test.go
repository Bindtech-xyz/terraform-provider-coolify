package client

import (
	"context"
	"net/http"
	"testing"
)

func TestScheduledTaskLifecycle(t *testing.T) {
	var paths []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"task1"}`))
		case http.MethodDelete:
			_, _ = w.Write([]byte(`{"message":"deleted"}`))
		default:
			_, _ = w.Write([]byte(`[{"uuid":"task1","name":"cleanup","command":"true","frequency":"@daily","enabled":true}]`))
		}
	}))

	ctx := context.Background()
	name, cmd, freq := "cleanup", "true", "@daily"
	task, err := c.CreateScheduledTask(ctx, ScheduledTaskParentApplication, "app1", ScheduledTaskRequest{
		Name: &name, Command: &cmd, Frequency: &freq,
	})
	if err != nil {
		t.Fatalf("CreateScheduledTask: %v", err)
	}
	if task.Frequency != "@daily" || !task.Enabled {
		t.Errorf("task = %+v", task)
	}
	if err := c.DeleteScheduledTask(ctx, ScheduledTaskParentApplication, "app1", "task1"); err != nil {
		t.Fatalf("DeleteScheduledTask: %v", err)
	}
	if want := "POST /api/v1/applications/app1/scheduled-tasks"; paths[0] != want {
		t.Errorf("create path = %q, want %q", paths[0], want)
	}
	if want := "DELETE /api/v1/applications/app1/scheduled-tasks/task1"; paths[len(paths)-1] != want {
		t.Errorf("delete path = %q, want %q", paths[len(paths)-1], want)
	}
}

func TestScheduledTaskServicePath(t *testing.T) {
	got, err := scheduledTaskBase(ScheduledTaskParentService, "svc1")
	if err != nil {
		t.Fatal(err)
	}
	if want := "/services/svc1/scheduled-tasks"; got != want {
		t.Errorf("base = %q, want %q", got, want)
	}
	if _, err := scheduledTaskBase("database", "x"); err == nil {
		t.Error("databases do not support scheduled tasks; expected error")
	}
}
