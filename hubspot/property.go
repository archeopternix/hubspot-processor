package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type propertyDefinition struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Options []struct {
		Value  string `json:"value"`
		Hidden bool   `json:"hidden"`
	} `json:"options"`
}

// ReadPropertyOptions returns the active internal option values configured for
// one HubSpot enumeration property.
func (c *Client) ReadPropertyOptions(
	ctx context.Context,
	objectType string,
	propertyName string,
) ([]string, error) {
	objectType = strings.TrimSpace(objectType)
	propertyName = strings.TrimSpace(propertyName)
	if objectType == "" {
		return nil, fmt.Errorf("HubSpot property read: object type is empty")
	}
	if propertyName == "" {
		return nil, fmt.Errorf("HubSpot property read: property name is empty")
	}
	if c.accessToken == "" {
		return nil, fmt.Errorf("HubSpot property read: access token is empty")
	}

	endpoint, err := url.Parse(
		strings.TrimRight(c.baseURL, "/") +
			"/crm/properties/2026-03/" + url.PathEscape(objectType) +
			"/" + url.PathEscape(propertyName),
	)
	if err != nil {
		return nil, fmt.Errorf("HubSpot property read: parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("HubSpot property read: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HubSpot property read: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf(
			"HubSpot property read: returned %s for %s.%s: %s",
			resp.Status,
			objectType,
			propertyName,
			strings.TrimSpace(string(body)),
		)
	}

	var property propertyDefinition
	if err := json.NewDecoder(resp.Body).Decode(&property); err != nil {
		return nil, fmt.Errorf("HubSpot property read: decode response: %w", err)
	}
	if property.Type != "enumeration" {
		return nil, fmt.Errorf(
			"HubSpot property read: %s.%s has type %q, want enumeration",
			objectType,
			propertyName,
			property.Type,
		)
	}

	values := make([]string, 0, len(property.Options))
	seen := make(map[string]struct{}, len(property.Options))
	for _, option := range property.Options {
		value := strings.TrimSpace(option.Value)
		if option.Hidden || value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"HubSpot property read: %s.%s has no active options",
			objectType,
			propertyName,
		)
	}

	return values, nil
}
