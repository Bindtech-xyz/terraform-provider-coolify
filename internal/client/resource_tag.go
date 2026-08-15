package client

import (
	"context"
	"fmt"
	"net/url"
)

// TaggableResourceType is the kind of resource a tag can be attached to.
type TaggableResourceType string

const (
	TaggableApplication TaggableResourceType = "application"
	TaggableDatabase    TaggableResourceType = "database"
	TaggableService     TaggableResourceType = "service"
)

func taggableBase(resourceType TaggableResourceType, resourceUUID string) (string, error) {
	var segment string
	switch resourceType {
	case TaggableApplication:
		segment = "applications"
	case TaggableDatabase:
		segment = "databases"
	case TaggableService:
		segment = "services"
	default:
		return "", fmt.Errorf("unknown taggable resource type %q", resourceType)
	}
	return "/" + segment + "/" + url.PathEscape(resourceUUID) + "/tags", nil
}

// ListResourceTags returns every tag currently attached to a resource.
func (c *Client) ListResourceTags(ctx context.Context, resourceType TaggableResourceType, resourceUUID string) ([]Tag, error) {
	base, err := taggableBase(resourceType, resourceUUID)
	if err != nil {
		return nil, err
	}
	var out []Tag
	if err := c.get(ctx, base, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AttachResourceTag attaches a tag to a resource by name — Coolify creates
// the team-wide tag if it doesn't already exist (create-or-attach in one
// call) and the attach is idempotent (re-attaching an already-attached tag
// is not an error). Returns the resource's full tag list after attaching,
// from which the caller can find the tag's uuid by name.
func (c *Client) AttachResourceTag(ctx context.Context, resourceType TaggableResourceType, resourceUUID, tagName string) ([]Tag, error) {
	base, err := taggableBase(resourceType, resourceUUID)
	if err != nil {
		return nil, err
	}
	body := struct {
		TagName string `json:"tag_name"`
	}{TagName: tagName}
	var out []Tag
	if err := c.post(ctx, base, body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DetachResourceTag detaches a tag from a resource by the tag's uuid.
// Coolify deletes the team-wide tag itself if this was its last attachment
// (deleteIfOrphaned), which is not surfaced here — nothing to act on.
func (c *Client) DetachResourceTag(ctx context.Context, resourceType TaggableResourceType, resourceUUID, tagUUID string) error {
	base, err := taggableBase(resourceType, resourceUUID)
	if err != nil {
		return err
	}
	return c.delete(ctx, base+"/"+url.PathEscape(tagUUID))
}
