package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matcxz/codeberg-cli/internal/config"
)

func TestLoadConfigDoesNotSendSavedTokenToOverrideHost(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEBERG_HOST", "https://other.example")
	t.Setenv("CODEBERG_TOKEN", "")
	if err := config.Save(config.Config{Host: "https://codeberg.org", Token: "saved-secret"}); err != nil {
		t.Fatal(err)
	}

	got, err := loadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "" {
		t.Fatalf("token for another host was retained: %#v", got)
	}
	if got.Host != "https://other.example" {
		t.Fatalf("host = %q", got.Host)
	}
}

func TestLoadConfigRefreshesExpiringOAuthToken(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse refresh request: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" ||
			r.Form.Get("refresh_token") != "old-refresh" ||
			r.Form.Get("client_id") != "client-id" {
			t.Errorf("refresh request form = %v", r.Form)
		}
		fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
	}))
	defer server.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEBERG_HOST", "")
	t.Setenv("CODEBERG_TOKEN", "")
	expired := time.Now().Add(-time.Minute)
	if err := config.Save(config.Config{
		Host:           server.URL,
		Token:          "old-access",
		Username:       "matcxz",
		OAuthClientID:  "client-id",
		RefreshToken:   "old-refresh",
		TokenExpiresAt: &expired,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := loadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "new-access" || got.RefreshToken != "new-refresh" {
		t.Fatalf("refreshed config = %#v", got)
	}
	if requests != 1 {
		t.Fatalf("refresh requests = %d, want 1", requests)
	}
	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Token != got.Token {
		t.Fatalf("saved token = %q, want %q", saved.Token, got.Token)
	}

}
