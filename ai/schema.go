package ai

import "github.com/archeopternix/hubspot-processor/domain"

func buildResearchSchema(definition domain.ObjectDefinition) map[string]any {
	names := researchAttributeNames(definition)

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"attributes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
							"enum": names,
						},
						"proposal": map[string]any{
							"type": "string",
						},
						"score": map[string]any{
							"type":    "number",
							"minimum": 0,
							"maximum": 1,
						},
					},
					"required":             []string{"name", "proposal", "score"},
					"additionalProperties": false,
				},
				"minItems": len(names),
				"maxItems": len(names),
			},
		},
		"required":             []string{"attributes"},
		"additionalProperties": false,
	}
}
