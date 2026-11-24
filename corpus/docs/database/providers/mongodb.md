---
title: MongoDB Provider
type: docs
prev: docs/database/providers/bun
next: docs/database/providers/redis
weight: 70
sidebar:
  open: true
---

The MongoDB provider (`github.com/lemmego/gpamongo`) wraps the official MongoDB Go driver to provide GPA-compatible document database access. It supports document-specific operations like aggregation, text search, and geospatial queries.

## Setup

```go
import (
    "github.com/lemmego/gpa"
    "github.com/lemmego/gpamongo"
)

config := gpa.Config{
    Host:     "localhost",
    Port:     27017,
    Database: "mydb",
    Username: "admin",
    Password: "password",
}

provider, err := gpamongo.NewProvider(config)
```

## Creating Repositories

```go
// The collection name is auto-derived from the type name (lowercased, pluralized)
repo := gpamongo.GetRepository[User](provider)
```

Returns `gpa.DocumentRepository[T]`.

## Document-Specific Operations

### Find by Document

```go
users, err := repo.FindByDocument(ctx, map[string]interface{}{
    "status": "active",
    "age": map[string]interface{}{"$gte": 18},
})
```

### Text Search

```go
results, err := repo.TextSearch(ctx, "golang developer", gpa.Limit(5))
```

### Geospatial Queries

```go
// Find near a point (longitude, latitude)
results, err := repo.FindNear(ctx, "location", []float64{-73.9857, 40.7484}, 1000)

// Find within a polygon
results, err := repo.FindWithinPolygon(ctx, "location", polygon)
```

### Aggregation Pipeline

```go
pipeline := []map[string]interface{}{
    {"$match": map[string]interface{}{"status": "active"}},
    {"$group": map[string]interface{}{
        "_id": "$category",
        "count": map[string]interface{}{"$sum": 1},
    }},
    {"$sort": map[string]interface{}{"count": -1}},
}

results, err := repo.Aggregate(ctx, pipeline)
```

### Distinct Values

```go
values, err := repo.Distinct(ctx, "category", nil)
```

## Collection Management

```go
// Create collection
err := repo.CreateCollection(ctx)

// Drop collection
err := repo.DropCollection(ctx)

// Index management
err := repo.CreateIndex(ctx, map[string]interface{}{"email": 1}, true)
err := repo.DropIndex(ctx, "email_1")
```

## Features

| Feature | Support |
|---------|---------|
| Transactions | Yes (MongoDB sessions) |
| Text Search | Yes |
| Geospatial | Yes (Near, WithinPolygon) |
| Aggregation | Yes |
| Index Management | Yes |
| Hooks | None |

## Entity ID Handling

The MongoDB provider uses reflection-based ID extraction and supports both `primitive.ObjectID` and hex string IDs:

```go
type User struct {
    ID    primitive.ObjectID `bson:"_id,omitempty"`
    Name  string
    Email string
}
```

## Next steps

- [Generic repository](/docs/database/generic-repository) — the
  document-specific repositories in context.
- [Provider registry](/docs/database/provider-registry) — registering it beside
  a SQL provider.
