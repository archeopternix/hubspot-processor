package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the default root URL for HubSpot API requests.
const DefaultBaseURL = "https://api.hubapi.com"

const maxErrorBodySize = 64 * 1024

// Client is a small HubSpot CRM API client.
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	pageSize    int
}

// NewClient creates a HubSpot client using the supplied bearer token.
func NewClient(accessToken string) *Client {
	return &Client{
		baseURL:     DefaultBaseURL,
		accessToken: strings.TrimSpace(accessToken),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		pageSize:    100,
	}
}

// WithBaseURL overrides the HubSpot base URL, primarily for testing.
func (c *Client) WithBaseURL(baseURL string) *Client {
	if value := strings.TrimSpace(baseURL); value != "" {
		c.baseURL = strings.TrimRight(value, "/")
	}
	return c
}

// WithHTTPClient overrides the HTTP client.
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	if client != nil {
		c.httpClient = client
	}
	return c
}

// WithPageSize overrides the number of records requested per page.
func (c *Client) WithPageSize(pageSize int) *Client {
	if pageSize > 0 && pageSize <= 100 {
		c.pageSize = pageSize
	}
	return c
}

func (c *Client) do(
	ctx context.Context,
	operation string,
	method string,
	path string,
	query url.Values,
	body io.Reader,
	out any,
) error {
	if c.accessToken == "" {
		return fmt.Errorf("HubSpot %s: access token is empty", operation)
	}

	endpoint, err := url.Parse(strings.TrimRight(c.baseURL, "/") + path)
	if err != nil {
		return fmt.Errorf("HubSpot %s: parse URL: %w", operation, err)
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("HubSpot %s: create request: %w", operation, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HubSpot %s: request failed: %w", operation, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		return &APIError{
			Category:   classifyHTTPStatus(resp.StatusCode),
			Operation:  operation,
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(raw)),
		}
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("HubSpot %s: decode response: %w", operation, err)
	}
	return nil
}
