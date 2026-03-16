package server

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/admin"
	"mrtang-pim/internal/config"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

func registerAdminRoutes(se *core.ServeEvent, cfg config.Config, service *pim.Service, miniappService *miniappservice.Service) {
	se.Router.GET("/api/pim/healthz", func(re *core.RequestEvent) error {
		return re.JSON(http.StatusOK, map[string]any{
			"service": "mrtang-pim",
			"status":  "ok",
		})
	})

	se.Router.GET("/_/mrtang-admin", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		return re.HTML(http.StatusOK, admin.RenderMrtangAdminHTML(re.Request.Context(), re.App, cfg, service, miniappService))
	})

	se.Router.GET("/_/procurement-workbench", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		summary, err := service.ProcurementWorkbenchSummary(re.Request.Context(), re.App, 20)
		if err != nil {
			return re.InternalServerError("load procurement workbench failed", err)
		}

		return re.HTML(http.StatusOK, admin.RenderProcurementWorkbenchHTML(summary))
	})

	se.Router.POST("/_/procurement-workbench/order/status", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		status := strings.TrimSpace(re.Request.FormValue("status"))
		note := strings.TrimSpace(re.Request.FormValue("note"))
		if id == "" || status == "" {
			return re.BadRequestError("missing procurement order id or status", nil)
		}

		if _, err := service.UpdateProcurementOrderStatus(re.Request.Context(), re.App, id, status, note); err != nil {
			return re.BadRequestError("update procurement order status failed", err)
		}

		return re.Redirect(http.StatusSeeOther, "/_/procurement-workbench")
	})

	se.Router.POST("/_/procurement-workbench/order/export", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		if _, err := service.ExportProcurementOrder(re.Request.Context(), re.App, id); err != nil {
			return re.BadRequestError("export procurement order failed", err)
		}

		return re.Redirect(http.StatusSeeOther, "/_/procurement-workbench")
	})

	se.Router.POST("/_/procurement-workbench/order/review", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		note := strings.TrimSpace(re.Request.FormValue("note"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		if _, err := service.ReviewProcurementOrder(re.Request.Context(), re.App, id, note); err != nil {
			return re.BadRequestError("review procurement order failed", err)
		}

		return re.Redirect(http.StatusSeeOther, "/_/procurement-workbench")
	})
}
