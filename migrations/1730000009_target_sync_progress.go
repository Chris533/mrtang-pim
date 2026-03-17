package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "target_sync_runs")
		if err != nil {
			return err
		}
		collection.Fields.Add(
			&core.NumberField{Name: "progress_total"},
			&core.NumberField{Name: "progress_done"},
			&core.TextField{Name: "current_stage", Max: 128},
			&core.TextField{Name: "current_item", Max: 512},
			&core.TextField{Name: "last_progress_at", Max: 64},
			&core.JSONField{Name: "progress_logs_json"},
		)
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := findOrCreateCollection(app, "target_sync_runs")
		if err != nil {
			return err
		}
		for _, name := range []string{"progress_total", "progress_done", "current_stage", "current_item", "last_progress_at", "progress_logs_json"} {
			if field := collection.Fields.GetByName(name); field != nil {
				collection.Fields.RemoveById(field.GetId())
			}
		}
		return app.Save(collection)
	})
}
