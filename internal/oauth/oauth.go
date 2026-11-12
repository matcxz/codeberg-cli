package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	callbackPath = "/"
	maxBodySize  = 64 << 10
	loginTimeout = 3 * time.Minute
)

type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

type callbackResult struct {
	code string
	err  error
}

func Authorize(ctx context.Context, host, clientID string, open func(string) error) (Token, error) {
	if strings.TrimSpace(clientID) == "" {
		return Token{}, errors.New("OAuth client ID is required")
	}
	if err := ctx.Err(); err != nil {
		return Token{}, err
	}
	baseURL, err := parseHost(host)
	if err != nil {
		return Token{}, err
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return Token{}, fmt.Errorf("start OAuth callback listener: %w", err)
	}
	defer listener.Close()

	redirectURL := "http://" + listener.Addr().String() + callbackPath
	state, err := randomURLSafe(32)
	if err != nil {
		return Token{}, fmt.Errorf("generate OAuth state: %w", err)
	}
	verifier, err := randomURLSafe(64)
	if err != nil {
		return Token{}, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	challengeSum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeSum[:])

	authorizeURL := *baseURL
	authorizeURL.Path = strings.TrimRight(authorizeURL.Path, "/") + "/login/oauth/authorize"
	query := authorizeURL.Query()
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURL)
	query.Set("response_type", "code")
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	authorizeURL.RawQuery = query.Encode()

	results := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		values := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(values.Get("state")), []byte(state)) != 1 {
			http.Error(w, "Invalid OAuth state. You can close this window.", http.StatusBadRequest)
			return
		}
		if oauthError := values.Get("error"); oauthError != "" {
			description := values.Get("error_description")
			if description == "" {
				description = oauthError
			}
			select {
			case results <- callbackResult{err: fmt.Errorf("OAuth authorization failed: %s", description)}:
			default:
			}
			fmt.Fprintln(w, "Authorization was not completed. You can close this window.")
			return
		}
		code := values.Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code. You can close this window.", http.StatusBadRequest)
			return
		}
		select {
		case results <- callbackResult{code: code}:
		default:
		}
		fmt.Fprintln(w, "Authentication complete. You can close this window and return to cb.")
	})
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := open(authorizeURL.String()); err != nil {
		return Token{}, fmt.Errorf("open authorization page: %w", err)
	}

	loginCtx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()
	var callback callbackResult
	select {
	case callback = <-results:
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return Token{}, fmt.Errorf("OAuth callback server: %w", err)
		}
		return Token{}, errors.New("OAuth callback server closed before authorization completed")
	case <-loginCtx.Done():
		if errors.Is(loginCtx.Err(), context.DeadlineExceeded) {
			return Token{}, errors.New("OAuth login timed out; run `cb auth login --web` again")
		}
		return Token{}, loginCtx.Err()
	}
	if callback.err != nil {
		return Token{}, callback.err
	}
	token, err := exchange(loginCtx, baseURL, url.Values{
		"client_id":     {clientID},
		"code":          {callback.code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURL},
	})
	if err != nil {
		return Token{}, fmt.Errorf("exchange authorization code: %w", err)
	}
	return token, nil
}

func Refresh(ctx context.Context, host, clientID, refreshToken string) (Token, error) {
	if clientID == "" || refreshToken == "" {
		return Token{}, errors.New("OAuth client ID and refresh token are required")
	}
	baseURL, err := parseHost(host)
	if err != nil {
		return Token{}, err
	}
	token, err := exchange(ctx, baseURL, url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
	if err != nil {
		return Token{}, err
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	return token, nil
}

func exchange(ctx context.Context, baseURL *url.URL, values url.Values) (Token, error) {
	endpoint := *baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/login/oauth/access_token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return Token{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("request OAuth token: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return Token{}, fmt.Errorf("read OAuth token response: %w", err)
	}
	var response tokenResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return Token{}, fmt.Errorf("decode OAuth token response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := response.Description
		if message == "" {
			message = response.Error
		}
		if message == "" {
			message = strings.TrimSpace(string(data))
		}
		return Token{}, fmt.Errorf("OAuth token endpoint returned HTTP %d: %s", resp.StatusCode, message)
	}
	if response.Error != "" {
		message := response.Description
		if message == "" {
			message = response.Error
		}
		return Token{}, fmt.Errorf("OAuth token endpoint: %s", message)
	}
	if response.AccessToken == "" {
		return Token{}, errors.New("OAuth token endpoint returned no access token")
	}
	token := Token{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
	}
	if response.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)
	}
	return token, nil
}

func parseHost(host string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(host), "/"))
	if err != nil || baseURL.Host == "" ||
		(baseURL.Scheme != "https" && !(baseURL.Scheme == "http" && isLoopbackHost(baseURL.Hostname()))) {
		return nil, errors.New("instance URL must use HTTPS (HTTP is allowed only for loopback testing)")
	}
	if baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, errors.New("instance URL must not contain credentials, query, or fragment")
	}
	return baseURL, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func randomURLSafe(byteCount int) (string, error) {
	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
