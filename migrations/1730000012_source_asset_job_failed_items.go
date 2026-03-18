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

		collection.Fields.Add(&core.JSONField{Name: "failed_items_json"})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("source_asset_jobs")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("failed_items_json")
		return app.Save(collection)
	})
}
