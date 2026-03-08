# M25-Auto-Clipping-AI

## Module Metadata
- Module ID: M25
- Canonical Name: M25-Auto-Clipping-AI
- Runtime Cluster: data-ai
- Category: AI & Automation
- Architecture: microservice

## Primary Responsibility
See canonical service specification.

## Dependency Snapshot
### DBR Dependencies
- M24-Clipping-Tool-Service (owner_api)

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes (health + out-of-MVP status only)

## Implementation Notes
- Business APIs/events are disabled in MVP scope per canonical spec.
- Runtime advertises a canonical error envelope (`SERVICE_OUT_OF_MVP`) for
  non-health endpoints.
- Admin deploy endpoint `/v1/admin/models/deploy` persists idempotency replay
  records to disk for restart-safe behavior (`M25_IDEMPOTENCY_STORE_PATH`).
- In production runtime (`M25_RUNTIME_MODE=production`), both
  `M24_CLIPPING_TOOL_OWNER_API_URL` and `M25_IDEMPOTENCY_STORE_PATH` must be
  explicitly configured (no implicit fallback).
- Follow canonical contracts from `viralForge/specs/M25-Auto-Clipping-AI.md`.
