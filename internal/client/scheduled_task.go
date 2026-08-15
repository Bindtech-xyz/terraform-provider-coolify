package client

import (
	"context"
	"fmt"
	"net/url"
)

// ScheduledTaskParent is the kind of resource a scheduled task runs against.
// Only applications and services support them.
type ScheduledTaskParent string

const (
	ScheduledTaskParentApplication ScheduledTaskParent = "application"
	ScheduledTaskParentService     ScheduledTaskParent = "service"
)

// ScheduledTask mirrors the `ScheduledTask` schema (cron jobs executed in the
// resource's container; docs: knowledge-base/cron-syntax).
type ScheduledTask struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	Frequency string `json:"frequency"`
	Container string `json:"container"`
	Timeout   int64  `json:"timeout"`
	Enabled   bool   `json:"enabled"`
}

// ScheduledTaskRequest is the create/update body.
type ScheduledTaskRequest struct {
	Name      *string `json:"name,omitempty"`
	Command   *string `json:"command,omitempty"`
	Frequency *string `json:"frequency,omitempty"`
	Container *string `json:"container,omitempty"`
	Timeout   *int64  `json:"timeout,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

func scheduledTaskBase(parent ScheduledTaskParent, parentUUID string) (string, error) {
	switch parent {
	case ScheduledTaskParentApplication:
		return "/applications/" + url.PathEscape(parentUUID) + "/scheduled-tasks", nil
	case ScheduledTaskParentService:
		return "/services/" + url.PathEscape(parentUUID) + "/scheduled-tasks", nil
	default:
		return "", fmt.Errorf("unknown scheduled task parent %q", parent)
	}
}

// ListScheduledTasks returns the tasks of an application or service.
func (c *Client) ListScheduledTasks(ctx context.Context, parent ScheduledTaskParent, parentUUID string) ([]ScheduledTask, error) {
	base, err := scheduledTaskBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	var out []ScheduledTask
	if err := c.get(ctx, base, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetScheduledTask returns one task by UUID, via the list endpoint.
func (c *Client) GetScheduledTask(ctx context.Context, parent ScheduledTaskParent, parentUUID, uuid string) (*ScheduledTask, error) {
	tasks, err := c.ListScheduledTasks(ctx, parent, parentUUID)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.UUID == uuid {
			return &t, nil
		}
	}
	return nil, &Error{Method: "GET", Path: string(parent) + " scheduled-tasks", StatusCode: 404, Message: "Scheduled task not found."}
}

// CreateScheduledTask creates a task and returns it refreshed.
func (c *Client) CreateScheduledTask(ctx context.Context, parent ScheduledTaskParent, parentUUID string, req ScheduledTaskRequest) (*ScheduledTask, error) {
	base, err := scheduledTaskBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	var created uuidResponse
	if err := c.post(ctx, base, req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST %s: API returned no uuid", base)
	}
	return c.GetScheduledTask(ctx, parent, parentUUID, created.UUID)
}

// UpdateScheduledTask updates a task by UUID and returns it refreshed.
func (c *Client) UpdateScheduledTask(ctx context.Context, parent ScheduledTaskParent, parentUUID, uuid string, req ScheduledTaskRequest) (*ScheduledTask, error) {
	base, err := scheduledTaskBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	if err := c.patch(ctx, base+"/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetScheduledTask(ctx, parent, parentUUID, uuid)
}

// DeleteScheduledTask removes a task by UUID.
func (c *Client) DeleteScheduledTask(ctx context.Context, parent ScheduledTaskParent, parentUUID, uuid string) error {
	base, err := scheduledTaskBase(parent, parentUUID)
	if err != nil {
		return err
	}
	return c.delete(ctx, base+"/"+url.PathEscape(uuid))
}
