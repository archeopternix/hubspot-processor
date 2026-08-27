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
		{Name: "hs_linkedin_url", Description: "Return only the contact’s current, verified public LinkedIn individual-profile URL. Match the full name with the current employer or company and other professional signals; never guess or return search-result, company, directory, recruiter, or unrelated URLs. Remove tracking parameters and fragments, and return an empty string if no unambiguous match can be verified.", Research: true, Export: true},
		{Name: "career_summary", Description: "Create a concise, factual career summary in plain text using only verified profile information. Include the contact's confirmed current role and employer, relevant career progression, education, and explicitly documented skills where available. Write 2–4 sentences without reproducing the full job history. Omit all unverified or inferred information; do not infer seniority, expertise, personality, nationality, hobbies, interests, or career goals. Do not include unsupported claims, sales language, links, URLs, citations, references, HTML, RTF, Markdown, or code fences. Return an empty value if there is insufficient verified information.", Research: true, Export: true},
		{Name: "xing_url", Description: "Find and return the contact's verified public XING individual-profile URL. Search using the contact's full name and company. Accept valid XING profile URL variants, including profile subpaths such as /web_profiles; when it is verified as the contact's profile. The company is a supporting identity signal, not a mandatory match when the profile name matches and no contradictory information is found. Do not reject a supported URL only because the page is dynamically rendered, not indexed, requires login to view some content, or cannot be fully fetched. Remove only tracking parameters or fragments when normalizing the URL. Never guess URLs or return search-result, company, directory, or unrelated URLs. If the profile is ambiguous or cannot be verified, return an empty value.", Research: true, Export: true},
		{Name: "state", Description: "Publicly verified professional state or region; do not infer a private residence", Research: true, Export: true},
		{Name: "zip", Description: "Publicly verified professional postal code; do not infer a private residence", Research: true, Export: true},
		{Name: "country", Description: "Publicly verified professional country or region", Research: true, Export: true},
		{Name: "ct_quick_context", Description: "Compile a concise, structured professional context for a contact as plain text. Do not use HTML, RTF, Markdown heading markers, code fences, external links, URLs, hyperlinks, citations in the text. Add references, and source attributions in a last section REFERENCES. Use only verified information from the contact profile and omit anything that cannot be verified. Start with a brief Summary that does not repeat information covered in later sections. Use the sections in exactly this order: Summary [+ date of enrichment]; Current role; Professional experience; Education; Skills. Put one blank line before and one blank line after every heading. Put each list item on its own line, prefix it with '- ', and terminate every line with CRLF (\\r\\n). Use clear, simple professional language. Keep the context brief and factual. Do not infer nationality, hobbies, skills, personality traits, interests, seniority, or other personal characteristics. Do not include nationality unless it is explicitly verified and required for the business use case. Do not repeat the same fact in multiple sections. Summary should contain two to three concise sentences covering the person's confirmed current role and professional focus. Current role should contain the confirmed job title and employer. Professional experience should list relevant jobs from newest to oldest using the format '- YYYY-MM or YYYY–YYYY: Role at Employer'; use 'present' only when the role is explicitly ongoing. Education should list confirmed qualifications and institutions, including years only when verified. Skills should list only explicitly documented professional or technical skills. Omit the section content when none are verified. Never fabricate or complete missing information from assumptions. Insert links where the information was collected from", Research: true, Export: true},
		{Name: "ai_enriched_date", Description: "Timestamp of the AI enrichment; set by the export workflow", Research: false, Export: true},
		{Name: "hs_lastmodifieddate", Description: "HubSpot last modified timestamp used as technical context", Research: false, Export: false},
	},
}

// ContactResearchPrompt defines the business-level research policy used when
// enriching contact objects. It is compiled into the application so no prompt
// file needs to be read at runtime.
const ContactResearchPrompt = `You are a B2B contact-enrichment agent. Research the professional contact represented by the supplied HubSpot data and populate the fields defined in Attributes.

Rules:
- Use only public, professional sources. 
- Verify the identity using multiple professional signals, such as name, company, role, location, or career history. Do not combine information from namesakes.
- Never guess, construct, or extrapolate values. Return an empty string when a value cannot be verified.
- Follow each Attribute description exactly. 
- Use current, reliable information for roles, employers, locations, and profile URLs.
- Return verified individual LinkedIn or XING URLs. Omit ambiguous, guessed or unrelated URLs.
- Return exactly one valid JSON object using the exact exported attribute names and types. Do not include extra fields, Markdown, HTML, comments, explanations, or reasoning.`
