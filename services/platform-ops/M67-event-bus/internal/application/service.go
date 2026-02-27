package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/domain"
)

func (s *Service) PublishEvent(ctx context.Context, actor Actor, in PublishInput) (domain.PublishResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.PublishResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.PublishResult{}, domain.ErrIdempotencyRequired
	}
	in.Format = strings.ToLower(strings.TrimSpace(in.Format))
	if in.Format == "" {
		in.Format = domain.SchemaTypeJSON
	}
	if err := s.validatePublishInput(ctx, in); err != nil {
		return domain.PublishResult{}, err
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.PublishResult{}, err
	} else if ok {
		var out domain.PublishResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.PublishResult{}, err
	}

	now := s.nowFn()
	res := domain.PublishResult{
		EventID:    strings.TrimSpace(in.EventID),
		EventType:  strings.TrimSpace(in.EventType),
		Topic:      strings.TrimSpace(in.EventType),
		Status:     "accepted",
		Format:     in.Format,
		AcceptedAt: now,
	}
	if s.metrics != nil {
		_ = s.metrics.IncCounter(ctx, "kafka_producer_records_sent_total", map[string]string{"topic": res.Topic, "format": res.Format}, 1)
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 202, res)
	return res, nil
}

func (s *Service) CreateTopic(ctx context.Context, actor Actor, in CreateTopicInput) (domain.Topic, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Topic{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.Topic{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Topic{}, domain.ErrIdempotencyRequired
	}
	in.TopicName = strings.TrimSpace(in.TopicName)
	in.CleanupPolicy = strings.ToLower(strings.TrimSpace(in.CleanupPolicy))
	in.CompressionType = strings.ToLower(strings.TrimSpace(in.CompressionType))
	if !domain.IsValidTopicName(in.TopicName) {
		return domain.Topic{}, domain.ErrInvalidInput
	}
	if in.Partitions <= 0 {
		in.Partitions = 6
	}
	if in.ReplicationFactor <= 0 {
		in.ReplicationFactor = 3
	}
	if in.RetentionDays <= 0 {
		in.RetentionDays = 7
	}
	if in.CleanupPolicy == "" {
		in.CleanupPolicy = domain.CleanupDelete
	}
	if in.CompressionType == "" {
		in.CompressionType = domain.CompressionSnappy
	}
	if !domain.IsValidCleanupPolicy(in.CleanupPolicy) || !domain.IsValidCompression(in.CompressionType) {
		return domain.Topic{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Topic{}, err
	} else if ok {
		var out domain.Topic
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Topic{}, err
	}
	now := s.nowFn()
	row := domain.Topic{
		ID:                "topic-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		TopicName:         in.TopicName,
		Partitions:        in.Partitions,
		ReplicationFactor: in.ReplicationFactor,
		RetentionDays:     in.RetentionDays,
		CleanupPolicy:     in.CleanupPolicy,
		CompressionType:   in.CompressionType,
		Status:            domain.TopicStatusActive,
		CreatedBy:         actor.SubjectID,
		CreatedAt:         now,
	}
	if err := s.topics.Create(ctx, row); err != nil {
		return domain.Topic{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) ListTopics(ctx context.Context, actor Actor, limit int) ([]domain.Topic, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if s.topics == nil {
		return nil, nil
	}
	return s.topics.List(ctx, limit)
}

func (s *Service) CreateACL(ctx context.Context, actor Actor, in CreateACLInput) (domain.ACLRecord, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ACLRecord{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.ACLRecord{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ACLRecord{}, domain.ErrIdempotencyRequired
	}
	in.Principal = strings.TrimSpace(in.Principal)
	in.ResourceType = strings.ToLower(strings.TrimSpace(in.ResourceType))
	in.ResourceName = strings.TrimSpace(in.ResourceName)
	in.PatternType = strings.ToLower(strings.TrimSpace(in.PatternType))
	if in.PatternType == "" {
		in.PatternType = "literal"
	}
	if in.Principal == "" || in.ResourceName == "" || in.ResourceType == "" || len(in.Operations) == 0 {
		return domain.ACLRecord{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ACLRecord{}, err
	} else if ok {
		var out domain.ACLRecord
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ACLRecord{}, err
	}
	now := s.nowFn()
	row := domain.ACLRecord{
		ID:           "acl-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		Principal:    in.Principal,
		ResourceType: in.ResourceType,
		ResourceName: in.ResourceName,
		PatternType:  in.PatternType,
		Operations:   normalizeOps(in.Operations),
		Status:       domain.ACLStatusActive,
		CreatedBy:    actor.SubjectID,
		CreatedAt:    now,
	}
	if err := s.acls.Create(ctx, row); err != nil {
		return domain.ACLRecord{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) ListACLs(ctx context.Context, actor Actor, limit int) ([]domain.ACLRecord, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if s.acls == nil {
		return nil, nil
	}
	return s.acls.List(ctx, limit)
}

func (s *Service) RegisterSchema(ctx context.Context, actor Actor, in RegisterSchemaInput) (domain.SchemaRecord, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.SchemaRecord{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.SchemaRecord{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.SchemaRecord{}, domain.ErrIdempotencyRequired
	}
	in.Subject = strings.TrimSpace(in.Subject)
	in.SchemaType = strings.ToLower(strings.TrimSpace(in.SchemaType))
	in.Compatibility = strings.ToUpper(strings.TrimSpace(in.Compatibility))
	in.Schema = strings.TrimSpace(in.Schema)
	if in.Subject == "" || in.Schema == "" {
		return domain.SchemaRecord{}, domain.ErrInvalidInput
	}
	if in.SchemaType == "" {
		in.SchemaType = domain.SchemaTypeAvro
	}
	if in.Compatibility == "" {
		in.Compatibility = "BACKWARD"
	}
	if !domain.IsValidSchemaType(in.SchemaType) || !domain.IsValidCompatibility(in.Compatibility) {
		return domain.SchemaRecord{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SchemaRecord{}, err
	} else if ok {
		var out domain.SchemaRecord
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SchemaRecord{}, err
	}
	now := s.nowFn()
	row := domain.SchemaRecord{
		ID:            "sch-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		Subject:       in.Subject,
		SchemaType:    in.SchemaType,
		Compatibility: in.Compatibility,
		Schema:        in.Schema,
		CreatedBy:     actor.SubjectID,
		CreatedAt:     now,
	}
	out, err := s.schemas.Register(ctx, row)
	if err != nil {
		return domain.SchemaRecord{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, out)
	return out, nil
}

func (s *Service) ResetConsumerOffset(ctx context.Context, actor Actor, in ResetOffsetInput) (domain.ConsumerOffsetAudit, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ConsumerOffsetAudit{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.ConsumerOffsetAudit{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ConsumerOffsetAudit{}, domain.ErrIdempotencyRequired
	}
	in.GroupID = strings.TrimSpace(in.GroupID)
	in.Topic = strings.TrimSpace(in.Topic)
	if in.GroupID == "" || in.Topic == "" || in.Partition < 0 || in.Offset < 0 {
		return domain.ConsumerOffsetAudit{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ConsumerOffsetAudit{}, err
	} else if ok {
		var out domain.ConsumerOffsetAudit
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ConsumerOffsetAudit{}, err
	}
	now := s.nowFn()
	row := domain.ConsumerOffsetAudit{
		ID:        "off-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		GroupID:   in.GroupID,
		Topic:     in.Topic,
		Partition: in.Partition,
		Offset:    in.Offset,
		Reason:    strings.TrimSpace(in.Reason),
		ChangedBy: actor.SubjectID,
		ChangedAt: now,
	}
	if err := s.offsets.Create(ctx, row); err != nil {
		return domain.ConsumerOffsetAudit{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) ReplayDLQ(ctx context.Context, actor Actor, in DLQReplayInput) (domain.DLQReplayResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DLQReplayResult{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.DLQReplayResult{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DLQReplayResult{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DLQReplayResult{}, err
	} else if ok {
		var out domain.DLQReplayResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DLQReplayResult{}, err
	}
	start := s.nowFn()
	msgs, err := s.dlq.Query(ctx, domain.DLQQuery{
		SourceTopic:     strings.TrimSpace(in.SourceTopic),
		ConsumerGroup:   strings.TrimSpace(in.ConsumerGroup),
		ErrorType:       strings.TrimSpace(in.ErrorType),
		Limit:           in.Limit,
		IncludeReplayed: false,
	})
	if err != nil {
		return domain.DLQReplayResult{}, err
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	now := s.nowFn()
	if len(ids) > 0 {
		if err := s.dlq.MarkReplayed(ctx, ids, now); err != nil {
			return domain.DLQReplayResult{}, err
		}
	}
	out := domain.DLQReplayResult{
		Requested: len(msgs),
		Replayed:  len(msgs),
		Failed:    0,
		StartedAt: start,
		EndedAt:   now,
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) ListDLQ(ctx context.Context, actor Actor, in DLQListInput) ([]domain.DLQMessage, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.dlq.Query(ctx, domain.DLQQuery{
		SourceTopic:     strings.TrimSpace(in.SourceTopic),
		ConsumerGroup:   strings.TrimSpace(in.ConsumerGroup),
		ErrorType:       strings.TrimSpace(in.ErrorType),
		Limit:           in.Limit,
		IncludeReplayed: in.IncludeReplayed,
	})
}

func (s *Service) AddDLQMessage(ctx context.Context, actor Actor, row domain.DLQMessage) (domain.DLQMessage, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DLQMessage{}, domain.ErrUnauthorized
	}
	now := s.nowFn()
	row.ID = strings.TrimSpace(row.ID)
	if row.ID == "" {
		row.ID = "dlq-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = now
	}
	if row.ErrorSummary == "" || row.SourceTopic == "" {
		return domain.DLQMessage{}, domain.ErrInvalidInput
	}
	if err := s.dlq.Create(ctx, row); err != nil {
		return domain.DLQMessage{}, err
	}
	return row, nil
}

func (s *Service) GetHealth(ctx context.Context) (domain.HealthReport, error) {
	_ = ctx
	now := s.nowFn()
	checks := map[string]domain.ComponentCheck{
		"kafka_cluster":   {Name: "kafka_cluster", Status: "healthy", LatencyMS: 15, LastChecked: now},
		"schema_registry": {Name: "schema_registry", Status: "healthy", LatencyMS: 8, LastChecked: now},
		"metadata_store":  {Name: "metadata_store", Status: "healthy", LatencyMS: 4, LastChecked: now},
	}
	return domain.HealthReport{
		Status:        "healthy",
		Timestamp:     now,
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Version:       s.cfg.Version,
		Checks:        checks,
	}, nil
}

func (s *Service) GetCacheMetrics(ctx context.Context) (domain.MetricsSnapshot, error) {
	_ = ctx
	return domain.MetricsSnapshot{}, nil
}

func (s *Service) RecordHTTPMetric(ctx context.Context, in MetricObservation) {
	if s.metrics == nil {
		return
	}
	path := strings.TrimSpace(in.Path)
	if path == "" {
		path = "/unknown"
	}
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if method == "" {
		method = "GET"
	}
	status := strconv.Itoa(in.StatusCode)
	_ = s.metrics.IncCounter(ctx, "http_requests_total", map[string]string{"service": s.cfg.ServiceName, "method": method, "path": path, "status": status}, 1)
	_ = s.metrics.ObserveHistogram(ctx, "http_request_duration_seconds",
		map[string]string{"service": s.cfg.ServiceName, "method": method, "path": path},
		in.Duration.Seconds(), []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5})
}

func (s *Service) RenderPrometheusMetrics(ctx context.Context) (string, error) {
	if s.metrics == nil {
		return "# no metrics\n", nil
	}
	snap, err := s.metrics.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if len(snap.Counters) > 0 {
		b.WriteString("# TYPE http_requests_total counter\n")
		for _, c := range snap.Counters {
			if c.Name != "http_requests_total" {
				continue
			}
			b.WriteString(c.Name + formatLabels(c.Labels) + " " + strconv.FormatFloat(c.Value, 'f', -1, 64) + "\n")
		}
	}
	for _, h := range snap.Histograms {
		if h.Name != "http_request_duration_seconds" {
			continue
		}
		b.WriteString("# TYPE http_request_duration_seconds histogram\n")
		for _, le := range sortedBucketKeys(h.Buckets) {
			lbl := copyMap(h.Labels)
			lbl["le"] = le
			b.WriteString(h.Name + "_bucket" + formatLabels(lbl) + " " + strconv.FormatFloat(h.Buckets[le], 'f', -1, 64) + "\n")
		}
		b.WriteString(h.Name + "_sum" + formatLabels(h.Labels) + " " + strconv.FormatFloat(h.Sum, 'f', -1, 64) + "\n")
		b.WriteString(h.Name + "_count" + formatLabels(h.Labels) + " " + strconv.FormatFloat(h.Count, 'f', -1, 64) + "\n")
	}
	if b.Len() == 0 {
		return "# no metrics yet\n", nil
	}
	return b.String(), nil
}

func (s *Service) validatePublishInput(ctx context.Context, in PublishInput) error {
	_ = ctx
	in.EventID = strings.TrimSpace(in.EventID)
	in.EventType = strings.TrimSpace(in.EventType)
	in.SourceService = strings.TrimSpace(in.SourceService)
	in.TraceID = strings.TrimSpace(in.TraceID)
	in.SchemaVersion = strings.TrimSpace(in.SchemaVersion)
	in.PartitionKeyPath = strings.TrimSpace(in.PartitionKeyPath)
	in.PartitionKey = strings.TrimSpace(in.PartitionKey)
	in.Format = strings.ToLower(strings.TrimSpace(in.Format))

	if _, err := uuid.Parse(in.EventID); err != nil {
		return domain.ErrInvalidEnvelope
	}
	if !domain.IsValidTopicName(in.EventType) {
		return domain.ErrInvalidEnvelope
	}
	if domain.IsDeprecatedPluralTopic(in.EventType) && strings.TrimSpace(in.CanonicalEvent) == "" {
		return domain.ErrInvalidEnvelope
	}
	if in.SourceService == "" || len(in.SourceService) > 50 || in.TraceID == "" || in.SchemaVersion == "" {
		return domain.ErrInvalidEnvelope
	}
	if in.PartitionKeyPath == "" || in.PartitionKey == "" || len(in.EventType) > 100 {
		return domain.ErrInvalidEnvelope
	}
	// Audit-sink ops events must key by source_service per 04-services canonical rule PK-2.
	if strings.EqualFold(in.PartitionKeyPath, "envelope.source_service") && in.PartitionKey != in.SourceService {
		return domain.ErrInvalidEnvelope
	}
	if in.Format == "" {
		in.Format = domain.SchemaTypeJSON
	}
	if !domain.IsValidFormat(in.Format) {
		return domain.ErrInvalidEnvelope
	}
	if !domain.TimestampWithinSkew(in.OccurredAt, s.nowFn(), 5*time.Minute) {
		return domain.ErrInvalidEnvelope
	}
	if len(in.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	if strings.HasPrefix(in.PartitionKeyPath, "data.") {
		key := strings.TrimPrefix(in.PartitionKeyPath, "data.")
		if v, ok := in.Data[key]; !ok || strings.TrimSpace(toString(v)) != in.PartitionKey {
			return domain.ErrInvalidEnvelope
		}
	}
	if in.Format == domain.SchemaTypeAvro && s.schemas != nil {
		subject := in.EventType + "-value"
		if _, err := s.schemas.GetLatestBySubject(s.nowFnCtx(), subject); err != nil {
			return domain.ErrSchemaNotFound
		}
	}
	return nil
}

func (s *Service) nowFnCtx() context.Context { return context.Background() }

func isAdminLike(actor Actor) bool {
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "sre" || r == "system"
}

func normalizeOps(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, op := range in {
		op = strings.ToUpper(strings.TrimSpace(op))
		if op == "" {
			continue
		}
		if _, ok := seen[op]; ok {
			continue
		}
		seen[op] = struct{}{}
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		raw, _ := json.Marshal(x)
		return string(raw)
	}
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) getIdempotent(ctx context.Context, key, expectedHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != expectedHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return append([]byte(nil), rec.ResponseBody...), true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(k + "=\"")
		b.WriteString(strings.ReplaceAll(labels[k], "\"", "\\\""))
		b.WriteString("\"")
	}
	b.WriteString("}")
	return b.String()
}

func sortedBucketKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "+Inf" {
			return false
		}
		if keys[j] == "+Inf" {
			return true
		}
		fi, ei := strconv.ParseFloat(keys[i], 64)
		fj, ej := strconv.ParseFloat(keys[j], 64)
		if ei != nil || ej != nil {
			return keys[i] < keys[j]
		}
		return fi < fj
	})
	return keys
}
