package server

import (
	"context"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/pim"
)

func RegisterHooks(app *pocketbase.PocketBase, cfg config.Config, service *pim.Service) {
	app.OnRecordAfterCreateSuccess(pim.CollectionSupplierProducts).BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		record := e.Record
		if record == nil {
			return nil
		}

		if !cfg.Workflow.AutoProcessOnIngest {
			return nil
		}

		if record.GetString("sync_status") != pim.StatusPending {
			return nil
		}

		go service.ProcessRecord(context.Background(), e.App, record.Id)
		return nil
	})

	app.OnRecordAfterUpdateSuccess(pim.CollectionSupplierProducts).BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		record := e.Record
		if record == nil {
			return nil
		}

		if cfg.Workflow.AutoSyncApproved && record.GetString("sync_status") == pim.StatusApproved {
			go service.SyncRecord(context.Background(), e.App, record.Id)
		}

		return nil
	})
}
