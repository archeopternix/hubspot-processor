package domain

import "strings"

// Object is the generic domain representation of a HubSpot CRM object.
// Attribute keys are the configured property names.
type Object struct {
	ID         string                    `json:"id"`
	Attributes map[string]AttributeState `json:"attributes"`
}

// NewObject creates an Object with all configured attributes initialized.
func NewObject(id string, definitions []AttributeDefinition) Object {
	object := Object{
		ID:         id,
		Attributes: make(map[string]AttributeState, len(definitions)),
	}

	for _, definition := range definitions {
		if definition.Name == "" {
			continue
		}
		object.Attributes[definition.Name] = AttributeState{}
	}

	return object
}

// ImportedValue returns one imported attribute value with surrounding
// whitespace removed. Nil objects and missing attributes return an empty value.
func (o *Object) ImportedValue(name string) string {
	if o == nil {
		return ""
	}
	attribute, ok := o.Attributes[name]
	if !ok {
		return ""
	}
	return strings.TrimSpace(attribute.Import)
}

// Name returns a display name for company and contact objects.
func (o *Object) Name() string {
	if name := o.ImportedValue("name"); name != "" {
		return name
	}

	name := strings.TrimSpace(
		o.ImportedValue("firstname") + " " + o.ImportedValue("lastname"),
	)
	if name != "" {
		return name
	}

	return o.ImportedValue("email")
}

// SetImport stores the value read from the source system.
func (o *Object) SetImport(name, value string) {
	attribute := o.Attributes[name]
	attribute.Import = value
	o.Attributes[name] = attribute
}

// SetProposal updates only the AI-proposed value and its confidence score.
// Imported and export-related values are preserved.
func (o *Object) SetProposal(name, value string, score float64) {
	attribute := o.Attributes[name]
	attribute.Proposal = value
	attribute.Score = score
	o.Attributes[name] = attribute
}
