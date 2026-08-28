package service

import (
	"context"
	"fmt"

	"github.com/archeopternix/hubspot-processor/domain"
)

// EnrichObject AI-enriches one object in place.
func (s *Service) EnrichObject(ctx context.Context, object *domain.Object) (Status, error) {
	if object == nil {
		return StatusFailed, fmt.Errorf("service enrichment: object is nil")
	}
	if err := ctx.Err(); err != nil {
		return StatusFailed, err
	}
	operationCtx, cancel := withOperationTimeout(ctx, s.operationTimeout)
	defer cancel()
	if err := s.enricher.EnrichObject(operationCtx, object, s.definition, s.prompt); err != nil {
		return StatusFailed, err
	}
	return StatusProcessed, nil
}
