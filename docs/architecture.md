# Rankster Backend Architecture (Scalable Baseline)

This document describes a pragmatic modular-monolith that scales into services as load grows.
Start with one deployable unit + one PostgreSQL DB; split by bounded context later without rewrites.

## Tech Choices (Recommended)

- Runtime: Node.js (TypeScript)
- HTTP: (later) Fastify or NestJS
- DB: PostgreSQL
- ORM: Prisma
- Validation: Zod
- Storage: Object storage for images (S3/GCS/R2); DB stores metadata only
- Queue/Workers: (later) Redis + BullMQ (or similar) for aggregation and ad delivery accounting
- Cache: (later) Redis for hot feeds and leaderboards

## Modules (Bounded Contexts)

### Users & Social Graph

- Auth: local email/password + OAuth providers (extendable)
- Profiles: public profile + customization settings (subscription-gated)
- Social graph: follow/unfollow, privacy controls (later)
- Pinned ranks: per-user pinned posts
- Stats: materialized counters for fast reads (`UserStats`)

### Tier Lists & Ranking

- Categories: global taxonomy (`Category`)
- Templates: master (global) and user-generated (`TierListTemplate`, `TierDefinition`)
- Rank posts: user posts an image into a category + tier position (`Post` + `RankPost`)
- Overall review: aggregated category stats for "average tier" and distributions
- Leaderboards: snapshots computed by workers

### Feed & Interactions

- Feed posts: mix chronological + popularity
- Interactions: likes, comments, shares (SNS share logs)
- Metrics: `PostMetrics` stores counters and a `hotScore` used by feed ranking

### Surveys & Ads

- Surveys are specialized posts (`Post` + `SurveyPost`)
- Campaigns: sponsorship, budget, delivery windows, targeting rules
- Delivery: injected into main feed at a configurable interval
- Accounting: impressions/clicks/responses tracked in DB (or streaming later)

### Monetization & Data Market

- Subscription: plans, active periods, provider references
- Perks: advanced stats endpoints, ad-free browsing, profile customization
- Data market: aggregated, anonymized metrics exported for B2B (`DataMarketAggregate`)

## Feed Algorithm (Baseline)

Main feed ranking is a weighted blend:

- Primary: `createdAt` (freshness)
- Secondary: `hotScore` (popularity)

Suggested `hotScore` (computed periodically):

`hotScore = likes*1.0 + comments*1.5 + shares*2.0 + recencyBoost`

Where `recencyBoost` decays over time (e.g. exponential decay).

Survey injection:

- For every N organic posts returned (e.g. 7), inject 1 eligible survey ad if:
  - campaign is active (time window)
  - budget is available
  - user is not ad-free
  - targeting matches (optional)

## Data Flows / Background Jobs

- Update `PostMetrics` (counters, hotScore)
- Update `UserStats` (ranks count, follower counts)
- Compute `CategoryOverallReview` and tier distributions
- Generate `LeaderboardSnapshot` (daily/weekly/all-time)
- Survey campaign pacing + budget decrements

## Evolution Path

If/when needed:

1) Extract workers into a separate process (same repo, same DB).
2) Add read-optimized caches (Redis) for feed and leaderboard.
3) Split into services by context (users/feed/surveys) with events (Kafka/SNS/SQS).

