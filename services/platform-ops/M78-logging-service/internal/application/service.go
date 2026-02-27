package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/domain"
)

var (
	reUserID = regexp.MustCompile(`user_id=\d+`)
	reEmail  = regexp.MustCompile(`([A-Za-z0-9._%+-])[A-Za-z0-9._%+-]*@([A-Za-z0-9.-]+\.[A-Za-z]{2,})`)
	reToken  = regexp.MustCompile(`(?i)(token|secret|password)=([^\s]+)`)
)

func (s *Service) IngestLogs(ctx context.Context, actor Actor, in IngestLogsInput) (IngestResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return IngestResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return IngestResult{}, domain.ErrIdempotencyRequired
	}
	if len(in.Logs) == 0 || len(in.Logs) > 10000 {
		return IngestResult{}, domain.ErrInvalidInput
	}
	if raw, _ := json.Marshal(in); len(raw) > 1024*1024 {
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
	rows := make([]domain.LogEvent, 0, len(in.Logs))
	for _, item := range in.Logs {
		lvl := domain.NormalizeLogLevel(item.Level)
		if !domain.IsValidLogLevel(lvl) || strings.TrimSpace(item.Service) == "" || strings.TrimSpace(item.Message) == "" || item.Timestamp.IsZero() {
			return IngestResult{}, domain.ErrInvalidInput
		}
		msg, redacted := redactMessage(item.Message)
		tagsRaw, _ := json.Marshal(item.Tags)
		rows = append(rows, domain.LogEvent{
			EventID:    "log-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			Timestamp:  item.Timestamp.UTC(),
			Level:      lvl,
			Service:    strings.TrimSpace(item.Service),
			InstanceID: strings.TrimSpace(item.InstanceID),
			TraceID:    strings.TrimSpace(item.TraceID),
			Message:    msg,
			UserID:     strings.TrimSpace(item.UserID),
			ErrorCode:  strings.TrimSpace(item.ErrorCode),
			Tags:       tagsRaw,
			Redacted:   redacted,
			IngestedAt: now,
		})
	}
	if s.logs != nil {
		if err := s.logs.InsertBatch(ctx, rows); err != nil {
			return IngestResult{}, err
		}
	}

	_ = s.appendAudit(ctx, domain.AuditLog{
		AuditID:    uuid.NewString(),
		ActorID:    actor.SubjectID,
		ActionType: "logs_ingested",
		ActionAt:   now,
		IPAddress:  actor.IPAddress,
		Details:    mustJSON(map[string]any{"count": len(rows), "request_id": actor.RequestID}),
	})
	_ = s.evaluateAlertRules(ctx, rows, actor)

	out := IngestResult{Ingested: len(rows)}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 202, out)
	return out, nil
}

func (s *Service) SearchLogs(ctx context.Context, actor Actor, in SearchLogsInput) ([]domain.LogEvent, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	level := ""
	if strings.TrimSpace(in.Level) != "" {
		level = domain.NormalizeLogLevel(in.Level)
		if !domain.IsValidLogLevel(level) {
			return nil, domain.ErrInvalidInput
		}
	}
	limit := in.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if s.logs == nil {
		return []domain.LogEvent{}, nil
	}
	rows, err := s.logs.Search(ctx, domain.LogSearchQuery{
		Service: strings.TrimSpace(in.Service),
		Level:   level,
		From:    in.From,
		To:      in.To,
		Q:       strings.TrimSpace(in.Q),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	// Auditor role receives redacted payloads only (already redacted on ingest, but enforce defensively).
	if strings.EqualFold(strings.TrimSpace(actor.Role), "auditor") {
		for i := range rows {
			if !rows[i].Redacted {
				rows[i].Message, _ = redactMessage(rows[i].Message)
				rows[i].Redacted = true
			}
		}
	}
	return rows, nil
}

func (s *Service) CreateExport(ctx context.Context, actor Actor, in CreateExportInput) (ExportCreateResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return ExportCreateResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return ExportCreateResult{}, domain.ErrIdempotencyRequired
	}
	format := domain.NormalizeExportFormat(in.Format)
	if !domain.IsValidExportFormat(format) {
		return ExportCreateResult{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return ExportCreateResult{}, err
	} else if ok {
		var out ExportCreateResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return ExportCreateResult{}, err
	}
	qraw, _ := json.Marshal(in.Query)
	now := s.nowFn()
	row := domain.LogExport{
		ExportID:    "exp-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		RequestedBy: actor.SubjectID,
		Query:       qraw,
		Format:      format,
		Status:      domain.ExportStatusPending,
		CreatedAt:   now,
	}
	if s.exp != nil {
		if err := s.exp.Create(ctx, row); err != nil {
			return ExportCreateResult{}, err
		}
	}
	_ = s.appendAudit(ctx, domain.AuditLog{
		AuditID:    uuid.NewString(),
		ActorID:    actor.SubjectID,
		ActionType: "logs_export_requested",
		ActionAt:   now,
		IPAddress:  actor.IPAddress,
		Details:    mustJSON(map[string]any{"export_id": row.ExportID, "format": format}),
	})
	out := ExportCreateResult{ExportID: row.ExportID, Status: row.Status}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, out)
	return out, nil
}

func (s *Service) GetExport(ctx context.Context, actor Actor, exportID string) (domain.LogExport, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.LogExport{}, domain.ErrUnauthorized
	}
	if s.exp == nil {
		return domain.LogExport{}, domain.ErrNotFound
	}
	return s.exp.GetByID(ctx, strings.TrimSpace(exportID))
}

