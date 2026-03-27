package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "harvest_runs")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.SelectField{Name: "trigger_type", Required: true, Values: []string{"manual", "cron", "api"}, MaxSelect: 1},
			&core.TextField{Name: "triggered_by_email", Max: 255},
			&core.TextField{Name: "triggered_by_name", Max: 255},
			&core.TextField{Name: "connector", Max: 64},
			&core.TextField{Name: "source_mode", Max: 32},
			&core.SelectField{Name: "status", Required: true, Values: []string{"running", "success", "partial", "failed"}, MaxSelect: 1},
			&core.TextField{Name: "started_at", Max: 64},
			&core.TextField{Name: "finished_at", Max: 64},
			&core.NumberField{Name: "duration_ms"},
			&core.NumberField{Name: "processed_count"},
			&core.NumberField{Name: "created_count"},
			&core.NumberField{Name: "updated_count"},
			&core.NumberField{Name: "skipped_count"},
			&core.NumberField{Name: "offline_count"},
			&core.NumberField{Name: "failed_count"},
			&core.TextField{Name: "error_message", Max: 2000},
			&core.JSONField{Name: "failure_items_json"},
		)

		collection.AddIndex("idx_harvest_runs_started", false, "started_at", "")
		collection.AddIndex("idx_harvest_runs_status", false, "status", "")
		collection.AddIndex("idx_harvest_runs_trigger", false, "trigger_type", "")
		return app.Save(collection)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "harvest_runs")
	})
}
