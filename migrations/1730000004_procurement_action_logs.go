package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return upsertProcurementActionLogsCollection(app)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "procurement_action_logs")
	})
}

func upsertProcurementActionLogsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "procurement_action_logs")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "order_id", Required: true, Max: 64},
		&core.TextField{Name: "external_ref", Max: 255},
		&core.TextField{Name: "action_type", Required: true, Max: 64},
		&core.SelectField{Name: "status", Required: true, Values: []string{"success", "failed"}, MaxSelect: 1},
		&core.TextField{Name: "message", Max: 500},
		&core.TextField{Name: "actor_email", Max: 255},
		&core.TextField{Name: "actor_name", Max: 255},
		&core.TextField{Name: "note", Max: 1000},
		&core.JSONField{Name: "details_json"},
	)

	collection.AddIndex("idx_procurement_action_logs_order", false, "order_id", "")
	collection.AddIndex("idx_procurement_action_logs_actor", false, "actor_email", "")
	return app.Save(collection)
}
