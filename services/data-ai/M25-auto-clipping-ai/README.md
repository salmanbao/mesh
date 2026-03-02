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
- Follow canonical contracts from `viralForge/specs/M25-Auto-Clipping-AI.md`.
