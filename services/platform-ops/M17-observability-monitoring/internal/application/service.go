package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/domain"
)

func (s *Service) GetHealth(ctx context.Context) (domain.HealthReport, error) {
	rows, err := s.components.List(ctx)
	if err != nil {
		return domain.HealthReport{}, err
	}
	checks := map[string]domain.ComponentCheck{}
	for _, row := range rows {
		checks[row.Name] = row
	}
	now := s.nowFn()
	ensure := func(name string, latency int, critical bool) {
		if _, ok := checks[name]; ok {
			return
		}
		checks[name] = domain.ComponentCheck{
			Name:        name,
			Status:      domain.StatusHealthy,
			Critical:    critical,
			LatencyMS:   latency,
			LastChecked: now,
		}
	}
	ensure("database", 12, true)
	ensure("redis", 2, true)
	if _, ok := checks["kafka"]; !ok {
		checks["kafka"] = domain.ComponentCheck{Name: "kafka", Status: domain.StatusHealthy, Critical: true, BrokersConnected: 3, LastChecked: now}
	}
	overall := domain.StatusHealthy
	for _, row := range checks {
		if !row.Critical {
			continue
		}
		if row.Status == domain.StatusUnhealthy {
			overall = domain.StatusUnhealthy
			break
		}
		if row.Status == domain.StatusDegraded && overall != domain.StatusUnhealthy {
			overall = domain.StatusUnhealthy
		}
	}
	return domain.HealthReport{
		Status:        overall,
		Timestamp:     now,
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Version:       s.cfg.Version,
		Checks:        checks,
	}, nil
}

func (s *Service) ListComponents(ctx context.Context) ([]domain.ComponentCheck, error) {
	rows, err := s.components.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}

func (s *Service) UpsertComponent(ctx context.Context, actor Actor, in UpsertComponentInput) (domain.ComponentCheck, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ComponentCheck{}, domain.ErrUnauthorized
	}
	if !isAdmin(actor) {
		return domain.ComponentCheck{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ComponentCheck{}, domain.ErrIdempotencyRequired
	}
	in.Name = strings.ToLower(strings.TrimSpace(in.Name))
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	if in.Name == "" || domain.NormalizeStatus(in.Status) == "" {
		return domain.ComponentCheck{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{
		"op": "upsert_component", "name": in.Name, "status": in.Status, "critical": in.Critical,
		"latency_ms": in.LatencyMS, "brokers_connected": in.BrokersConnected, "error": strings.TrimSpace(in.Error),
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ComponentCheck{}, err
	} else if ok {
		var out domain.ComponentCheck
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ComponentCheck{}, err
	}
	now := s.nowFn()
	row := domain.ComponentCheck{
		Name:        in.Name,
		Status:      in.Status,
		Critical:    true,
		Error:       strings.TrimSpace(in.Error),
		LastChecked: now,
		Metadata:    copyMap(in.Metadata),
	}
	if in.Critical != nil {
		row.Critical = *in.Critical
	}
	if in.LatencyMS != nil {
		row.LatencyMS = *in.LatencyMS
	}
	if in.BrokersConnected != nil {
		row.BrokersConnected = *in.BrokersConnected
	}
	if err := s.components.Upsert(ctx, row); err != nil {
		return domain.ComponentCheck{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
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
	status := "0"
	if in.StatusCode > 0 {
		status = strconvItoa(in.StatusCode)
	}
	labels := map[string]string{
		"service": s.cfg.ServiceName,
		"method":  method,
		"path":    path,
		"status":  status,
	}
	_ = s.metrics.IncCounter(ctx, "http_requests_total", labels, 1)
	_ = s.metrics.ObserveHistogram(ctx, "http_request_duration_seconds", map[string]string{
		"service": s.cfg.ServiceName,
		"method":  method,
		"path":    path,
	}, in.Duration.Seconds(), []float64{0.1, 0.5, 1.0, 2.5, 5.0})
}

func (s *Service) RenderPrometheusMetrics(ctx context.Context) (string, error) {
	snapshot, err := s.metrics.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if len(snapshot.Counters) > 0 {
		b.WriteString("# HELP http_requests_total Total HTTP requests\n")
		b.WriteString("# TYPE http_requests_total counter\n")
		for _, c := range snapshot.Counters {
			if c.Name != "http_requests_total" {
				continue
			}
			b.WriteString(c.Name)
			b.WriteString(formatLabels(c.Labels))
			b.WriteString(" ")
			b.WriteString(trimFloat(c.Value))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	for _, h := range snapshot.Histograms {
		if h.Name != "http_request_duration_seconds" {
			continue
		}
		b.WriteString("# HELP http_request_duration_seconds Request latency\n")
		b.WriteString("# TYPE http_request_duration_seconds histogram\n")
		ordered := sortedBucketKeys(h.Buckets)
		for _, le := range ordered {
			lbl := copyMap(h.Labels)
			lbl["le"] = le
			b.WriteString(h.Name + "_bucket" + formatLabels(lbl) + " " + trimFloat(h.Buckets[le]) + "\n")
		}
		b.WriteString(h.Name + "_sum" + formatLabels(h.Labels) + " " + trimFloat(h.Sum) + "\n")
		b.WriteString(h.Name + "_count" + formatLabels(h.Labels) + " " + trimFloat(h.Count) + "\n\n")
	}
	if b.Len() == 0 {
		return "# no metrics yet\n", nil
	}
	return b.String(), nil
}

func isAdmin(actor Actor) bool { return strings.ToLower(strings.TrimSpace(actor.Role)) == "admin" }

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

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
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
	return rec.ResponseBody, true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, v any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(v)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}

func strconvItoa(v int) string {
	return strconv.Itoa(v)
}

func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
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
		b.WriteString(k)
		b.WriteString("=\"")
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
		ai, errI := strconv.ParseFloat(keys[i], 64)
		aj, errJ := strconv.ParseFloat(keys[j], 64)
		if errI != nil && errJ != nil {
			return keys[i] < keys[j]
		}
		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}
		return ai < aj
	})
	return keys
}
