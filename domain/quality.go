package domain

import "strings"

// Quality contains the lifecycle of a single attribute during data-quality processing.
type Quality struct {
	Import   string  `json:"import,omitempty"`
	Proposal string  `json:"proposal,omitempty"`
	Export   string  `json:"export,omitempty"`
	Score    float64 `json:"score,omitempty"`
	IsExport bool    `json:"is_export"`
}

// Evaluate decides whether a proposal is eligible for export.
// Existing imported values are never overwritten.
func (q Quality) Evaluate(definition AttributeDefinition, minScore float64) Quality {
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
