package ai

import (
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "gpt-5.6-luna"
)

// Client is a slim OpenAI Responses API client.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		baseURL:    DefaultBaseURL,
		model:      DefaultModel,
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) WithBaseURL(baseURL string) *Client {
	if value := strings.TrimSpace(baseURL); value != "" {
		c.baseURL = strings.TrimRight(value, "/")
	}
	return c
}

func (c *Client) WithModel(model string) *Client {
	if value := strings.TrimSpace(model); value != "" {
		c.model = value
	}
	return c
}

func (c *Client) WithHTTPClient(client *http.Client) *Client {
	if client != nil {
		c.httpClient = client
	}
	return c
}