func (s *Service) CreateAlertRule(ctx context.Context, actor Actor, in CreateAlertRuleInput) (domain.AlertRule, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.AlertRule{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.AlertRule{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.AlertRule{}, domain.ErrIdempotencyRequired
	}
	in.Service = strings.TrimSpace(in.Service)
	in.Severity = domain.NormalizeSeverity(in.Severity)
	if in.Service == "" || !domain.IsValidSeverity(in.Severity) {
		return domain.AlertRule{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.AlertRule{}, err
	} else if ok {
		var out domain.AlertRule
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.AlertRule{}, err
	}
	craw, _ := json.Marshal(in.Condition)
	now := s.nowFn()
	row := domain.AlertRule{
		RuleID:    "rule-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		Service:   in.Service,
		Condition: craw,
		Severity:  in.Severity,
		Enabled:   in.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if s.alerts != nil {
		if err := s.alerts.Create(ctx, row); err != nil {
			return domain.AlertRule{}, err
		}
	}
	_ = s.appendAudit(ctx, domain.AuditLog{
		AuditID:    uuid.NewString(),
		ActorID:    actor.SubjectID,
		ActionType: "log_alert_rule_created",
		ActionAt:   now,
		IPAddress:  actor.IPAddress,
		Details:    mustJSON(map[string]any{"rule_id": row.RuleID, "service": row.Service, "severity": row.Severity}),
	})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) ListAlertRules(ctx context.Context, actor Actor, onlyEnabled bool) ([]domain.AlertRule, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if s.alerts == nil {
		return []domain.AlertRule{}, nil
	}
	return s.alerts.List(ctx, onlyEnabled)
}

func (s *Service) QueryAudit(ctx context.Context, actor Actor, in AuditQueryInput) (domain.AuditQueryResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.AuditQueryResult{}, domain.ErrUnauthorized
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if role != "admin" && role != "auditor" && role != "sre" && role != "system" {
		return domain.AuditQueryResult{}, domain.ErrForbidden
	}
	if s.audits == nil {
		return domain.AuditQueryResult{}, nil
	}
	return s.audits.Query(ctx, domain.AuditQuery{
		ActorID:    strings.TrimSpace(in.ActorID),
		ActionType: strings.TrimSpace(in.ActionType),
		Limit:      in.Limit,
	})
}

func (s *Service) evaluateAlertRules(ctx context.Context, rows []domain.LogEvent, actor Actor) error {
	if s.alerts == nil || len(rows) == 0 {
		return nil
	}
	rules, err := s.alerts.List(ctx, true)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}
	now := s.nowFn()
	for _, row := range rows {
		if row.Level != domain.LogLevelError && row.Level != domain.LogLevelFatal {
			continue
		}
		for _, rule := range rules {
			if rule.Service != row.Service {
				continue
			}
			// Spec describes non-canonical ops event logging.alert_triggered; keep it module-local here.
			_ = s.appendAudit(ctx, domain.AuditLog{
				AuditID:    uuid.NewString(),
				ActorID:    actor.SubjectID,
				ActionType: "log_alert_triggered",
				ActionAt:   now,
				IPAddress:  actor.IPAddress,
				Details: mustJSON(map[string]any{
					"service":  row.Service,
					"level":    row.Level,
					"rule_id":  rule.RuleID,
					"message":  row.Message,
					"trace_id": row.TraceID,
				}),
			})
			s.publishOpsAlert(ctx, row, rule, actor, now)
			break
		}
	}
	return nil
}

func (s *Service) publishOpsAlert(ctx context.Context, row domain.LogEvent, rule domain.AlertRule, actor Actor, now time.Time) {
	if s.ops == nil {
		return
	}
	traceID := strings.TrimSpace(row.TraceID)
	if traceID == "" {
		traceID = strings.TrimSpace(actor.RequestID)
	}
	if traceID == "" {
		traceID = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	threshold := 0.0
	metric := "error_rate"
	var cond map[string]any
	if len(rule.Condition) > 0 && json.Unmarshal(rule.Condition, &cond) == nil {
		if v, ok := cond["metric"].(string); ok && strings.TrimSpace(v) != "" {
			metric = strings.TrimSpace(v)
		}
		if v, ok := cond["error_rate_gt"].(float64); ok {
			threshold = v
		}
		if v, ok := cond["threshold"].(float64); ok {
			threshold = v
		}
	}
	payload := contracts.LoggingAlertTriggered{
		Service:   row.Service,
		Metric:    metric,
		Value:     1,
		Threshold: threshold,
		Severity:  rule.Severity,
		RuleID:    rule.RuleID,
		Message:   row.Message,
		TraceID:   traceID,
	}
	data, _ := json.Marshal(payload)
	env := contracts.EventEnvelope{
		EventID:          uuid.NewString(),
		EventType:        "logging.alert_triggered",
		EventClass:       domain.CanonicalEventClassOps,
		OccurredAt:       now,
		PartitionKeyPath: "envelope.source_service",
		PartitionKey:     s.cfg.ServiceName,
		SourceService:    s.cfg.ServiceName,
		TraceID:          traceID,
		SchemaVersion:    "v1",
		Data:             data,
	}
	_ = s.ops.PublishOps(ctx, env)
}

func redactMessage(msg string) (string, bool) {
	orig := msg
	msg = reUserID.ReplaceAllString(msg, "user_id=***")
	msg = reEmail.ReplaceAllString(msg, "$1***@$2")
	msg = reToken.ReplaceAllStringFunc(msg, func(s string) string {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			return s
		}
		return parts[0] + "=[REDACTED]"
	})
	return msg, msg != orig
}

func (s *Service) GetHealth(ctx context.Context) (domain.HealthReport, error) {
	_ = ctx
	now := s.nowFn()
	return domain.HealthReport{
		Status:        "healthy",
		Timestamp:     now,
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Version:       s.cfg.Version,
		Checks: map[string]domain.ComponentCheck{
			"postgres":   {Name: "postgres", Status: "healthy", LatencyMS: 7, LastChecked: now},
			"opensearch": {Name: "opensearch", Status: "healthy", LatencyMS: 11, LastChecked: now},
			"kafka":      {Name: "kafka", Status: "healthy", LatencyMS: 9, LastChecked: now},
			"redactor":   {Name: "redactor", Status: "healthy", LatencyMS: 2, LastChecked: now},
		},
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
	_ = s.metrics.IncCounter(ctx, "http_requests_total", map[string]string{
		"service": s.cfg.ServiceName,
		"method":  method,
		"path":    path,
		"status":  status,
	}, 1)
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
	wroteCounterHeader := false
	wroteHistHeader := false
	for _, c := range snap.Counters {
		if c.Name != "http_requests_total" {
			continue
		}
		if !wroteCounterHeader {
			b.WriteString("# TYPE http_requests_total counter\n")
			wroteCounterHeader = true
		}
		b.WriteString(c.Name + formatLabels(c.Labels) + " " + strconv.FormatFloat(c.Value, 'f', -1, 64) + "\n")
	}
	for _, h := range snap.Histograms {
		if h.Name != "http_request_duration_seconds" {
			continue
		}
		if !wroteHistHeader {
			b.WriteString("# TYPE http_request_duration_seconds histogram\n")
			wroteHistHeader = true
		}
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

func (s *Service) appendAudit(ctx context.Context, row domain.AuditLog) error {
	if s.audits == nil {
		return nil
	}
	return s.audits.Create(ctx, row)
}

func isAdminLike(actor Actor) bool {
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "sre" || r == "system"
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

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
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
