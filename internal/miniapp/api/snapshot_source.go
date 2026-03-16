package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"mrtang-pim/internal/miniapp/model"
)

type SnapshotSource struct {
	path string

	mu     sync.RWMutex
	loaded bool
	data   model.Dataset
}

func NewSnapshotSource(path string) *SnapshotSource {
	return &SnapshotSource{path: path}
}

func (s *SnapshotSource) FetchDataset(ctx context.Context) (*model.Dataset, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	if s.loaded {
		data := s.data
		s.mu.RUnlock()
		return &data, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded {
		data := s.data
		return &data, nil
	}

	body, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read miniapp snapshot: %w", err)
	}

	var dataset model.Dataset
	if err := json.Unmarshal(body, &dataset); err != nil {
		return nil, fmt.Errorf("decode miniapp snapshot: %w", err)
	}

	s.data = dataset
	s.loaded = true

	data := s.data
	return &data, nil
}
