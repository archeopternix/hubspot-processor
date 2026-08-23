package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/archeopternix/hubspot-processor/domain"
)

type writeRequest struct {
	Properties map[string]string `json:"properties"`
}

// Write updates one CRM object in HubSpot using the supplied definition.
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

	path := "/crm/v3/objects/" + url.PathEscape(objectType) + "/" +
		url.PathEscape(object.ID)
	return c.do(
		ctx,
		"write "+objectType+" "+object.ID,
		http.MethodPatch,
		path,
		nil,
		bytes.NewReader(payload),
		nil,
	)
}
