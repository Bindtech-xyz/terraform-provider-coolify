package client

import (
	"context"
	"fmt"
	"net/url"
)

// Tag mirrors the `Tag` schema. Tags are team-wide labels that can be attached
// to applications, services and databases, and used with POST /deploy?tag=.
type Tag struct {
	ID   int64  `json:"id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// TagRequest is the body for create and update (name only).
type TagRequest struct {
	Name *string `json:"name,omitempty"`
}

// ListTags returns every tag of the token's team.
func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	var out []Tag
	if err := c.get(ctx, "/tags", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTag fetches one tag by UUID. The API has no single-tag endpoint, so this
// filters the list.
func (c *Client) GetTag(ctx context.Context, uuid string) (*Tag, error) {
	tags, err := c.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		if t.UUID == uuid {
			return &t, nil
		}
	}
	return nil, &Error{Method: "GET", Path: "/tags", StatusCode: 404, Message: "Tag not found."}
}

// CreateTag creates a tag (name must be at least 2 characters).
func (c *Client) CreateTag(ctx context.Context, req TagRequest) (*Tag, error) {
	var created struct {
		UUID string `json:"uuid"`
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := c.post(ctx, "/tags", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /tags: API returned no uuid")
	}
	return c.GetTag(ctx, created.UUID)
}

// UpdateTag renames a tag.
func (c *Client) UpdateTag(ctx context.Context, uuid string, req TagRequest) (*Tag, error) {
	if err := c.patch(ctx, "/tags/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetTag(ctx, uuid)
}

// DeleteTag removes a tag from the team.
func (c *Client) DeleteTag(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/tags/"+url.PathEscape(uuid))
}
