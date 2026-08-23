package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/archeopternix/hubspot-processor/ai"
	"github.com/archeopternix/hubspot-processor/domain"
	"github.com/archeopternix/hubspot-processor/hubspot"
	"github.com/archeopternix/hubspot-processor/report"
	"github.com/archeopternix/hubspot-processor/service"
)

const (
	resultFile        = "result.md"
	minimalConfidence = 0.75
	highConfidence    = 0.90
	operationTimeout  = 2 * time.Minute
)

func main() {
	ctx := context.Background()

	var err error
	switch strings.ToLower(strings.TrimSpace(os.Getenv("HUBSPOT_OBJECT_TYPE"))) {
	case "", "companies":
		err = runCompanies(ctx)
	case "contacts":
		err = runContacts(ctx)
	default:
		err = fmt.Errorf(
			"unsupported HUBSPOT_OBJECT_TYPE %q: expected companies or contacts",
			os.Getenv("HUBSPOT_OBJECT_TYPE"),
		)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type objectRunConfig struct {
	Definition            domain.ObjectDefinition
	Prompt                string
	EnumerationProperties []string
}

func runCompanies(ctx context.Context) error {
	return runObject(ctx, objectRunConfig{
		Definition:            domain.CompanyDefinition,
		Prompt:                domain.CompanyResearchPrompt,
		EnumerationProperties: []string{"industry"},
	})
}

func runContacts(ctx context.Context) error {
	return runObject(ctx, objectRunConfig{
		Definition: domain.ContactDefinition,
		Prompt:     domain.ContactResearchPrompt,
	})
}

func runObject(ctx context.Context, config objectRunConfig) error {
	hubSpotClient, err := newHubSpotClient()
	if err != nil {
		return err
	}
	aiClient, err := newAIClient()
	if err != nil {
		return err
	}

	processor, err := service.New(
		ctx,
		hubSpotClient,
		aiClient,
		report.NewMarkdownPrinter(config.Definition),
		config.Definition,
		service.Options{
			Prompt:                config.Prompt,
			MinimalConfidence:     minimalConfidence,
			HighConfidence:        highConfidence,
			EnumerationProperties: config.EnumerationProperties,
			OperationTimeout:      operationTimeout,
		},
	)
	if err != nil {
		return err
	}

	objects, err := processor.ReadAll(ctx)
	if err != nil {
		return err
	}
	log.Printf(
		"Hubspot read completed objects=%d",
		len(objects),
	)

	enrichedObject, enrichResult, err := processor.EnrichFirstEligible(ctx, objects)
	if err != nil {
		return fmt.Errorf("enrich objects: %w", err)
	}
	logEnrichmentResult(enrichResult, objects)

	var writeErr error
	if enrichedObject != nil {
		status, err := processor.WriteOne(ctx, enrichedObject)
		writeErr = err
		logWriteResult(enrichedObject, status, err)
	}

	var output bytes.Buffer
	processedCount := 0
	if enrichedObject != nil {
		processedCount = 1
		if err := processor.PrintOne(&output, enrichedObject); err != nil {
			return err
		}
	} else if err := processor.PrintAll(&output, nil); err != nil {
		return err
	}
	if err := os.WriteFile(resultFile, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", resultFile, err)
	}

	fmt.Printf("written %d objects to %s\n", processedCount, resultFile)

	if writeErr != nil {
		return fmt.Errorf("HubSpot write failed: %w", writeErr)
	}

	return nil
}

func newHubSpotClient() (*hubspot.Client, error) {
	accessToken := strings.TrimSpace(os.Getenv("HUBSPOT_ACCESS_TOKEN"))
	if accessToken == "" {
		return nil, fmt.Errorf("HUBSPOT_ACCESS_TOKEN is not set")
	}

	client := hubspot.NewClient(accessToken)
	if baseURL := strings.TrimSpace(os.Getenv("HUBSPOT_BASE_URL")); baseURL != "" {
		client.WithBaseURL(baseURL)
	}
	return client, nil
}

func newAIClient() (*ai.Client, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	client := ai.NewClient(apiKey)
	if model := strings.TrimSpace(os.Getenv("OPENAI_MODEL")); model != "" {
		client.WithModel(model)
	} else {
		client.WithModel("gpt-5.6-luna")
	}
	if baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")); baseURL != "" {
		client.WithBaseURL(baseURL)
	}
	return client, nil
}

func logEnrichmentResult(result service.BatchResult, objects []domain.Object) {
	objectsByID := make(map[string]*domain.Object, len(objects))
	for i := range objects {
		objectsByID[objects[i].ID] = &objects[i]
	}

	for _, item := range result.Items {
		object := objectsByID[item.ObjectID]
		switch item.Status {
		case service.StatusProcessed:
			log.Printf(
				"AI enrichment completed id=%s name=%q",
				item.ObjectID,
				object.Name(),
			)
		case service.StatusSkipped:
			log.Printf(
				"AI enrichment skipped id=%s name=%q ai_enriched_date=%q",
				item.ObjectID,
				object.Name(),
				object.ImportedValue("ai_enriched_date"),
			)
		case service.StatusFailed:
			log.Printf(
				"AI enrichment failed id=%s name=%q error=%v",
				item.ObjectID,
				object.Name(),
				item.Err,
			)
		}
	}
}

func logWriteResult(object *domain.Object, status service.Status, err error) {
	if err != nil {
		log.Printf("HubSpot write failed id=%s name=%q error=%v", object.ID, object.Name(), err)
		return
	}
	if status == service.StatusSkipped {
		log.Printf("HubSpot write skipped id=%s name=%q", object.ID, object.Name())
		return
	}
	log.Printf("HubSpot write completed id=%s name=%q", object.ID, object.Name())
}
