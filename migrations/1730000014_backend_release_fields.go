package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := upsertSupplierProductsBackendFields(app); err != nil {
			return err
		}
		return upsertBackendCategoryMappingsCollection(app)
	}, func(app core.App) error {
		if err := deleteCollectionIfExists(app, "backend_category_mappings"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "source_product_id"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "source_type"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "conversion_rate"); err != nil {
			return err
		}
		return removeCollectionFieldIfExists(app, "supplier_products", "target_audience")
	})
}

func upsertSupplierProductsBackendFields(app core.App) error {
	collection, err := findOrCreateCollection(app, "supplier_products")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "source_product_id", Max: 128},
		&core.TextField{Name: "source_type", Max: 64},
		&core.NumberField{Name: "conversion_rate"},
		&core.SelectField{Name: "target_audience", Values: []string{"ALL", "B_ONLY", "C_ONLY"}, MaxSelect: 1},
	)

	collection.AddIndex("idx_supplier_products_source_product", false, "source_product_id", "")
	return app.Save(collection)
}

func upsertBackendCategoryMappingsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "backend_category_mappings")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "source_key", Required: true, Max: 128},
		&core.TextField{Name: "source_path", Max: 500},
		&core.TextField{Name: "backend_collection", Max: 255},
		&core.TextField{Name: "backend_path", Max: 500},
		&core.SelectField{Name: "publish_status", Values: []string{"pending", "mapped", "published", "error"}, MaxSelect: 1},
		&core.TextField{Name: "last_error", Max: 500},
		&core.EditorField{Name: "note"},
	)

	collection.AddIndex("idx_backend_category_mappings_source_key", true, "source_key", "")
	collection.AddIndex("idx_backend_category_mappings_publish_status", false, "publish_status", "")
	return app.Save(collection)
}
