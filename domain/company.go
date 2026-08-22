package domain

// CompanyDefinition is only a configuration preset. The actual data is read
// into the generic Object type.
var CompanyDefinition = ObjectDefinition{
	Type: "companies",
	Attributes: []AttributeDefinition{
		{Name: "name", Required: true, Research: true, Export: true},
		{Name: "annualrevenue", Research: true, Export: true},
		{Name: "city", Required: true, Research: true, Export: true},
		{Name: "country", Required: true, Research: true, Export: true},
		{Name: "description", Required: true, Research: true, Export: true},
		{Name: "industry", Required: true, Research: true, Export: true},
		{Name: "zip", Required: true, Research: true, Export: true},
		{Name: "address", Required: true, Research: true, Export: true},
		{Name: "state", Research: true, Export: true},
		{Name: "founded_year", Research: true, Export: true},
		{Name: "kiunternehmensprofil", Required: true, Research: true, Export: true},
		{Name: "numberofemployees", Research: true, Export: true},
		{Name: "domain", Research: true, Export: true},
		{Name: "about_us", Research: true, Export: true},
		{Name: "hs_quick_context", Research: false, Export: false},
		{Name: "ai_enriched_date", Research: false, Export: true},
		{Name: "hs_lastmodifieddate", Research: false, Export: false},
	},
}
