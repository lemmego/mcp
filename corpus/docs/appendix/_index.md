---
title: Appendix
type: docs
prev: docs/architecture/
next: docs/appendix/upgrading
weight: 161
sidebar:
  open: true
---

## Environment Variables

{{< tabs >}}
{{< tab name="Application" >}}

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_NAME` | `Lemmego` | Application name |
| `APP_ENV` | `development` | Environment (`production`, `development`) |
| `APP_DEBUG` | `false` | Enable debug mode |
| `APP_PORT` | `1490` | HTTP server port |
| `APP_URL` | `http://localhost:1490` | Application URL |
| `APP_KEY` | — | 32-byte base64 encryption key |

{{< /tab >}}
{{< tab name="Database" >}}

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_CONNECTION` | `sqlite` | Database driver |
| `DB_HOST` | `127.0.0.1` | Database host |
| `DB_PORT` | (driver default) | Database port |
| `DB_DATABASE` | `storage/database.sqlite` | Database name/path |
| `DB_USERNAME` | `root` | Database user |
| `DB_PASSWORD` | — | Database password |

{{< /tab >}}
{{< tab name="Session" >}}

| Variable | Default | Description |
|----------|---------|-------------|
| `SESSION_DRIVER` | `file` | Session store (`memory`, `file`, `redis`, `database`) |
| `SESSION_LIFETIME` | `120` | Session lifetime in minutes |
| `SESSION_COOKIE` | `lemmego_session` | Session cookie name |
| `REDIS_HOST` | `127.0.0.1` | Redis host |

{{< /tab >}}
{{< tab name="Filesystem" >}}

| Variable | Default | Description |
|----------|---------|-------------|
| `FILESYSTEM_DISK` | `local` | Default storage disk |
| `AWS_ACCESS_KEY_ID` | — | S3 access key |
| `AWS_SECRET_ACCESS_KEY` | — | S3 secret key |
| `AWS_DEFAULT_REGION` | `us-east-1` | S3 region |
| `AWS_BUCKET` | — | S3 bucket |

{{< /tab >}}
{{< tab name="Mail" >}}

| Variable | Default | Description |
|----------|---------|-------------|
| `MAIL_MAILER` | `smtp` | Mail driver |
| `MAIL_HOST` | `smtp.mailtrap.io` | SMTP host |
| `MAIL_PORT` | `2525` | SMTP port |

{{< /tab >}}
{{< /tabs >}}

## Package Summary

| Module | Path | Description |
|--------|------|-------------|
| `api` | `github.com/lemmego/api` | Core framework |
| `auth` | `github.com/lemmego/auth` | Authentication |
| `cli` | `github.com/lemmego/cli` | CLI tool |
| `fsys` | `github.com/lemmego/fsys` | File system abstraction |
| `gpa` | `github.com/lemmego/gpa` | Go Persistence API |
| `gpagorm` | `github.com/lemmego/gpagorm` | GORM provider |
| `gpabun` | `github.com/lemmego/gpabun` | Bun provider |
| `gpamongo` | `github.com/lemmego/gpamongo` | MongoDB provider |
| `gparedis` | `github.com/lemmego/gparedis` | Redis provider |
| `orm` | `github.com/lemmego/orm` | Built-in SQL ORM |
| `gpaorm` | `github.com/lemmego/gpaorm` | GPA provider backed by the built-in ORM |
| `ormconnector` | `github.com/lemmego/ormconnector` | ORM app provider |
| `gormconnector` | `github.com/lemmego/gormconnector` | GORM app provider |
| `bunconnector` | `github.com/lemmego/bunconnector` | Bun app provider |
| `inertia` | `github.com/lemmego/inertia` | Inertia.js adapter |
| `templ` | `github.com/lemmego/templ` | Templ bridge |
| `migration` | `github.com/lemmego/migration` | Database migrations |

## Module Dependencies

```
api → fsys, migration
gpa → (standalone)
gpagorm → gpa, GORM
gpabun → gpa, Bun
gpamongo → gpa, MongoDB driver
gparedis → gpa, go-redis
orm → (standalone, database/sql only)
gpaorm → gpa, orm
ormconnector → api, gpa, orm, gpaorm
gormconnector → api, gpa, gpagorm
bunconnector → api, gpa, gpabun
inertia → api, gonertia
templ → api, a-h/templ
auth → api
cli → (standalone, embeds scaffold templates)
lemmego → api, gpa, orm, gparedis, inertia, auth, ormconnector, migration, queue
```

## Next steps

- [Upgrade notes](/docs/appendix/upgrading) — every change that needs action on
  your project.
- [FAQ](/docs/appendix/faq) — the questions that come up most.
- [Configuration](/docs/configuration) — reading these values in your own code.
