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

		collection.Fields.Add(
			&core.SelectField{Name: "supplier_status", Values: []string{"active", "out_of_stock", "offline", "price_changed", "spec_changed"}, MaxSelect: 1},
			&core.TextField{Name: "offline_reason", Max: 255},
			&core.DateField{Name: "last_seen_at"},
			&core.DateField{Name: "last_price_sync_at"},
			&core.DateField{Name: "last_stock_sync_at"},
			&core.TextField{Name: "source_snapshot_hash", Max: 128},
			&core.JSONField{Name: "last_snapshot_json"},
		)
		collection.AddIndex("idx_supplier_products_supplier_status", false, "supplier_status", "")
		collection.AddIndex("idx_supplier_products_last_seen_at", false, "last_seen_at", "")
		return app.Save(collection)
	}, func(app core.App) error {
		if err := removeCollectionFieldIfExists(app, "supplier_products", "supplier_status"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "offline_reason"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "last_seen_at"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "last_price_sync_at"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "last_stock_sync_at"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "supplier_products", "source_snapshot_hash"); err != nil {
			return err
		}
		return removeCollectionFieldIfExists(app, "supplier_products", "last_snapshot_json")
	})
}
