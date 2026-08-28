package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/archeopternix/hubspot-processor/domain"
)

const (
	primaryCompanyAssociationTypeID = 1
	associationBatchSize            = 1000
	objectBatchSize                 = 100
)

type batchInput struct {
	ID    string `json:"id"`
	After string `json:"after,omitempty"`
}

type associationBatchRequest struct {
	Inputs []batchInput `json:"inputs"`
}

type associationType struct {
	TypeID int `json:"typeId"`
}

type objectID string

func (id *objectID) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*id = ""
		return nil
	}
	if strings.HasPrefix(raw, "\"") {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("decode object ID: %w", err)
		}
		*id = objectID(strings.TrimSpace(value))
		return nil
	}
	for _, character := range raw {
		if character < '0' || character > '9' {
			return fmt.Errorf("decode object ID: invalid value %q", raw)
		}
	}
	*id = objectID(raw)
	return nil
}

type associatedObject struct {
	AssociationTypes []associationType `json:"associationTypes"`
	ToObjectID       objectID          `json:"toObjectId"`
}

type associationBatchResult struct {
	From struct {
		ID objectID `json:"id"`
	} `json:"from"`
	To     []associatedObject `json:"to"`
	Paging *struct {
		Next *struct {
			After string `json:"after"`
		} `json:"next"`
	} `json:"paging"`
}

type batchResponseError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type associationBatchResponse struct {
	Results []associationBatchResult `json:"results"`
	Errors  []batchResponseError     `json:"errors"`
}

type objectBatchRequest struct {
	Inputs     []batchInput `json:"inputs"`
	Properties []string     `json:"properties"`
}

type objectBatchResponse struct {
	Results []crmObject          `json:"results"`
	Errors  []batchResponseError `json:"errors"`
}

// ReadPrimaryCompanies returns the primary associated company for each
// supplied contact ID. Successful partial results are returned with any error.
func (c *Client) ReadPrimaryCompanies(
	ctx context.Context,
	contactIDs []string,
) (map[string]domain.Object, error) {
	contactToCompany, associationErr := c.readPrimaryCompanyIDs(ctx, contactIDs)

	companyIDs := make([]string, 0, len(contactToCompany))
	seenCompanies := make(map[string]struct{}, len(contactToCompany))
	for _, companyID := range contactToCompany {
		if _, exists := seenCompanies[companyID]; exists {
			continue
		}
		seenCompanies[companyID] = struct{}{}
		companyIDs = append(companyIDs, companyID)
	}

	companies, companyErr := c.readCompaniesByID(ctx, companyIDs)
	result := make(map[string]domain.Object, len(contactToCompany))
	for contactID, companyID := range contactToCompany {
		if company, exists := companies[companyID]; exists {
			result[contactID] = company
		}
	}

	return result, errors.Join(associationErr, companyErr)
}

func (c *Client) readPrimaryCompanyIDs(
	ctx context.Context,
	contactIDs []string,
) (map[string]string, error) {
	primaryCompanies := make(map[string]string)
	pending := make([]batchInput, 0, len(contactIDs))
	seenContacts := make(map[string]struct{}, len(contactIDs))
	for _, contactID := range contactIDs {
		contactID = strings.TrimSpace(contactID)
		if contactID == "" {
			continue
		}
		if _, duplicate := seenContacts[contactID]; duplicate {
			continue
		}
		seenContacts[contactID] = struct{}{}
		pending = append(pending, batchInput{ID: contactID})
	}

	seenPages := make(map[string]struct{})
	var batchErrors []error
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			batchErrors = append(batchErrors, err)
			break
		}

		batchLength := min(len(pending), associationBatchSize)
		inputs := pending[:batchLength]
		pending = pending[batchLength:]
		var response associationBatchResponse
		if err := c.postJSON(
			ctx,
			"read primary company associations",
			"/crm/associations/2026-03/contacts/companies/batch/read",
			associationBatchRequest{Inputs: inputs},
			&response,
		); err != nil {
			batchErrors = append(batchErrors, err)
			continue
		}
		batchErrors = append(batchErrors, responseErrors("read primary company associations", response.Errors)...)

		for _, item := range response.Results {
			contactID := strings.TrimSpace(string(item.From.ID))
			if contactID == "" {
				batchErrors = append(batchErrors, fmt.Errorf(
					"read primary company associations: response contains an empty contact ID",
				))
				continue
			}
			for _, associated := range item.To {
				companyID := strings.TrimSpace(string(associated.ToObjectID))
				if companyID != "" && hasAssociationType(associated.AssociationTypes, primaryCompanyAssociationTypeID) {
					primaryCompanies[contactID] = companyID
					break
				}
			}
			if primaryCompanies[contactID] != "" ||
				item.Paging == nil ||
				item.Paging.Next == nil ||
				strings.TrimSpace(item.Paging.Next.After) == "" {
				continue
			}

			after := strings.TrimSpace(item.Paging.Next.After)
			pageKey := contactID + "\x00" + after
			if _, duplicate := seenPages[pageKey]; duplicate {
				batchErrors = append(batchErrors, fmt.Errorf(
					"read primary company associations: repeated page for contact %s",
					contactID,
				))
				continue
			}
			seenPages[pageKey] = struct{}{}
			pending = append(pending, batchInput{ID: contactID, After: after})
		}
	}

	return primaryCompanies, errors.Join(batchErrors...)
}

func (c *Client) readCompaniesByID(
	ctx context.Context,
	companyIDs []string,
) (map[string]domain.Object, error) {
	companies := make(map[string]domain.Object, len(companyIDs))
	var batchErrors []error
	for start := 0; start < len(companyIDs); start += objectBatchSize {
		if err := ctx.Err(); err != nil {
			batchErrors = append(batchErrors, err)
			break
		}

		end := min(start+objectBatchSize, len(companyIDs))
		inputs := make([]batchInput, 0, end-start)
		for _, companyID := range companyIDs[start:end] {
			inputs = append(inputs, batchInput{ID: companyID})
		}

		var response objectBatchResponse
		if err := c.postJSON(
			ctx,
			"read primary companies",
			"/crm/objects/2026-03/companies/batch/read",
			objectBatchRequest{
				Inputs:     inputs,
				Properties: []string{"name", "address", "city", "zip", "state"},
			},
			&response,
		); err != nil {
			batchErrors = append(batchErrors, err)
			continue
		}
		batchErrors = append(batchErrors, responseErrors("read primary companies", response.Errors)...)
		for _, record := range response.Results {
			companies[record.ID] = toDomainObject(record, domain.CompanyDefinition.Attributes)
		}
	}

	return companies, errors.Join(batchErrors...)
}

func (c *Client) postJSON(
	ctx context.Context,
	operation string,
	path string,
	requestBody any,
	responseBody any,
) error {
	raw, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("HubSpot %s: encode request: %w", operation, err)
	}
	return c.do(
		ctx,
		operation,
		http.MethodPost,
		path,
		nil,
		bytes.NewReader(raw),
		responseBody,
	)
}

func hasAssociationType(types []associationType, typeID int) bool {
	for _, association := range types {
		if association.TypeID == typeID {
			return true
		}
	}
	return false
}

func responseErrors(operation string, responseErrors []batchResponseError) []error {
	result := make([]error, 0, len(responseErrors))
	for _, responseError := range responseErrors {
		result = append(result, fmt.Errorf(
			"HubSpot %s for object %s: %s",
			operation,
			strings.TrimSpace(responseError.ID),
			strings.TrimSpace(responseError.Message),
		))
	}
	return result
}
