# Rankster Backend

Rankster backend is a Go API built with Gin, GORM, and PostgreSQL.
It powers:
- social rank posts
- survey posts in the feed
- category discovery
- subscriber-only user stats

The API currently runs on port `8000` by default.

## Stack

- Go
- Gin
- GORM
- PostgreSQL

## Prerequisites

Make sure these are installed before running the project:
- Go `1.21+`
- PostgreSQL `14+`

You can verify with:

```bash
go version
psql --version
```

## Clone And Run

1. Clone the repository and enter it:

```bash
git clone <your-repo-url>
cd rankster-backend
```

2. Create the PostgreSQL database:

```bash
createdb rankster
```

If `createdb` is not available, use:

```bash
psql -d postgres -c 'CREATE DATABASE rankster;'
```

3. Export the database connection string:

```bash
export DATABASE_URL='postgresql://<your-postgres-user>@localhost:5432/rankster?sslmode=disable'
```

Example:

```bash
export DATABASE_URL='postgresql://tntons@localhost:5432/rankster?sslmode=disable'
```

4. If your Go install tries to auto-download a newer toolchain, force the local one:

```bash
export GOTOOLCHAIN=local
```

5. Download dependencies:

```bash
go mod download
```

6. Start the backend:

```bash
go run ./cmd/api
```

The API will start on:

```text
http://localhost:8000
```

## What Happens On Startup

The backend currently bootstraps itself automatically:
- GORM auto-migrates the required tables
- seed data is inserted if the database is empty

That means a fresh local database is enough to get started.

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `HOST`: API bind host, default `0.0.0.0`
- `PORT`: API port, default `8000`
- `GOTOOLCHAIN`: recommended `local` if your machine has an older but compatible local Go toolchain

## Quick Checks

Health check:

```bash
curl http://localhost:8000/healthz
```

Main feed:

```bash
curl 'http://localhost:8000/feed/main'
```

Search categories:

```bash
curl 'http://localhost:8000/search/categories?q=coffee'
```

## Seeded Local Data

The local seed creates sample users, categories, assets, rank posts, and one survey post.

Sample seeded user:
- `alice`
- user id: `c17147fe-5ca5-4d36-9c8a-b8215660f12f`

Use this header for authenticated local testing:

```bash
-H 'Authorization: Bearer c17147fe-5ca5-4d36-9c8a-b8215660f12f'
```

Subscriber-only stats endpoint:

```bash
curl -H 'Authorization: Bearer c17147fe-5ca5-4d36-9c8a-b8215660f12f' \
  http://localhost:8000/user/stats
```

Create a rank post:

```bash
curl -X POST http://localhost:8000/rank/create \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer c17147fe-5ca5-4d36-9c8a-b8215660f12f' \
  -d '{
    "categoryId": "1a669446-b6ce-4d60-8787-ace8343d1940",
    "templateId": "a46cc484-84fe-4c3c-91e5-d42195391be0",
    "tierKey": "B",
    "imageAssetId": "17ded82a-e65e-419e-9f14-6800a932b6fd",
    "caption": "Cold Brew belongs in B tier",
    "subjectTitle": "Cold Brew"
  }'
```

## Database Notes

Recommended local database:
- PostgreSQL

Why PostgreSQL:
- strong relational support for social graph + rankings
- good JSON support for survey targeting and flexible metadata
- reliable indexing and query performance for feed/search workloads

## Project Structure

- `cmd/api`: application entrypoint
- `internal/config`: environment config
- `internal/db`: database connection, migration, and seeding
- `internal/models`: GORM models
- `internal/handlers`: HTTP handlers
- `internal/server`: router setup
- `docs/architecture.md`: architecture notes
- `docs/erd.mmd`: Mermaid ERD
- `openapi.yaml`: API contract

## Troubleshooting

If Go fails with a toolchain download error:

```bash
export GOTOOLCHAIN=local
```

If PostgreSQL is not running:

```bash
brew services start postgresql@14
```

If the port is already in use:

```bash
lsof -nP -iTCP:8000 -sTCP:LISTEN
```

Then either stop that process or override the port:

```bash
PORT=8001 go run ./cmd/api
```

## Docs

- `docs/architecture.md`
- `docs/erd.mmd`
- `openapi.yaml`
