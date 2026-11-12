package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/acme/widget/issues":
			if r.URL.Query().Get("state") != "open" ||
				r.URL.Query().Get("type") != "issues" ||
				r.URL.Query().Get("page") != "1" ||
				r.URL.Query().Get("limit") != "5" {
				t.Errorf("unexpected issue query: %v", r.URL.Query())
			}
			fmt.Fprint(w, `[{"number":12,"title":"Bug","state":"open","html_url":"https://codeberg.org/acme/widget/issues/12"}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/acme/widget/issues/12":
			fmt.Fprint(w, `{"number":12,"title":"Bug","body":"Details","state":"open"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/acme/widget/issues":
			var body CreateIssueOptions
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create issue body: %v", err)
			}
			if body != (CreateIssueOptions{Title: "Bug", Body: "Details"}) {
				t.Errorf("create issue body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"number":13,"title":"Bug","state":"open"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	issues, err := client.ListIssues(context.Background(), "acme/widget", "open", 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Index != 12 {
		t.Fatalf("issues = %#v", issues)
	}
	issue, err := client.GetIssue(context.Background(), "acme/widget", 12)
	if err != nil {
		t.Fatal(err)
	}
	if issue.Body != "Details" {
		t.Fatalf("issue body = %q", issue.Body)
	}
	created, err := client.CreateIssue(context.Background(), "acme/widget", CreateIssueOptions{Title: "Bug", Body: "Details"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Index != 13 {
		t.Fatalf("created issue = %#v", created)
	}
}

func TestPullRequestEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/acme/widget/pulls":
			if r.URL.Query().Get("state") != "open" || r.URL.Query().Get("limit") != "3" {
				t.Errorf("unexpected pull request query: %v", r.URL.Query())
			}
			fmt.Fprint(w, `[{"number":21,"title":"Feature","state":"open","draft":true}]`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/acme/widget/pulls/21":
			fmt.Fprint(w, `{"number":21,"title":"Feature","body":"Details","state":"open","draft":true}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/acme/widget/pulls":
			var body CreatePullRequestOptions
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create pull request body: %v", err)
			}
			want := CreatePullRequestOptions{Head: "feature", Base: "main", Title: "Feature", Body: "Details"}
			if body != want {
				t.Errorf("create pull request body = %#v, want %#v", body, want)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"number":22,"title":"Feature","state":"open"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	pulls, err := client.ListPullRequests(context.Background(), "acme/widget", "open", 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(pulls) != 1 || !pulls[0].Draft {
		t.Fatalf("pull requests = %#v", pulls)
	}
	pull, err := client.GetPullRequest(context.Background(), "acme/widget", 21)
	if err != nil {
		t.Fatal(err)
	}
	if pull.Body != "Details" {
		t.Fatalf("pull request body = %q", pull.Body)
	}
	created, err := client.CreatePullRequest(context.Background(), "acme/widget", CreatePullRequestOptions{
		Head: "feature", Base: "main", Title: "Feature", Body: "Details",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Index != 22 {
		t.Fatalf("created pull request = %#v", created)
	}
}

func TestRepositoryPathRejectsInvalidRepositoryNames(t *testing.T) {
	for _, repository := range []string{"", "owner", "/name", "owner/", "owner/name/extra"} {
		if _, err := repositoryPath(repository, "/issues"); err == nil {
			t.Errorf("repositoryPath(%q) succeeded, want error", repository)
		}
	}
}
