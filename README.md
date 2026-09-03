# HubSpot Processor

HubSpot Processor is a Go application and reusable package set for improving
HubSpot company and contact data with AI-assisted research.

The application reads CRM objects from HubSpot, enriches configured attributes
through the OpenAI Responses API and web search, evaluates proposals against
confidence and value constraints, writes eligible changes back to HubSpot, and
creates a Markdown result report.

The core domain and service packages are independent of HTTP and credentials.
HubSpot, OpenAI, and report generation are adapters behind small interfaces,
allowing the processing workflow to be embedded in another Go application or
tested with replacement implementations.

> [!IMPORTANT]
> The current executable is a synchronous batch processor. The proposed design
> for scheduled execution and invocation from ChatGPT or Langdock is documented
> in [Agent architecture concept](docs/agent-architecture-concept.md).

## Features

- Generic HubSpot CRM object model driven by object definitions.
- Built-in definitions and research prompts for companies and contacts.
- Complete HubSpot pagination and contact-to-primary-company preprocessing.
- AI research with strict structured output and server-side validation.
- Runtime resolution of HubSpot enumeration options.
- Confidence-based export decisions with separate thresholds for missing and
  existing CRM values.
- Guarded HubSpot writes that only send explicitly approved export properties.
- Caller-provided contexts, operation timeouts, and configurable HTTP clients.
- Structured HubSpot API errors.
- Human-readable Markdown reports.

## Processing flow

```text
HubSpot read
    -> optional contact/company preprocessing
    -> OpenAI research
    -> proposal validation
    -> confidence evaluation
    -> guarded HubSpot write
    -> Markdown report
```

The executable processes records whose `ai_enriched_date` property is empty.
When at least one eligible change is written, it also sets
`ai_enriched_date` to the current UTC date. Failures on individual records are
logged and skipped so that the remaining records can continue.

## Package overview

| Package | Public role | Main exported API |
| --- | --- | --- |
| [`domain`](domain/) | Provider-neutral CRM objects, attribute state, validation constraints, and built-in definitions | `Object`, `AttributeState`, `AttributeDefinition`, `ObjectDefinition`, `CompanyDefinition`, `ContactDefinition` |
| [`hubspot`](hubspot/) | HubSpot CRM read, association, property-option, and write adapter | `Client`, `NewClient`, `ReadAll`, `ReadPrimaryCompanies`, `ReadPropertyOptions`, `Write`, `APIError` |
| [`ai`](ai/) | OpenAI Responses API enrichment adapter | `Client`, `NewClient`, `EnrichObject` |
| [`service`](service/) | Application-facing orchestration and confidence-based export policy | `Service`, `New`, `Options`, `ReadAll`, `Preprocess`, `EnrichObject`, `WriteOne`, `PrintOne`, `PrintAll` |
| [`report`](report/) | Markdown rendering adapter | `MarkdownPrinter`, `NewMarkdownPrinter` |

Detailed adapter documentation:

- [AI package reference](ai/ai.md)
- [HubSpot package reference](hubspot/hubspot.md)

## Public service interface

`service.Service` is the main entry point for embedding the processing workflow.
It depends on three public interfaces:

```go
type HubSpotGateway interface {
    ReadAll(context.Context, domain.ObjectDefinition) ([]domain.Object, error)
    ReadPrimaryCompanies(context.Context, []string) (map[string]domain.Object, error)
    ReadPropertyOptions(context.Context, string, string) ([]string, error)
    Write(context.Context, *domain.Object, domain.ObjectDefinition) error
}

type AIEnricher interface {
    EnrichObject(context.Context, *domain.Object, domain.ObjectDefinition, string) error
}

type ObjectPrinter interface {
    PrintOne(io.Writer, *domain.Object) error
    PrintAll(io.Writer, []domain.Object) error
}
```

The included `hubspot.Client`, `ai.Client`, and `report.MarkdownPrinter`
implement these interfaces.

### Construct a service

```go
definition := domain.CompanyDefinition

hubSpotClient := hubspot.NewClient(os.Getenv("HUBSPOT_ACCESS_TOKEN"))
aiClient := ai.NewClient(os.Getenv("OPENAI_API_KEY"))
printer := report.NewMarkdownPrinter(definition)

processor, err := service.New(
    ctx,
    hubSpotClient,
    aiClient,
    printer,
    definition,
    service.Options{
        Prompt:                domain.CompanyResearchPrompt,
        MinimalConfidence:     0.75,
        HighConfidence:        0.90,
        EnumerationProperties: []string{"industry"},
        OperationTimeout:      2 * time.Minute,
    },
)
if err != nil {
    return err
}
```

`service.New` validates its dependencies and options. It also reads configured
HubSpot enumeration options during initialization and adds them to a copy of
the supplied object definition.

### Process objects

