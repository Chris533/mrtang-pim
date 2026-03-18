package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateTargetSyncEntityValuesV2(app, "target_sync_jobs"); err != nil {
			return err
		}
		return updateTargetSyncEntityValuesV2(app, "target_sync_runs")
	}, func(app core.App) error {
		return nil
	})
}

func updateTargetSyncEntityValuesV2(app core.App, collectionName string) error {
	collection, err := findOrCreateCollection(app, collectionName)
	if err != nil {
		return err
	}
	field := collection.Fields.GetByName("entity_type")
	selectField, ok := field.(*core.SelectField)
	if !ok || selectField == nil {
		return app.Save(collection)
	}
	selectField.Values = []string{"category_tree", "category_sources", "category_rebuild", "products", "assets"}
	return app.Save(collection)
}
