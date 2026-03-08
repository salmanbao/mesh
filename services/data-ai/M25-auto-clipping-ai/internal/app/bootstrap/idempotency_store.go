package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const deployModelIdempotencyTTL = 7 * 24 * time.Hour

type deployIdempotencyStore struct {
	path string
	ttl  time.Duration

	mu      sync.Mutex
	records map[string]deployedModelRecord
}

func newDeployIdempotencyStore(path string, ttl time.Duration) (*deployIdempotencyStore, error) {
	storePath := strings.TrimSpace(path)
	if storePath == "" {
		return nil, fmt.Errorf("idempotency store path is required")
	}
	if ttl <= 0 {
		ttl = deployModelIdempotencyTTL
	}

	store := &deployIdempotencyStore{
		path:    storePath,
		ttl:     ttl,
		records: map[string]deployedModelRecord{},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *deployIdempotencyStore) replayFor(
	key string,
	requestHash string,
	now time.Time,
) (deployModelResponse, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dirty := s.pruneExpiredLocked(now)

	row, ok := s.records[key]
	if !ok {
		if dirty {
			if err := s.persistLocked(); err != nil {
				return deployModelResponse{}, false, err
			}
		}
		return deployModelResponse{}, false, nil
	}
	if row.RequestHash != requestHash {
		return deployModelResponse{}, false, errIdempotencyCollision
	}
	if dirty {
		if err := s.persistLocked(); err != nil {
			return deployModelResponse{}, false, err
		}
	}
	return row.Response, true, nil
}

func (s *deployIdempotencyStore) store(
	key string,
	requestHash string,
	response deployModelResponse,
	now time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked(now)

	if existing, ok := s.records[key]; ok {
		if existing.RequestHash != requestHash {
			return errIdempotencyCollision
		}
		return nil
	}

	s.records[key] = deployedModelRecord{
		RequestHash: requestHash,
		Response:    response,
		ExpiresAt:   now.Add(s.ttl),
	}
	return s.persistLocked()
}

func (s *deployIdempotencyStore) pruneExpiredLocked(now time.Time) bool {
	changed := false
	for key, row := range s.records {
		if now.After(row.ExpiresAt) {
			delete(s.records, key)
			changed = true
		}
	}
	return changed
}

func (s *deployIdempotencyStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("prepare idempotency store dir: %w", err)
	}

	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.records = map[string]deployedModelRecord{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read idempotency store: %w", err)
	}
	if len(raw) == 0 {
		s.records = map[string]deployedModelRecord{}
		return nil
	}

	var payload deployIdempotencyFile
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode idempotency store: %w", err)
	}

	s.records = make(map[string]deployedModelRecord, len(payload.Records))
	for key, row := range payload.Records {
		row.Response = normalizeDeployModelResponse(row.Response)
		s.records[key] = row
	}
	return nil
}

func (s *deployIdempotencyStore) persistLocked() error {
	payload := deployIdempotencyFile{
		Records: make(map[string]deployedModelRecord, len(s.records)),
	}
	for key, row := range s.records {
		row.Response = normalizeDeployModelResponse(row.Response)
		payload.Records[key] = row
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode idempotency store: %w", err)
	}

	tempFile := s.path + ".tmp"
	if err := os.WriteFile(tempFile, raw, 0o600); err != nil {
		return fmt.Errorf("write idempotency temp store: %w", err)
	}
	if err := os.Rename(tempFile, s.path); err != nil {
		_ = os.Remove(s.path)
		if errRetry := os.Rename(tempFile, s.path); errRetry != nil {
			_ = os.Remove(tempFile)
			return fmt.Errorf("commit idempotency store: %w", errRetry)
		}
	}
	return nil
}

func normalizeDeployModelResponse(resp deployModelResponse) deployModelResponse {
	resp.ModelVersionID = strings.TrimSpace(resp.ModelVersionID)
	resp.DeploymentStatus = strings.TrimSpace(resp.DeploymentStatus)
	resp.DeployedAt = strings.TrimSpace(resp.DeployedAt)
	resp.Message = strings.TrimSpace(resp.Message)
	return resp
}

type deployIdempotencyFile struct {
	Records map[string]deployedModelRecord `json:"records"`
}

type deployedModelRecord struct {
	RequestHash string              `json:"request_hash"`
	Response    deployModelResponse `json:"response"`
	ExpiresAt   time.Time           `json:"expires_at"`
}

var errIdempotencyCollision = errors.New("idempotency key reused with different payload")
