package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) publishEvent(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.PublishEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	ts := strings.TrimSpace(req.OccurredAt)
	if ts == "" {
		ts = strings.TrimSpace(req.Timestamp)
	}
	occurredAt, _ := time.Parse(time.RFC3339Nano, ts)
	out, err := h.service.PublishEvent(r.Context(), actor, application.PublishInput{
		EventID:          req.EventID,
		EventType:        req.EventType,
		CanonicalEvent:   req.CanonicalEvent,
		OccurredAt:       occurredAt,
		SourceService:    req.SourceService,
		TraceID:          req.TraceID,
		SchemaVersion:    req.SchemaVersion,
		PartitionKeyPath: req.PartitionKeyPath,
		PartitionKey:     req.PartitionKey,
		Format:           req.Format,
		Data:             req.Data,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusAccepted, "", contracts.PublishEventResponse{
		EventID:    out.EventID,
		EventType:  out.EventType,
		Topic:      out.Topic,
		Status:     out.Status,
		Format:     out.Format,
		AcceptedAt: out.AcceptedAt.Format(time.RFC3339Nano),
	})
}

func (h *Handler) createTopic(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreateTopic(r.Context(), actor, application.CreateTopicInput{
		TopicName:         req.TopicName,
		Partitions:        req.Partitions,
		ReplicationFactor: req.ReplicationFactor,
		RetentionDays:     req.RetentionDays,
		CleanupPolicy:     req.CleanupPolicy,
		CompressionType:   req.CompressionType,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", toTopicResponse(row))
}

func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	limit := parseLimit(r, 100)
	rows, err := h.service.ListTopics(r.Context(), actor, limit)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	out := make([]contracts.TopicResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTopicResponse(row))
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) createACL(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreateACL(r.Context(), actor, application.CreateACLInput{
		Principal:    req.Principal,
		ResourceType: req.ResourceType,
		ResourceName: req.ResourceName,
		PatternType:  req.PatternType,
		Operations:   req.Operations,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.ACLResponse{
		ID:           row.ID,
		Principal:    row.Principal,
		ResourceType: row.ResourceType,
		ResourceName: row.ResourceName,
		PatternType:  row.PatternType,
		Operations:   row.Operations,
		Status:       row.Status,
		CreatedAt:    row.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) listACLs(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	rows, err := h.service.ListACLs(r.Context(), actor, parseLimit(r, 100))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	out := make([]contracts.ACLResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, contracts.ACLResponse{
			ID:           row.ID,
			Principal:    row.Principal,
			ResourceType: row.ResourceType,
			ResourceName: row.ResourceName,
			PatternType:  row.PatternType,
			Operations:   row.Operations,
			Status:       row.Status,
			CreatedAt:    row.CreatedAt.Format(time.RFC3339),
		})
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) registerSchema(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.RegisterSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.RegisterSchema(r.Context(), actor, application.RegisterSchemaInput{
		Subject:       req.Subject,
		SchemaType:    req.SchemaType,
		Compatibility: req.Compatibility,
		Schema:        req.Schema,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.SchemaResponse{
		ID:            row.ID,
		Subject:       row.Subject,
		SchemaType:    row.SchemaType,
		Compatibility: row.Compatibility,
		Version:       row.Version,
		CreatedAt:     row.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) resetConsumerOffset(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	groupID := chi.URLParam(r, "group_id")
	var req contracts.ResetOffsetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.ResetConsumerOffset(r.Context(), actor, application.ResetOffsetInput{
		GroupID:   groupID,
		Topic:     req.Topic,
		Partition: req.Partition,
		Offset:    req.Offset,
		Reason:    req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.OffsetAuditResponse{
		ID:        row.ID,
		GroupID:   row.GroupID,
		Topic:     row.Topic,
		Partition: row.Partition,
		Offset:    row.Offset,
		Reason:    row.Reason,
		ChangedAt: row.ChangedAt.Format(time.RFC3339),
	})
}

func (h *Handler) replayDLQ(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ReplayDLQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	out, err := h.service.ReplayDLQ(r.Context(), actor, application.DLQReplayInput{
		SourceTopic:   req.SourceTopic,
		ConsumerGroup: req.ConsumerGroup,
		ErrorType:     req.ErrorType,
		Limit:         req.Limit,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.ReplayDLQResponse{
		Requested: out.Requested,
		Replayed:  out.Replayed,
		Failed:    out.Failed,
		StartedAt: out.StartedAt.Format(time.RFC3339),
		EndedAt:   out.EndedAt.Format(time.RFC3339),
	})
}

func (h *Handler) listDLQ(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	includeReplayed := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("include_replayed")), "true")
	rows, err := h.service.ListDLQ(r.Context(), actor, application.DLQListInput{
		SourceTopic:     strings.TrimSpace(r.URL.Query().Get("source_topic")),
		ConsumerGroup:   strings.TrimSpace(r.URL.Query().Get("consumer_group")),
		ErrorType:       strings.TrimSpace(r.URL.Query().Get("error_type")),
		Limit:           parseLimit(r, 100),
		IncludeReplayed: includeReplayed,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	out := make([]contracts.DLQMessageResponse, 0, len(rows))
	for _, row := range rows {
		resp := contracts.DLQMessageResponse{
			ID:            row.ID,
			SourceTopic:   row.SourceTopic,
			ConsumerGroup: row.ConsumerGroup,
			ErrorType:     row.ErrorType,
			ErrorSummary:  row.ErrorSummary,
			RetryCount:    row.RetryCount,
			EventID:       row.EventID,
			CreatedAt:     row.CreatedAt.Format(time.RFC3339),
		}
		if row.ReplayedAt != nil {
			resp.ReplayedAt = row.ReplayedAt.Format(time.RFC3339)
		}
		out = append(out, resp)
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) getHealth(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetHealth(r.Context())
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	status := http.StatusOK
	if out.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}
	writeSuccess(w, status, "", out)
}

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/plain") {
		payload, err := h.service.RenderPrometheusMetrics(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
		return
	}
	m, err := h.service.GetCacheMetrics(r.Context())
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.CacheMetricsResponse{
		Hits:            m.Hits,
		Misses:          m.Misses,
		Evictions:       m.Evictions,
		MemoryUsedBytes: m.MemoryUsedBytes,
	})
}

func parseLimit(r *http.Request, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func toTopicResponse(row domain.Topic) contracts.TopicResponse {
	return contracts.TopicResponse{
		ID:                row.ID,
		TopicName:         row.TopicName,
		Partitions:        row.Partitions,
		ReplicationFactor: row.ReplicationFactor,
		RetentionDays:     row.RetentionDays,
		CleanupPolicy:     row.CleanupPolicy,
		CompressionType:   row.CompressionType,
		Status:            row.Status,
		CreatedAt:         row.CreatedAt.Format(time.RFC3339),
	}
}
