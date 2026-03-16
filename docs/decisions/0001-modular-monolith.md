# 0001 - Start as a Modular Monolith

## Status

Accepted

## Context

Rankster needs multiple product areas (social graph, feed, ranks, surveys/ads, subscriptions).
Early velocity and schema correctness matter more than service boundaries.

## Decision

Start with a modular monolith:
- One deployable API process (later split workers)
- One PostgreSQL database
- Clear module boundaries in code

## Consequences

- Fast iteration and a single transactional datastore.
- Easy to extract heavy modules later (feed, surveys) when required by load.

