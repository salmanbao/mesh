package contracts

import "encoding/json"

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type PutCacheRequest struct {
	Value      json.RawMessage `json:"value"`
	TTLSeconds int             `json:"ttl_seconds,omitempty"`
}

type GetCacheResponse struct {
	Key        string          `json:"key"`
	Value      json.RawMessage `json:"value,omitempty"`
	TTLSeconds int             `json:"ttl_seconds"`
	Found      bool            `json:"found"`
}

type PutCacheResponse struct {
	Key        string `json:"key"`
	StoredAt   string `json:"stored_at"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type DeleteCacheResponse struct {
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

type InvalidateCacheRequest struct {
	Keys []string `json:"keys"`
}

type InvalidateCacheResponse struct {
	InvalidatedCount int `json:"invalidated_count"`
}

type MetricsResponse struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}
