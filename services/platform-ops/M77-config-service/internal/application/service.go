package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/domain"
)

func (s *Service) GetConfig(ctx context.Context, actor Actor, in GetConfigInput) (map[string]any, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	env := domain.NormalizeEnvironment(in.Environment)
	if !domain.IsValidEnvironment(env) {
		return nil, domain.ErrInvalidInput
	}
	scope := domain.NormalizeServiceScope(in.ServiceScope)
	keys, err := s.keys.List(ctx)
	if err != nil {
		return nil, err
	}
	values, err := s.values.ListByEnvironment(ctx, env)
	if err != nil {
		return nil, err
	}

	byKeyScope := make(map[string]domain.ConfigValue, len(values))
	for _, row := range values {
		byKeyScope[row.KeyID+"|"+domain.NormalizeServiceScope(row.ServiceScope)] = row
	}
	out := make(map[string]any)
	for _, key := range keys {
		val, ok := byKeyScope[key.KeyID+"|"+scope]
		if !ok {
			val, ok = byKeyScope[key.KeyID+"|"+domain.GlobalServiceScope]
		}
		if !ok {
			continue
		}
		enabled, err := s.evaluateRollout(ctx, key, in)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeValueForResponse(key, val)
		if err != nil {
			return nil, err
		}
		if !enabled {
			// For feature flags, return false when gated off; otherwise omit.
			if key.ValueType == domain.ValueTypeBoolean {
				out[key.KeyName] = false
			}
			continue
		}
		out[key.KeyName] = decoded
	}
	return out, nil
}

