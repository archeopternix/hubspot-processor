package hubspot

import (
	"context"
	"fmt"
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

	path := "/crm/properties/2026-03/" + url.PathEscape(objectType) +
		"/" + url.PathEscape(propertyName)
	var property propertyDefinition
	if err := c.do(
		ctx,
		"property read "+objectType+"."+propertyName,
		http.MethodGet,
		path,
		nil,
		nil,
		&property,
	); err != nil {
		return nil, err
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
