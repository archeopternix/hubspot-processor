package domain

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
