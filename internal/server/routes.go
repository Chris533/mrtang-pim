package server

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

func RegisterRoutes(app *pocketbase.PocketBase, cfg config.Config, service *pim.Service, miniappService *miniappservice.Service) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerAdminRoutes(se, cfg, service, miniappService)
		registerProcurementRoutes(se, cfg, service)
		registerMiniAppRoutes(se, cfg, miniappService)
		registerPIMWorkflowRoutes(se, cfg, service)
		return se.Next()
	})
}
