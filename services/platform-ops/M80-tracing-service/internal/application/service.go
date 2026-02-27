package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/domain"
)

func (s *Service) IngestSpans(ctx context.Context, actor Actor, in IngestInput) (IngestResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return IngestResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return IngestResult{}, domain.ErrIdempotencyRequired
	}
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if format == "" {
		format = "zipkin"
	}
	if format != "otlp" && format != "zipkin" {
		return IngestResult{}, domain.ErrInvalidInput
	}
	if len(in.Spans) == 0 {
		return IngestResult{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return IngestResult{}, err
	} else if ok {
		var out IngestResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return IngestResult{}, err
	}

	now := s.nowFn()
	validSpans := make([]domain.SpanRecord, 0, len(in.Spans))
	allTags := make([]domain.SpanTag, 0)
	environment := ""
	rejected := 0
	for _, raw := range in.Spans {
		if !domain.IsHexTraceID(raw.TraceID) || !domain.IsHexSpanID(raw.SpanID) {
			rejected++
			continue
		}
		if raw.ParentSpanID != "" && !domain.IsHexSpanID(raw.ParentSpanID) {
			rejected++
			continue
		}
		if strings.TrimSpace(raw.ServiceName) == "" || strings.TrimSpace(raw.OperationName) == "" || raw.StartTime.IsZero() || raw.EndTime.IsZero() {
			rejected++
			continue
		}
		if raw.EndTime.Before(raw.StartTime) {
			rejected++
			continue
		}
		sp := domain.SpanRecord{
			SpanID:         strings.TrimSpace(raw.SpanID),
			TraceID:        strings.TrimSpace(raw.TraceID),
			ParentSpanID:   strings.TrimSpace(raw.ParentSpanID),
			ServiceName:    strings.TrimSpace(raw.ServiceName),
			OperationName:  strings.TrimSpace(raw.OperationName),
			StartTime:      raw.StartTime.UTC(),
			EndTime:        raw.EndTime.UTC(),
			DurationMS:     raw.EndTime.Sub(raw.StartTime).Milliseconds(),
			Error:          raw.Error,
			HTTPStatusCode: raw.HTTPStatusCode,
			CreatedAt:      now,
		}
		validSpans = append(validSpans, sp)
		if environment == "" {
			environment = strings.TrimSpace(raw.Environment)
		}
		for k, v := range raw.Tags {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" {
				continue
			}
			allTags = append(allTags, domain.SpanTag{TagID: sp.TraceID + ":" + sp.SpanID + ":" + k, SpanID: sp.SpanID, Key: k, Value: v, CreatedAt: now})
		}
	}
	if len(validSpans) == 0 {
		return IngestResult{}, domain.ErrInvalidInput
	}

	inserted, duplicates, err := s.spans.UpsertBatch(ctx, validSpans)
	if err != nil {
		return IngestResult{}, err
	}
	if len(allTags) > 0 {
		_ = s.tags.ReplaceForSpans(ctx, allTags)
	}
	_, err = s.traces.UpsertFromSpans(ctx, validSpans, environment)
	if err != nil {
		return IngestResult{}, err
	}
	_ = s.appendAudit(ctx, actor.SubjectID, "trace.ingest", "trace_batch", "", map[string]string{"format": format, "accepted": strconv.Itoa(inserted), "duplicates": strconv.Itoa(duplicates), "rejected": strconv.Itoa(rejected)})
	if s.metrics != nil {
		_ = s.metrics.IncCounter(ctx, "tracing_ingested_spans_total", map[string]string{"format": format}, float64(inserted))
		_ = s.metrics.IncCounter(ctx, "tracing_ingest_duplicates_total", map[string]string{"format": format}, float64(duplicates))
		if rejected > 0 {
			_ = s.metrics.IncCounter(ctx, "tracing_ingest_rejected_total", map[string]string{"format": format}, float64(rejected))
		}
	}
	out := IngestResult{Accepted: inserted, Duplicates: duplicates, Rejected: rejected}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 202, out)
	return out, nil
}

func (s *Service) SearchTraces(ctx context.Context, actor Actor, in SearchInput) ([]domain.TraceSearchHit, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	q := domain.TraceSearchQuery{TraceID: strings.TrimSpace(in.TraceID), ServiceName: strings.TrimSpace(in.ServiceName), ErrorOnly: in.ErrorOnly, DurationGTMS: in.DurationGTMS, Limit: in.Limit}
	return s.traces.Search(ctx, q)
}

func (s *Service) GetTraceDetail(ctx context.Context, actor Actor, traceID string) (domain.TraceDetail, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.TraceDetail{}, domain.ErrUnauthorized
	}
	traceID = strings.TrimSpace(traceID)
	if !domain.IsHexTraceID(traceID) {
		return domain.TraceDetail{}, domain.ErrInvalidInput
	}
	tr, err := s.traces.GetByID(ctx, traceID)
	if err != nil {
		if s.metrics != nil {
			_ = s.metrics.IncCounter(ctx, "cache_misses_total", map[string]string{"resource": "trace_detail"}, 1)
		}
		return domain.TraceDetail{}, err
	}
	spans, err := s.spans.ListByTraceID(ctx, traceID)
	if err != nil {
		return domain.TraceDetail{}, err
	}
	tags, _ := s.tags.ListByTraceID(ctx, traceID)
	if s.metrics != nil {
		_ = s.metrics.IncCounter(ctx, "cache_hits_total", map[string]string{"resource": "trace_detail"}, 1)
	}
	return domain.TraceDetail{Trace: tr, Spans: spans, Tags: tags}, nil
}

