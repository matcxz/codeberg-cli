package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matcxz/codeberg-cli/internal/forgejo"
)

func TestAPIReadRetriesWithoutInsufficientScopeToken(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") == "token limited-token" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"token does not have at least one of required scope(s): [read:issue]"}`)
			return
		}
		fmt.Fprint(w, `{"number":1,"title":"Public issue","state":"open"}`)
	}))
	defer server.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEBERG_HOST", server.URL)
	t.Setenv("CODEBERG_TOKEN", "limited-token")

	issue, err := apiRead(func(client *forgejo.Client) (forgejo.Issue, error) {
		return client.GetIssue(context.Background(), "owner/repo", 1)
	})
	if err != nil {
		t.Fatal(err)
	}
	if issue.Title != "Public issue" {
		t.Fatalf("issue = %#v", issue)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want authenticated request plus public retry", requests)
	}
}
