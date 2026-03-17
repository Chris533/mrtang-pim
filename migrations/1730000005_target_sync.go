package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := upsertTargetSyncJobsCollection(app); err != nil {
			return err
		}
		return upsertTargetSyncRunsCollection(app)
	}, func(app core.App) error {
		if err := deleteCollectionIfExists(app, "target_sync_runs"); err != nil {
			return err
		}
		return deleteCollectionIfExists(app, "target_sync_jobs")
	})
}

func upsertTargetSyncJobsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "target_sync_jobs")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "job_key", Required: true, Max: 255},
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.SelectField{Name: "entity_type", Required: true, Values: []string{"category_tree", "products", "assets"}, MaxSelect: 1},
		&core.SelectField{Name: "scope_type", Required: true, Values: []string{"all", "top_level"}, MaxSelect: 1},
		&core.TextField{Name: "scope_key", Max: 128},
		&core.TextField{Name: "scope_label", Max: 255},
		&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "running", "success", "partial", "failed"}, MaxSelect: 1},
		&core.TextField{Name: "source_mode", Max: 32},
		&core.TextField{Name: "last_run_at", Max: 64},
		&core.TextField{Name: "last_success_at", Max: 64},
		&core.TextField{Name: "last_error", Max: 1000},
		&core.JSONField{Name: "config_json"},
	)

	collection.AddIndex("idx_target_sync_jobs_key", true, "job_key", "")
	collection.AddIndex("idx_target_sync_jobs_entity_scope", false, "entity_type, scope_type, scope_key", "")
	return app.Save(collection)
}

func upsertTargetSyncRunsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "target_sync_runs")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "job_key", Required: true, Max: 255},
		&core.TextField{Name: "job_name", Max: 255},
		&core.SelectField{Name: "entity_type", Required: true, Values: []string{"category_tree", "products", "assets"}, MaxSelect: 1},
		&core.SelectField{Name: "scope_type", Required: true, Values: []string{"all", "top_level"}, MaxSelect: 1},
		&core.TextField{Name: "scope_key", Max: 128},
		&core.TextField{Name: "scope_label", Max: 255},
		&core.SelectField{Name: "status", Required: true, Values: []string{"running", "success", "partial", "failed"}, MaxSelect: 1},
		&core.TextField{Name: "source_mode", Max: 32},
		&core.TextField{Name: "started_at", Max: 64},
		&core.TextField{Name: "finished_at", Max: 64},
		&core.TextField{Name: "triggered_by_email", Max: 255},
		&core.TextField{Name: "triggered_by_name", Max: 255},
		&core.NumberField{Name: "created_count"},
		&core.NumberField{Name: "updated_count"},
		&core.NumberField{Name: "unchanged_count"},
		&core.NumberField{Name: "missing_count"},
		&core.NumberField{Name: "scoped_node_count"},
		&core.TextField{Name: "error_message", Max: 1000},
		&core.JSONField{Name: "summary_json"},
	)

	collection.AddIndex("idx_target_sync_runs_job_key", false, "job_key", "")
	collection.AddIndex("idx_target_sync_runs_started", false, "started_at", "")
	return app.Save(collection)
}
