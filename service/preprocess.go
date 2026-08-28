package service

import (
	"context"

	"github.com/archeopternix/hubspot-processor/domain"
)

type companyPropertyMapping struct {
	company string
	contact string
}

var primaryCompanyMappings = []companyPropertyMapping{
	{company: "name", contact: "company"},
	{company: "address", contact: "address"},
	{company: "city", contact: "city"},
	{company: "zip", contact: "zip"},
	{company: "state", contact: "state"},
}

// Preprocess fills missing contact data from the primary associated company.
// Copied values are retained as AI context and prepared for HubSpot export.
func (s *Service) Preprocess(ctx context.Context, objects []domain.Object) error {
	if s.definition.Type != "contacts" || len(objects) == 0 {
		return nil
	}

	contactIDs := make([]string, 0, len(objects))
	for i := range objects {
		if objects[i].ID != "" && needsPrimaryCompanyData(&objects[i]) {
			contactIDs = append(contactIDs, objects[i].ID)
		}
	}
	if len(contactIDs) == 0 {
		return nil
	}

	operationCtx, cancel := withOperationTimeout(ctx, s.operationTimeout)
	defer cancel()
	companies, err := s.hubSpot.ReadPrimaryCompanies(operationCtx, contactIDs)

	for i := range objects {
		company, exists := companies[objects[i].ID]
		if !exists {
			continue
		}
		for _, mapping := range primaryCompanyMappings {
			if objects[i].ImportedValue(mapping.contact) != "" {
				continue
			}
			value := company.ImportedValue(mapping.company)
			if value == "" {
				continue
			}
			objects[i].SetImport(mapping.contact, value)
			objects[i].SetExport(mapping.contact, value)
		}
	}

	return err
}

func needsPrimaryCompanyData(object *domain.Object) bool {
	for _, mapping := range primaryCompanyMappings {
		if object.ImportedValue(mapping.contact) == "" {
			return true
		}
	}
	return false
}