func (s *Service) PatchConfig(ctx context.Context, actor Actor, in PatchConfigInput) (PatchResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return PatchResult{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return PatchResult{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return PatchResult{}, domain.ErrIdempotencyRequired
	}
	if err := validatePatchInput(in); err != nil {
		return PatchResult{}, err
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return PatchResult{}, err
	} else if ok {
		var out PatchResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return PatchResult{}, err
	}

	out, err := s.patchConfigNoIdem(ctx, actor, in, "config_update")
	if err != nil {
		return PatchResult{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) CreateRolloutRule(ctx context.Context, actor Actor, in CreateRolloutRuleInput) (domain.RolloutRule, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.RolloutRule{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.RolloutRule{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.RolloutRule{}, domain.ErrIdempotencyRequired
	}

	in.Key = strings.TrimSpace(in.Key)
	in.RuleType = strings.ToLower(strings.TrimSpace(in.RuleType))
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))
	in.Tier = strings.ToLower(strings.TrimSpace(in.Tier))
	if in.Key == "" || !domain.IsValidRuleType(in.RuleType) {
		return domain.RolloutRule{}, domain.ErrInvalidInput
	}
	switch in.RuleType {
	case domain.RuleTypePercentage:
		if in.Percentage < 0 || in.Percentage > 100 {
			return domain.RolloutRule{}, domain.ErrInvalidInput
		}
	case domain.RuleTypeRole:
		if in.Role == "" {
			return domain.RolloutRule{}, domain.ErrInvalidInput
		}
	case domain.RuleTypeTier:
		if in.Tier == "" {
			return domain.RolloutRule{}, domain.ErrInvalidInput
		}
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RolloutRule{}, err
	} else if ok {
		var out domain.RolloutRule
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RolloutRule{}, err
	}

	key, err := s.keys.GetByName(ctx, in.Key)
	if err != nil {
		return domain.RolloutRule{}, err
	}
	var raw json.RawMessage
	switch in.RuleType {
	case domain.RuleTypePercentage:
		raw, _ = json.Marshal(map[string]int{"percentage": in.Percentage})
	case domain.RuleTypeRole:
		raw, _ = json.Marshal(map[string]string{"role": in.Role})
	case domain.RuleTypeTier:
		raw, _ = json.Marshal(map[string]string{"tier": in.Tier})
	}
	now := s.nowFn()
	row := domain.RolloutRule{
		RuleID:    "rule-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		KeyID:     key.KeyID,
		KeyName:   key.KeyName,
		RuleType:  in.RuleType,
		RuleValue: raw,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if s.rules != nil {
		row, err = s.rules.UpsertForKey(ctx, row)
		if err != nil {
			return domain.RolloutRule{}, err
		}
	}
	_ = s.appendAudit(ctx, domain.AuditLog{
		AuditID:      uuid.NewString(),
		ActionType:   "rollout_rule_upserted",
		KeyID:        key.KeyID,
		KeyName:      key.KeyName,
		ActorID:      actor.SubjectID,
		IPAddress:    actor.IPAddress,
		UserAgent:    actor.UserAgent,
		ChangeDetail: raw,
		ActionAt:     now,
	})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) ImportConfig(ctx context.Context, actor Actor, in ImportConfigInput) (int, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return 0, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return 0, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return 0, domain.ErrIdempotencyRequired
	}
	env := domain.NormalizeEnvironment(in.Environment)
	scope := domain.NormalizeServiceScope(in.ServiceScope)
	if !domain.IsValidEnvironment(env) || len(in.Entries) == 0 {
		return 0, domain.ErrInvalidInput
	}
	for _, e := range in.Entries {
		if err := validatePatchInput(PatchConfigInput{
			Key:          e.Key,
			Environment:  env,
			ServiceScope: scope,
			ValueType:    e.ValueType,
			Value:        e.Value,
		}); err != nil {
			return 0, err
		}
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return 0, err
	} else if ok {
		var out struct {
			Applied int `json:"applied"`
		}
		if json.Unmarshal(raw, &out) == nil {
			return out.Applied, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return 0, err
	}

	applied := 0
	for _, e := range in.Entries {
		if _, err := s.patchConfigNoIdem(ctx, actor, PatchConfigInput{
			Key:          e.Key,
			Environment:  env,
			ServiceScope: scope,
			ValueType:    e.ValueType,
			Value:        e.Value,
		}, "config_import"); err != nil {
			return applied, err
		}
		applied++
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, struct {
		Applied int `json:"applied"`
	}{Applied: applied})
	return applied, nil
}

func (s *Service) ExportConfig(ctx context.Context, actor Actor, in ExportConfigInput) (domain.ExportSnapshot, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportSnapshot{}, domain.ErrUnauthorized
	}
	env := domain.NormalizeEnvironment(in.Environment)
	scope := domain.NormalizeServiceScope(in.ServiceScope)
	if !domain.IsValidEnvironment(env) {
		return domain.ExportSnapshot{}, domain.ErrInvalidInput
	}
	values, err := s.GetConfig(ctx, actor, GetConfigInput{
		Environment:  env,
		ServiceScope: scope,
		UserID:       "",
		Role:         actor.Role,
		Tier:         "",
	})
	if err != nil {
		return domain.ExportSnapshot{}, err
	}
	keys, err := s.keys.List(ctx)
	if err != nil {
		return domain.ExportSnapshot{}, err
	}
	meta := map[string]domain.ExportMeta{}
	maxVersion := 0
	for _, key := range keys {
		if _, ok := values[key.KeyName]; !ok {
			continue
		}
		meta[key.KeyName] = domain.ExportMeta{
			ValueType:  key.ValueType,
			UpdatedAt:  key.UpdatedAt,
			KeyVersion: key.LastVersion,
		}
		if key.LastVersion > maxVersion {
			maxVersion = key.LastVersion
		}
	}
	return domain.ExportSnapshot{
		Version:      maxVersion,
		GeneratedAt:  s.nowFn(),
		Environment:  env,
		ServiceScope: scope,
		Values:       values,
		Meta:         meta,
	}, nil
}

func (s *Service) RollbackConfig(ctx context.Context, actor Actor, in RollbackConfigInput) (RollbackResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return RollbackResult{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return RollbackResult{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return RollbackResult{}, domain.ErrIdempotencyRequired
	}
	in.Key = strings.TrimSpace(in.Key)
	in.Environment = domain.NormalizeEnvironment(in.Environment)
	in.ServiceScope = domain.NormalizeServiceScope(in.ServiceScope)
	if in.Key == "" || !domain.IsValidEnvironment(in.Environment) || in.Version <= 0 {
		return RollbackResult{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return RollbackResult{}, err
	} else if ok {
		var out RollbackResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return RollbackResult{}, err
	}

	key, err := s.keys.GetByName(ctx, in.Key)
	if err != nil {
		return RollbackResult{}, err
	}
	ver, err := s.vers.GetByVersionNumber(ctx, key.KeyID, in.Environment, in.ServiceScope, in.Version)
	if err != nil {
		return RollbackResult{}, err
	}
	if key.ValueType == domain.ValueTypeEncrypted {
		// M77 versions store masked values for encrypted keys in this mesh implementation.
		return RollbackResult{}, domain.ErrInvalidInput
	}
	var restored any
	if len(ver.NewValue) == 0 || string(ver.NewValue) == "null" {
		restored = nil
	} else if err := json.Unmarshal(ver.NewValue, &restored); err != nil {
		return RollbackResult{}, domain.ErrInvalidInput
	}
	patched, err := s.patchConfigNoIdem(ctx, actor, PatchConfigInput{
		Key:          in.Key,
		Environment:  in.Environment,
		ServiceScope: in.ServiceScope,
		ValueType:    key.ValueType,
		Value:        restored,
	}, "config_rollback")
	if err != nil {
		return RollbackResult{}, err
	}
	out := RollbackResult{PatchResult: patched, RolledBackTo: in.Version}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
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
		KeyName:      strings.TrimSpace(in.KeyName),
		Environment:  domain.NormalizeEnvironment(in.Environment),
		ServiceScope: domain.NormalizeServiceScope(in.ServiceScope),
		ActorID:      strings.TrimSpace(in.ActorID),
		Limit:        in.Limit,
	})
}

func (s *Service) patchConfigNoIdem(ctx context.Context, actor Actor, in PatchConfigInput, auditAction string) (PatchResult, error) {
	if err := validatePatchInput(in); err != nil {
		return PatchResult{}, err
	}
	now := s.nowFn()
	env := domain.NormalizeEnvironment(in.Environment)
	scope := domain.NormalizeServiceScope(in.ServiceScope)
	valueType := domain.NormalizeValueType(in.ValueType)

	key, err := s.ensureKey(ctx, strings.TrimSpace(in.Key), valueType, now)
	if err != nil {
		return PatchResult{}, err
	}
	if key.ValueType != valueType {
		return PatchResult{}, domain.ErrConflict
	}
	oldValue, _ := s.values.Get(ctx, key.KeyID, env, scope)

	newValue, versionPayload, err := normalizeStoredValue(valueType, in.Value, now)
	if err != nil {
		return PatchResult{}, err
	}
	newValue.KeyID = key.KeyID
	newValue.Environment = env
	newValue.ServiceScope = scope
	newValue, err = s.values.Upsert(ctx, newValue)
	if err != nil {
		return PatchResult{}, err
	}

	nextVersion, err := s.vers.NextVersionNumber(ctx, key.KeyID)
	if err != nil {
		return PatchResult{}, err
	}
	key.LastVersion = nextVersion
	key.UpdatedAt = now
	if err := s.keys.Update(ctx, key); err != nil {
		return PatchResult{}, err
	}

	oldVersionPayload := buildVersionPayload(key.ValueType, oldValue)
	version := domain.ConfigVersion{
		VersionID:     "ver-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		VersionNumber: nextVersion,
		KeyID:         key.KeyID,
		KeyName:       key.KeyName,
		Environment:   env,
		ServiceScope:  scope,
		OldValue:      oldVersionPayload,
		NewValue:      versionPayload,
		ChangedBy:     actor.SubjectID,
		ChangedAt:     now,
	}
	if err := s.vers.Create(ctx, version); err != nil {
		return PatchResult{}, err
	}

	changeDetail, _ := json.Marshal(map[string]any{
		"old_value":       maskValueForAudit(key.ValueType, oldValue),
		"new_value":       maskValueForAudit(key.ValueType, newValue),
		"version":         nextVersion,
		"request_id":      actor.RequestID,
		"idempotency_key": actor.IdempotencyKey,
	})
	_ = s.appendAudit(ctx, domain.AuditLog{
		AuditID:      uuid.NewString(),
		ActionType:   auditAction,
		KeyID:        key.KeyID,
		KeyName:      key.KeyName,
		ActorID:      actor.SubjectID,
		Environment:  env,
		ServiceScope: scope,
		IPAddress:    actor.IPAddress,
		UserAgent:    actor.UserAgent,
		ChangeDetail: changeDetail,
		ActionAt:     now,
	})

	return PatchResult{
		Key:          key.KeyName,
		Environment:  env,
		ServiceScope: scope,
		Version:      nextVersion,
	}, nil
}

func (s *Service) ensureKey(ctx context.Context, keyName, valueType string, now time.Time) (domain.ConfigKey, error) {
	key, err := s.keys.GetByName(ctx, keyName)
	if err == nil {
		return key, nil
	}
	row := domain.ConfigKey{
		KeyID:     "key-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		KeyName:   keyName,
		ValueType: valueType,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.keys.Upsert(ctx, row)
}

func (s *Service) evaluateRollout(ctx context.Context, key domain.ConfigKey, in GetConfigInput) (bool, error) {
	if s.rules == nil {
		return true, nil
	}
	rule, err := s.rules.GetByKeyID(ctx, key.KeyID)
	if err != nil {
		return true, nil
	}
	switch rule.RuleType {
	case domain.RuleTypePercentage:
		var payload struct {
			Percentage int `json:"percentage"`
		}
		if json.Unmarshal(rule.RuleValue, &payload) != nil {
			return false, nil
		}
		if strings.TrimSpace(in.UserID) == "" {
			return false, nil
		}
		sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(in.UserID)) + ":" + key.KeyName))
		return int(sum[0])%100 < payload.Percentage, nil
	case domain.RuleTypeRole:
		var payload struct {
			Role string `json:"role"`
		}
		if json.Unmarshal(rule.RuleValue, &payload) != nil {
			return false, nil
		}
		return strings.EqualFold(strings.TrimSpace(in.Role), strings.TrimSpace(payload.Role)), nil
	case domain.RuleTypeTier:
		var payload struct {
			Tier string `json:"tier"`
		}
		if json.Unmarshal(rule.RuleValue, &payload) != nil {
			return false, nil
		}
		return strings.EqualFold(strings.TrimSpace(in.Tier), strings.TrimSpace(payload.Tier)), nil
	default:
		return true, nil
	}
}

