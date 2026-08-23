package domain

// ContactDefinition configures public professional contact enrichment. Values
// must be verified; private or inferred personal data must not be proposed.
var ContactDefinition = ObjectDefinition{
	Type: "contacts",
	Attributes: []AttributeDefinition{
		{Name: "firstname", Description: "Verified first name of the contact", Required: true, Research: true, Export: true},
		{Name: "lastname", Description: "Verified last name of the contact", Required: true, Research: true, Export: true},
		{Name: "email", Description: "Publicly listed professional email address; do not infer or construct an address", Research: true, Export: true},
		{Name: "phone", Description: "Publicly listed professional phone number; do not use a private number", Research: true, Export: true},
		{Name: "mobilephone", Description: "Publicly listed professional mobile number; do not use a private number", Research: true, Export: true},
		{Name: "jobtitle", Description: "Current verified professional job title", Research: true, Export: true},
		{Name: "company", Description: "Current verified employer or company name", Research: true, Export: true},
		{Name: "city", Description: "Publicly verified professional city or work location; do not infer a private residence", Research: true, Export: true},
		{Name: "state", Description: "Publicly verified professional state or region; do not infer a private residence", Research: true, Export: true},
		{Name: "zip", Description: "Publicly verified professional postal code; do not infer a private residence", Research: true, Export: true},
		{Name: "country", Description: "Publicly verified professional country or region", Research: true, Export: true},
		{Name: "ai_enriched_date", Description: "Timestamp of the AI enrichment; set by the export workflow", Research: false, Export: true},
		{Name: "hs_lastmodifieddate", Description: "HubSpot last modified timestamp used as technical context", Research: false, Export: false},
	},
}

// ContactResearchPrompt defines the business-level research policy used when
// enriching contact objects.
const ContactResearchPrompt = `Research the professional contact represented by the supplied HubSpot CRM data.

Identify the exact person before extracting facts and distinguish them from people with similar names.
Use current, reliable, publicly available professional sources.
Prefer official employer websites, official professional biographies, and other primary sources.
Return email addresses and phone numbers only when they are explicitly published as professional contact details.
Do not infer or construct contact details, use private personal information, or invent information.`
