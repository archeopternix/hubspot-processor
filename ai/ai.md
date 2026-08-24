# AI package

Package `ai` enriches a `domain.Object` through the OpenAI Responses API. It asks the model to research configured attributes using web search, validates the complete structured response, and then stores proposed values and confidence scores on the object.

Import the package as:

```go
import "github.com/archeopternix/hubspot-processor/ai"
```

## Public API

| Symbol | Purpose |
| --- | --- |
| `DefaultBaseURL` | Default OpenAI API root: `https://api.openai.com/v1`. |
| `DefaultModel` | Default model used by new clients. |
| `Client` | Configured Responses API client. |
| `NewClient` | Creates a client from an OpenAI API key. |
| `(*Client).WithBaseURL` | Overrides the API root. |
| `(*Client).WithModel` | Overrides the model. |
| `(*Client).WithHTTPClient` | Replaces the HTTP client. |
| `(*Client).EnrichObject` | Researches one object and applies validated proposals in place. |

## Create and configure a client

`NewClient` trims the supplied API key and creates a client using `DefaultBaseURL`, `DefaultModel`, and an HTTP client with a 90-second timeout.

```go
client := ai.NewClient(os.Getenv("OPENAI_API_KEY"))
```

Configuration methods are chainable:

```go
client := ai.NewClient(os.Getenv("OPENAI_API_KEY")).
	WithModel("gpt-5.6-luna").
	WithBaseURL("https://api.openai.com/v1")
```

`WithBaseURL`, `WithModel`, and `WithHTTPClient` mutate and return the same client. Call them during application setup, before requests begin. Do not call them concurrently with `EnrichObject`. Blank base URLs and model names are ignored, as is a nil HTTP client.

To control transport behavior or request timeouts, supply a standard `http.Client`:

```go
httpClient := &http.Client{
	Timeout: 2 * time.Minute,
}

client := ai.NewClient(os.Getenv("OPENAI_API_KEY")).
	WithHTTPClient(httpClient)
```

`WithBaseURL` accepts an API root, not the full Responses endpoint. The client appends `/responses` when it sends a request. Surrounding whitespace and trailing slashes are removed during setup.

## Enrich an object

`EnrichObject` accepts a context, an object to update, its definition, and the business-level research prompt:

```go
func (c *Client) EnrichObject(
	ctx context.Context,
	object *domain.Object,
	definition domain.ObjectDefinition,
	prompt string,
) error
```

The method updates the object in place. For attributes whose definitions have `Research: true`, it may replace only `AttributeState.Proposal` and `AttributeState.Score`. Imported values and export state are preserved.

The entire provider result is validated before any proposal is applied. An error therefore leaves the object unchanged. If the definition has no research attributes, the call returns successfully without validating the API key or making a network request. A nil object always returns an error.

### Company enrichment example

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/archeopternix/hubspot-processor/ai"
	"github.com/archeopternix/hubspot-processor/domain"
)

func main() {
	definition := domain.CompanyDefinition
	object := domain.NewObject("12345", definition.Attributes)
	object.SetImport("name", "Example GmbH")
	object.SetImport("domain", "example.com")

	client := ai.NewClient(os.Getenv("OPENAI_API_KEY"))
	if err := client.EnrichObject(
		context.Background(),
		&object,
		definition,
		domain.CompanyResearchPrompt,
	); err != nil {
		panic(err)
	}

	for name, state := range object.Attributes {
		if state.Proposal != "" {
			fmt.Printf("%s=%q confidence=%.2f\n", name, state.Proposal, state.Score)
		}
	}
}
```

The same pattern works with `domain.ContactDefinition` and `domain.ContactResearchPrompt`.

### Custom definition example

Definitions determine which imported values provide context and which attributes the AI must research:

```go
definition := domain.ObjectDefinition{
	Type: "companies",
	Attributes: []domain.AttributeDefinition{
		{
			Name:        "name",
			Description: "Official company name",
			Required:    true,
			Research:    true,
		},
		{
			Name:          "market_segment",
			Description:   "Primary market segment",
			AllowedValues: []string{"Enterprise", "Mid-market", "Small business"},
			Research:      true,
		},
		{
			Name:        "numberofemployees",
			Description: "Most recent employee count",
			ValueType:   domain.AttributeValueInteger,
			Research:    true,
		},
		{
			Name:     "domain",
			Research: false,
		},
	},
}

object := domain.NewObject("12345", definition.Attributes)
object.SetImport("domain", "example.com")

err := client.EnrichObject(
	ctx,
	&object,
	definition,
	"Research the exact company represented by the imported CRM data.",
)
```

All non-empty imported values in the definition are supplied as untrusted identity evidence. Only attributes marked `Research: true` are requested from the model. Every research attribute must still appear exactly once in the structured response; `Required` is also included in the research instructions.

## Validation and mutation rules

Before changing the object, `EnrichObject` requires that:

- every research attribute appears exactly once;
- no unexpected attributes are present;
- every score is between `0` and `1`;
- an empty proposal has score `0`;
- a constrained proposal exactly matches one of `AllowedValues`;
- an `AttributeValueInteger` proposal is a non-negative integer without signs, separators, units, or approximation text.

String and non-negative integer proposals are accepted from the structured provider response. Stored proposals have surrounding whitespace removed. The `hs_quick_context` domain attribute additionally uses CRLF line endings and ends with CRLF.

## Contexts and errors

Pass a request-scoped context to set cancellation or deadline behavior:

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

if err := client.EnrichObject(ctx, &object, definition, prompt); err != nil {
	if errors.Is(err, context.DeadlineExceeded) {
		// Handle the timeout.
	}
	return err
}
```

Errors are returned for invalid arguments or configuration, request construction or transport failures, non-success provider responses, malformed response bodies, missing output text, and structured-result validation failures. Provider errors are currently ordinary wrapped or formatted Go errors; the package does not expose a typed provider error.
