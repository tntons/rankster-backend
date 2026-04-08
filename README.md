# Rankster Backend (Boilerplate)

Go + PostgreSQL backend scaffold focused on:
- Social tier-list ranking posts (Rank posts)
- Survey posts (in-feed ads)
- OpenAPI documentation for core endpoints

This repo currently contains architecture + schema + API spec only (no HTTP routes yet).

## Quick Start (DB)

1) Create a Postgres database.
2) Copy env example:

```bash
cp .env.example .env
```

## Run API

```bash
go run ./cmd/api
```

Then call:
- `GET http://localhost:3000/feed/main`
- `POST http://localhost:3000/rank/create`
- `GET http://localhost:3000/search/categories?q=coffee`
- `GET http://localhost:3000/user/stats` (requires `Authorization: Bearer <userId>` and an active PRO/BUSINESS subscription)

## Docs

- `docs/architecture.md`
- `docs/erd.mmd` (Mermaid ERD)
- `openapi.yaml`
