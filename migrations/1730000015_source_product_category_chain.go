package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "source_products")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "leaf_category_key", Max: 128},
			&core.TextField{Name: "leaf_category_path", Max: 500},
			&core.JSONField{Name: "category_keys_json"},
		)

		collection.AddIndex("idx_source_products_leaf_category_key", false, "leaf_category_key", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := findOrCreateCollection(app, "source_products")
		if err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_products", "leaf_category_key"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_products", "leaf_category_path"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_products", "category_keys_json"); err != nil {
			return err
		}
		return app.Save(collection)
	})
}
