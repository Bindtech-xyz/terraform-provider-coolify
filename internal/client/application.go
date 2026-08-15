package client

import (
	"context"
	"fmt"
	"net/url"
)

// ApplicationType selects the Coolify create endpoint. The API models each
// source kind as a distinct POST endpoint sharing one attribute set.
type ApplicationType string

const (
	ApplicationTypePublic           ApplicationType = "public"
	ApplicationTypePrivateGithubApp ApplicationType = "private-github-app"
	ApplicationTypePrivateDeployKey ApplicationType = "private-deploy-key"
	ApplicationTypeDockerfile       ApplicationType = "dockerfile"
	ApplicationTypeDockerImage      ApplicationType = "dockerimage"
)

// Application mirrors the `Application` schema (subset the provider surfaces).
type Application struct {
	ID                      int64  `json:"id"`
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	Description             string `json:"description"`
	FQDN                    string `json:"fqdn"`
	GitRepository           string `json:"git_repository"`
	GitBranch               string `json:"git_branch"`
	GitCommitSHA            string `json:"git_commit_sha"`
	BuildPack               string `json:"build_pack"`
	StaticImage             string `json:"static_image"`
	InstallCommand          string `json:"install_command"`
	BuildCommand            string `json:"build_command"`
	StartCommand            string `json:"start_command"`
	BaseDirectory           string `json:"base_directory"`
	PublishDirectory        string `json:"publish_directory"`
	PortsExposes            string `json:"ports_exposes"`
	PortsMappings           string `json:"ports_mappings"`
	Dockerfile              string `json:"dockerfile"`
	DockerfileLocation      string `json:"dockerfile_location"`
	DockerRegistryImageName string `json:"docker_registry_image_name"`
	DockerRegistryImageTag  string `json:"docker_registry_image_tag"`
	DockerComposeLocation   string `json:"docker_compose_location"`
	DockerComposeRaw        string `json:"docker_compose_raw"`
	CustomLabels            string `json:"custom_labels"`
	CustomDockerRunOptions  string `json:"custom_docker_run_options"`
	PreDeploymentCommand    string `json:"pre_deployment_command"`
	PostDeploymentCommand   string `json:"post_deployment_command"`
	WatchPaths              string `json:"watch_paths"`
	RedirectBehaviour       string `json:"redirect"`
	LimitsMemory            string `json:"limits_memory"`
	LimitsCPUs              string `json:"limits_cpus"`
	Status                  string `json:"status"`
	EnvironmentID           int64  `json:"environment_id"`
	DestinationID           int64  `json:"destination_id"`
}

