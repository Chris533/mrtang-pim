package server

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/pim"
)

func registerPIMWorkflowRoutes(se *core.ServeEvent, cfg config.Config, service *pim.Service) {
	se.Router.POST("/api/pim/harvest", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		result, err := service.HarvestWithOptions(re.Request.Context(), re.App, pim.HarvestOptions{
			TriggerType: pim.HarvestTriggerAPI,
		})
		if err != nil {
			return re.BadRequestError("harvest failed", err)
		}

		return re.JSON(http.StatusOK, result)
	})

	se.Router.POST("/api/pim/process", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		result, err := service.ProcessPending(re.Request.Context(), re.App, cfg.Workflow.ProcessBatchSize)
		if err != nil {
			return re.BadRequestError("image processing failed", err)
		}

		return re.JSON(http.StatusOK, result)
	})

	se.Router.POST("/api/pim/sync", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		result, err := service.SyncApproved(re.Request.Context(), re.App, cfg.Workflow.SyncBatchSize)
		if err != nil {
			return re.BadRequestError("sync failed", err)
		}

		return re.JSON(http.StatusOK, result)
	})
}
