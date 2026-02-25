# M55-Dashboard-Service

## Module Metadata
- Module ID: M55
- Canonical Name: M55-Dashboard-Service
- Runtime Cluster: data-ai
- Category: Analytics & Reporting
- Architecture: microservice

## Primary Responsibility
System must serve tailored dashboards per user role: Creator (campaigns, earnings, submissions, payouts), Clipper (submissions, views, rewards, history), Developer (app installs, revenue, ratings), Admin (platform GMV, top creators, disputes, system health).  

## Dependency Snapshot
### DBR Dependencies
- M02-Profile-Service
- M05-Billing-Service
- M09-Content-Library-Marketplace
- M13-Escrow-Ledger-Service
- M22-Onboarding-Service
- M39-Finance-Service
- M41-Reward-Engine
- M47-Gamification-Service
- M54-Analytics-Service
- M60-Product-Service

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes

## Implementation Notes
- Internal service calls: gRPC.
- External/public interfaces: REST.
- Follow canonical contracts from viralForge/specs/M55-*.md.
