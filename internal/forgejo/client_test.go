package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrentUserUsesTokenAndInstanceBasePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forgejo/api/v1/user" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "token test-secret" {
			t.Errorf("Authorization = %q", got)
		}
		fmt.Fprint(w, `{"login":"matcxz"}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/forgejo/", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "matcxz" {
		t.Fatalf("username = %q, want matcxz", user.Username)
	}
}

func TestListRepositoriesUsesOwnerAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/matcxz/repos" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page = %q, want 2", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}
		fmt.Fprint(w, `[{"full_name":"matcxz/project","private":false,"stars_count":3}]`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	repositories, err := client.ListRepositories(context.Background(), "matcxz", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 1 || repositories[0].FullName != "matcxz/project" {
		t.Fatalf("repositories = %#v", repositories)
	}
}

func TestCreateRepositorySendsExpectedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/user/repos" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var got CreateRepositoryOptions
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		want := CreateRepositoryOptions{Name: "new-project", Description: "demo", Private: true, AutoInit: true}
		if got != want {
			t.Errorf("payload = %#v, want %#v", got, want)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"full_name":"matcxz/new-project"}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	repository, err := client.CreateRepository(context.Background(), CreateRepositoryOptions{
		Name: "new-project", Description: "demo", Private: true, AutoInit: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.FullName != "matcxz/new-project" {
		t.Fatalf("created repository = %#v", repository)
	}
}

func TestNewClientRejectsInsecureRemoteHost(t *testing.T) {
	if _, err := NewClient("http://codeberg.example", "token"); err == nil {
		t.Fatal("NewClient accepted insecure non-loopback HTTP")
	}
}

func TestAPIErrorIncludesStatusAndMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"token is invalid"}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CurrentUser(context.Background())
	if err == nil || err.Error() != "Codeberg API returned HTTP 401: token is invalid" {
		t.Fatalf("error = %v", err)
	}
}

func TestIsInsufficientScope(t *testing.T) {
	if !IsInsufficientScope(&APIError{
		StatusCode: http.StatusForbidden,
		Message:    "token does not have required scope read:issue",
	}) {
		t.Fatal("IsInsufficientScope did not recognize a missing-scope API error")
	}
	if IsInsufficientScope(&APIError{StatusCode: http.StatusForbidden, Message: "access denied"}) {
		t.Fatal("IsInsufficientScope matched an unrelated forbidden error")
	}
}
