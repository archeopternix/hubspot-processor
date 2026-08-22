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
	Name     string  `json:"name"`
	Proposal string  `json:"proposal"`
	Score    float64 `json:"score"`
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
func (c *Client) EnrichObject(
	ctx context.Context,
	object *domain.Object,
	definition domain.ObjectDefinition,
	prompt string,
) error {
	if object == nil {
		return fmt.Errorf("ai: object is nil")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return fmt.Errorf("ai: OpenAI API key is empty")
	}
	if len(definition.ResearchAttributes()) == 0 {
		return nil
	}

	fullPrompt := buildPrompt(prompt, *object, definition)
	result, err := c.research(ctx, fullPrompt, definition)
	if err != nil {
		return err
	}
	if err := validateResearchResult(result, definition); err != nil {
		return err
	}

	// Apply only after the complete result passed validation.
	for _, researched := range result.Attributes {
		proposal := strings.TrimSpace(researched.Proposal)
		if researched.Name == "hs_quick_context" {
			proposal = formatQuickContextLineEndings(proposal)
		}
		object.SetProposal(
			researched.Name,
			proposal,
			researched.Score,
		)
	}

	return nil
}

func formatQuickContextLineEndings(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ReplaceAll(value, "\n", "\r\n") + "\r\n"
}

func (c *Client) research(
	ctx context.Context,
	prompt string,
	definition domain.ObjectDefinition,
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
				"schema": buildResearchSchema(definition),
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
		strings.TrimRight(c.baseURL, "/")+"/responses",
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

func validateResearchResult(result researchResult, definition domain.ObjectDefinition) error {
	required := make(map[string]domain.AttributeDefinition)
	for _, attribute := range definition.ResearchAttributes() {
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

		proposal := strings.TrimSpace(attribute.Proposal)
		if proposal == "" && attribute.Score != 0 {
			return fmt.Errorf("ai: empty proposal for %q must have score 0", attribute.Name)
		}
		if proposal != "" && !attributeDefinition.AcceptsValue(proposal) {
			return fmt.Errorf(
				"ai: proposal %q is not an allowed value for %q",
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
