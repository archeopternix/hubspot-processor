package service

import (
	"context"
	"io"

	"github.com/archeopternix/hubspot-processor/domain"
)

// HubSpotGateway describes the HubSpot operations required by the service.
type HubSpotGateway interface {
	ReadAll(context.Context, domain.ObjectDefinition) ([]domain.Object, error)
	ReadPrimaryCompanies(context.Context, []string) (map[string]domain.Object, error)
	ReadPropertyOptions(context.Context, string, string) ([]string, error)
	Write(context.Context, *domain.Object, domain.ObjectDefinition) error
}

// AIEnricher describes the AI operation required by the service.
type AIEnricher interface {
	EnrichObject(context.Context, *domain.Object, domain.ObjectDefinition, string) error
}

// ObjectPrinter renders one or more domain objects.
type ObjectPrinter interface {
	PrintOne(io.Writer, *domain.Object) error
	PrintAll(io.Writer, []domain.Object) error
}
