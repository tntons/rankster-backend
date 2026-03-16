# Source Layout

This folder is organized by bounded context (modules) so a modular monolith can scale into services later.

No HTTP routes are implemented yet (schema-first workflow). The OpenAPI spec lives at `openapi.yaml`.

## Suggested Module Responsibilities

- `modules/feed`: feed query, ranking, and survey injection
- `modules/ranks`: rank post creation + tier validation against templates
- `modules/search`: category/template discovery
- `modules/users`: profiles, social graph, pinned posts, stats
- `modules/surveys`: survey CRUD, responses, campaign accounting
- `modules/leaderboards`: aggregation queries + snapshot reads

