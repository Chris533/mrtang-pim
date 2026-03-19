package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "supplier_products")
		if err != nil {
			return err
		}

		collection.Fields.Add(&core.JSONField{Name: "vendure_variants_json"})
		return app.Save(collection)
	}, func(app core.App) error {
		return removeCollectionFieldIfExists(app, "supplier_products", "vendure_variants_json")
	})
}
