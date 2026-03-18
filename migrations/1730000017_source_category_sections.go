package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "source_category_sections")
		if err != nil {
			return err
		}

		collection.Fields.Add(
			&core.TextField{Name: "category_key", Required: true, Max: 128},
			&core.TextField{Name: "label", Max: 255},
			&core.TextField{Name: "category_path", Max: 500},
			&core.TextField{Name: "subject_path", Max: 500},
			&core.TextField{Name: "top_level_key", Max: 128},
			&core.TextField{Name: "top_level_label", Max: 255},
			&core.NumberField{Name: "product_count"},
			&core.TextField{Name: "source_mode", Max: 64},
			&core.JSONField{Name: "category_keys_json"},
			&core.JSONField{Name: "product_ids_json"},
			&core.JSONField{Name: "product_spu_ids_json"},
			&core.JSONField{Name: "product_sku_ids_json"},
			&core.JSONField{Name: "source_payload"},
		)

		collection.AddIndex("idx_source_category_sections_key", true, "category_key", "")
		collection.AddIndex("idx_source_category_sections_top_level", false, "top_level_key", "")
		return app.Save(collection)
	}, func(app core.App) error {
		return deleteCollectionIfExists(app, "source_category_sections")
	})
}
