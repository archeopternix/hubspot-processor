package service

import (
	"context"
	"fmt"

	"github.com/archeopternix/hubspot-processor/domain"
)

// EnrichOne AI-enriches one object in place. Objects with an imported
// ai_enriched_date are skipped without making an AI request.
func (s *Service) EnrichOne(ctx context.Context, object *domain.Object) (Status, error) {
	if object == nil {
		return StatusFailed, fmt.Errorf("service enrichment: object is nil")
	}
	if err := ctx.Err(); err != nil {
		return StatusFailed, err
	}
	if s.isAlreadyEnriched(object) {
		return StatusSkipped, nil
	}
	operationCtx, cancel := withOperationTimeout(ctx, s.operationTimeout)
	defer cancel()
	if err := s.enricher.EnrichObject(operationCtx, object, s.definition, s.prompt); err != nil {
		return StatusFailed, err
	}
	return StatusProcessed, nil
}

// EnrichAll AI-enriches every supplied object in place. Object-level failures
// are collected and do not prevent later records from being processed.
func (s *Service) EnrichAll(
	ctx context.Context,
	objects []domain.Object,
) (BatchResult, error) {
	result, _, err := s.enrichObjects(ctx, objects, false)
	return result, err
}

// EnrichFirstEligible skips previously enriched objects and continues after
// object-level failures until the first object is enriched successfully.
func (s *Service) EnrichFirstEligible(
	ctx context.Context,
	objects []domain.Object,
) (*domain.Object, BatchResult, error) {
	result, object, err := s.enrichObjects(ctx, objects, true)
	return object, result, err
}

func (s *Service) enrichObjects(
	ctx context.Context,
	objects []domain.Object,
	stopAfterFirstSuccess bool,
) (BatchResult, *domain.Object, error) {
	result := newBatchResult(len(objects))
	for i := range objects {
		if err := ctx.Err(); err != nil {
			return result, nil, err
		}

		status, err := s.EnrichOne(ctx, &objects[i])
		result.add(objects[i].ID, status, err)
		if err != nil && ctx.Err() != nil {
			return result, nil, ctx.Err()
		}
		if status == StatusProcessed && stopAfterFirstSuccess {
			return result, &objects[i], nil
		}
	}
	return result, nil, nil
}

func (s *Service) isAlreadyEnriched(object *domain.Object) bool {
	return object.ImportedValue(s.enrichedDateProperty) != ""
}
