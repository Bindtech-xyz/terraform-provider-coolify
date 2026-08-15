package client

import (
	"context"
	"strconv"
)

// Team mirrors the `Team` schema (read-only through the API).
type Team struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TeamMember is a user belonging to a team.
type TeamMember struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ListTeams returns every team the token's user belongs to.
func (c *Client) ListTeams(ctx context.Context) ([]Team, error) {
	var out []Team
	if err := c.get(ctx, "/teams", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTeam fetches a team by numeric id.
func (c *Client) GetTeam(ctx context.Context, id int64) (*Team, error) {
	var out Team
	if err := c.get(ctx, "/teams/"+strconv.FormatInt(id, 10), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CurrentTeam returns the team the API token belongs to.
func (c *Client) CurrentTeam(ctx context.Context) (*Team, error) {
	var out Team
	if err := c.get(ctx, "/team", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CurrentTeamMembers returns the members of the token's team.
func (c *Client) CurrentTeamMembers(ctx context.Context) ([]TeamMember, error) {
	var out []TeamMember
	if err := c.get(ctx, "/team/members", &out); err != nil {
		return nil, err
	}
	return out, nil
}
