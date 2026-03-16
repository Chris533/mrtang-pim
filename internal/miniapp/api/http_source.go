package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mrtang-pim/internal/miniapp/model"
)

type HTTPSourceConfig struct {
	URL                 string
	AuthorizedAccountID string
	UserAgent           string
	Timeout             time.Duration
}

type HTTPSource struct {
	cfg    HTTPSourceConfig
	client *http.Client
}

func NewHTTPSource(cfg HTTPSourceConfig) *HTTPSource {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	return &HTTPSource{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *HTTPSource) FetchDataset(ctx context.Context) (*model.Dataset, error) {
	if strings.TrimSpace(s.cfg.URL) == "" {
		return nil, fmt.Errorf("miniapp source url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.URL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(s.cfg.UserAgent) != "" {
		req.Header.Set("User-Agent", s.cfg.UserAgent)
	}
	if strings.TrimSpace(s.cfg.AuthorizedAccountID) != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.AuthorizedAccountID)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request miniapp http source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("miniapp http source returned status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var dataset model.Dataset
	if err := json.NewDecoder(resp.Body).Decode(&dataset); err != nil {
		return nil, fmt.Errorf("decode miniapp http source: %w", err)
	}

	return &dataset, nil
}
