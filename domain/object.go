package domain

// Object is the generic domain representation of a HubSpot CRM object.
// Attribute keys are the configured property names.
type Object struct {
	ID         string             `json:"id"`
	Attributes map[string]Quality `json:"attributes"`
}

// NewObject creates an Object with all configured attributes initialized.
func NewObject(id string, definitions []AttributeDefinition) Object {
	object := Object{
		ID:         id,
		Attributes: make(map[string]Quality, len(definitions)),
	}

	for _, definition := range definitions {
		if definition.Name == "" {
			continue
		}
		object.Attributes[definition.Name] = Quality{}
	}

	return object
}

// SetImport stores the value read from the source system.
func (o *Object) SetImport(name, value string) {
	quality := o.Attributes[name]
	quality.Import = value
	o.Attributes[name] = quality
}
