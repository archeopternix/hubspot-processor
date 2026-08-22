package ai

import "github.com/archeopternix/hubspot-processor/domain"

func buildResearchSchema(definition domain.ObjectDefinition) map[string]any {
	attributes := definition.ResearchAttributes()
	attributeSchemas := make([]any, 0, len(attributes))
	for _, attribute := range attributes {
		attributeSchemas = append(attributeSchemas, map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type": "string",
					"enum": []string{attribute.Name},
				},
				"proposal": buildProposalSchema(attribute),
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

func buildProposalSchema(attribute domain.AttributeDefinition) map[string]any {
	if len(attribute.AllowedValues) > 0 {
		allowed := make([]string, 0, len(attribute.AllowedValues)+1)
		allowed = append(allowed, "")
		allowed = append(allowed, attribute.AllowedValues...)
		return map[string]any{
			"type": "string",
			"enum": allowed,
		}
	}

	if attribute.ValueType == domain.AttributeValueInteger {
		return map[string]any{
			"anyOf": []any{
				map[string]any{
					"type": "string",
					"enum": []string{""},
				},
				map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		}
	}

	return map[string]any{"type": "string"}
}
