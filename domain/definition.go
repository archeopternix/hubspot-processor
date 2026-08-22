package domain

import "strings"

// AttributeDefinition configures how an attribute participates in the
// data-quality workflow. Name is also used as the HubSpot property name by
// the HubSpot adapter.
type AttributeDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Research    bool   `json:"research"`
	Export      bool   `json:"export"`
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
