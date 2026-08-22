package ai

import "github.com/archeopternix/hubspot-processor/domain"

func buildResearchSchema(definition domain.ObjectDefinition) map[string]any {
	attributes := definition.ResearchAttributes()
	attributeSchemas := make([]any, 0, len(attributes))
	for _, attribute := range attributes {
		proposalSchema := map[string]any{
			"type": "string",
		}
		if len(attribute.AllowedValues) > 0 {
			allowed := make([]string, 0, len(attribute.AllowedValues)+1)
			allowed = append(allowed, "")
			allowed = append(allowed, attribute.AllowedValues...)
			proposalSchema["enum"] = allowed
		}

		attributeSchemas = append(attributeSchemas, map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type": "string",
					"enum": []string{attribute.Name},
				},
				"proposal": proposalSchema,
				"score": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
			},
			"required":             []string{"name", "proposal", "score"},
			"additionalProperties": false,
		})
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"attributes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"anyOf": attributeSchemas,
				},
				"minItems": len(attributes),
				"maxItems": len(attributes),
			},
		},
		"required":             []string{"attributes"},
		"additionalProperties": false,
	}
}
