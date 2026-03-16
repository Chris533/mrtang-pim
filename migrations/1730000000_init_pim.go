package migrations

import (
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := upsertSupplierProductsCollection(app); err != nil {
			return err
		}

		if err := upsertCategoryMappingsCollection(app); err != nil {
			return err
		}

		if err := upsertProcurementOrdersCollection(app); err != nil {
			return err
		}

		return ensureSuperuser(app)
	}, func(app core.App) error {
		if err := deleteCollectionIfExists(app, "procurement_orders"); err != nil {
			return err
		}

		if err := deleteCollectionIfExists(app, "category_mappings"); err != nil {
			return err
		}

		return deleteCollectionIfExists(app, "supplier_products")
	})
}

func upsertSupplierProductsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "supplier_products")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "supplier_code", Required: true, Max: 64},
		&core.TextField{Name: "original_sku", Required: true, Max: 128},
		&core.TextField{Name: "raw_title", Required: true, Max: 255},
		&core.TextField{Name: "normalized_title", Max: 255},
		&core.EditorField{Name: "raw_description"},
		&core.EditorField{Name: "marketing_description"},
		&core.TextField{Name: "raw_category", Max: 128},
		&core.TextField{Name: "normalized_category", Max: 128},
		&core.URLField{Name: "raw_image_url"},
		&core.FileField{Name: "processed_image"},
		&core.TextField{Name: "processed_image_source", Max: 255},
		&core.SelectField{Name: "sync_status", Required: true, Values: []string{"pending", "ai_processing", "ready", "approved", "synced", "offline", "error"}, MaxSelect: 1},
		&core.SelectField{Name: "image_processing_status", Values: []string{"pending", "processing", "processed", "failed"}, MaxSelect: 1},
		&core.TextField{Name: "image_processing_error", Max: 500},
		&core.TextField{Name: "last_sync_error", Max: 500},
		&core.TextField{Name: "vendure_product_id", Max: 64},
		&core.TextField{Name: "vendure_variant_id", Max: 64},
		&core.NumberField{Name: "cost_price"},
		&core.NumberField{Name: "b_price"},
		&core.NumberField{Name: "c_price"},
		&core.TextField{Name: "currency_code", Max: 16},
		&core.DateField{Name: "supplier_updated_at"},
		&core.DateField{Name: "last_synced_at"},
		&core.DateField{Name: "offline_at"},
		&core.JSONField{Name: "supplier_payload"},
	)

	collection.AddIndex("idx_supplier_products_supplier_sku", true, "supplier_code, original_sku", "")
	return app.Save(collection)
}

func upsertCategoryMappingsCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "category_mappings")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "supplier_code", Required: true, Max: 64},
		&core.TextField{Name: "supplier_category", Required: true, Max: 128},
		&core.TextField{Name: "normalized_category", Required: true, Max: 128},
	)

	collection.AddIndex("idx_category_mapping_unique", true, "supplier_code, supplier_category", "")
	return app.Save(collection)
}

func upsertProcurementOrdersCollection(app core.App) error {
	collection, err := findOrCreateCollection(app, "procurement_orders")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "external_ref", Required: true, Max: 128},
		&core.SelectField{Name: "status", Required: true, Values: []string{"draft", "reviewed", "exported", "ordered", "received", "canceled"}, MaxSelect: 1},
		&core.TextField{Name: "connector", Max: 32},
		&core.TextField{Name: "delivery_address", Max: 255},
		&core.EditorField{Name: "notes"},
		&core.TextField{Name: "last_action_note", Max: 500},
		&core.NumberField{Name: "supplier_count"},
		&core.NumberField{Name: "item_count"},
		&core.NumberField{Name: "total_qty"},
		&core.NumberField{Name: "total_cost_amount"},
		&core.NumberField{Name: "risky_item_count"},
		&core.JSONField{Name: "summary_json"},
		&core.JSONField{Name: "results_json"},
		&core.EditorField{Name: "export_csv"},
		&core.DateField{Name: "reviewed_at"},
		&core.DateField{Name: "exported_at"},
		&core.DateField{Name: "ordered_at"},
		&core.DateField{Name: "received_at"},
		&core.DateField{Name: "canceled_at"},
	)

	collection.AddIndex("idx_procurement_orders_external_ref", true, "external_ref", "")
	collection.AddIndex("idx_procurement_orders_status", false, "status", "")
	return app.Save(collection)
}

func ensureSuperuser(app core.App) error {
	email := strings.TrimSpace(os.Getenv("PIM_SUPERUSER_EMAIL"))
	password := strings.TrimSpace(os.Getenv("PIM_SUPERUSER_PASSWORD"))
	if email == "" || password == "" {
		return nil
	}

	record, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
	if err == nil && record != nil {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}

	record = core.NewRecord(collection)
	record.Set("email", email)
	record.Set("password", password)
	return app.Save(record)
}

func findOrCreateCollection(app core.App, name string) (*core.Collection, error) {
	collection, err := app.FindCollectionByNameOrId(name)
	if err == nil {
		return collection, nil
	}

	return core.NewBaseCollection(name), nil
}

func deleteCollectionIfExists(app core.App, name string) error {
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return nil
	}

	return app.Delete(collection)
}
