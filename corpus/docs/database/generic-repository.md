---
title: Generic Repository
type: docs
prev: docs/database/connection-seam
next: docs/database/query-system
weight: 63
sidebar:
  open: true
---

A **repository** is an object that holds all the database operations for one
kind of record. You get a `Repository[User]` and ask *it* for users, rather
than writing queries wherever you happen to need one. Your SQL — or whatever
the store underneath speaks — ends up in one place, and your handlers stay
about HTTP.

The `[User]` part is what keeps it pleasant to use. `FindByID` on a
`Repository[User]` hands you a `*User`, not something you have to convert:

```go
userRepo := gpagorm.GetRepository[User]()
user, err := userRepo.FindByID(ctx, 42)   // *User
```

## Creating repositories

```go
// From the provider registry (recommended)
userRepo := gpagorm.GetRepository[User]()
postRepo := gpagorm.GetRepository[Post]()

// From a specific provider instance
provider := gpa.MustGet[*gpagorm.Provider]("primary")
userRepo := gpagorm.GetRepository[User](provider)
```

## CRUD Operations

### Create

```go
user := &User{Name: "John", Email: "john@example.com"}
err := userRepo.Create(ctx, user)
// user.ID is now populated
```

### Create Batch

```go
users := []*User{
    {Name: "Alice"},
    {Name: "Bob"},
}
err := userRepo.CreateBatch(ctx, users)
```

### Find By ID

```go
user, err := userRepo.FindByID(ctx, 1)
// user is *User, not interface{}
```

### Find All

```go
users, err := userRepo.FindAll(ctx)
// users is []*User
```

### Update

```go
user.Name = "Jane"
err := userRepo.Update(ctx, user)
```

### Partial Update

```go
err := userRepo.UpdatePartial(ctx, 1, map[string]interface{}{
    "status": "inactive",
    "age":    31,
})
```

### Delete

```go
err := userRepo.Delete(ctx, 1)
```

### Delete by Condition

```go
err := userRepo.DeleteByCondition(ctx, gpa.Where("status", gpa.OpEqual, "inactive"))
```

## Query Operations

```go
// Find with conditions
users, err := userRepo.Query(ctx,
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("status", gpa.OpEqual, "active"),
)

// Find single result
user, err := userRepo.QueryOne(ctx,
    gpa.Where("email", gpa.OpEqual, "user@example.com"),
)

// Count
count, err := userRepo.Count(ctx, gpa.Where("active", gpa.OpEqual, true))

// Check existence
exists, err := userRepo.Exists(ctx, gpa.Where("email", gpa.OpEqual, "user@example.com"))
```

## Repository Interface

```go
type Repository[T any] interface {
    // Basic CRUD
    Create(ctx context.Context, entity *T) error
    CreateBatch(ctx context.Context, entities []*T) error
    FindByID(ctx context.Context, id interface{}) (*T, error)
    FindAll(ctx context.Context, opts ...QueryOption) ([]*T, error)
    Update(ctx context.Context, entity *T) error
    UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error
    Delete(ctx context.Context, id interface{}) error
    DeleteByCondition(ctx context.Context, condition Condition) error

    // Query
    Query(ctx context.Context, opts ...QueryOption) ([]*T, error)
    QueryOne(ctx context.Context, opts ...QueryOption) (*T, error)
    Count(ctx context.Context, opts ...QueryOption) (int64, error)
    Exists(ctx context.Context, opts ...QueryOption) (bool, error)

    // Advanced
    Transaction(ctx context.Context, fn TransactionFunc[T]) error
    RawQuery(ctx context.Context, query string, args []interface{}) ([]*T, error)
    RawExec(ctx context.Context, query string, args []interface{}) (Result, error)

    // Metadata
    GetEntityInfo() (*EntityInfo, error)
    Close() error
}
```

## SQL-Specific Repositories

SQL providers support additional operations through specialized interfaces:

### SQLRepository[T]

```go
type SQLRepository[T any] interface {
    Repository[T]
    FindBySQL(ctx, sql string, args []interface{}) ([]*T, error)
    ExecSQL(ctx, sql string, args ...interface{}) (Result, error)
    FindWithRelations(ctx, relations []string, opts ...QueryOption) ([]*T, error)
    FindByIDWithRelations(ctx, id interface{}, relations []string) (*T, error)
    CreateTable(ctx) error
    DropTable(ctx) error
    CreateIndex(ctx, fields []string, unique bool) error
    DropIndex(ctx, indexName string) error
}
```

### MigratableRepository[T]

```go
type MigratableRepository[T any] interface {
    SQLRepository[T]
    MigrateTable(ctx) error
    GetMigrationStatus(ctx) (MigrationStatus, error)
    GetTableInfo(ctx) (TableInfo, error)
}
```

## KV-Specific Repositories

Redis and other key-value stores use specialized interfaces:

```go
// Basic KV operations
type BasicKeyValueRepository[T any] interface {
    Get(ctx, key) (*T, error)
    Set(ctx, key, value *T) error
    DeleteKey(ctx, key) error
    KeyExists(ctx, key) (bool, error)
}

// TTL support
type TTLKeyValueRepository[T any] interface {
    SetWithTTL(ctx, key, value, ttl) error
    TTL(ctx, key) (time.Duration, error)
    SetTTL(ctx, key, ttl) error
    RemoveTTL(ctx, key) error
}
```

## Document-Specific Repositories

MongoDB and other document databases:

```go
type DocumentRepository[T any] interface {
    Repository[T]
    FindByDocument(ctx, doc map[string]interface{}) ([]*T, error)
    TextSearch(ctx, query string, opts ...QueryOption) ([]*T, error)
    FindNear(ctx, field string, point []float64, maxDistance float64) ([]*T, error)
    Aggregate(ctx, pipeline []map[string]interface{}) ([]map[string]interface{}, error)
    CreateCollection(ctx) error
    DropCollection(ctx) error
}
```

## Next steps

- [Query system](/docs/database/query-system) — conditions, options and the
  fluent builder.
- [Entity hooks](/docs/database/entity-hooks) — validating and stamping rows on
  the way through.
- [Transactions](/docs/database/transactions) — grouping writes so they land
  together.
