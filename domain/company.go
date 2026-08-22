package domain

// CompanyDefinition is only a configuration preset. The actual data is read
// into the generic Object type.
var CompanyDefinition = ObjectDefinition{
	Type: "companies",
	Attributes: []AttributeDefinition{
		{Name: "name", Description: "Official company name", Required: true, Research: true, Export: true},
		{Name: "annualrevenue", Description: "Most recent annual revenue as an absolute numeric value", Research: true, Export: true},
		{Name: "city", Description: "City of the company headquarters", Required: true, Research: true, Export: true},
		{Name: "country", Description: "Country or region of the company headquarters", Required: true, Research: true, Export: true},
		{Name: "industry", Description: "Primary industry of the company", Required: true, Research: true, Export: true},
		{Name: "zip", Description: "Postal code of the company headquarters", Required: true, Research: true, Export: true},
		{Name: "address", Description: "Street address of the company headquarters", Required: true, Research: true, Export: true},
		{Name: "state", Description: "State or region of the company headquarters", Research: true, Export: true},
		{Name: "founded_year", Description: "Four-digit year in which the company was founded", Research: true, Export: true},
		{Name: "numberofemployees", Description: "Most recent known number of employees", Research: true, Export: true},
		{Name: "domain", Description: "Official primary internet domain of the company", Research: true, Export: true},
		{Name: "hs_quick_context", Description: "Write a concise, structured company overview covering its organizational structure, services, products, major clients or customer groups, business strategy, and key differentiators. Start with a brief summary that does not repeat the details in the sections below. Use clear, simple professional language with descriptive headings and bullet points where helpful. Include a History and major milestones section covering events such as founding, renaming, mergers, acquisitions, and takeovers. Use only verified information, avoid unsupported claims and excessive marketing language, and format the content for easy reading in a rich text field.", Research: true, Export: true},
		{Name: "ai_enriched_date", Description: "Timestamp of the AI enrichment; set by the export workflow", Research: false, Export: true},
		{Name: "hs_lastmodifieddate", Description: "HubSpot last modified timestamp used as technical context", Research: false, Export: false},
	},
}