```go
objects, err := processor.ReadAll(ctx)
if err != nil {
    return err
}

if err := processor.Preprocess(ctx, objects); err != nil {
    // Preprocessing can return partial association errors. Decide whether the
    // caller should continue with the successfully prepared objects.
    log.Printf("preprocess: %v", err)
}

for i := range objects {
    if _, err := processor.EnrichObject(ctx, &objects[i]); err != nil {
        log.Printf("enrich %s: %v", objects[i].ID, err)
        continue
    }
    if _, err := processor.WriteOne(ctx, &objects[i]); err != nil {
        log.Printf("write %s: %v", objects[i].ID, err)
    }
}

if err := processor.PrintAll(os.Stdout, objects); err != nil {
    return err
}
```

Service operations return one of the following statuses where applicable:

| Status | Meaning |
| --- | --- |
| `service.StatusProcessed` | AI enrichment completed |
| `service.StatusWritten` | Eligible properties were written to HubSpot |
| `service.StatusSkipped` | No property met the export policy |
| `service.StatusFailed` | The operation failed |

## Domain model

CRM data is represented by a generic object whose attribute states preserve the
complete processing lifecycle:

```go
type Object struct {
    ID         string
    Attributes map[string]AttributeState
}

type AttributeState struct {
    Import   string
    Proposal string
    Export   string
    Score    float64
    IsExport bool
}
```

- `Import` is the value read from HubSpot.
- `Proposal` and `Score` are supplied by the AI adapter after validation.
- `Export` is the value selected by the service policy.
- `IsExport` explicitly authorizes the HubSpot adapter to write that property.

Object behavior is configured through `domain.ObjectDefinition`. Each
`AttributeDefinition` can specify whether a property is read, researched, or
exported, along with allowed values and value-type constraints.

```go
definition := domain.ObjectDefinition{
    Type: "companies",
    Attributes: []domain.AttributeDefinition{
        {Name: "name", Required: true},
        {
            Name:          "market_segment",
            Research:      true,
            Export:        true,
            AllowedValues: []string{"Enterprise", "Mid-market", "SMB"},
        },
    },
}
```

## Export policy and write safety

For a proposed value to become an export candidate:

- The property must be configured with `Export: true`.
- The proposal must be non-empty and valid for the attribute definition.
- A missing HubSpot value must meet `MinimalConfidence`.
- An existing HubSpot value must meet `HighConfidence`.
- A proposal equal to the imported value is not exported.

The HubSpot adapter sends only values with `IsExport: true`. It rejects
unconfigured properties, properties not enabled for export, invalid values,
and attempts to export `id` or `name`. Imported and proposed values are never
written directly.

## Requirements

- Go 1.23 or newer.
- A HubSpot private-app access token with the CRM scopes required by the chosen
  object definitions.
- An OpenAI API key with access to the configured model and Responses API.
- A HubSpot property with the internal name `ai_enriched_date` for each
  processed object type.

The built-in company definition also expects its configured HubSpot properties,
including an `industry` enumeration. Review
[`domain/company.go`](domain/company.go) and
[`domain/contact.go`](domain/contact.go) before running the application against
a production portal.

## Run the batch application

Set the required credentials and run the module:

```powershell
$env:HUBSPOT_ACCESS_TOKEN="your-hubspot-token"
$env:OPENAI_API_KEY="your-openai-key"
go run .
```

Companies are processed by default. Select contacts explicitly:

```powershell
$env:HUBSPOT_OBJECT_TYPE="contacts"
go run .
```

The batch command writes its Markdown report to `result.md` in the current
working directory.

## Configuration

| Environment variable | Required | Default | Description |
| --- | --- | --- | --- |
| `HUBSPOT_ACCESS_TOKEN` | Yes | - | HubSpot bearer token |
| `OPENAI_API_KEY` | Yes | - | OpenAI API key |
| `HUBSPOT_OBJECT_TYPE` | No | `companies` | Object workflow: `companies` or `contacts` |
| `HUBSPOT_BASE_URL` | No | `https://api.hubapi.com` | Override the HubSpot API root, primarily for testing |
| `OPENAI_BASE_URL` | No | `https://api.openai.com/v1` | Override the OpenAI API root |
| `OPENAI_MODEL` | No | `gpt-5.6-luna` | Responses API model used for research |

Configuration methods on the public clients mutate and return the same client.
Call them during application setup, before concurrent use.

## Build and verify

```powershell
go build .
go test ./...
```

## Project structure

```text
.
|-- ai/          OpenAI enrichment adapter and documentation
|-- docs/        Architecture and design documents
|-- domain/      Provider-neutral data and workflow definitions
|-- hubspot/     HubSpot adapter and documentation
|-- report/      Markdown report adapter
|-- service/     Application service and ports
|-- main.go      Current batch executable
|-- go.mod
|-- LICENSE
`-- README.md
```

## Security notes

- Supply credentials through environment variables or a deployment secret
  manager; never commit them to the repository.
- Treat HubSpot property values and web content as untrusted input.
- Use caller-provided contexts and appropriate deadlines.
- Review proposed changes and confidence policies before enabling automated
  writes in a production portal.
- Avoid logging complete CRM records, prompts, tokens, or sensitive provider
  responses.

## License

This project is available under the [MIT License](LICENSE).
