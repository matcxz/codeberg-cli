package forgejo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type User struct {
	Username string `json:"login"`
}

type Repository struct {
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Stars         int    `json:"stars_count"`
	UpdatedAt     string `json:"updated_at"`
}

type CreateRepositoryOptions struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private"`
	AutoInit    bool   `json:"auto_init"`
}

func (c *Client) CurrentUser(ctx context.Context) (User, error) {
	var user User
	err := c.Get(ctx, "/user", nil, &user)
	return user, err
}

func (c *Client) ListRepositories(ctx context.Context, owner string, page, limit int) ([]Repository, error) {
	path := "/user/repos"
	if owner != "" {
		path = "/users/" + owner + "/repos"
	}
	query := url.Values{
		"page":  {fmt.Sprint(page)},
		"limit": {fmt.Sprint(limit)},
	}
	var repositories []Repository
	err := c.Get(ctx, path, query, &repositories)
	return repositories, err
}

func (c *Client) GetRepository(ctx context.Context, fullName string) (Repository, error) {
	owner, name, ok := strings.Cut(fullName, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return Repository{}, fmt.Errorf("repository must be in OWNER/NAME format")
	}
	path := "/repos/" + owner + "/" + name
	var repository Repository
	err := c.Get(ctx, path, nil, &repository)
	return repository, err
}

func (c *Client) CreateRepository(ctx context.Context, options CreateRepositoryOptions) (Repository, error) {
	var repository Repository
	err := c.Post(ctx, "/user/repos", options, &repository)
	return repository, err
}
