package pim

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	CollectionHarvestRuns = "harvest_runs"

	HarvestRunStatusRunning = "running"
	HarvestRunStatusSuccess = "success"
	HarvestRunStatusPartial = "partial"
	HarvestRunStatusFailed  = "failed"

	HarvestTriggerManual = "manual"
	HarvestTriggerCron   = "cron"
	HarvestTriggerAPI    = "api"
)

const (
	harvestRunFailureItemLimit = 30
	harvestRunSuccessKeepCount = 200
	harvestRunFailureKeepCount = 200
	harvestRunSuccessRetention = 60 * 24 * time.Hour
	harvestRunFailureRetention = 180 * 24 * time.Hour
	harvestRunStaleAfter       = 20 * time.Minute
)

type HarvestActionActor struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type HarvestOptions struct {
	TriggerType string             `json:"triggerType"`
	Actor       HarvestActionActor `json:"actor"`
}

type HarvestFailureItem struct {
	SKU       string `json:"sku,omitempty"`
	ProductID string `json:"productId,omitempty"`
	Step      string `json:"step"`
	Error     string `json:"error"`
}

type HarvestRun struct {
	ID               string               `json:"id"`
	TriggerType      string               `json:"triggerType"`
	TriggeredByEmail string               `json:"triggeredByEmail,omitempty"`
	TriggeredByName  string               `json:"triggeredByName,omitempty"`
	Connector        string               `json:"connector"`
	SourceMode       string               `json:"sourceMode,omitempty"`
	Status           string               `json:"status"`
	StartedAt        string               `json:"startedAt"`
	FinishedAt       string               `json:"finishedAt,omitempty"`
	DurationMs       int                  `json:"durationMs,omitempty"`
	Processed        int                  `json:"processed"`
	Created          int                  `json:"created"`
	Updated          int                  `json:"updated"`
	Skipped          int                  `json:"skipped"`
	Offline          int                  `json:"offline"`
	Failed           int                  `json:"failed"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	FailureItems     []HarvestFailureItem `json:"failureItems,omitempty"`
}

type harvestExecutionState struct {
	Result
	startedAt        time.Time
	lastPersistAt    time.Time
	lastPersistCount int
	errorMessage     string
	failureItems     []HarvestFailureItem
}

func (s *Service) createHarvestRun(app core.App, options HarvestOptions) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionHarvestRuns)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	startedAt := now.Format(time.RFC3339)
	record := core.NewRecord(collection)
	record.Set("trigger_type", normalizeHarvestTriggerType(options.TriggerType))
	record.Set("triggered_by_email", strings.TrimSpace(options.Actor.Email))
	record.Set("triggered_by_name", strings.TrimSpace(options.Actor.Name))
	record.Set("connector", strings.TrimSpace(s.cfg.Supplier.Connector))
	record.Set("source_mode", strings.TrimSpace(s.cfg.MiniApp.SourceMode))
	record.Set("status", HarvestRunStatusRunning)
	record.Set("started_at", startedAt)
	record.Set("finished_at", "")
	record.Set("duration_ms", 0)
	record.Set("processed_count", 0)
	record.Set("created_count", 0)
	record.Set("updated_count", 0)
	record.Set("skipped_count", 0)
	record.Set("offline_count", 0)
	record.Set("failed_count", 0)
	record.Set("error_message", "")
	if err := setJSON(record, "failure_items_json", []HarvestFailureItem{}); err != nil {
		return nil, err
	}
	return record, app.Save(record)
}

func (s *Service) finalizeHarvestRun(app core.App, record *core.Record, state harvestExecutionState) (HarvestRun, error) {
	if record == nil {
		return HarvestRun{}, nil
	}

	finishedAt := time.Now()
	status := harvestRunStatus(state)
	record.Set("status", status)
	record.Set("finished_at", finishedAt.Format(time.RFC3339))
	record.Set("duration_ms", int(finishedAt.Sub(state.startedAt).Milliseconds()))
	record.Set("processed_count", state.Processed)
	record.Set("created_count", state.Created)
	record.Set("updated_count", state.Updated)
	record.Set("skipped_count", state.Skipped)
	record.Set("offline_count", state.Offline)
	record.Set("failed_count", state.Failed)
	record.Set("error_message", strings.TrimSpace(state.errorMessage))
	if err := setJSON(record, "failure_items_json", truncateHarvestFailureItems(state.failureItems)); err != nil {
		return HarvestRun{}, err
	}
	if err := app.Save(record); err != nil {
		return HarvestRun{}, err
	}
	_ = pruneHarvestRuns(app)
	return harvestRunFromRecord(record), nil
}

func (s *Service) updateHarvestRunProgress(app core.App, record *core.Record, state *harvestExecutionState, force bool) error {
	if app == nil || record == nil || state == nil {
		return nil
	}
	now := time.Now()
	currentCount := state.Processed + state.Failed
	if !force {
		if currentCount == 0 {
			return nil
		}
		if currentCount != 1 && currentCount-state.lastPersistCount < 5 && now.Sub(state.lastPersistAt) < 2*time.Second {
			return nil
		}
	}

	record.Set("status", HarvestRunStatusRunning)
	record.Set("duration_ms", int(now.Sub(state.startedAt).Milliseconds()))
	record.Set("processed_count", state.Processed)
	record.Set("created_count", state.Created)
	record.Set("updated_count", state.Updated)
	record.Set("skipped_count", state.Skipped)
	record.Set("offline_count", state.Offline)
	record.Set("failed_count", state.Failed)
	record.Set("error_message", strings.TrimSpace(state.errorMessage))
	if err := setJSON(record, "failure_items_json", truncateHarvestFailureItems(state.failureItems)); err != nil {
		return err
	}
	if err := app.Save(record); err != nil {
		return err
	}
	state.lastPersistAt = now
	state.lastPersistCount = currentCount
	return nil
}

func ListHarvestRuns(app core.App, limit int) ([]HarvestRun, error) {
	if limit <= 0 {
		limit = 8
	}
	if err := recoverStaleHarvestRuns(app, harvestRunStaleAfter); err != nil {
		return nil, err
	}
	collection, err := app.FindCollectionByNameOrId(CollectionHarvestRuns)
	if err != nil {
		return []HarvestRun{}, nil
	}
	sortExpr := safeCollectionSortExpr(collection, "started_at")
	records, err := app.FindRecordsByFilter(CollectionHarvestRuns, "", sortExpr, limit, 0, nil)
	if err != nil {
		return nil, err
	}
	items := make([]HarvestRun, 0, len(records))
	for _, record := range records {
		items = append(items, harvestRunFromRecord(record))
	}
	return items, nil
}

func GetHarvestRun(app core.App, id string) (HarvestRun, error) {
	record, err := app.FindRecordById(CollectionHarvestRuns, strings.TrimSpace(id))
	if err != nil {
		return HarvestRun{}, err
	}
	return harvestRunFromRecord(record), nil
}

func FindRunningHarvestRun(app core.App) (HarvestRun, error) {
	if err := recoverStaleHarvestRuns(app, harvestRunStaleAfter); err != nil {
		return HarvestRun{}, err
	}
	collection, err := app.FindCollectionByNameOrId(CollectionHarvestRuns)
	if err != nil {
		return HarvestRun{}, nil
	}
	sortExpr := safeCollectionSortExpr(collection, "started_at")
	records, err := app.FindRecordsByFilter(
		CollectionHarvestRuns,
		"status = {:status}",
		sortExpr,
		1,
		0,
		map[string]any{"status": HarvestRunStatusRunning},
	)
	if err != nil {
		return HarvestRun{}, err
	}
	if len(records) == 0 {
		return HarvestRun{}, nil
	}
	return harvestRunFromRecord(records[0]), nil
}

func recoverStaleHarvestRuns(app core.App, maxAge time.Duration) error {
	if app == nil || maxAge <= 0 {
		return nil
	}
	records, err := app.FindRecordsByFilter(
		CollectionHarvestRuns,
		"status = {:status}",
		"-started_at",
		50,
		0,
		map[string]any{"status": HarvestRunStatusRunning},
	)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, record := range records {
		startedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(record.GetString("started_at")))
		if parseErr != nil || startedAt.IsZero() {
			startedAt = now
		}
		if now.Sub(startedAt) < maxAge {
			continue
		}
		record.Set("status", HarvestRunStatusFailed)
		record.Set("finished_at", now.Format(time.RFC3339))
		record.Set("duration_ms", int(now.Sub(startedAt).Milliseconds()))
		if strings.TrimSpace(record.GetString("error_message")) == "" {
			record.Set("error_message", fmt.Sprintf("harvest run marked stale after %d minutes without completion", int(maxAge/time.Minute)))
		}
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func harvestRunFromRecord(record *core.Record) HarvestRun {
	return HarvestRun{
		ID:               record.Id,
		TriggerType:      strings.TrimSpace(record.GetString("trigger_type")),
		TriggeredByEmail: strings.TrimSpace(record.GetString("triggered_by_email")),
		TriggeredByName:  strings.TrimSpace(record.GetString("triggered_by_name")),
		Connector:        strings.TrimSpace(record.GetString("connector")),
		SourceMode:       strings.TrimSpace(record.GetString("source_mode")),
		Status:           strings.TrimSpace(record.GetString("status")),
		StartedAt:        strings.TrimSpace(record.GetString("started_at")),
		FinishedAt:       strings.TrimSpace(record.GetString("finished_at")),
		DurationMs:       record.GetInt("duration_ms"),
		Processed:        record.GetInt("processed_count"),
		Created:          record.GetInt("created_count"),
		Updated:          record.GetInt("updated_count"),
		Skipped:          record.GetInt("skipped_count"),
		Offline:          record.GetInt("offline_count"),
		Failed:           record.GetInt("failed_count"),
		ErrorMessage:     strings.TrimSpace(record.GetString("error_message")),
		FailureItems:     decodeHarvestFailureItems(record.GetString("failure_items_json")),
	}
}

func harvestRunStatus(state harvestExecutionState) string {
	if strings.TrimSpace(state.errorMessage) != "" {
		return HarvestRunStatusFailed
	}
	if state.Failed > 0 {
		return HarvestRunStatusPartial
	}
	return HarvestRunStatusSuccess
}

func normalizeHarvestTriggerType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case HarvestTriggerManual:
		return HarvestTriggerManual
	case HarvestTriggerCron:
		return HarvestTriggerCron
	default:
		return HarvestTriggerAPI
	}
}

func appendHarvestFailure(items []HarvestFailureItem, item HarvestFailureItem) []HarvestFailureItem {
	item.Step = strings.TrimSpace(item.Step)
	item.Error = strings.TrimSpace(item.Error)
	item.SKU = strings.TrimSpace(item.SKU)
	item.ProductID = strings.TrimSpace(item.ProductID)
	if item.Step == "" || item.Error == "" {
		return items
	}
	if len(items) >= harvestRunFailureItemLimit {
		return items
	}
	return append(items, item)
}

func truncateHarvestFailureItems(items []HarvestFailureItem) []HarvestFailureItem {
	if len(items) <= harvestRunFailureItemLimit {
		return append([]HarvestFailureItem(nil), items...)
	}
	return append([]HarvestFailureItem(nil), items[:harvestRunFailureItemLimit]...)
}

func decodeHarvestFailureItems(raw string) []HarvestFailureItem {
	items := []HarvestFailureItem{}
	decodeTargetSyncJSONField(raw, &items)
	return items
}

func pruneHarvestRuns(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(CollectionHarvestRuns)
	if err != nil {
		return nil
	}
	sortExpr := safeCollectionSortExpr(collection, "started_at")
	records, err := app.FindRecordsByFilter(CollectionHarvestRuns, "", sortExpr, 500, 0, nil)
	if err != nil {
		return err
	}

	now := time.Now()
	successKept := 0
	failureKept := 0
	for _, record := range records {
		status := strings.ToLower(strings.TrimSpace(record.GetString("status")))
		startedAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(record.GetString("started_at")))
		shouldDelete := false
		switch status {
		case HarvestRunStatusSuccess, HarvestRunStatusPartial:
			successKept++
			if successKept > harvestRunSuccessKeepCount || (!startedAt.IsZero() && now.Sub(startedAt) > harvestRunSuccessRetention) {
				shouldDelete = true
			}
		case HarvestRunStatusFailed:
			failureKept++
			if failureKept > harvestRunFailureKeepCount || (!startedAt.IsZero() && now.Sub(startedAt) > harvestRunFailureRetention) {
				shouldDelete = true
			}
		default:
			if status != HarvestRunStatusRunning && strings.TrimSpace(record.GetString("started_at")) == "" {
				shouldDelete = true
			}
		}
		if shouldDelete {
			if err := app.Delete(record); err != nil {
				return err
			}
		}
	}

	return nil
}
