package domain

import "strings"

// AttributeValueType describes additional value constraints that apply to an
// attribute proposal and export value.
type AttributeValueType string

const (
	AttributeValueText    AttributeValueType = ""
	AttributeValueInteger AttributeValueType = "integer"
)

// AttributeDefinition configures how an attribute participates in the
// data-quality workflow. Name is also used as the HubSpot property name by
// the HubSpot adapter.
type AttributeDefinition struct {
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	AllowedValues []string           `json:"allowed_values,omitempty"`
	ValueType     AttributeValueType `json:"value_type,omitempty"`
	Required      bool               `json:"required"`
	Research      bool               `json:"research"`
	Export        bool               `json:"export"`
}

// AcceptsValue reports whether value satisfies the attribute's type and
// optional allowed-values constraints.
func (d AttributeDefinition) AcceptsValue(value string) bool {
	value = strings.TrimSpace(value)
	if d.ValueType == AttributeValueInteger && !isNonNegativeInteger(value) {
		return false
	}

	if len(d.AllowedValues) == 0 {
		return true
	}
	for _, allowed := range d.AllowedValues {
		if value == strings.TrimSpace(allowed) {
			return true
		}
	}
	return false
}

// NormalizeProposal prepares a researched value for storage. All proposals are
// trimmed, and attribute-specific storage formatting is applied where needed.
func (d AttributeDefinition) NormalizeProposal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || d.Name != "hs_quick_context" {
		return value
	}

	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n") + "\r\n"
}

func isNonNegativeInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// ObjectDefinition describes one CRM object type and the attributes to read.
type ObjectDefinition struct {
	Type       string                `json:"type"`
	Attributes []AttributeDefinition `json:"attributes"`
}

// PropertyNames returns the unique configured property names in stable order.
func (d ObjectDefinition) PropertyNames() []string {
	properties := make([]string, 0, len(d.Attributes))
	seen := make(map[string]struct{}, len(d.Attributes))

	for _, definition := range d.Attributes {
		name := strings.TrimSpace(definition.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}

		seen[name] = struct{}{}
		properties = append(properties, name)
	}

	return properties
}

// Attribute returns the definition for one attribute.
func (d ObjectDefinition) Attribute(name string) (AttributeDefinition, bool) {
	for _, definition := range d.Attributes {
		if definition.Name == name {
			return definition, true
		}
	}
	return AttributeDefinition{}, false
}

// WithAllowedValues returns a copy of the definition with an allowed-values
// constraint applied to one attribute. The original definition is unchanged.
func (d ObjectDefinition) WithAllowedValues(name string, values []string) ObjectDefinition {
	result := d
	result.Attributes = append([]AttributeDefinition(nil), d.Attributes...)

	name = strings.TrimSpace(name)
	for i := range result.Attributes {
		if result.Attributes[i].Name != name {
			continue
		}
		result.Attributes[i].AllowedValues = append([]string(nil), values...)
		break
	}

	return result
}

// ResearchAttributes returns all attributes that should be researched.
func (d ObjectDefinition) ResearchAttributes() []AttributeDefinition {
	result := make([]AttributeDefinition, 0, len(d.Attributes))
	for _, attribute := range d.Attributes {
		if attribute.Research && strings.TrimSpace(attribute.Name) != "" {
			result = append(result, attribute)
		}
	}
	return result
}
