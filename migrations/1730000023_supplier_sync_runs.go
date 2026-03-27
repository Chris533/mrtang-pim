package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "supplier_sync_runs")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.SelectField{Name: "status", Required: true, Values: []string{"running", "success", "partial", "failed"}, MaxSelect: 1},
			&core.TextField{Name: "started_at", Max: 64},
			&core.TextField{Name: "finished_at", Max: 64},
			&core.NumberField{Name: "duration_ms"},
			&core.NumberField{Name: "total_count"},
			&core.NumberField{Name: "processed_count"},
			&core.NumberField{Name: "failed_count"},
			&core.TextField{Name: "current_item", Max: 255},
			&core.TextField{Name: "error_message", Max: 2000},
		)

		collection.AddIndex("idx_supplier_sync_runs_started", false, "started_at", "")
		collection.AddIndex("idx_supplier_sync_runs_status", false, "status", "")
		return app.Save(collection)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "supplier_sync_runs")
	})
}
