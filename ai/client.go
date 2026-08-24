package ai

import (
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the default OpenAI API root used by NewClient.
	DefaultBaseURL = "https://api.openai.com/v1"
	// DefaultModel is the default model used for enrichment requests.
	DefaultModel = "gpt-5.6-luna"
)

// Client is a slim OpenAI Responses API client. Configure a client before it
// is used for requests; concurrent calls to its With methods are not supported.
type Client struct {
	apiKey     string
	endpoint   string
	model      string
	httpClient *http.Client
}

// NewClient creates an OpenAI client with DefaultBaseURL, DefaultModel, and a
// 90-second HTTP timeout. The API key is trimmed before it is stored.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		endpoint:   DefaultBaseURL,
		model:      DefaultModel,
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

// WithBaseURL replaces the API base URL when baseURL is not blank and returns
// the same Client. It mutates the client and should be called during setup.
func (c *Client) WithBaseURL(baseURL string) *Client {
	if value := strings.TrimSpace(baseURL); value != "" {
		c.endpoint = strings.TrimRight(value, "/")
	}
	return c
}

// WithModel replaces the model when model is not blank and returns the same
// Client. It mutates the client and should be called during setup.
func (c *Client) WithModel(model string) *Client {
	if value := strings.TrimSpace(model); value != "" {
		c.model = value
	}
	return c
}

// WithHTTPClient replaces the HTTP client when client is non-nil and returns
// the same Client. It mutates the client and should be called during setup.
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	if client != nil {
		c.httpClient = client
	}
	return c
}
