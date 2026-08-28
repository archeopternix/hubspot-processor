package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/archeopternix/hubspot-processor/domain"
)

// WriteOne evaluates one object's proposals and writes eligible exports to
// HubSpot. Objects without eligible changes are skipped.
func (s *Service) WriteOne(ctx context.Context, object *domain.Object) (Status, error) {
	if object == nil {
		return StatusFailed, fmt.Errorf("service write: object is nil")
	}
	if err := ctx.Err(); err != nil {
		return StatusFailed, err
	}
	if !s.evaluateObject(object) {
		return StatusSkipped, nil
	}
	operationCtx, cancel := withOperationTimeout(ctx, s.operationTimeout)
	defer cancel()
	if err := s.hubSpot.Write(operationCtx, object, s.definition); err != nil {
		return StatusFailed, err
	}
	return StatusWritten, nil
}

func (s *Service) evaluateObject(object *domain.Object) bool {
	candidates := make(map[string]domain.AttributeState, len(s.definition.Attributes))
	changed := false

	for _, attribute := range s.definition.Attributes {
		name := strings.TrimSpace(attribute.Name)
		if name == "" {
			continue
		}

		state, exists := object.Attributes[name]
		if !exists {
			continue
		}

		preparedExport := state.Export
		isPreparedExport := state.IsExport && strings.TrimSpace(preparedExport) != ""
		state.Export = ""
		state.IsExport = false
		candidates[name] = state

		if name == "id" || name == "name" || name == s.enrichedDateProperty {
			continue
		}
		if !attribute.Export {
			continue
		}
		if isPreparedExport {
			state.Export = preparedExport
			state.IsExport = true
			changed = true
			candidates[name] = state
			continue
		}

		evaluated := state.Evaluate(s.minimalConfidence, s.highConfidence)
		if evaluated.IsExport &&
			strings.TrimSpace(evaluated.Export) == strings.TrimSpace(state.Import) {
			evaluated.Export = ""
			evaluated.IsExport = false
		}
		if evaluated.IsExport {
			changed = true
		}
		candidates[name] = evaluated
	}

	if changed {
		if enrichedDate, exists := candidates[s.enrichedDateProperty]; exists {
			enrichedDate.Export = time.Now().UTC().Format(time.DateOnly)
			enrichedDate.IsExport = true
			candidates[s.enrichedDateProperty] = enrichedDate
		}
	}

	for _, attribute := range s.definition.Attributes {
		name := strings.TrimSpace(attribute.Name)
		if candidate, exists := candidates[name]; exists {
			object.Attributes[name] = candidate
		}
	}

	return changed
}
