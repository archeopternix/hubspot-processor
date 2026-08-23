package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/archeopternix/hubspot-processor/domain"
)

// Options configures the generic object processing service.
type Options struct {
	Prompt                string
	MinimalConfidence     float64
	HighConfidence        float64
	EnumerationProperties []string
	// OperationTimeout limits each external HubSpot or AI operation. A zero
	// value leaves deadline management to the caller and the underlying client.
	OperationTimeout time.Duration
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
	operationTimeout     time.Duration
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
	if options.OperationTimeout < 0 {
		return nil, fmt.Errorf("service: operation timeout must not be negative")
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

		operationCtx, cancel := withOperationTimeout(ctx, options.OperationTimeout)
		values, err := hubSpot.ReadPropertyOptions(operationCtx, definition.Type, propertyName)
		cancel()
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
		operationTimeout:     options.OperationTimeout,
	}, nil
}

// ReadAll reads all objects described by the service definition from HubSpot.
func (s *Service) ReadAll(ctx context.Context) ([]domain.Object, error) {
	operationCtx, cancel := withOperationTimeout(ctx, s.operationTimeout)
	defer cancel()
	return s.hubSpot.ReadAll(operationCtx, s.definition)
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

func withOperationTimeout(
	ctx context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}
