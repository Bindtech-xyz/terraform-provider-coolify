package client

import (
	"context"
	"fmt"
	"net/url"
)

// S3Storage is an S3-compatible backup destination.
type S3Storage struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	Key         string `json:"key"`
	Secret      string `json:"secret"`
	IsUsable    bool   `json:"is_usable"`
}

// S3StorageRequest is the body for POST /s3-storages and PATCH /s3-storages/{uuid}.
// endpoint, bucket, region, key and secret are required on create.
type S3StorageRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Endpoint    *string `json:"endpoint,omitempty"`
	Bucket      *string `json:"bucket,omitempty"`
	Region      *string `json:"region,omitempty"`
	Key         *string `json:"key,omitempty"`
	Secret      *string `json:"secret,omitempty"`
}

// ListS3Storages returns every S3 storage of the token's team.
func (c *Client) ListS3Storages(ctx context.Context) ([]S3Storage, error) {
	var out []S3Storage
	if err := c.get(ctx, "/s3-storages", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetS3Storage fetches one S3 storage by UUID.
func (c *Client) GetS3Storage(ctx context.Context, uuid string) (*S3Storage, error) {
	var out S3Storage
	if err := c.get(ctx, "/s3-storages/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateS3Storage registers an S3-compatible storage and returns it refreshed.
func (c *Client) CreateS3Storage(ctx context.Context, req S3StorageRequest) (*S3Storage, error) {
	var created uuidResponse
	if err := c.post(ctx, "/s3-storages", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /s3-storages: API returned no uuid")
	}
	return c.GetS3Storage(ctx, created.UUID)
}

// UpdateS3Storage applies a partial update and returns the refreshed object.
func (c *Client) UpdateS3Storage(ctx context.Context, uuid string, req S3StorageRequest) (*S3Storage, error) {
	if err := c.patch(ctx, "/s3-storages/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetS3Storage(ctx, uuid)
}

// DeleteS3Storage removes the storage definition (not the bucket contents).
func (c *Client) DeleteS3Storage(ctx context.Context, uuid string) error {
	return c.delete(ctx, "/s3-storages/"+url.PathEscape(uuid))
}

// ValidateS3Storage asks Coolify to test connectivity to the bucket.
func (c *Client) ValidateS3Storage(ctx context.Context, uuid string) error {
	return c.post(ctx, "/s3-storages/"+url.PathEscape(uuid)+"/validate", nil, nil)
}
