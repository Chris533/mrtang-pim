package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := upsertSourceProductAuditFields(app); err != nil {
			return err
		}
		return upsertSourceActionLogAuditFields(app)
	}, func(app core.App) error {
		if err := removeCollectionFieldIfExists(app, "source_products", "review_note"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_products", "reviewed_by_email"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_products", "reviewed_by_name"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_products", "reviewed_at"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_action_logs", "actor_email"); err != nil {
			return err
		}
		if err := removeCollectionFieldIfExists(app, "source_action_logs", "actor_name"); err != nil {
			return err
		}
		return removeCollectionFieldIfExists(app, "source_action_logs", "note")
	})
}

func upsertSourceProductAuditFields(app core.App) error {
	collection, err := findOrCreateCollection(app, "source_products")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "review_note", Max: 1000},
		&core.TextField{Name: "reviewed_by_email", Max: 255},
		&core.TextField{Name: "reviewed_by_name", Max: 255},
		&core.TextField{Name: "reviewed_at", Max: 64},
	)

	return app.Save(collection)
}

func upsertSourceActionLogAuditFields(app core.App) error {
	collection, err := findOrCreateCollection(app, "source_action_logs")
	if err != nil {
		return err
	}

	collection.Fields.Add(
		&core.TextField{Name: "actor_email", Max: 255},
		&core.TextField{Name: "actor_name", Max: 255},
		&core.TextField{Name: "note", Max: 1000},
	)

	collection.AddIndex("idx_source_action_logs_actor", false, "actor_email", "")
	return app.Save(collection)
}

func removeCollectionFieldIfExists(app core.App, collectionName string, fieldName string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil
	}

	field := collection.Fields.GetByName(fieldName)
	if field == nil {
		return nil
	}

	collection.Fields.RemoveById(field.GetId())
	return app.Save(collection)
}
