package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/archeopternix/hubspot-processor/domain"
)

type researchResult struct {
	Attributes []researchAttribute `json:"attributes"`
}

type researchAttribute struct {
	Name     string        `json:"name"`
	Proposal proposalValue `json:"proposal"`
	Score    float64       `json:"score"`
}

type proposalValue string

func (p *proposalValue) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("proposal is empty JSON")
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*p = proposalValue(value)
		return nil
	}

	for _, character := range data {
		if character < '0' || character > '9' {
			return fmt.Errorf("numeric proposal %q is not a non-negative integer", string(data))
		}
	}
	*p = proposalValue(string(data))
	return nil
}

type responsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Tools []any  `json:"tools,omitempty"`
	Text  any    `json:"text,omitempty"`
}

type responsesResponse struct {
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// EnrichObject researches exactly one object and updates only Proposal and
// Score on Research=true attributes. Import, Export and IsExport are preserved.
// The object is not mutated unless the complete provider result is valid. A
// definition without research attributes is a no-op and does not require an
// API key.
func (c *Client) EnrichObject(
	ctx context.Context,
	object *domain.Object,
	definition domain.ObjectDefinition,
	prompt string,
) error {
	if object == nil {
		return fmt.Errorf("ai: object is nil")
	}
	researchAttributes := definition.ResearchAttributes()
	if len(researchAttributes) == 0 {
		return nil
	}
	if c.apiKey == "" {
		return fmt.Errorf("ai: OpenAI API key is empty")
	}

	fullPrompt := buildPrompt(prompt, *object, definition, researchAttributes)
	result, err := c.research(ctx, fullPrompt, researchAttributes)
	if err != nil {
		return err
	}
	if err := validateResearchResult(result, researchAttributes); err != nil {
		return err
	}

	applyResearchResult(object, researchAttributes, result)
	return nil
}

func applyResearchResult(
	object *domain.Object,
	researchAttributes []domain.AttributeDefinition,
	result researchResult,
) {
	definitions := make(
		map[string]domain.AttributeDefinition,
		len(researchAttributes),
	)
	for _, attribute := range researchAttributes {
		definitions[attribute.Name] = attribute
	}

	for _, researched := range result.Attributes {
		attribute := definitions[researched.Name]
		proposal := attribute.NormalizeProposal(string(researched.Proposal))
		object.SetProposal(
			researched.Name,
			proposal,
			researched.Score,
		)
	}
}

func (c *Client) research(
	ctx context.Context,
	prompt string,
	researchAttributes []domain.AttributeDefinition,
) (researchResult, error) {
	requestBody := responsesRequest{
		Model: c.model,
		Input: prompt,
		Tools: []any{
			map[string]any{"type": "web_search"},
		},
		Text: map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "object_research",
				"strict": true,
				"schema": buildResearchSchema(researchAttributes),
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return researchResult{}, fmt.Errorf("ai: encode OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint+"/responses",
		bytes.NewReader(body),
	)
	if err != nil {
		return researchResult{}, fmt.Errorf("ai: create OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return researchResult{}, fmt.Errorf("ai: OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return researchResult{}, fmt.Errorf(
			"ai: OpenAI returned %s: %s",
			resp.Status,
			strings.TrimSpace(string(raw)),
		)
	}

	var response responsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return researchResult{}, fmt.Errorf("ai: decode OpenAI response: %w", err)
	}
	if response.Error != nil {
		return researchResult{}, fmt.Errorf("ai: OpenAI error: %s", response.Error.Message)
	}

	text := responseOutputText(response)
	if strings.TrimSpace(text) == "" {
		return researchResult{}, fmt.Errorf("ai: OpenAI response contained no output text")
	}

	var result researchResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return researchResult{}, fmt.Errorf("ai: decode structured research result: %w", err)
	}

	return result, nil
}

func responseOutputText(response responsesResponse) string {
	for _, item := range response.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}
	return ""
}

func validateResearchResult(
	result researchResult,
	researchAttributes []domain.AttributeDefinition,
) error {
	required := make(map[string]domain.AttributeDefinition)
	for _, attribute := range researchAttributes {
		required[attribute.Name] = attribute
	}

	seen := make(map[string]struct{}, len(result.Attributes))
	for _, attribute := range result.Attributes {
		attributeDefinition, ok := required[attribute.Name]
		if !ok {
			return fmt.Errorf("ai: unexpected research attribute %q", attribute.Name)
		}
		if _, duplicate := seen[attribute.Name]; duplicate {
			return fmt.Errorf("ai: duplicate research attribute %q", attribute.Name)
		}
		if attribute.Score < 0 || attribute.Score > 1 {
			return fmt.Errorf("ai: score for %q must be between 0 and 1", attribute.Name)
		}

		proposal := strings.TrimSpace(string(attribute.Proposal))
		if proposal == "" && attribute.Score != 0 {
			return fmt.Errorf("ai: empty proposal for %q must have score 0", attribute.Name)
		}
		if proposal != "" && !attributeDefinition.AcceptsValue(proposal) {
			return fmt.Errorf(
				"ai: proposal %q is not valid for %q",
				proposal,
				attribute.Name,
			)
		}
		seen[attribute.Name] = struct{}{}
	}

	for name := range required {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("ai: missing research attribute %q", name)
		}
	}
	return nil
}
