package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
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
	DockerfileTargetBuild   string `json:"dockerfile_target_build"`
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
	// LimitsMemorySwappiness and LimitsCPUShares are genuine integer columns
	// (confirmed against the migrations), unlike every other limits_* field —
	// see the identical note on client.Database.
	LimitsMemorySwap        string `json:"limits_memory_swap"`
	LimitsMemorySwappiness  int64  `json:"limits_memory_swappiness"`
	LimitsMemoryReservation string `json:"limits_memory_reservation"`
	LimitsCPUSet            string `json:"limits_cpuset"`
	LimitsCPUShares         int64  `json:"limits_cpu_shares"`
	// ManualWebhookSecret* are `$hidden` on the Application model — echoed
	// only to a token with the read:sensitive ability, same as
	// GithubApp.client_secret/webhook_secret.
	ManualWebhookSecretGithub    string `json:"manual_webhook_secret_github"`
	ManualWebhookSecretGitlab    string `json:"manual_webhook_secret_gitlab"`
	ManualWebhookSecretBitbucket string `json:"manual_webhook_secret_bitbucket"`
	ManualWebhookSecretGitea     string `json:"manual_webhook_secret_gitea"`

	// Health check. HealthCheckReturnCode/Interval/Timeout/Retries/StartPeriod
	// are genuine integer columns (confirmed against the migration), like
	// limits_memory_swappiness/limits_cpu_shares above; every other
	// health_check_* field is a string column.
	HealthCheckEnabled      bool   `json:"health_check_enabled"`
	HealthCheckPath         string `json:"health_check_path"`
	HealthCheckPort         string `json:"health_check_port"`
	HealthCheckHost         string `json:"health_check_host"`
	HealthCheckMethod       string `json:"health_check_method"`
	HealthCheckReturnCode   int64  `json:"health_check_return_code"`
	HealthCheckScheme       string `json:"health_check_scheme"`
	HealthCheckResponseText string `json:"health_check_response_text"`
	HealthCheckInterval     int64  `json:"health_check_interval"`
	HealthCheckTimeout      int64  `json:"health_check_timeout"`
	HealthCheckRetries      int64  `json:"health_check_retries"`
	HealthCheckStartPeriod  int64  `json:"health_check_start_period"`
	HealthCheckType         string `json:"health_check_type"`
	HealthCheckCommand      string `json:"health_check_command"`

	Status        string `json:"status"`
	EnvironmentID int64  `json:"environment_id"`
	DestinationID int64  `json:"destination_id"`
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
	Dockerfile         *string `json:"dockerfile,omitempty"` // base64-encoded, create-only (see CLAUDE.md)
	DockerfileLocation *string `json:"dockerfile_location,omitempty"`
	// DockerfileTargetBuild is the inverse of Dockerfile: accepted by
	// update_by_uuid but absent from every create endpoint's allowed-fields
	// list (confirmed against the controller source) — set it via a
	// follow-up PATCH after create, never in the create body itself.
	DockerfileTargetBuild   *string `json:"dockerfile_target_build,omitempty"`
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

	// Manual webhook secrets (for Coolify's own /webhooks/source/*/events/manual
	// endpoints, independent of any git-App-managed webhook). Accepted by both
	// create and update — no whitelist mismatch, unlike dockerfile*. Coolify
	// generates a random 40-char value itself when left unset (Application's
	// `creating` model event, `??=`), so these are Optional+Computed, not
	// Required.
	ManualWebhookSecretGithub    *string `json:"manual_webhook_secret_github,omitempty"`
	ManualWebhookSecretGitlab    *string `json:"manual_webhook_secret_gitlab,omitempty"`
	ManualWebhookSecretBitbucket *string `json:"manual_webhook_secret_bitbucket,omitempty"`
	ManualWebhookSecretGitea     *string `json:"manual_webhook_secret_gitea,omitempty"`

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
	HealthCheckEnabled      *bool   `json:"health_check_enabled,omitempty"`
	HealthCheckPath         *string `json:"health_check_path,omitempty"`
	HealthCheckPort         *string `json:"health_check_port,omitempty"`
	HealthCheckHost         *string `json:"health_check_host,omitempty"`
	HealthCheckMethod       *string `json:"health_check_method,omitempty"`
	HealthCheckReturnCode   *int64  `json:"health_check_return_code,omitempty"`
	HealthCheckScheme       *string `json:"health_check_scheme,omitempty"`
	HealthCheckResponseText *string `json:"health_check_response_text,omitempty"`
	HealthCheckInterval     *int64  `json:"health_check_interval,omitempty"`
	HealthCheckTimeout      *int64  `json:"health_check_timeout,omitempty"`
	HealthCheckRetries      *int64  `json:"health_check_retries,omitempty"`
	HealthCheckStartPeriod  *int64  `json:"health_check_start_period,omitempty"`
	HealthCheckType         *string `json:"health_check_type,omitempty"`
	HealthCheckCommand      *string `json:"health_check_command,omitempty"`

	// Limits.
	LimitsMemory            *string `json:"limits_memory,omitempty"`
	LimitsCPUs              *string `json:"limits_cpus,omitempty"`
	LimitsMemorySwap        *string `json:"limits_memory_swap,omitempty"`
	LimitsMemorySwappiness  *int64  `json:"limits_memory_swappiness,omitempty"`
	LimitsMemoryReservation *string `json:"limits_memory_reservation,omitempty"`
	LimitsCPUSet            *string `json:"limits_cpuset,omitempty"`
	LimitsCPUShares         *int64  `json:"limits_cpu_shares,omitempty"`
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

// DeleteApplication removes an application and waits until Coolify has
// actually finished tearing it down (deletion is asynchronous server-side).
// Nil flags keep the API defaults (delete configurations, volumes, networks,
// and run docker cleanup).
func (c *Client) DeleteApplication(ctx context.Context, uuid string, deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks *bool) error {
	if err := c.deleteWithQuery(ctx, "/applications/"+url.PathEscape(uuid),
		deletionQuery(deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks)); err != nil {
		return err
	}
	return c.waitForDeletion(ctx, 2*time.Minute, func(ctx context.Context) error {
		_, err := c.GetApplication(ctx, uuid)
		return err
	})
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
