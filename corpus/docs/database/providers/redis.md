---
title: Redis Provider
type: docs
prev: docs/database/providers/mongodb
next: docs/database/gormconnector
weight: 71
sidebar:
  open: true
---

The Redis provider (`github.com/lemmego/gparedis`) wraps the [go-redis](https://github.com/redis/go-redis) client to provide GPA-compatible key-value storage. It supports TTL, batch operations, atomic counters, and pattern matching.

## Setup

```go
import (
    "github.com/lemmego/gpa"
    "github.com/lemmego/gparedis"
)

config := gpa.Config{
    Host: "localhost",
    Port: 6379,
}

provider, err := gparedis.NewProvider(config)
```

## Creating Repositories

```go
// With a key prefix
userRepo := gparedis.NewRepository[User](provider, provider.Client(), "user:")

// Or from the registry (empty prefix)
userRepo := gparedis.GetRepository[User](provider)
```

Returns `gpa.AdvancedKeyValueRepository[T]`.

## Basic Key-Value Operations

```go
// Set
userRepo.Set(ctx, "123", &User{Name: "John"})

// Get
user, err := userRepo.Get(ctx, "123")

// Check existence
exists, err := userRepo.KeyExists(ctx, "123")

// Delete
userRepo.DeleteKey(ctx, "123")
```

## TTL Operations

```go
// Set with expiration
userRepo.SetWithTTL(ctx, "session:abc", session, 30*time.Minute)

// Get remaining TTL
ttl, err := userRepo.TTL(ctx, "session:abc")

// Update TTL
userRepo.SetTTL(ctx, "session:abc", 1*time.Hour)

// Remove TTL (make persistent)
userRepo.RemoveTTL(ctx, "session:abc")
```

## Batch Operations

```go
// Multi-get
users, err := userRepo.MGet(ctx, []string{"1", "2", "3"})

// Multi-set
userRepo.MSet(ctx, map[string]*User{
    "4": {Name: "Alice"},
    "5": {Name: "Bob"},
})

// Multi-delete
deleted, err := userRepo.MDelete(ctx, []string{"1", "2"})
```

## Atomic Counters

```go
count, err := counterRepo.Increment(ctx, "page_views", 1)
count, err = counterRepo.Decrement(ctx, "page_views", 2)
```

## Pattern Matching

```go
// List keys by pattern
keys, err := userRepo.Keys(ctx, "user:*")

// Scan with cursor (for large key sets)
keys, cursor, err := userRepo.Scan(ctx, 0, "user:*", 10)
```

## Hooks

Redis supports hooks on specific operations:

```go
func (s *Session) AfterFind(ctx context.Context) error {
    // Refresh session on access
    return nil
}

func (s *Session) BeforeCreate(ctx context.Context) error {
    s.LastAccessed = time.Now()
    return nil
}
```

| Hook | Triggered By |
|------|-------------|
| `BeforeCreate` / `AfterCreate` | `SetWithTTL` |
| `AfterFind` | `Get` |
| `BeforeDelete` / `AfterDelete` | `DeleteKey` |

## Standard CRUD Unsupported

Standard `Repository[T]` CRUD methods (`Create`, `FindByID`, `FindAll`, `Update`, `Delete`, `Query`, `Transaction`) are not supported for Redis and return `gpa.ErrorTypeUnsupported`.

## Provider-Level Operations

```go
// Direct provider access (for raw operations)
provider := gpa.MustGet[*gparedis.Provider]("cache")
provider.Set(ctx, "key", value, 5*time.Minute)
val, err := provider.Get(ctx, "key")
provider.Delete(ctx, "key")
provider.Exists(ctx, "key")
provider.Keys(ctx, "pattern:*")
provider.Expire(ctx, "key", ttl)
provider.TTL(ctx, "key")
```

## Next steps

- [Caching](/docs/cache/) — the higher-level cache that can sit on Redis.
- [Provider registry](/docs/database/provider-registry) — running it alongside
  your primary store.
