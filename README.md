# HubSpot Object Reader

A slim Go application with a generic HubSpot read adapter and provider-neutral domain model.

## Structure

```text
hubspot-object-reader/
├── main.go
├── go.mod
├── domain/
│   ├── object.go
│   ├── quality.go
│   ├── definition.go
│   └── company.go
└── hubspot/
    ├── client.go
    └── read.go
```

## Domain model

All CRM data is represented as:

```go
type Object struct {
    ID         string
    Attributes map[string]Quality
}
```

The domain package contains no HTTP or token handling. Object types are configured through `ObjectDefinition` presets such as `domain.CompanyDefinition`.

## Generic HubSpot read

The HubSpot package can read any CRM object type as long as an `ObjectDefinition` is supplied:

```go
objects, err := client.ReadAll(ctx, definition)
```

Example contact definition:

```go
definition := domain.ObjectDefinition{
    Type: "contacts",
    Attributes: []domain.AttributeDefinition{
        {Name: "firstname"},
        {Name: "lastname"},
        {Name: "email"},
    },
}

objects, err := client.ReadAll(ctx, definition)
```

No new HubSpot reader code is needed for another object type.

## Run

PowerShell:

```powershell
$env:HUBSPOT_ACCESS_TOKEN="your-token"
go run .
```

Optional:

```powershell
$env:HUBSPOT_BASE_URL="https://api.hubapi.com"
```
