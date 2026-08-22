package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/archeopternix/hubspot-processor/ai"
	"github.com/archeopternix/hubspot-processor/domain"
	"github.com/archeopternix/hubspot-processor/hubspot"
)

const (
	resultFile        = "result.md"
	minimalConfidence = 0.75
	highConfidence    = 0.90

	companyResearchPrompt = `Research the company represented by the supplied HubSpot CRM data.

Identify the exact legal/business entity before extracting facts.
Use current and reliable web sources.
Prefer the official company website, annual reports, official registries and other primary sources.
Do not invent information.`
)

func main() {
	ctx := context.Background()

	// Initialization of external services and clients is done here. The actual business logic is in the main loop below.
	// HubSpot
	hubSpotToken := strings.TrimSpace(os.Getenv("HUBSPOT_ACCESS_TOKEN"))
	if hubSpotToken == "" {
		fmt.Fprintln(os.Stderr, "HUBSPOT_ACCESS_TOKEN is not set")
		os.Exit(1)
	}

	hubSpotClient := hubspot.NewClient(hubSpotToken)
	if baseURL := strings.TrimSpace(os.Getenv("HUBSPOT_BASE_URL")); baseURL != "" {
		hubSpotClient.WithBaseURL(baseURL)
	}

	// OpenAI
	openAIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if openAIKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	aiClient := ai.NewClient(openAIKey)
	if model := strings.TrimSpace(os.Getenv("OPENAI_MODEL")); model != "" {
		aiClient.WithModel(model)
	} else {
		aiClient.WithModel("gpt-5.6-luna")
	}
	if baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")); baseURL != "" {
		aiClient.WithBaseURL(baseURL)
	}

	// business logic
	definition := domain.CompanyDefinition
	industryOptions, err := hubSpotClient.ReadPropertyOptions(
		ctx,
		definition.Type,
		"industry",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	definition = definition.WithAllowedValues("industry", industryOptions)

	objects, err := hubSpotClient.ReadAll(ctx, definition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	batchStarted := time.Now()
	enriched := 0
	failed := 0
	processed := make([]int, 0, 1)

	for i := range objects {
		object := &objects[i]
		started := time.Now()

		log.Printf(
			"AI enrichment started id=%s name=%q",
			object.ID,
			objectName(object),
		)

		err := aiClient.EnrichObject(
			ctx,
			object,
			definition,
			companyResearchPrompt,
		)
		elapsed := time.Since(started)

		if err != nil {
			failed++
			log.Printf(
				"AI enrichment failed id=%s name=%q duration=%s error=%v",
				object.ID,
				objectName(object),
				elapsed,
				err,
			)
			continue
		}

		enriched++
		processed = append(processed, i)
		log.Printf(
			"AI enrichment completed id=%s name=%q duration=%s",
			object.ID,
			objectName(object),
			elapsed,
		)
		// TODO: Remove this limit when the AI enrichment is working reliably.
		break
	}

	log.Printf(
		"AI enrichment finished objects=%d enriched=%d failed=%d duration=%s",
		len(objects),
		enriched,
		failed,
		time.Since(batchStarted),
	)

	updated := 0
	updatedIndexes := make([]int, 0, len(processed))
	for _, index := range processed {
		if evaluateObject(
			&objects[index],
			definition,
			minimalConfidence,
			highConfidence,
		) {
			updated++
			updatedIndexes = append(updatedIndexes, index)
		}
	}
	log.Printf(
		"evaluation finished processed=%d updated=%d",
		len(processed),
		updated,
	)

	written := 0
	writeFailed := 0
	for _, index := range updatedIndexes {
		object := &objects[index]
		log.Printf(
			"HubSpot write started id=%s name=%q",
			object.ID,
			objectName(object),
		)

		if err := hubSpotClient.Write(ctx, object, definition); err != nil {
			writeFailed++
			log.Printf(
				"HubSpot write failed id=%s name=%q error=%v",
				object.ID,
				objectName(object),
				err,
			)
			continue
		}

		written++
		log.Printf(
			"HubSpot write completed id=%s name=%q",
			object.ID,
			objectName(object),
		)
	}
	log.Printf(
		"HubSpot writes finished eligible=%d written=%d failed=%d",
		len(updatedIndexes),
		written,
		writeFailed,
	)

	processedObjects := make([]domain.Object, 0, len(processed))
	for _, index := range processed {
		processedObjects = append(processedObjects, objects[index])
	}

	markdown := renderMarkdown(processedObjects, definition)
	if err := os.WriteFile(resultFile, []byte(markdown), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", resultFile, err)
		os.Exit(1)
	}

	fmt.Printf("written %d objects to %s\n", len(processedObjects), resultFile)
	if writeFailed > 0 {
		fmt.Fprintf(os.Stderr, "%d HubSpot write(s) failed\n", writeFailed)
		os.Exit(1)
	}
}

func evaluateObject(
	object *domain.Object,
	definition domain.ObjectDefinition,
	minimalConfidence float64,
	highConfidence float64,
) bool {
	if object == nil {
		return false
	}

	candidates := make(map[string]domain.AttributeState, len(definition.Attributes))
	changed := false

	for _, attribute := range definition.Attributes {
		name := strings.TrimSpace(attribute.Name)
		if name == "" {
			continue
		}

		state, exists := object.Attributes[name]
		if !exists {
			continue
		}

		state.Export = ""
		state.IsExport = false
		candidates[name] = state

		if name == "id" || name == "name" || name == "ai_enriched_date" {
			continue
		}
		if !attribute.Export {
			continue
		}

		evaluated := state.Evaluate(minimalConfidence, highConfidence)
		if evaluated.IsExport &&
			strings.TrimSpace(evaluated.Export) == strings.TrimSpace(state.Import) {
			evaluated.Export = ""
			evaluated.IsExport = false
		}
		if evaluated.IsExport {
			changed = true
		}
		candidates[name] = evaluated
	}

	if changed {
		if enrichedDate, exists := candidates["ai_enriched_date"]; exists {
			enrichedDate.Export = time.Now().UTC().Format(time.DateOnly)
			enrichedDate.IsExport = true
			candidates["ai_enriched_date"] = enrichedDate
		}
	}

	for _, attribute := range definition.Attributes {
		name := strings.TrimSpace(attribute.Name)
		if candidate, exists := candidates[name]; exists {
			object.Attributes[name] = candidate
		}
	}

	return changed
}

func objectName(object *domain.Object) string {
	if object == nil {
		return ""
	}
	attribute, ok := object.Attributes["name"]
	if !ok {
		return ""
	}
	return strings.TrimSpace(attribute.Import)
}

func renderMarkdown(objects []domain.Object, definition domain.ObjectDefinition) string {
	var b strings.Builder
	b.WriteString("# HubSpot Company Result\n\n")

	for i, object := range objects {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}

		name := objectName(&object)
		if name == "" {
			name = "Unnamed Company"
		}
		fmt.Fprintf(&b, "## %s\n\n", escapeMarkdownText(name))
		fmt.Fprintf(&b, "**ID:** %s\n\n", escapeMarkdownText(object.ID))

		for _, attribute := range definition.Attributes {
			if attribute.Export {
				continue
			}
			state := object.Attributes[attribute.Name]
			fmt.Fprintf(
				&b,
				"**%s:** %s\n\n",
				escapeMarkdownText(attribute.Name),
				escapeMarkdownText(truncate(state.Import, 25)),
			)
		}

		b.WriteString("| Attribute | Import | Proposal | Score | Export |\n")
		b.WriteString("|---|---|---|---|---|\n")
		for _, attribute := range definition.Attributes {
			if !attribute.Export {
				continue
			}
			state := object.Attributes[attribute.Name]
			fmt.Fprintf(
				&b,
				"| %s | %s | %s | %s | %s |\n",
				escapeMarkdownCell(attribute.Name),
				escapeMarkdownCell(truncate(state.Import, 25)),
				escapeMarkdownCell(truncate(state.Proposal, 25)),
				renderScore(state.Score, attribute.Research),
				escapeMarkdownCell(truncate(state.Export, 25)),
			)
		}

		fmt.Fprintf(&b, "**HubSpot Context:** \n%s\n", escapeMarkdownText(object.Attributes["hs_quick_context"].Proposal))

	}

	return b.String()
}

func renderScore(score float64, researched bool) string {
	if !researched {
		return ""
	}
	return fmt.Sprintf("%.2f", score)
}

func truncate(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if maxChars <= 0 || value == "" {
		return value
	}
	if utf8.RuneCountInString(value) <= maxChars {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxChars]) + "..."
}

func escapeMarkdownText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r\n", "<br>")
	value = strings.ReplaceAll(value, "\r", "<br>")
	value = strings.ReplaceAll(value, "\n", "<br>")
	return strings.TrimSpace(value)
}
