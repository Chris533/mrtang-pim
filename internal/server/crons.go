package server

import (
	"context"
	"strings"

	"github.com/pocketbase/pocketbase"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/pim"
)

func RegisterCrons(app *pocketbase.PocketBase, cfg config.Config, service *pim.Service) {
	if strings.TrimSpace(cfg.Workflow.CronHarvest) != "" {
		app.Cron().MustAdd("pim-harvest", cfg.Workflow.CronHarvest, func() {
			if _, err := service.Harvest(context.Background(), app); err != nil {
				app.Logger().Error("scheduled harvest failed", "error", err)
			}
		})
	}

	if strings.TrimSpace(cfg.Workflow.CronProcess) != "" {
		app.Cron().MustAdd("pim-process", cfg.Workflow.CronProcess, func() {
			if _, err := service.ProcessPending(context.Background(), app, cfg.Workflow.ProcessBatchSize); err != nil {
				app.Logger().Error("scheduled processing failed", "error", err)
			}
		})
	}

	if strings.TrimSpace(cfg.Workflow.CronSync) != "" {
		app.Cron().MustAdd("pim-sync", cfg.Workflow.CronSync, func() {
			if _, err := service.SyncApproved(context.Background(), app, cfg.Workflow.SyncBatchSize); err != nil {
				app.Logger().Error("scheduled sync failed", "error", err)
			}
		})
	}
}