func (s *Service) CreateSamplingPolicy(ctx context.Context, actor Actor, in CreateSamplingPolicyInput) (domain.SamplingPolicy, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.SamplingPolicy{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.SamplingPolicy{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.SamplingPolicy{}, domain.ErrIdempotencyRequired
	}
	in.ServiceName = strings.TrimSpace(in.ServiceName)
	in.RuleType = strings.TrimSpace(in.RuleType)
	if in.ServiceName == "" || !domain.IsValidSamplingRuleType(in.RuleType) {
		return domain.SamplingPolicy{}, domain.ErrInvalidInput
	}
	if in.Probability != nil && (*in.Probability < 0 || *in.Probability > 1) {
		return domain.SamplingPolicy{}, domain.ErrInvalidInput
	}
	if in.RuleType == "rate_limited" && (in.MaxTracesPerMin == nil || *in.MaxTracesPerMin <= 0) {
		return domain.SamplingPolicy{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SamplingPolicy{}, err
	} else if ok {
		var out domain.SamplingPolicy
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SamplingPolicy{}, err
	}
	now := s.nowFn()
	row := domain.SamplingPolicy{PolicyID: "pol-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8], ServiceName: in.ServiceName, RuleType: in.RuleType, Probability: in.Probability, MaxTracesPerMin: in.MaxTracesPerMin, Enabled: true, CreatedAt: now, UpdatedAt: now}
	if err := s.policies.Create(ctx, row); err != nil {
		return domain.SamplingPolicy{}, err
	}
	_ = s.appendAudit(ctx, actor.SubjectID, "sampling_policy.created", "sampling_policy", row.PolicyID, map[string]string{"service_name": row.ServiceName, "rule_type": row.RuleType})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) ListSamplingPolicies(ctx context.Context, actor Actor) ([]domain.SamplingPolicy, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.policies.List(ctx)
}

func (s *Service) CreateExport(ctx context.Context, actor Actor, in CreateExportInput) (domain.ExportJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportJob{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ExportJob{}, domain.ErrIdempotencyRequired
	}
	in.TraceID = strings.TrimSpace(in.TraceID)
	if in.TraceID != "" && !domain.IsHexTraceID(in.TraceID) {
		return domain.ExportJob{}, domain.ErrInvalidInput
	}
	in.Format = strings.ToLower(strings.TrimSpace(in.Format))
	if in.Format == "" {
		in.Format = "parquet"
	}
	if in.Format != "parquet" && in.Format != "json" {
		return domain.ExportJob{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportJob{}, err
	} else if ok {
		var out domain.ExportJob
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportJob{}, err
	}
	now := s.nowFn()
	exportID := "exp-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	job := domain.ExportJob{ExportID: exportID, RequestedBy: actor.SubjectID, Status: domain.ExportStatusQueued, TraceID: in.TraceID, Format: in.Format, Filters: cleanMap(in.Filters), CreatedAt: now, UpdatedAt: now}
	if strings.TrimSpace(in.TraceID) != "" {
		job.OutputURI = "s3://traces/exports/" + exportID + "." + in.Format
		job.Status = domain.ExportStatusCompleted
	}
	if err := s.exports.Create(ctx, job); err != nil {
		return domain.ExportJob{}, err
	}
	_ = s.appendAudit(ctx, actor.SubjectID, "trace_export.created", "export", job.ExportID, map[string]string{"format": job.Format})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, job)
	return job, nil
}

func (s *Service) GetExport(ctx context.Context, actor Actor, exportID string) (domain.ExportJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportJob{}, domain.ErrUnauthorized
	}
	return s.exports.GetByID(ctx, strings.TrimSpace(exportID))
}

func (s *Service) GetHealth(ctx context.Context) (domain.HealthReport, error) {
	now := s.nowFn()
	traceCount, _ := s.traces.Count(ctx)
	checks := map[string]domain.ComponentCheck{
		"trace_store": {Name: "trace_store", Status: "healthy", LatencyMS: 10, LastChecked: now},
		"trace_index": {Name: "trace_index", Status: "healthy", LatencyMS: 8, LastChecked: now},
		"cache":       {Name: "cache", Status: "healthy", LatencyMS: 2, LastChecked: now},
	}
	if traceCount == 0 {
		checks["trace_store"] = domain.ComponentCheck{Name: "trace_store", Status: "degraded", LatencyMS: 10, LastChecked: now}
	}
	status := "healthy"
	for _, c := range checks {
		if c.Status != "healthy" {
			status = "degraded"
			break
		}
	}
	return domain.HealthReport{Status: status, Timestamp: now, UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()), Version: s.cfg.Version, Checks: checks}, nil
}

func (s *Service) GetCacheMetrics(ctx context.Context) (domain.MetricsSnapshot, error) {
	_ = ctx
	traceCount, _ := s.traces.Count(context.Background())
	return domain.MetricsSnapshot{Hits: 0, Misses: 0, Evictions: 0, MemoryUsedBytes: 0, StoredTraces: int64(traceCount)}, nil
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
	_ = s.metrics.ObserveHistogram(ctx, "http_request_duration_seconds", map[string]string{"service": s.cfg.ServiceName, "method": method, "path": path}, in.Duration.Seconds(), []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5})
}

func (s *Service) RenderPrometheusMetrics(ctx context.Context) (string, error) {
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

func (s *Service) appendAudit(ctx context.Context, actor, action, targetType, targetID string, meta map[string]string) error {
	if s.auditLogs == nil {
		return nil
	}
	return s.auditLogs.Create(ctx, domain.AuditLog{AuditID: uuid.NewString(), ActorUserID: strings.TrimSpace(actor), Action: strings.TrimSpace(action), TargetType: strings.TrimSpace(targetType), TargetID: strings.TrimSpace(targetID), Metadata: cleanMap(meta), OccurredAt: s.nowFn()})
}

func isAdminLike(actor Actor) bool {
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "sre" || r == "system"
}

func cleanMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
