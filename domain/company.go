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
		{Name: "hs_quick_context", Description: "Write a concise, structured company overview as plain text; do not use HTML, RTF, Markdown heading markers, or code fences. Start with a brief Summary that does not repeat information covered in later sections. Use the sections in exactly this order: Summary; Services and products; Major clients or customer groups; Business strategy; Key differentiators; History and major milestones; Company structure. Company structure must be the final section in the field. Put one blank line before and one blank line after every heading. Put each list item on its own line, prefix it with '- ', and terminate every line with CRLF (\\r\\n). Format History and major milestones as '- ' bullet points as well. Use only verified information and clear, simple professional language. Omit any information that cannot be verified. Do not include external links, URLs, hyperlinks, citations, references, or source attributions. Avoid unsupported claims, excessive marketing language, and repetition.", Research: true, Export: true},
		{Name: "ai_enriched_date", Description: "Timestamp of the AI enrichment; set by the export workflow", Research: false, Export: true},
		{Name: "hs_lastmodifieddate", Description: "HubSpot last modified timestamp used as technical context", Research: false, Export: false},
	},
}
