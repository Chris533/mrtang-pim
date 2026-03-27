package pim

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	CollectionSupplierSyncRuns = "supplier_sync_runs"

	SupplierSyncRunStatusRunning = "running"
	SupplierSyncRunStatusSuccess = "success"
	SupplierSyncRunStatusPartial = "partial"
	SupplierSyncRunStatusFailed  = "failed"

	supplierSyncRunStaleAfter = 30 * time.Minute
)

func (s *Service) createSupplierSyncRun(app core.App) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionSupplierSyncRuns)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	record := core.NewRecord(collection)
	record.Set("status", SupplierSyncRunStatusRunning)
	record.Set("started_at", now.Format(time.RFC3339))
	record.Set("finished_at", "")
	record.Set("duration_ms", 0)
	record.Set("total_count", 0)
	record.Set("processed_count", 0)
	record.Set("failed_count", 0)
	record.Set("current_item", "")
	record.Set("error_message", "")
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) updateSupplierSyncRunProgress(app core.App, record *core.Record, total int, processed int, failed int, current string) error {
	if app == nil || record == nil {
		return nil
	}
	record.Set("status", SupplierSyncRunStatusRunning)
	record.Set("total_count", total)
	record.Set("processed_count", processed)
	record.Set("failed_count", failed)
	record.Set("current_item", strings.TrimSpace(current))
	return app.Save(record)
}

func (s *Service) finalizeSupplierSyncRun(app core.App, record *core.Record, total int, processed int, failed int, errorMessage string) error {
	if app == nil || record == nil {
		return nil
	}
	startedAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(record.GetString("started_at")))
	finishedAt := time.Now().UTC()
	status := SupplierSyncRunStatusSuccess
	if strings.TrimSpace(errorMessage) != "" {
		status = SupplierSyncRunStatusFailed
	} else if failed > 0 {
		status = SupplierSyncRunStatusPartial
	}
	record.Set("status", status)
	record.Set("finished_at", finishedAt.Format(time.RFC3339))
	if !startedAt.IsZero() {
		record.Set("duration_ms", int(finishedAt.Sub(startedAt).Milliseconds()))
	}
	record.Set("total_count", total)
	record.Set("processed_count", processed)
	record.Set("failed_count", failed)
	record.Set("current_item", "")
	record.Set("error_message", strings.TrimSpace(errorMessage))
	return app.Save(record)
}

func supplierSyncProgressFromRecord(record *core.Record) SupplierSyncProgress {
	if record == nil {
		return SupplierSyncProgress{}
	}
	return SupplierSyncProgress{
		ID:          record.Id,
		Status:      strings.TrimSpace(record.GetString("status")),
		Total:       record.GetInt("total_count"),
		Processed:   record.GetInt("processed_count"),
		Failed:      record.GetInt("failed_count"),
		CurrentItem: strings.TrimSpace(record.GetString("current_item")),
		StartedAt:   strings.TrimSpace(record.GetString("started_at")),
		FinishedAt:  strings.TrimSpace(record.GetString("finished_at")),
		Error:       strings.TrimSpace(record.GetString("error_message")),
	}
}

func findRunningSupplierSyncRun(app core.App) (*core.Record, error) {
	if err := recoverStaleSupplierSyncRuns(app, supplierSyncRunStaleAfter); err != nil {
		return nil, err
	}
	collection, err := app.FindCollectionByNameOrId(CollectionSupplierSyncRuns)
	if err != nil {
		return nil, nil
	}
	sortExpr := safeCollectionSortExpr(collection, "started_at")
	records, err := app.FindRecordsByFilter(
		CollectionSupplierSyncRuns,
		"status = {:status}",
		sortExpr,
		1,
		0,
		map[string]any{"status": SupplierSyncRunStatusRunning},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func latestSupplierSyncRun(app core.App) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionSupplierSyncRuns)
	if err != nil {
		return nil, nil
	}
	sortExpr := safeCollectionSortExpr(collection, "started_at")
	records, err := app.FindRecordsByFilter(CollectionSupplierSyncRuns, "", sortExpr, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func recoverStaleSupplierSyncRuns(app core.App, maxAge time.Duration) error {
	if app == nil || maxAge <= 0 {
		return nil
	}
	records, err := app.FindRecordsByFilter(
		CollectionSupplierSyncRuns,
		"status = {:status}",
		"-started_at",
		20,
		0,
		map[string]any{"status": SupplierSyncRunStatusRunning},
	)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, record := range records {
		startedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(record.GetString("started_at")))
		if parseErr != nil || startedAt.IsZero() {
			startedAt = now
		}
		if now.Sub(startedAt) < maxAge {
			continue
		}
		record.Set("status", SupplierSyncRunStatusFailed)
		record.Set("finished_at", now.Format(time.RFC3339))
		record.Set("duration_ms", int(now.Sub(startedAt).Milliseconds()))
		if strings.TrimSpace(record.GetString("error_message")) == "" {
			record.Set("error_message", fmt.Sprintf("supplier sync run marked stale after %d minutes without completion", int(maxAge/time.Minute)))
		}
		record.Set("current_item", "")
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}
