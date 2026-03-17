package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "miniapp_action_logs")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "source_mode", Max: 32},
			&core.TextField{Name: "operation_id", Required: true, Max: 64},
			&core.TextField{Name: "operation_label", Required: true, Max: 255},
			&core.TextField{Name: "contract_id", Max: 255},
			&core.SelectField{Name: "status", Required: true, Values: []string{"success", "failed"}, MaxSelect: 1},
			&core.TextField{Name: "message", Max: 1000},
			&core.JSONField{Name: "request_json"},
			&core.JSONField{Name: "response_json"},
		)

		collection.AddIndex("idx_miniapp_action_logs_operation", false, "operation_id, status", "")
		return app.Save(collection)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "miniapp_action_logs")
	})
}
