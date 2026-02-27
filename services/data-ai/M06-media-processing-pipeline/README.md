# M06-Media-Processing-Pipeline

## Module Metadata
- Module ID: M06
- Canonical Name: M06-Media-Processing-Pipeline
- Runtime Cluster: data-ai
- Category: Editorial Workflow
- Architecture: microservice

## Primary Responsibility
Handle media upload orchestration and processing lifecycle (transcode, aspect conversions, thumbnails, watermarking) with status APIs for internal callers.

## Dependency Snapshot
### DBR Dependencies
- M04-Campaign-Service (owner_api)

### Event Dependencies
- none

### Event Provides
- none

### HTTP Provides
- yes (internal)

## Owned Tables (Canonical)
- `media_assets`
- `media_jobs`
- `media_outputs`
- `media_thumbnails`
- `watermark_records`

## API Surface
- `POST /v1/media/uploads` (requires `Idempotency-Key`, TTL 7 days)
- `GET /v1/media/assets/{asset_id}`
- `POST /v1/media/assets/{asset_id}/retry` (admin + `Idempotency-Key`, TTL 7 days)

## gRPC Surface
- Contract: `contracts/proto/media/v1/media_internal.proto`
- Service: `viralforge.media.v1.MediaInternalService`
- Methods:
  - `GetPreviewUrl`
  - `GetAssetMetadata`
- Internal health service registration

## Operational Notes
- No canonical Kafka events consumed or emitted.
- Event dedup repository exists for compliance but no canonical event handlers are active.
- Cross-service reads/writes occur only through the declared `M04-Campaign-Service` owner API client.
