package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return upsertSourceActionLogsCollection(app)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "source_action_logs")
	})
}

func upsertSourceActionLogsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "source_action_logs")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "target_type", Required: true, Max: 32},
		&core.TextField{Name: "target_id", Required: true, Max: 64},
		&core.TextField{Name: "target_label", Max: 255},
		&core.TextField{Name: "action_type", Required: true, Max: 64},
		&core.SelectField{Name: "status", Required: true, Values: []string{"success", "failed"}, MaxSelect: 1},
		&core.TextField{Name: "message", Max: 500},
		&core.JSONField{Name: "details_json"},
	)

	collection.AddIndex("idx_source_action_logs_target", false, "target_type, target_id", "")
	collection.AddIndex("idx_source_action_logs_action", false, "action_type, status", "")
	return app.Save(collection)
}