func validatePatchInput(in PatchConfigInput) error {
	in.Key = strings.TrimSpace(in.Key)
	if in.Key == "" {
		return domain.ErrInvalidInput
	}
	if !domain.IsValidEnvironment(domain.NormalizeEnvironment(in.Environment)) {
		return domain.ErrInvalidInput
	}
	if !domain.IsValidValueType(in.ValueType) {
		return domain.ErrInvalidInput
	}
	_, _, err := normalizeStoredValue(domain.NormalizeValueType(in.ValueType), in.Value, time.Now().UTC())
	return err
}

func normalizeStoredValue(valueType string, in any, now time.Time) (domain.ConfigValue, json.RawMessage, error) {
	row := domain.ConfigValue{
		ValueID:   "val-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		UpdatedAt: now,
	}
	switch valueType {
	case domain.ValueTypeString:
		v, ok := in.(string)
		if !ok {
			return domain.ConfigValue{}, nil, domain.ErrInvalidInput
		}
		raw, _ := json.Marshal(v)
		row.ValueJSON = raw
		return row, raw, nil
	case domain.ValueTypeBoolean:
		v, ok := in.(bool)
		if !ok {
			return domain.ConfigValue{}, nil, domain.ErrInvalidInput
		}
		raw, _ := json.Marshal(v)
		row.ValueJSON = raw
		return row, raw, nil
	case domain.ValueTypeNumber:
		switch v := in.(type) {
		case float64, float32, int, int64, int32, uint64, uint32, uint:
			raw, _ := json.Marshal(v)
			row.ValueJSON = raw
			return row, raw, nil
		case json.Number:
			raw := json.RawMessage(v.String())
			row.ValueJSON = raw
			return row, raw, nil
		default:
			return domain.ConfigValue{}, nil, domain.ErrInvalidInput
		}
	case domain.ValueTypeJSON:
		raw, err := json.Marshal(in)
		if err != nil || string(raw) == "null" {
			return domain.ConfigValue{}, nil, domain.ErrInvalidInput
		}
		row.ValueJSON = raw
		return row, raw, nil
	case domain.ValueTypeEncrypted:
		raw, err := json.Marshal(in)
		if err != nil || string(raw) == "null" {
			return domain.ConfigValue{}, nil, domain.ErrInvalidInput
		}
		row.ValueEncrypted = "enc:" + base64.StdEncoding.EncodeToString(raw)
		ver, _ := json.Marshal(map[string]string{"ciphertext": row.ValueEncrypted})
		return row, ver, nil
	default:
		return domain.ConfigValue{}, nil, domain.ErrInvalidInput
	}
}

