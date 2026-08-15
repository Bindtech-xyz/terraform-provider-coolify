package client

import (
	"context"
	"net/url"
	"strconv"
)

// ServerDomain groups the domains served from one IP on a server.
type ServerDomain struct {
	IP      string   `json:"ip"`
	Domains []string `json:"domains"`
}

// ServerResource is a workload (application, database or service) defined on a
// server.
type ServerResource struct {
	ID     int64  `json:"id"`
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// GetServerDomains returns the domains grouped by IP for a server.
func (c *Client) GetServerDomains(ctx context.Context, serverUUID string) ([]ServerDomain, error) {
	var out []ServerDomain
	if err := c.get(ctx, "/servers/"+url.PathEscape(serverUUID)+"/domains", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetServerResources returns every workload defined on a server.
func (c *Client) GetServerResources(ctx context.Context, serverUUID string) ([]ServerResource, error) {
	var out []ServerResource
	if err := c.get(ctx, "/servers/"+url.PathEscape(serverUUID)+"/resources", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// BackupExecution is one run of a scheduled database backup.
type BackupExecution struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Size      int64  `json:"size"`
	Filename  string `json:"filename"`
	CreatedAt string `json:"created_at"`
}

// ListBackupExecutions returns the runs of one backup configuration.
func (c *Client) ListBackupExecutions(ctx context.Context, databaseUUID, backupUUID string) ([]BackupExecution, error) {
	var out struct {
		Executions []BackupExecution `json:"executions"`
	}
	path := "/databases/" + url.PathEscape(databaseUUID) + "/backups/" + url.PathEscape(backupUUID) + "/executions"
	if err := c.get(ctx, path, &out); err != nil {
		var bare []BackupExecution
		if err2 := c.get(ctx, path, &bare); err2 != nil {
			return nil, err
		}
		return bare, nil
	}
	return out.Executions, nil
}

// GithubRepository is a repository accessible through a GitHub App.
type GithubRepository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}

// ListGithubAppRepositories returns the repositories a GitHub App can access.
func (c *Client) ListGithubAppRepositories(ctx context.Context, githubAppID int64) ([]GithubRepository, error) {
	var out struct {
		Repositories []GithubRepository `json:"repositories"`
	}
	path := "/github-apps/" + strconv.FormatInt(githubAppID, 10) + "/repositories"
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Repositories, nil
}

// GithubBranch is a branch of a repository accessible through a GitHub App.
type GithubBranch struct {
	Name string `json:"name"`
}

// ListGithubAppBranches returns the branches of one repository.
func (c *Client) ListGithubAppBranches(ctx context.Context, githubAppID int64, owner, repo string) ([]GithubBranch, error) {
	var out []GithubBranch
	path := "/github-apps/" + strconv.FormatInt(githubAppID, 10) + "/repositories/" +
		url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/branches"
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Healthy reports whether GET /health answers OK.
func (c *Client) Healthy(ctx context.Context) bool {
	// /health returns a bare "OK" string, not JSON; a nil out ignores the body.
	return c.get(ctx, "/health", nil) == nil
}
