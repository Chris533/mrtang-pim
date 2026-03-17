package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := findOrCreateCollection(app, "source_assets")
		if err != nil {
			return err
		}
		collection.Fields.Add(
			&core.FileField{Name: "original_image"},
			&core.SelectField{Name: "original_image_status", Values: []string{"pending", "downloading", "downloaded", "failed"}, MaxSelect: 1},
			&core.TextField{Name: "original_image_error", Max: 500},
		)
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := findOrCreateCollection(app, "source_assets")
		if err != nil {
			return err
		}
		for _, name := range []string{"original_image", "original_image_status", "original_image_error"} {
			if field := collection.Fields.GetByName(name); field != nil {
				collection.Fields.RemoveById(field.GetId())
			}
		}
		return app.Save(collection)
	})
}