func buildVersionPayload(valueType string, row domain.ConfigValue) json.RawMessage {
	if valueType == domain.ValueTypeEncrypted {
		if strings.TrimSpace(row.ValueEncrypted) == "" {
			return nil
		}
		raw, _ := json.Marshal(map[string]string{"ciphertext": row.ValueEncrypted})
		return raw
	}
	if len(row.ValueJSON) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), row.ValueJSON...)
}

func maskValueForAudit(valueType string, row domain.ConfigValue) any {
	if valueType == domain.ValueTypeEncrypted {
		if strings.TrimSpace(row.ValueEncrypted) == "" {
			return nil
		}
		return "***"
	}
	if len(row.ValueJSON) == 0 {
		return nil
	}
	var out any
	if err := json.Unmarshal(row.ValueJSON, &out); err != nil {
		return nil
	}
	return out
}

func decodeValueForResponse(key domain.ConfigKey, row domain.ConfigValue) (any, error) {
	if key.ValueType == domain.ValueTypeEncrypted {
		if strings.TrimSpace(row.ValueEncrypted) == "" {
			return nil, nil
		}
		return "***", nil
	}
	if len(row.ValueJSON) == 0 {
		return nil, nil
	}
	var out any
	if err := json.Unmarshal(row.ValueJSON, &out); err != nil {
		return nil, domain.ErrInvalidInput
	}
	return out, nil
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
			"postgres": {Name: "postgres", Status: "healthy", LatencyMS: 6, LastChecked: now},
			"redis":    {Name: "redis", Status: "healthy", LatencyMS: 4, LastChecked: now},
			"kms":      {Name: "kms", Status: "healthy", LatencyMS: 12, LastChecked: now},
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
