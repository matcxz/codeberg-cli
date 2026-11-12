package forgejo

import (
	"bytes"
	"context"
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

const maxResponseSize = 4 << 20

type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("Codeberg API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("Codeberg API returned HTTP %d: %s", e.StatusCode, e.Message)
}

func IsInsufficientScope(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) &&
		apiErr.StatusCode == http.StatusForbidden &&
		strings.Contains(strings.ToLower(apiErr.Message), "scope")
}

func NewClient(host, token string) (*Client, error) {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	baseURL, err := url.Parse(host)
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "https" && baseURL.Scheme != "http") {
		return nil, fmt.Errorf("invalid instance URL %q: expected an http or https URL", host)
	}
	if baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("invalid instance URL %q: credentials, query, and fragment are not allowed", host)
	}
	if baseURL.Scheme != "https" && !isLoopbackHost(baseURL.Hostname()) {
		return nil, fmt.Errorf("refusing to send credentials over insecure HTTP; use HTTPS")
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/api/v1"
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) Get(ctx context.Context, path string, query url.Values, result any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, result)
}

func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, result)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, result any) error {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawPath = ""
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode API request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return fmt.Errorf("create API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request Codeberg API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		if readErr != nil {
			return fmt.Errorf("Codeberg API returned HTTP %d (could not read error response: %w)", resp.StatusCode, readErr)
		}
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &apiErr)
		if apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(data))
		}
		return &APIError{StatusCode: resp.StatusCode, Message: apiErr.Message}
	}
	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize))
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("decode Codeberg API response: %w", err)
	}
	return nil
}
