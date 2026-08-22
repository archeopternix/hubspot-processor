package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/archeopternix/hubspot-processor/domain"
)

type writeRequest struct {
	Properties map[string]string `json:"properties"`
}

// Write writes one Company record to HubSpot.
//
// Only attributes explicitly marked with AttributeState.IsExport == true are sent.
// The value written to HubSpot is AttributeState.Export. Import and Proposal are
// never written directly.
func (c *Client) Write(
	ctx context.Context,
	object *domain.Object,
	definition domain.ObjectDefinition,
) error {
	if object == nil {
		return fmt.Errorf("HubSpot write: object is nil")
	}
	if strings.TrimSpace(object.ID) == "" {
		return fmt.Errorf("HubSpot write: object ID is empty")
	}
	if c.accessToken == "" {
		return fmt.Errorf("HubSpot write: access token is empty")
	}
	objectType := strings.TrimSpace(definition.Type)
	if objectType == "" {
		return fmt.Errorf("HubSpot write: object type is empty")
	}

	properties := make(map[string]string)
	for name, state := range object.Attributes {
		if !state.IsExport {
			continue
		}
		if name == "id" || name == "name" {
			return fmt.Errorf("HubSpot write: property %q must never be exported", name)
		}

		attribute, configured := definition.Attribute(name)
		if !configured {
			return fmt.Errorf("HubSpot write: property %q is not configured", name)
		}
		if !attribute.Export {
			return fmt.Errorf("HubSpot write: property %q is not enabled for export", name)
		}
		if !attribute.AcceptsValue(state.Export) {
			return fmt.Errorf(
				"HubSpot write: value %q is not allowed for property %q",
				state.Export,
				name,
			)
		}

		properties[name] = state.Export
	}

	if len(properties) == 0 {
		return fmt.Errorf("HubSpot write: object %s has no attributes marked for export", object.ID)
	}

	payload, err := json.Marshal(writeRequest{Properties: properties})
	if err != nil {
		return fmt.Errorf("HubSpot write: encode request: %w", err)
	}

	endpoint, err := url.Parse(
		strings.TrimRight(c.baseURL, "/") +
			"/crm/v3/objects/" + url.PathEscape(objectType) + "/" +
			url.PathEscape(object.ID),
	)
	if err != nil {
		return fmt.Errorf("HubSpot write: parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		endpoint.String(),
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("HubSpot write: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HubSpot write: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf(
			"HubSpot write: returned %s for %s %s: %s",
			resp.Status,
			objectType,
			object.ID,
			strings.TrimSpace(string(body)),
		)
	}

	return nil
}
