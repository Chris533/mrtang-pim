package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("source_products")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.JSONField{Name: "observed_category_keys_json"},
			&core.JSONField{Name: "observed_category_paths_json"},
		)

		return app.Save(collection)
	}, func(app core.App) error {
		if err := removeCollectionFieldIfExists(app, "source_products", "observed_category_keys_json"); err != nil {
			return err
		}
		return removeCollectionFieldIfExists(app, "source_products", "observed_category_paths_json")
	})
}
