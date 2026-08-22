package hubspot

import (
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.hubapi.com"

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
