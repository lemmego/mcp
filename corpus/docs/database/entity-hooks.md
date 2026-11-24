---
title: Entity Hooks
type: docs
prev: docs/database/query-system
next: docs/database/transactions
weight: 65
sidebar:
  open: true
---

Entities can implement lifecycle hooks that are automatically called during CRUD operations. This enables automatic behavior like validation, timestamp management, and data transformation.

## Available Hooks

```go
type ValidationHook   interface { Validate(ctx context.Context) error }
type BeforeCreateHook interface { BeforeCreate(ctx context.Context) error }
type AfterCreateHook  interface { AfterCreate(ctx context.Context) error }
type BeforeUpdateHook interface { BeforeUpdate(ctx context.Context) error }
type AfterUpdateHook  interface { AfterUpdate(ctx context.Context) error }
type BeforeDeleteHook interface { BeforeDelete(ctx context.Context) error }
type AfterDeleteHook  interface { AfterDelete(ctx context.Context) error }
type BeforeFindHook   interface { BeforeFind(ctx context.Context) error }
type AfterFindHook    interface { AfterFind(ctx context.Context) error }
```

## Execution Order

When performing a create or update operation, hooks run in this order:

1. **`Validate()`** — validate the entity before anything else
2. **`BeforeCreate()` / `BeforeUpdate()`** — transform or enrich data
3. **Database operation** — the actual insert/update
4. **`AfterCreate()` / `AfterUpdate()`** — post-operation logic

For deletes:
1. **`BeforeDelete()`**
2. **Database operation**
3. **`AfterDelete()`**

For finds:
1. **`BeforeFind()`**
2. **Database query**
3. **`AfterFind()`** (called for each loaded entity)

## Examples

### Validation Hook

```go
type User struct {
    ID    int
    Email string
    Name  string
}

func (u *User) Validate(ctx context.Context) error {
    if u.Email == "" {
        return gpa.NewError(gpa.ErrorTypeValidation, "email is required")
    }
    if u.Name == "" {
        return gpa.NewError(gpa.ErrorTypeValidation, "name is required")
    }
    return nil
}
```

### Before Create Hook

```go
func (u *User) BeforeCreate(ctx context.Context) error {
    u.Email = strings.ToLower(strings.TrimSpace(u.Email))
    u.CreatedAt = time.Now()
    u.UpdatedAt = time.Now()
    return nil
}
```

### After Create Hook

```go
func (u *User) AfterCreate(ctx context.Context) error {
    // Send welcome email, log audit, etc.
    return sendWelcomeEmail(u.Email)
}
```

### Before Update Hook

```go
func (u *User) BeforeUpdate(ctx context.Context) error {
    u.UpdatedAt = time.Now()
    return nil
}
```

### After Find Hook

```go
func (u *User) AfterFind(ctx context.Context) error {
    // Compute derived fields, mask sensitive data, etc.
    u.DisplayName = fmt.Sprintf("%s (%s)", u.Name, u.Email)
    return nil
}
```

### Full Example

```go
type Task struct {
    ID          uint
    Title       string
    Description string
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func (t *Task) Validate(ctx context.Context) error {
    if t.Title == "" {
        return gpa.NewError(gpa.ErrorTypeValidation, "title is required")
    }
    return nil
}

func (t *Task) BeforeCreate(ctx context.Context) error {
    if t.Status == "" {
        t.Status = "todo"
    }
    t.CreatedAt = time.Now()
    t.UpdatedAt = time.Now()
    return nil
}

func (t *Task) BeforeUpdate(ctx context.Context) error {
    t.UpdatedAt = time.Now()
    return nil
}
```

## Provider-Specific Hook Support

| Provider | Validation | Before/After Create | Before/After Update | Before/After Delete | After Find |
|----------|-----------|---------------------|---------------------|--------------------|------------|
| GORM | Yes | Yes | Yes | Yes | Yes |
| Bun | No | Yes | Yes | Yes | Yes |
| MongoDB | No | No | No | No | No |
| Redis* | No | Yes (Set) | No | Yes (Delete) | Yes |

*Redis hooks apply to `SetWithTTL` and `DeleteKey` operations only.

## Next steps

- [Transactions](/docs/database/transactions) — where a hook's error takes the
  whole batch.
- [Generic repository](/docs/database/generic-repository) — the operations that
  fire these hooks.
