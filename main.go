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
	resultFile            = "result.md"
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
	objects, err := hubSpotClient.ReadAll(ctx, domain.CompanyDefinition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	batchStarted := time.Now()
	enriched := 0
	failed := 0

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
			domain.CompanyDefinition,
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

	markdown := renderMarkdown(objects, domain.CompanyDefinition)
	if err := os.WriteFile(resultFile, []byte(markdown), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", resultFile, err)
		os.Exit(1)
	}

	fmt.Printf("written %d objects to %s\n", len(objects), resultFile)
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
		// TODO: Remove this limit when the AI enrichment is working reliably.
		if i > 5 {
			break
		}
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
