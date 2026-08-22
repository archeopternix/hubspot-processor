package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"hubspot-object-reader/domain"
)

type crmObject struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
	CreatedAt  string            `json:"createdAt"`
	UpdatedAt  string            `json:"updatedAt"`
	Archived   bool              `json:"archived"`
}

type objectPage struct {
	Results []crmObject `json:"results"`
	Paging  *struct {
		Next *struct {
			After string `json:"after"`
			Link  string `json:"link"`
		} `json:"next"`
	} `json:"paging"`
}

// ReadAll reads all records of any HubSpot CRM object type described by the
// supplied definition and maps them into generic domain.Object values.
func (c *Client) ReadAll(ctx context.Context, definition domain.ObjectDefinition) ([]domain.Object, error) {
	if strings.TrimSpace(definition.Type) == "" {
		return nil, fmt.Errorf("HubSpot object type is empty")
	}
	if c.accessToken == "" {
		return nil, fmt.Errorf("HubSpot access token is empty")
	}

	properties := definition.PropertyNames()
	var (
		after   string
		objects []domain.Object
	)

	for {
		page, err := c.readPage(ctx, definition.Type, properties, after)
		if err != nil {
			return nil, err
		}

		for _, record := range page.Results {
			objects = append(objects, toDomainObject(record, definition.Attributes))
		}

		if page.Paging == nil ||
			page.Paging.Next == nil ||
			strings.TrimSpace(page.Paging.Next.After) == "" {
			break
		}

		after = page.Paging.Next.After
	}

	return objects, nil
}

func (c *Client) readPage(
	ctx context.Context,
	objectType string,
	properties []string,
	after string,
) (objectPage, error) {
	endpoint, err := url.Parse(
		strings.TrimRight(c.baseURL, "/") + "/crm/v3/objects/" + url.PathEscape(objectType),
	)
	if err != nil {
		return objectPage{}, fmt.Errorf("parse HubSpot URL: %w", err)
	}

	query := endpoint.Query()
	query.Set("limit", fmt.Sprintf("%d", c.pageSize))
	if len(properties) > 0 {
		query.Set("properties", strings.Join(properties, ","))
	}
	if after != "" {
		query.Set("after", after)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return objectPage{}, fmt.Errorf("create HubSpot request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return objectPage{}, fmt.Errorf("HubSpot %s request failed: %w", objectType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return objectPage{}, fmt.Errorf(
			"HubSpot returned %s for %s: %s",
			resp.Status,
			objectType,
			strings.TrimSpace(string(body)),
		)
	}

	var page objectPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return objectPage{}, fmt.Errorf("decode HubSpot %s response: %w", objectType, err)
	}

	return page, nil
}

func toDomainObject(record crmObject, definitions []domain.AttributeDefinition) domain.Object {
	object := domain.NewObject(record.ID, definitions)

	for _, definition := range definitions {
		name := strings.TrimSpace(definition.Name)
		if name == "" {
			continue
		}
		object.SetImport(name, record.Properties[name])
	}

	return object
}
