# Ranks Module

Responsibilities:
- Create rank posts (`POST /rank/create`)
- Validate `tierKey` belongs to template and template belongs to category
- Update counters (`UserStats`, `PostMetrics`) synchronously or via workers

Key tables:
- `Post`, `RankPost`, `TierListTemplate`, `TierDefinition`, `Asset`

