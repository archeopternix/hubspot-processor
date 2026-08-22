package domain

import "strings"

// AttributeState contains the lifecycle of a single object attribute.
type AttributeState struct {
	Import   string  `json:"import,omitempty"`
	Proposal string  `json:"proposal,omitempty"`
	Export   string  `json:"export,omitempty"`
	Score    float64 `json:"score,omitempty"`
	IsExport bool    `json:"is_export"`
}

// Evaluate decides whether a proposal is eligible for export.
// Existing imported values are never overwritten.
func (q AttributeState) Evaluate(definition AttributeDefinition, minScore float64) AttributeState {
	q.Export = ""
	q.IsExport = false

	if strings.TrimSpace(q.Import) != "" {
		return q
	}
	if !definition.Export {
		return q
	}
	if strings.TrimSpace(q.Proposal) == "" {
		return q
	}
	if q.Score < minScore {
		return q
	}

	q.Export = strings.TrimSpace(q.Proposal)
	q.IsExport = true
	return q
}
