---
title: Relationships
type: docs
prev: docs/orm/writes
next: docs/orm/field-descriptors
weight: 53
sidebar:
  open: true
---

Rows refer to each other. A post has an author; a user has many posts; an
article has many tags and each tag has many articles. In the database that is
done with a **foreign key** — a column on one table holding another table's
`id`.

Writing those joins by hand gets tedious fast, and gets slow faster: fetch
fifty posts, then fetch each author as you render them, and you have made
fifty-one trips to the database to show one page. That is common enough to
have a name, the **N+1 problem**.

So you declare the relationship once on your struct, and then ask for related
rows when you want them:

```go
posts, err := db.Model[Post]().With("Author").All(ctx)
```

`With` fetches every author in **one** extra query rather than one per post —
that is called **eager loading**, and it is the fix for N+1.

## Declaring relationships

Relationships are struct tags. The tag names the **field** that holds the foreign key, not the column:

```go
type User struct {
    ID      int      `orm:"primaryKey;autoIncrement"`
    Profile *Profile `orm:"hasOne:UserID"`     // UserID lives on Profile
    Posts   []Post   `orm:"hasMany:UserID"`    // UserID lives on Post
}

type Post struct {
    ID       int       `orm:"primaryKey;autoIncrement"`
    UserID   int
    Author   *User     `orm:"belongsTo:UserID"`   // UserID lives on Post
    Comments []Comment `orm:"hasMany:PostID"`
    Tags     []Tag     `orm:"many2many:post_tags"`
}
```

| Tag | Meaning |
|---|---|
| `hasOne:Field` | One related row; the named field is on the target |
| `hasMany:Field` | Many related rows; the named field is on the target |
| `belongsTo:Field` | The named field is on **this** model |
| `many2many:table` | Joined through a pivot table |

Add `references:Field` to join on something other than the primary key.

For many-to-many, pivot columns default to the conventional `<singular model>_id` on each side — `post_id` and `tag_id` above. Override with `joinForeignKey:` and `joinTargetKey:`.

## Eager loading

`With` loads relationships, including nested paths:

```go
posts, err := db.Model[Post]().With("Author", "Tags", "Comments").All(ctx)

users, err := db.Model[User]().With("Posts.Comments").All(ctx)
```

Loading is done with batched `WHERE fk IN (...)` lookups rather than joins, which avoids multiplying rows. The cost is **one query per relationship level**, independent of how many rows come back:

| Query | Statements |
|---|---|
| `With("Posts")` over 200 users | 2 |
| `With("Posts.Comments")` | 3 |
| `With("Tags")` (many-to-many) | 3 — the pivot, then the targets |

Soft-deleted rows are excluded from relationships, exactly as they are from a direct query.

## Association writes

Models stay plain structs. Associations are operated through the database handle rather than off the model:

```go
db.Relation[User, Post](&user, "Posts").Add(ctx, &post)
db.Relation[User, Post](&user, "Posts").Remove(ctx, &post)
db.Relation[User, Post](&user, "Posts").Load(ctx)
count, _ := db.Relation[User, Post](&user, "Posts").Count(ctx)

// Narrow it further before reading
recent, _ := db.Relation[User, Post](&user, "Posts").
    Query().Where(orm.Eq("published", true)).All(ctx)
```

`Add` inserts a new record or relinks an existing one. `Remove` clears the foreign key — it never deletes the row, since dropping a record is a separate, explicit decision.

A model that carried a live connection could not be safely serialised, copied between goroutines, or built in a test without a database. That is why `user.Posts().Add(...)` is not offered.

### Many-to-many

```go
tags := db.Relation[Post, Tag](&post, "Tags")

tags.Attach(ctx, &goTag, &sqlTag)   // idempotent; existing links are not duplicated
tags.Detach(ctx, &sqlTag)           // Detach(ctx) with no arguments clears everything
tags.Sync(ctx, &goTag, &ormTag)     // make the set exactly this
tags.Toggle(ctx, &goTag)            // flip membership

tags.AttachKeys(ctx, 1, 2)          // and DetachKeys / SyncKeys / ToggleKeys
```

The `...Keys` variants take identifiers directly, for when the related rows are known by id rather than loaded.

`Add` and `Remove` delegate to `Attach` and `Detach` on a many-to-many, so the handle behaves sensibly whichever verb you reach for.

Inside a transaction, use `tx.Relation[P, C]` so the writes join the same unit of work:

```go
err := db.Transaction(ctx, func(tx *orm.Tx) error {
    if _, err := tx.Model[Post]().Create(ctx, post); err != nil {
        return err
    }
    return tx.Relation[Post, Tag](post, "Tags").AttachKeys(ctx, tagIDs...)
})
```

## Composite keys and relationships

A relationship joins on a single column. A model with a composite primary key must therefore name which one with `references:`, or eager loading reports the problem rather than guessing.

## Limitations

`hasOneThrough`, `hasManyThrough` and polymorphic relationships are not available.

## Next steps

- [Field descriptors](/docs/orm/field-descriptors) — checked column names for
  the queries you build.
- [Queries](/docs/orm/queries) — predicates, pagination and scopes.
- [Schema builder](/docs/migrations/schema-builder) — the foreign keys behind a
  relationship.
