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

// Evaluate decides whether a proposed value meets the confidence threshold for
// export. Attribute-specific export policy is applied by the object evaluator.
func (q AttributeState) Evaluate(
	minimalConfidence float64,
	highConfidence float64,
) AttributeState {
	q.Export = ""
	q.IsExport = false

	proposal := strings.TrimSpace(q.Proposal)
	if proposal == "" {
		return q
	}

	requiredConfidence := highConfidence
	if strings.TrimSpace(q.Import) == "" {
		requiredConfidence = minimalConfidence
	}
	if q.Score < requiredConfidence {
		return q
	}

	q.Export = proposal
	q.IsExport = true
	return q
}
