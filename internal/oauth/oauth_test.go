package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthorizeUsesPKCEStateAndExchangesCode(t *testing.T) {
	expectedChallenge := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/oauth/access_token" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token request: %v", err)
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if got := r.Form.Get("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.Form.Get("client_id"); got != "test-client" {
			t.Errorf("client_id = %q", got)
		}
		if got := r.Form.Get("code"); got != "one-time-code" {
			t.Errorf("code = %q", got)
		}
		verifier := r.Form.Get("code_verifier")
		if len(verifier) < 43 || len(verifier) > 128 {
			t.Errorf("invalid PKCE verifier length %d", len(verifier))
		}
		digest := sha256.Sum256([]byte(verifier))
		if got := base64.RawURLEncoding.EncodeToString(digest[:]); got != expectedChallenge {
			t.Errorf("PKCE challenge = %q, want %q", got, expectedChallenge)
		}
		if !strings.HasPrefix(r.Form.Get("redirect_uri"), "http://127.0.0.1:") {
			t.Errorf("redirect_uri = %q", r.Form.Get("redirect_uri"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"access-secret","refresh_token":"refresh-secret","expires_in":3600}`)
	}))
	defer server.Close()

	token, err := Authorize(context.Background(), server.URL, "test-client", func(target string) error {
		authorizeURL, err := url.Parse(target)
		if err != nil {
			return err
		}
		query := authorizeURL.Query()
		if query.Get("response_type") != "code" || query.Get("code_challenge_method") != "S256" {
			t.Errorf("authorization parameters = %v", query)
		}
		if len(query.Get("code_challenge")) != 43 {
			t.Errorf("challenge length = %d", len(query.Get("code_challenge")))
		}
		expectedChallenge = query.Get("code_challenge")

		redirectURL, err := url.Parse(query.Get("redirect_uri"))
		if err != nil {
			return err
		}
		badCallback := *redirectURL
		badQuery := badCallback.Query()
		badQuery.Set("state", "wrong")
		badQuery.Set("code", "one-time-code")
		badCallback.RawQuery = badQuery.Encode()
		response, err := http.Get(badCallback.String())
		if err != nil {
			return err
		}
		response.Body.Close()
		if response.StatusCode != http.StatusBadRequest {
			t.Errorf("invalid state returned HTTP %d", response.StatusCode)
		}

		callback := *redirectURL
		callbackQuery := callback.Query()
		callbackQuery.Set("state", query.Get("state"))
		callbackQuery.Set("code", "one-time-code")
		callback.RawQuery = callbackQuery.Encode()
		response, err = http.Get(callback.String())
		if err == nil {
			response.Body.Close()
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access-secret" || token.RefreshToken != "refresh-secret" {
		t.Fatalf("token = %#v", token)
	}
	if time.Until(token.ExpiresAt) < 59*time.Minute {
		t.Fatalf("token expiry is too soon: %v", token.ExpiresAt)
	}
}

func TestRefreshRetainsRefreshTokenWhenProviderDoesNotRotate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse refresh request: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-secret" {
			t.Errorf("refresh form = %v", r.Form)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	token, err := Refresh(context.Background(), server.URL, "test-client", "refresh-secret")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "new-access" || token.RefreshToken != "refresh-secret" {
		t.Fatalf("refreshed token = %#v", token)
	}
}

func TestAuthorizeRejectsInsecureRemoteHost(t *testing.T) {
	_, err := Authorize(context.Background(), "http://example.org", "client", func(string) error {
		t.Fatal("browser opener called with an insecure remote host")
		return nil
	})
	if err == nil {
		t.Fatal("Authorize accepted insecure HTTP")
	}
}
