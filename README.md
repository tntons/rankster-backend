# Rankster Backend (Boilerplate)

TypeScript + PostgreSQL backend scaffold focused on:
- Social tier-list ranking posts (Rank posts)
- Survey posts (in-feed ads)
- Prisma-first schema design + seed data
- OpenAPI documentation for core endpoints

This repo currently contains architecture + schema + API spec only (no HTTP routes yet).

## Quick Start (DB + Prisma)

1) Create a Postgres database.
2) Copy env example:

```bash
cp .env.example .env
```

3) Install deps, generate client, run migrations, seed:

```bash
npm i
npx prisma generate
npx prisma migrate dev
npx prisma db seed
```

## Docs

- `docs/architecture.md`
- `docs/erd.mmd` (Mermaid ERD)
- `openapi.yaml`

