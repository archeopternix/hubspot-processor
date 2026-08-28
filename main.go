package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
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
	resultFile           = "result.md"
	enrichedDateProperty = "ai_enriched_date"
	minimalConfidence    = 0.75
	highConfidence       = 0.90
	operationTimeout     = 2 * time.Minute
)

type runMode uint8

const (
	runFirstEligible runMode = iota
	runAllEligible
	runAll
)

type errorMode uint8

const (
	stopOnFirstError errorMode = iota
	skipFailedRecord
)

func main() {
	ctx := context.Background()

	var err error
	switch strings.ToLower(strings.TrimSpace(os.Getenv("HUBSPOT_OBJECT_TYPE"))) {
	// run companies
	case "companies":
		err = runObject(ctx, objectRunConfig{
			Definition:            domain.CompanyDefinition,
			Prompt:                domain.CompanyResearchPrompt,
			EnumerationProperties: []string{"industry"},
			RunMode:               runFirstEligible,
			ErrorMode:             skipFailedRecord,
		})
	// run contacts
	case "", "contacts":
		err = runObject(ctx, objectRunConfig{
			Definition: domain.ContactDefinition,
			Prompt:     domain.ContactResearchPrompt,
			RunMode:    runFirstEligible,
			ErrorMode:  skipFailedRecord,
		})
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
	RunMode               runMode
	ErrorMode             errorMode
}

func runObject(ctx context.Context, config objectRunConfig) error {
	if config.RunMode != runFirstEligible &&
		config.RunMode != runAllEligible &&
		config.RunMode != runAll {
		return fmt.Errorf("unsupported run mode %d", config.RunMode)
	}
	if config.ErrorMode != stopOnFirstError && config.ErrorMode != skipFailedRecord {
		return fmt.Errorf("unsupported error mode %d", config.ErrorMode)
	}

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
	slog.Info("Hubspot read completed", "objects=", len(objects))

	objects = filterObjects(objects, config.RunMode)
	processedObjects := make([]domain.Object, 0, len(objects))
	for i := range objects {
		if err := enrichObject(ctx, processor, &objects[i]); err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return contextErr
			}
			if config.ErrorMode == stopOnFirstError {
				return err
			}
			continue
		}
		processedObjects = append(processedObjects, objects[i])
	}

	var output bytes.Buffer
	processedCount := len(processedObjects)
	if config.RunMode == runFirstEligible && processedCount == 1 {
		if err := processor.PrintOne(&output, &processedObjects[0]); err != nil {
			return err
		}
	} else if err := processor.PrintAll(&output, processedObjects); err != nil {
		return err
	}

	/* for debug purpose only
	if err := os.WriteFile(resultFile, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", resultFile, err)
	}


	*/

	slog.Info("written objects to file", "count", processedCount, "file", resultFile)

	return nil
}

func filterObjects(objects []domain.Object, mode runMode) []domain.Object {
	if mode == runAll {
		return objects
	}

	filtered := make([]domain.Object, 0, len(objects))
	for i := range objects {
		if objects[i].ImportedValue(enrichedDateProperty) != "" {
			continue
		}
		filtered = append(filtered, objects[i])
		if mode == runFirstEligible {
			break
		}
	}
	return filtered
}

func enrichObject(
	ctx context.Context,
	processor *service.Service,
	object *domain.Object,
) error {
	start := time.Now()
	status, err := processor.EnrichObject(ctx, object)
	logEnrichmentResult(object, status, err, time.Since(start))
	if err != nil {
		return fmt.Errorf("enrich object %s: %w", object.ID, err)
	}

	status, err = processor.WriteOne(ctx, object)
	logWriteResult(object, status, err)
	if err != nil {
		return fmt.Errorf("write object %s: %w", object.ID, err)
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

func logEnrichmentResult(
	object *domain.Object,
	status service.Status,
	err error,
	duration time.Duration,
) {
	if err != nil {
		slog.Error(
			"AI enrichment failed",
			"id", object.ID,
			"name", object.Name(),
			"duration", duration,
			"error", err,
		)
		return
	}
	slog.Info(
		"AI enrichment completed",
		"id", object.ID,
		"name", object.Name(),
		"status", status,
		"duration", duration,
	)
}

func logWriteResult(object *domain.Object, status service.Status, err error) {
	if err != nil {
		slog.Error("HubSpot write failed", "id", object.ID, "name", object.Name(), "error", err)
		return
	}
	if status == service.StatusSkipped {
		slog.Info("HubSpot write skipped", "id", object.ID, "name", object.Name())
		return
	}
	slog.Info("HubSpot write completed", "id", object.ID, "name", object.Name())
}
