package client

import (
	"context"
	"fmt"
	"net/url"
)

// CloudServerRequest is the union body for the three VM-provisioning endpoints
// (POST /servers/hetzner, /servers/digitalocean, /servers/vultr). Coolify
// creates the VM at the provider, registers it as a server and returns its
// UUID. Provider-specific fields are only valid for their provider.
type CloudServerRequest struct {
	CloudProviderTokenUUID *string `json:"cloud_provider_token_uuid,omitempty"`
	Name                   *string `json:"name,omitempty"`
	PrivateKeyUUID         *string `json:"private_key_uuid,omitempty"`
	CloudInitScript        *string `json:"cloud_init_script,omitempty"`
	InstantValidate        *bool   `json:"instant_validate,omitempty"`

	// Hetzner.
	Location          *string `json:"location,omitempty"`
	ServerType        *string `json:"server_type,omitempty"`
	HetznerImage      *int64  `json:"image,omitempty"` // Hetzner wants an integer image id
	EnableIPv4        *bool   `json:"enable_ipv4,omitempty"`
	EnableIPv6        *bool   `json:"enable_ipv6,omitempty"`
	EnableBackups     *bool   `json:"enable_backups,omitempty"`
	HetznerSSHKeyIDs  []int64 `json:"hetzner_ssh_key_ids,omitempty"`
	HetznerFirewallID []int64 `json:"hetzner_firewall_ids,omitempty"`
	HetznerNetworkIDs []int64 `json:"hetzner_network_ids,omitempty"`

	// DigitalOcean.
	Region                *string `json:"region,omitempty"`
	Size                  *string `json:"size,omitempty"`
	DigitalOceanImage     *string `json:"-"` // marshalled through HetznerImage slot conflict; see build()
	Monitoring            *bool   `json:"monitoring,omitempty"`
	DigitalOceanSSHKeyIDs []int64 `json:"digitalocean_ssh_key_ids,omitempty"`

	// Vultr.
	Plan *string `json:"plan,omitempty"`
	OSID *int64  `json:"os_id,omitempty"`
}

// cloudServerBody renders the request for a given provider, resolving the
// image field type difference (Hetzner: integer id, DigitalOcean: string slug).
func cloudServerBody(provider string, req CloudServerRequest) (map[string]any, error) {
	body := map[string]any{}
	set := func(key string, v any) { body[key] = v }

	if req.CloudProviderTokenUUID != nil {
		set("cloud_provider_token_uuid", *req.CloudProviderTokenUUID)
	}
	if req.Name != nil {
		set("name", *req.Name)
	}
	if req.PrivateKeyUUID != nil {
		set("private_key_uuid", *req.PrivateKeyUUID)
	}
	if req.CloudInitScript != nil {
		set("cloud_init_script", *req.CloudInitScript)
	}
	if req.InstantValidate != nil {
		set("instant_validate", *req.InstantValidate)
	}

	switch provider {
	case "hetzner":
		if req.Location != nil {
			set("location", *req.Location)
		}
		if req.ServerType != nil {
			set("server_type", *req.ServerType)
		}
		if req.HetznerImage != nil {
			set("image", *req.HetznerImage)
		}
		if req.EnableIPv4 != nil {
			set("enable_ipv4", *req.EnableIPv4)
		}
		if req.EnableIPv6 != nil {
			set("enable_ipv6", *req.EnableIPv6)
		}
		if req.EnableBackups != nil {
			set("enable_backups", *req.EnableBackups)
		}
		if len(req.HetznerSSHKeyIDs) > 0 {
			set("hetzner_ssh_key_ids", req.HetznerSSHKeyIDs)
		}
		if len(req.HetznerFirewallID) > 0 {
			set("hetzner_firewall_ids", req.HetznerFirewallID)
		}
		if len(req.HetznerNetworkIDs) > 0 {
			set("hetzner_network_ids", req.HetznerNetworkIDs)
		}
	case "digitalocean":
		if req.Region != nil {
			set("region", *req.Region)
		}
		if req.Size != nil {
			set("size", *req.Size)
		}
		if req.DigitalOceanImage != nil {
			set("image", *req.DigitalOceanImage)
		}
		if req.EnableIPv6 != nil {
			set("enable_ipv6", *req.EnableIPv6)
		}
		if req.Monitoring != nil {
			set("monitoring", *req.Monitoring)
		}
		if len(req.DigitalOceanSSHKeyIDs) > 0 {
			set("digitalocean_ssh_key_ids", req.DigitalOceanSSHKeyIDs)
		}
	case "vultr":
		if req.Region != nil {
			set("region", *req.Region)
		}
		if req.Plan != nil {
			set("plan", *req.Plan)
		}
		if req.OSID != nil {
			set("os_id", *req.OSID)
		}
	default:
		return nil, fmt.Errorf("unknown cloud provider %q", provider)
	}
	return body, nil
}

// CreateCloudServer provisions a VM at the provider and registers it as a
// Coolify server; the full server object is fetched before returning.
func (c *Client) CreateCloudServer(ctx context.Context, provider string, req CloudServerRequest) (*Server, error) {
	body, err := cloudServerBody(provider, req)
	if err != nil {
		return nil, err
	}
	var created uuidResponse
	path := "/servers/" + provider
	if err := c.post(ctx, path, body, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST %s: API returned no uuid", path)
	}
	return c.GetServer(ctx, created.UUID)
}

// CloudCatalog fetches a provider catalog endpoint (regions, sizes, images,
// ssh-keys, …) as raw JSON objects. The catalogs proxy the provider APIs, so
// their shapes vary; the data-source layer picks the fields it exposes.
func (c *Client) CloudCatalog(ctx context.Context, provider, section, tokenUUID string) ([]map[string]any, error) {
	q := url.Values{}
	if tokenUUID != "" {
		q.Set("cloud_provider_token_uuid", tokenUUID)
	}
	path := "/" + provider + "/" + section
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out []map[string]any
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
