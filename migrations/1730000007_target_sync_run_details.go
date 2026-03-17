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
		collection.Fields.Add(&core.JSONField{Name: "details_json"})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := findOrCreateCollection(app, "target_sync_runs")
		if err != nil {
			return err
		}
		field := collection.Fields.GetByName("details_json")
		if field == nil {
			return nil
		}
		collection.Fields.RemoveById(field.GetId())
		return app.Save(collection)
	})
}
