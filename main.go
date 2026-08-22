package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"hubspot-object-reader/domain"
	"hubspot-object-reader/hubspot"
)

func main() {
	ctx := context.Background()

	token := strings.TrimSpace(os.Getenv("HUBSPOT_ACCESS_TOKEN"))
	if token == "" {
		fmt.Fprintln(os.Stderr, "HUBSPOT_ACCESS_TOKEN is not set")
		os.Exit(1)
	}

	client := hubspot.NewClient(token)
	if baseURL := strings.TrimSpace(os.Getenv("HUBSPOT_BASE_URL")); baseURL != "" {
		client.WithBaseURL(baseURL)
	}

	objects, err := client.ReadAll(ctx, domain.CompanyDefinition)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(objects); err != nil {
		fmt.Fprintf(os.Stderr, "encode JSON: %v\n", err)
		os.Exit(1)
	}
}
