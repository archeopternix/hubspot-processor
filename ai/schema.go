package ai

import "github.com/archeopternix/hubspot-processor/domain"

func buildResearchSchema(attributes []domain.AttributeDefinition) map[string]any {
	attributeSchemas := make([]any, 0, len(attributes))
	for _, attribute := range attributes {
		attributeSchemas = append(attributeSchemas, objectSchema(
			map[string]any{
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
			"name", "proposal", "score",
		))
	}

	return objectSchema(
		map[string]any{
			"attributes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"anyOf": attributeSchemas,
				},
				"minItems": len(attributes),
				"maxItems": len(attributes),
			},
		},
		"attributes",
	)
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
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
