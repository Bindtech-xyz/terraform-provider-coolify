package client

import (
	"context"
	"fmt"
	"net/url"
)

// Service mirrors the `Service` schema (one-click services and raw
// docker-compose stacks).
type Service struct {
	ID               int64  `json:"id"`
	UUID             string `json:"uuid"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Type             string `json:"service_type"`
	DockerComposeRaw string `json:"docker_compose_raw"`
	DockerCompose    string `json:"docker_compose"`
	Status           string `json:"status"`
	EnvironmentID    int64  `json:"environment_id"`
	ServerID         int64  `json:"server_id"`
	DestinationID    int64  `json:"destination_id"`
}

// ServiceURL maps a compose service name to its URL(s), used at create time.
type ServiceURL struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// ServiceCreateRequest is the body for POST /services. Either Type (a one-click
// service like "plausible") or DockerComposeRaw (base64 compose file) is
// required.
type ServiceCreateRequest struct {
	Type             *string      `json:"type,omitempty"`
	DockerComposeRaw *string      `json:"docker_compose_raw,omitempty"` // base64
	Name             *string      `json:"name,omitempty"`
	Description      *string      `json:"description,omitempty"`
	ProjectUUID      *string      `json:"project_uuid,omitempty"`
	EnvironmentName  *string      `json:"environment_name,omitempty"`
	EnvironmentUUID  *string      `json:"environment_uuid,omitempty"`
	ServerUUID       *string      `json:"server_uuid,omitempty"`
	DestinationUUID  *string      `json:"destination_uuid,omitempty"`
	InstantDeploy    *bool        `json:"instant_deploy,omitempty"`
	URLs             []ServiceURL `json:"urls,omitempty"`
}

// ServiceUpdateRequest is the body for PATCH /services/{uuid}.
type ServiceUpdateRequest struct {
	Name                   *string      `json:"name,omitempty"`
	Description            *string      `json:"description,omitempty"`
	DockerComposeRaw       *string      `json:"docker_compose_raw,omitempty"` // base64
	InstantDeploy          *bool        `json:"instant_deploy,omitempty"`
	ConnectToDockerNetwork *bool        `json:"connect_to_docker_network,omitempty"`
	URLs                   []ServiceURL `json:"urls,omitempty"`
}

// ListServices returns every service of the token's team.
func (c *Client) ListServices(ctx context.Context) ([]Service, error) {
	var out []Service
	if err := c.get(ctx, "/services", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetService fetches one service by UUID.
func (c *Client) GetService(ctx context.Context, uuid string) (*Service, error) {
	var out Service
	if err := c.get(ctx, "/services/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateService creates a one-click or compose service. The API responds with
// {uuid, domains}; the full object is fetched before returning.
func (c *Client) CreateService(ctx context.Context, req ServiceCreateRequest) (*Service, error) {
	var created uuidResponse
	if err := c.post(ctx, "/services", req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST /services: API returned no uuid")
	}
	return c.GetService(ctx, created.UUID)
}

// UpdateService applies a partial update and returns the refreshed object.
func (c *Client) UpdateService(ctx context.Context, uuid string, req ServiceUpdateRequest) (*Service, error) {
	if err := c.patch(ctx, "/services/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetService(ctx, uuid)
}

// DeleteService removes a service. Nil flags keep the API defaults (all true).
func (c *Client) DeleteService(ctx context.Context, uuid string, deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks *bool) error {
	return c.deleteWithQuery(ctx, "/services/"+url.PathEscape(uuid),
		deletionQuery(deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks))
}

// StartService starts the service stack.
func (c *Client) StartService(ctx context.Context, uuid string) error {
	return c.post(ctx, "/services/"+url.PathEscape(uuid)+"/start", nil, nil)
}

// StopService stops the service stack.
func (c *Client) StopService(ctx context.Context, uuid string) error {
	return c.post(ctx, "/services/"+url.PathEscape(uuid)+"/stop", nil, nil)
}

// RestartService restarts the service stack.
func (c *Client) RestartService(ctx context.Context, uuid string) error {
	return c.post(ctx, "/services/"+url.PathEscape(uuid)+"/restart", nil, nil)
}
