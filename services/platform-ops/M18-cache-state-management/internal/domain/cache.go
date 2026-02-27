package domain

import "time"

type CacheEntry struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
	StoredAt  time.Time
}

type CacheItem struct {
	Key        string
	Value      []byte
	Found      bool
	TTLSeconds int
}

type CacheMetrics struct {
	Hits            int64
	Misses          int64
	Evictions       int64
	MemoryUsedBytes int64
}

type ComponentCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LatencyMS   int       `json:"latency_ms"`
	LastChecked time.Time `json:"last_checked"`
}

type HealthReport struct {
	Status        string                    `json:"status"`
	Timestamp     time.Time                 `json:"timestamp"`
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Version       string                    `json:"version"`
	Checks        map[string]ComponentCheck `json:"checks"`
}
