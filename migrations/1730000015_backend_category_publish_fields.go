package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "backend_category_mappings")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "backend_collection_id", Max: 64},
			&core.TextField{Name: "published_at", Max: 64},
		)
		return app.Save(collection)
	}, func(app core.App) error {
		if err := removeCollectionFieldIfExists(app, "backend_category_mappings", "backend_collection_id"); err != nil {
			return err
		}
		return removeCollectionFieldIfExists(app, "backend_category_mappings", "published_at")
	})
}
