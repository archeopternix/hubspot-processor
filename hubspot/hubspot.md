# HubSpot package

The `hubspot` package is a small client for reading and updating HubSpot CRM
objects. It supports caller-provided contexts, pagination, configurable HTTP
clients, structured API errors, and generic object definitions from the
`domain` package.

## API overview

| API | Purpose | Documentation |
| --- | --- | --- |
| `DefaultBaseURL` | Default HubSpot API root URL | [Create a client](#create-a-client) |
| `Client`, `NewClient` | Construct a HubSpot CRM client | [Create a client](#create-a-client) |
| `WithBaseURL` | Override the API root URL | [Create a client](#create-a-client) |
| `WithHTTPClient` | Supply a custom `http.Client` | [Create a client](#create-a-client) |
| `WithPageSize` | Set the CRM read page size | [Create a client](#create-a-client) |
| `ReadAll` | Read every page of one CRM object type | [Read CRM objects](#read-crm-objects) |
| `ReadPropertyOptions` | Read active enumeration values | [Read enumeration options](#read-enumeration-options) |
| `Write` | Update selected properties on one object | [Write an object](#write-an-object) |
| `APIError`, `ErrorCategory` | Inspect non-2xx HubSpot responses | [Structured errors](#structured-errors) |
| `service.Options.OperationTimeout` | Bound each external service operation | [Service operation timeout](#service-operation-timeout) |

## Create a client

`NewClient` trims and stores the bearer token, uses `DefaultBaseURL`, requests
100 records per page, and creates an HTTP client with a 30-second transport
timeout.

```go
client := hubspot.NewClient(os.Getenv("HUBSPOT_ACCESS_TOKEN"))
```

The fluent options mutate and return the same client:

```go
client := hubspot.NewClient(token).
	WithBaseURL("https://api.hubapi.com").
	WithPageSize(50).
	WithHTTPClient(&http.Client{Timeout: 45 * time.Second})
```

`WithBaseURL` ignores empty values. `WithHTTPClient` ignores `nil`.
`WithPageSize` accepts values from 1 through 100 and ignores values outside
that range.

## Read CRM objects

`ReadAll` reads every page for the object type and properties in a
`domain.ObjectDefinition`.

```go
definition := domain.ObjectDefinition{
	Type: "contacts",
	Attributes: []domain.AttributeDefinition{
		{Name: "firstname"},
		{Name: "lastname"},
		{Name: "email"},
	},
}

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

objects, err := client.ReadAll(ctx, definition)
if err != nil {
	return err
}
for i := range objects {
	fmt.Printf("%s: %s\n", objects[i].ID, objects[i].Name())
}
```

Each returned `domain.Object` contains one `AttributeState` per configured
attribute. Values returned by HubSpot are stored in `AttributeState.Import`.

## Read enumeration options

`ReadPropertyOptions` returns the unique, non-empty values of visible options
for an enumeration property. It returns an error if the property is not an
enumeration or has no active options.

```go
industries, err := client.ReadPropertyOptions(ctx, "companies", "industry")
if err != nil {
	return err
}

definition := domain.CompanyDefinition.WithAllowedValues("industry", industries)
```

## Write an object

`Write` sends only attributes whose state has `IsExport` set to `true`. The
attribute must exist in the supplied definition, be enabled for export, and
accept the value in `AttributeState.Export`. Imported and proposed values are
never written directly.

```go
definition := domain.ObjectDefinition{
	Type: "companies",
	Attributes: []domain.AttributeDefinition{
		{Name: "industry", Export: true},
	},
}

object := domain.NewObject("12345", definition.Attributes)
state := object.Attributes["industry"]
state.Export = "SOFTWARE"
state.IsExport = true
object.Attributes["industry"] = state

if err := client.Write(ctx, &object, definition); err != nil {
	return err
}
```

The `id` and `name` properties are explicitly protected from export.

## Structured errors

Non-2xx HubSpot responses are returned as `*hubspot.APIError`. Use `errors.As`
to read the normalized category, operation, status code, and bounded response
body. Caller-input validation and request-construction errors remain ordinary
Go errors. Transport, cancellation, deadline, and decoding errors wrap their
underlying cause, so they remain compatible with `errors.Is`.

```go
objects, err := client.ReadAll(ctx, definition)
if err != nil {
	var apiErr *hubspot.APIError
	if errors.As(err, &apiErr) {
		log.Printf(
			"HubSpot operation=%s category=%s status=%d body=%q",
			apiErr.Operation,
			apiErr.Category,
			apiErr.StatusCode,
			apiErr.Body,
		)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("HubSpot deadline exceeded: %w", err)
	}
	return err
}
```

HTTP statuses are classified centrally:

| HTTP status | `ErrorCategory` |
| --- | --- |
| 400, 422 | `ErrorCategoryValidation` |
| 401 | `ErrorCategoryAuthentication` |
| 403 | `ErrorCategoryAuthorization` |
| 404 | `ErrorCategoryNotFound` |
| 408, 504 | `ErrorCategoryTimeout` |
| 409 | `ErrorCategoryConflict` |
| 429 | `ErrorCategoryRateLimit` |
| Other 5xx | `ErrorCategoryServer` |
| Other non-2xx | `ErrorCategoryUnexpected` |

Local context deadlines can be detected with
`errors.Is(err, context.DeadlineExceeded)`. HTTP 408 and 504 responses use
`ErrorCategoryTimeout` and are available through `errors.As` as an `APIError`.

## Service operation timeout

The application service can add a deadline to each external HubSpot or AI
operation. Configure it once when constructing the service:

```go
processor, err := service.New(
	ctx,
	hubSpotClient,
	aiClient,
	printer,
	definition,
	service.Options{
		Prompt:            prompt,
		MinimalConfidence: 0.75,
		HighConfidence:    0.90,
		OperationTimeout:  2 * time.Minute,
	},
)
```

The timeout applies separately to each property-option lookup during service
initialization, the complete paginated `ReadAll` operation, each AI enrichment,
and each HubSpot write. A zero value adds no service-level deadline; a negative
value is rejected by `service.New`. Parent-context cancellation and earlier
deadlines always take precedence.