// ApplicationRequest is the union body for the five create endpoints and for
// PATCH /applications/{uuid}. Coolify rejects fields that are not allowed for
// the chosen endpoint, so callers only set what applies.
type ApplicationRequest struct {
	// Placement (create only).
	ProjectUUID     *string `json:"project_uuid,omitempty"`
	EnvironmentName *string `json:"environment_name,omitempty"`
	EnvironmentUUID *string `json:"environment_uuid,omitempty"`
	ServerUUID      *string `json:"server_uuid,omitempty"`
	DestinationUUID *string `json:"destination_uuid,omitempty"`

	// Identity.
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`

	// Git sources.
	GitRepository  *string `json:"git_repository,omitempty"`
	GitBranch      *string `json:"git_branch,omitempty"`
	GitCommitSHA   *string `json:"git_commit_sha,omitempty"`
	PrivateKeyUUID *string `json:"private_key_uuid,omitempty"`
	GithubAppUUID  *string `json:"github_app_uuid,omitempty"`

	// Docker sources.
	Dockerfile              *string `json:"dockerfile,omitempty"` // base64-encoded
	DockerRegistryImageName *string `json:"docker_registry_image_name,omitempty"`
	DockerRegistryImageTag  *string `json:"docker_registry_image_tag,omitempty"`

	// Build & runtime.
	BuildPack              *string `json:"build_pack,omitempty"` // nixpacks|static|dockerfile|dockercompose|railpack
	StaticImage            *string `json:"static_image,omitempty"`
	InstallCommand         *string `json:"install_command,omitempty"`
	BuildCommand           *string `json:"build_command,omitempty"`
	StartCommand           *string `json:"start_command,omitempty"`
	BaseDirectory          *string `json:"base_directory,omitempty"`
	PublishDirectory       *string `json:"publish_directory,omitempty"`
	PortsExposes           *string `json:"ports_exposes,omitempty"`
	PortsMappings          *string `json:"ports_mappings,omitempty"`
	CustomLabels           *string `json:"custom_labels,omitempty"`
	CustomDockerRunOptions *string `json:"custom_docker_run_options,omitempty"`
	PreDeploymentCommand   *string `json:"pre_deployment_command,omitempty"`
	PostDeploymentCommand  *string `json:"post_deployment_command,omitempty"`
	WatchPaths             *string `json:"watch_paths,omitempty"`

	// Domains.
	Domains            *string `json:"domains,omitempty"`
	Redirect           *string `json:"redirect,omitempty"` // www|non-www|both
	AutogenerateDomain *bool   `json:"autogenerate_domain,omitempty"`

	// Behaviour toggles.
	IsStatic                    *bool `json:"is_static,omitempty"`
	IsSPA                       *bool `json:"is_spa,omitempty"`
	IsAutoDeployEnabled         *bool `json:"is_auto_deploy_enabled,omitempty"`
	IsForceHTTPSEnabled         *bool `json:"is_force_https_enabled,omitempty"`
	IsPreviewDeploymentsEnabled *bool `json:"is_preview_deployments_enabled,omitempty"`
	ConnectToDockerNetwork      *bool `json:"connect_to_docker_network,omitempty"`
	UseBuildServer              *bool `json:"use_build_server,omitempty"`
	InstantDeploy               *bool `json:"instant_deploy,omitempty"`

	// Health check.
	HealthCheckEnabled *bool   `json:"health_check_enabled,omitempty"`
	HealthCheckPath    *string `json:"health_check_path,omitempty"`
	HealthCheckPort    *string `json:"health_check_port,omitempty"`

	// Limits.
	LimitsMemory *string `json:"limits_memory,omitempty"`
	LimitsCPUs   *string `json:"limits_cpus,omitempty"`
}

// applicationCreatePaths maps a type to its POST endpoint.
var applicationCreatePaths = map[ApplicationType]string{
	ApplicationTypePublic:           "/applications/public",
	ApplicationTypePrivateGithubApp: "/applications/private-github-app",
	ApplicationTypePrivateDeployKey: "/applications/private-deploy-key",
	ApplicationTypeDockerfile:       "/applications/dockerfile",
	ApplicationTypeDockerImage:      "/applications/dockerimage",
}

// ListApplications returns every application of the token's team.
func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	var out []Application
	if err := c.get(ctx, "/applications", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetApplication fetches one application by UUID.
func (c *Client) GetApplication(ctx context.Context, uuid string) (*Application, error) {
	var out Application
	if err := c.get(ctx, "/applications/"+url.PathEscape(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateApplication creates an application of the given type. The API responds
// with {uuid, domains}; the full object is fetched before returning.
func (c *Client) CreateApplication(ctx context.Context, appType ApplicationType, req ApplicationRequest) (*Application, error) {
	path, ok := applicationCreatePaths[appType]
	if !ok {
		return nil, fmt.Errorf("unknown application type %q", appType)
	}
	var created uuidResponse
	if err := c.post(ctx, path, req, &created); err != nil {
		return nil, err
	}
	if created.UUID == "" {
		return nil, fmt.Errorf("POST %s: API returned no uuid", path)
	}
	return c.GetApplication(ctx, created.UUID)
}

// UpdateApplication applies a partial update and returns the refreshed object.
// Placement fields (project/server/environment) are not updatable here — use
// the move/migrate endpoints for that.
func (c *Client) UpdateApplication(ctx context.Context, uuid string, req ApplicationRequest) (*Application, error) {
	if err := c.patch(ctx, "/applications/"+url.PathEscape(uuid), req, nil); err != nil {
		return nil, err
	}
	return c.GetApplication(ctx, uuid)
}

// DeleteApplication removes an application. Nil flags keep the API defaults
// (delete configurations, volumes, networks, and run docker cleanup).
func (c *Client) DeleteApplication(ctx context.Context, uuid string, deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks *bool) error {
	return c.deleteWithQuery(ctx, "/applications/"+url.PathEscape(uuid),
		deletionQuery(deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks))
}

// StartApplication queues a deployment (equivalent to pressing Deploy).
func (c *Client) StartApplication(ctx context.Context, uuid string) error {
	return c.post(ctx, "/applications/"+url.PathEscape(uuid)+"/start", nil, nil)
}

// StopApplication stops the application's containers.
func (c *Client) StopApplication(ctx context.Context, uuid string) error {
	return c.post(ctx, "/applications/"+url.PathEscape(uuid)+"/stop", nil, nil)
}

// RestartApplication restarts the application.
func (c *Client) RestartApplication(ctx context.Context, uuid string) error {
	return c.post(ctx, "/applications/"+url.PathEscape(uuid)+"/restart", nil, nil)
}
