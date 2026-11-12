package forgejo

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Issue struct {
	Index   int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

type CreateIssueOptions struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

type IssueComment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

type PullRequest struct {
	Index   int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
	Merged  bool   `json:"merged"`
}

type CreatePullRequestOptions struct {
	Head  string `json:"head"`
	Base  string `json:"base"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

func (c *Client) ListIssues(ctx context.Context, repository, state string, page, limit int) ([]Issue, error) {
	path, err := repositoryPath(repository, "/issues")
	if err != nil {
		return nil, err
	}
	query := url.Values{
		"state": {state},
		"page":  {strconv.Itoa(page)},
		"limit": {strconv.Itoa(limit)},
		"type":  {"issues"},
	}
	var issues []Issue
	err = c.Get(ctx, path, query, &issues)
	return issues, err
}

func (c *Client) GetIssue(ctx context.Context, repository string, index int) (Issue, error) {
	path, err := repositoryPath(repository, fmt.Sprintf("/issues/%d", index))
	if err != nil {
		return Issue{}, err
	}
	var issue Issue
	err = c.Get(ctx, path, nil, &issue)
	return issue, err
}

func (c *Client) CreateIssue(ctx context.Context, repository string, options CreateIssueOptions) (Issue, error) {
	path, err := repositoryPath(repository, "/issues")
	if err != nil {
		return Issue{}, err
	}
	var issue Issue
	err = c.Post(ctx, path, options, &issue)
	return issue, err
}

func (c *Client) SetIssueState(ctx context.Context, repository string, index int, state string) (Issue, error) {
	path, err := repositoryPath(repository, fmt.Sprintf("/issues/%d", index))
	if err != nil {
		return Issue{}, err
	}
	var issue Issue
	err = c.Patch(ctx, path, struct {
		State string `json:"state"`
	}{State: state}, &issue)
	return issue, err
}

func (c *Client) AddIssueComment(ctx context.Context, repository string, index int, body string) (IssueComment, error) {
	path, err := repositoryPath(repository, fmt.Sprintf("/issues/%d/comments", index))
	if err != nil {
		return IssueComment{}, err
	}
	var comment IssueComment
	err = c.Post(ctx, path, struct {
		Body string `json:"body"`
	}{Body: body}, &comment)
	return comment, err
}

func (c *Client) ListPullRequests(ctx context.Context, repository, state string, page, limit int) ([]PullRequest, error) {
	path, err := repositoryPath(repository, "/pulls")
	if err != nil {
		return nil, err
	}
	query := url.Values{
		"state": {state},
		"page":  {strconv.Itoa(page)},
		"limit": {strconv.Itoa(limit)},
	}
	var pulls []PullRequest
	err = c.Get(ctx, path, query, &pulls)
	return pulls, err
}

func (c *Client) GetPullRequest(ctx context.Context, repository string, index int) (PullRequest, error) {
	path, err := repositoryPath(repository, fmt.Sprintf("/pulls/%d", index))
	if err != nil {
		return PullRequest{}, err
	}
	var pull PullRequest
	err = c.Get(ctx, path, nil, &pull)
	return pull, err
}

func (c *Client) CreatePullRequest(ctx context.Context, repository string, options CreatePullRequestOptions) (PullRequest, error) {
	path, err := repositoryPath(repository, "/pulls")
	if err != nil {
		return PullRequest{}, err
	}
	var pull PullRequest
	err = c.Post(ctx, path, options, &pull)
	return pull, err
}

func repositoryPath(repository, suffix string) (string, error) {
	owner, name, ok := splitRepository(repository)
	if !ok {
		return "", fmt.Errorf("repository must be in OWNER/NAME format")
	}
	return "/repos/" + owner + "/" + name + suffix, nil
}

func splitRepository(repository string) (string, string, bool) {
	owner, name, ok := strings.Cut(repository, "/")
	return owner, name, ok && owner != "" && name != "" && !strings.Contains(name, "/")
}
