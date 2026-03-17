package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "source_asset_jobs")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "job_type", Required: true, Max: 32},
			&core.TextField{Name: "mode", Max: 32},
			&core.SelectField{Name: "status", Required: true, Values: []string{"running", "completed", "failed"}, MaxSelect: 1},
			&core.NumberField{Name: "total"},
			&core.NumberField{Name: "processed"},
			&core.NumberField{Name: "failed_count"},
			&core.TextField{Name: "current_item", Max: 512},
			&core.TextField{Name: "started_at", Max: 64},
			&core.TextField{Name: "finished_at", Max: 64},
			&core.TextField{Name: "error", Max: 1000},
			&core.JSONField{Name: "logs_json"},
		)

		collection.AddIndex("idx_source_asset_jobs_type", false, "job_type, status", "")
		return app.Save(collection)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "source_asset_jobs")
	})
}
