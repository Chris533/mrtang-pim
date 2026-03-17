package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := upsertSourceCategoriesCollection(app); err != nil {
			return err
		}

		if err := upsertSourceProductsCollection(app); err != nil {
			return err
		}

		return upsertSourceAssetsCollection(app)
	}, func(app core.App) error {
		if err := deleteCollectionIfExists(app, "source_assets"); err != nil {
			return err
		}
		if err := deleteCollectionIfExists(app, "source_products"); err != nil {
			return err
		}
		return deleteCollectionIfExists(app, "source_categories")
	})
}

func upsertSourceCategoriesCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "source_categories")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "source_key", Required: true, Max: 128},
		&core.TextField{Name: "label", Required: true, Max: 255},
		&core.TextField{Name: "path_name", Max: 255},
		&core.TextField{Name: "category_path", Max: 500},
		&core.TextField{Name: "parent_key", Max: 128},
		&core.URLField{Name: "image_url"},
		&core.NumberField{Name: "depth"},
		&core.NumberField{Name: "sort"},
		&core.BoolField{Name: "has_children"},
		&core.JSONField{Name: "source_payload"},
	)

	collection.AddIndex("idx_source_categories_key", true, "source_key", "")
	collection.AddIndex("idx_source_categories_parent", false, "parent_key", "")
	return app.Save(collection)
}

func upsertSourceProductsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "source_products")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "product_id", Required: true, Max: 128},
		&core.TextField{Name: "spu_id", Required: true, Max: 64},
		&core.TextField{Name: "sku_id", Required: true, Max: 64},
		&core.TextField{Name: "name", Required: true, Max: 255},
		&core.TextField{Name: "sku_name", Max: 255},
		&core.TextField{Name: "cover_url", Max: 500},
		&core.TextField{Name: "default_unit", Max: 64},
		&core.TextField{Name: "default_unit_id", Max: 64},
		&core.TextField{Name: "base_unit_id", Max: 64},
		&core.TextField{Name: "category_key", Max: 128},
		&core.TextField{Name: "category_path", Max: 500},
		&core.TextField{Name: "source_type", Max: 64},
		&core.SelectField{Name: "review_status", Values: []string{"imported", "approved", "rejected", "promoted"}, MaxSelect: 1},
		&core.NumberField{Name: "unit_count"},
		&core.BoolField{Name: "has_multi_unit"},
		&core.NumberField{Name: "default_price"},
		&core.NumberField{Name: "default_stock_qty"},
		&core.TextField{Name: "stock_text", Max: 128},
		&core.JSONField{Name: "source_sections"},
		&core.JSONField{Name: "tags_json"},
		&core.JSONField{Name: "promotion_texts_json"},
		&core.JSONField{Name: "unit_options_json"},
		&core.JSONField{Name: "order_units_json"},
		&core.JSONField{Name: "summary_json"},
		&core.JSONField{Name: "detail_json"},
		&core.JSONField{Name: "pricing_json"},
		&core.JSONField{Name: "package_json"},
		&core.JSONField{Name: "context_json"},
		&core.NumberField{Name: "asset_count"},
	)

	collection.AddIndex("idx_source_products_product_id", true, "product_id", "")
	collection.AddIndex("idx_source_products_category_key", false, "category_key", "")
	collection.AddIndex("idx_source_products_review_status", false, "review_status", "")
	return app.Save(collection)
}

func upsertSourceAssetsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "source_assets")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "asset_key", Required: true, Max: 255},
		&core.TextField{Name: "product_id", Required: true, Max: 128},
		&core.TextField{Name: "spu_id", Max: 64},
		&core.TextField{Name: "sku_id", Max: 64},
		&core.TextField{Name: "name", Max: 255},
		&core.URLField{Name: "source_url"},
		&core.TextField{Name: "asset_role", Max: 64},
		&core.NumberField{Name: "sort"},
		&core.FileField{Name: "processed_image"},
		&core.TextField{Name: "processed_image_source", Max: 255},
		&core.SelectField{Name: "image_processing_status", Values: []string{"pending", "processing", "processed", "failed"}, MaxSelect: 1},
		&core.TextField{Name: "image_processing_error", Max: 500},
		&core.JSONField{Name: "source_payload"},
	)

	collection.AddIndex("idx_source_assets_key", true, "asset_key", "")
	collection.AddIndex("idx_source_assets_product_id", false, "product_id", "")
	return app.Save(collection)
}
