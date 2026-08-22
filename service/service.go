package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/archeopternix/hubspot-processor/domain"
)

// Options configures the generic object processing service.
type Options struct {
	Prompt                string
	MinimalConfidence     float64
	HighConfidence        float64
	EnumerationProperties []string
}

// Service is the application-facing API for reading, printing, enriching and
// writing CRM objects.
type Service struct {
	hubSpot              HubSpotGateway
	enricher             AIEnricher
	printer              ObjectPrinter
	definition           domain.ObjectDefinition
	prompt               string
	minimalConfidence    float64
	highConfidence       float64
	enrichedDateProperty string
}

// New creates a ready-to-use service and resolves configured HubSpot
// enumeration values once during initialization.
func New(
	ctx context.Context,
	hubSpot HubSpotGateway,
	enricher AIEnricher,
	printer ObjectPrinter,
	definition domain.ObjectDefinition,
	options Options,
) (*Service, error) {
	if hubSpot == nil {
		return nil, fmt.Errorf("service: HubSpot gateway is nil")
	}
	if enricher == nil {
		return nil, fmt.Errorf("service: AI enricher is nil")
	}
	if printer == nil {
		return nil, fmt.Errorf("service: object printer is nil")
	}
	if strings.TrimSpace(definition.Type) == "" {
		return nil, fmt.Errorf("service: object type is empty")
	}
	if options.MinimalConfidence < 0 || options.MinimalConfidence > 1 {
		return nil, fmt.Errorf("service: minimal confidence must be between 0 and 1")
	}
	if options.HighConfidence < 0 || options.HighConfidence > 1 {
		return nil, fmt.Errorf("service: high confidence must be between 0 and 1")
	}

	resolvedDefinition := definition
	seenProperties := make(map[string]struct{}, len(options.EnumerationProperties))
	for _, propertyName := range options.EnumerationProperties {
		propertyName = strings.TrimSpace(propertyName)
		if propertyName == "" {
			continue
		}
		if _, duplicate := seenProperties[propertyName]; duplicate {
			continue
		}
		seenProperties[propertyName] = struct{}{}

		values, err := hubSpot.ReadPropertyOptions(ctx, definition.Type, propertyName)
		if err != nil {
			return nil, fmt.Errorf("service: resolve %s options: %w", propertyName, err)
		}
		resolvedDefinition = resolvedDefinition.WithAllowedValues(propertyName, values)
	}

	return &Service{
		hubSpot:              hubSpot,
		enricher:             enricher,
		printer:              printer,
		definition:           resolvedDefinition,
		prompt:               strings.TrimSpace(options.Prompt),
		minimalConfidence:    options.MinimalConfidence,
		highConfidence:       options.HighConfidence,
		enrichedDateProperty: "ai_enriched_date",
	}, nil
}

// ReadAll reads all objects described by the service definition from HubSpot.
func (s *Service) ReadAll(ctx context.Context) ([]domain.Object, error) {
	return s.hubSpot.ReadAll(ctx, s.definition)
}

// PrintOne renders one object to writer.
func (s *Service) PrintOne(writer io.Writer, object *domain.Object) error {
	if object == nil {
		return fmt.Errorf("service print: object is nil")
	}
	if writer == nil {
		return fmt.Errorf("service print: writer is nil")
	}
	return s.printer.PrintOne(writer, object)
}

// PrintAll renders all supplied objects to writer.
func (s *Service) PrintAll(writer io.Writer, objects []domain.Object) error {
	if writer == nil {
		return fmt.Errorf("service print: writer is nil")
	}
	return s.printer.PrintAll(writer, objects)
}
