package client

import (
	"context"
	"fmt"
	"net/url"
)

// Storage is a persistent volume or file mount attached to an application,
// service or database (docs: knowledge-base/persistent-storage).
//
// Coolify models "persistent" and "file" as two separate Eloquent models
// (LocalPersistentVolume, LocalFileVolume) with no shared discriminator
// column — there is no "type" field in the response to unmarshal, unlike the
// request body, which does take one to select which endpoint behavior to
// apply. Name is only a real column on LocalPersistentVolume (absent, so
// always "", for file mounts). Content is LocalFileVolume's declared
// `$hidden` and is therefore never present in any response, regardless of
// the token's abilities. Provider code (see storageToModel) reads only UUID
// and Name here — the rest is kept for completeness/future use, not because
// it is currently trustworthy to adopt into state.
type Storage struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Name      string `json:"name"` // persistent volumes only; "" for file mounts
	MountPath string `json:"mount_path"`
	HostPath  string `json:"host_path"`
	Content   string `json:"content"` // always "": see doc comment above
	FsPath    string `json:"fs_path"`
}

// StorageRequest is the body for creating and updating storages. Type is
// required; persistent storages use Name/MountPath/HostPath, file mounts use
// MountPath/Content.
type StorageRequest struct {
	// UUID identifies the storage on update (the PATCH endpoint has no
	// storage segment in its path).
	UUID      *string `json:"uuid,omitempty"`
	Type      *string `json:"type,omitempty"`
	Name      *string `json:"name,omitempty"`
	MountPath *string `json:"mount_path,omitempty"`
	HostPath  *string `json:"host_path,omitempty"`
	Content   *string `json:"content,omitempty"`
}

// storageBase reuses the env-var parent mapping: applications, services and
// databases all expose /{parent}/{uuid}/storages.
func storageBase(parent EnvVarParent, parentUUID string) (string, error) {
	base, ok := envVarParentPaths[parent]
	if !ok {
		return "", fmt.Errorf("unknown storage parent %q", parent)
	}
	return base + url.PathEscape(parentUUID) + "/storages", nil
}

// ListStorages returns the storages of a resource.
// ListStorages returns the storages of a resource. The endpoint does not
// return a flat array: persistent volumes and file mounts are Coolify's two
// separate underlying models (see the Storage doc comment), so the response
// is {"persistent_storages": [...], "file_storages": [...]} — this
// concatenates both into one list.
func (c *Client) ListStorages(ctx context.Context, parent EnvVarParent, parentUUID string) ([]Storage, error) {
	base, err := storageBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	var wrapped struct {
		Persistent []Storage `json:"persistent_storages"`
		File       []Storage `json:"file_storages"`
	}
	if err := c.get(ctx, base, &wrapped); err != nil {
		return nil, err
	}
	return append(wrapped.Persistent, wrapped.File...), nil
}

// GetStorage returns one storage by UUID, via the list endpoint.
func (c *Client) GetStorage(ctx context.Context, parent EnvVarParent, parentUUID, uuid string) (*Storage, error) {
	storages, err := c.ListStorages(ctx, parent, parentUUID)
	if err != nil {
		return nil, err
	}
	for _, s := range storages {
		if s.UUID == uuid {
			return &s, nil
		}
	}
	return nil, &Error{Method: "GET", Path: string(parent) + " storages", StatusCode: 404, Message: "Storage not found."}
}

// CreateStorage attaches a storage; the API returns the full object.
func (c *Client) CreateStorage(ctx context.Context, parent EnvVarParent, parentUUID string, req StorageRequest) (*Storage, error) {
	base, err := storageBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	var out Storage
	if err := c.post(ctx, base, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateStorage updates a storage (identified by req.UUID, which this method
// fills in) and returns it refreshed.
func (c *Client) UpdateStorage(ctx context.Context, parent EnvVarParent, parentUUID, uuid string, req StorageRequest) (*Storage, error) {
	base, err := storageBase(parent, parentUUID)
	if err != nil {
		return nil, err
	}
	req.UUID = &uuid
	if err := c.patch(ctx, base, req, nil); err != nil {
		return nil, err
	}
	return c.GetStorage(ctx, parent, parentUUID, uuid)
}

// DeleteStorage detaches a storage by UUID.
func (c *Client) DeleteStorage(ctx context.Context, parent EnvVarParent, parentUUID, uuid string) error {
	base, err := storageBase(parent, parentUUID)
	if err != nil {
		return err
	}
	return c.delete(ctx, base+"/"+url.PathEscape(uuid))
}
