package server

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/pim"
)

func registerProcurementRoutes(se *core.ServeEvent, cfg config.Config, service *pim.Service) {
	se.Router.GET("/api/pim/procurement/capabilities", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		return re.JSON(http.StatusOK, map[string]any{
			"connector":    cfg.Supplier.Connector,
			"capabilities": service.ConnectorCapabilities(),
		})
	})

	se.Router.GET("/api/pim/procurement/workbench-summary", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		summary, err := service.ProcurementWorkbenchSummary(re.Request.Context(), re.App, readIntQuery(re, "limit", 20))
		if err != nil {
			return re.BadRequestError("load procurement workbench summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})

	se.Router.GET("/api/pim/procurement/orders", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		orders, err := service.ListProcurementOrders(
			re.Request.Context(),
			re.App,
			readIntQuery(re, "limit", 20),
			re.Request.URL.Query().Get("status"),
		)
		if err != nil {
			return re.BadRequestError("load procurement orders failed", err)
		}

		return re.JSON(http.StatusOK, map[string]any{
			"items": orders,
			"count": len(orders),
		})
	})

	se.Router.GET("/api/pim/procurement/order", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		order, err := service.GetProcurementOrder(re.Request.Context(), re.App, id)
		if err != nil {
			return re.BadRequestError("load procurement order failed", err)
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.GET("/api/pim/procurement/actions", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		orderID := strings.TrimSpace(re.Request.URL.Query().Get("orderId"))
		actions, err := service.ListProcurementActions(re.App, orderID, readIntQuery(re, "limit", 20))
		if err != nil {
			return re.BadRequestError("load procurement actions failed", err)
		}

		return re.JSON(http.StatusOK, map[string]any{
			"items": actions,
			"count": len(actions),
		})
	})

	se.Router.POST("/api/pim/procurement/orders", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		request, err := readProcurementRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement request", err)
		}

		order, err := service.CreateProcurementOrder(re.Request.Context(), re.App, request)
		if err != nil {
			return re.BadRequestError("create procurement order failed", err)
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.POST("/api/pim/procurement/order/review", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		update, err := readProcurementStatusUpdateRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement review request", err)
		}

		order, err := service.ReviewProcurementOrder(re.Request.Context(), re.App, id, update.Note)
		if err != nil {
			return re.BadRequestError("review procurement order failed", err)
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.POST("/api/pim/procurement/order/export", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		order, err := service.ExportProcurementOrder(re.Request.Context(), re.App, id)
		if err != nil {
			return re.BadRequestError("export procurement order failed", err)
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.POST("/api/pim/procurement/order/submit", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		update, err := readProcurementStatusUpdateRequest(re)
		if err != nil {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"message": "invalid procurement submit request",
				"error":   err.Error(),
			})
		}

		order, err := service.SubmitProcurementOrder(re.Request.Context(), re.App, id, update.Note)
		if err != nil {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"message": "submit procurement order failed",
				"error":   err.Error(),
				"id":      id,
			})
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.POST("/api/pim/procurement/order/status", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		update, err := readProcurementStatusUpdateRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement status request", err)
		}

		order, err := service.UpdateProcurementOrderStatus(re.Request.Context(), re.App, id, update.Status, update.Note)
		if err != nil {
			return re.BadRequestError("update procurement order status failed", err)
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.POST("/api/pim/procurement/summary", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		request, err := readProcurementRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement request", err)
		}

		summary, err := service.ProcurementSummary(re.Request.Context(), re.App, request)
		if err != nil {
			return re.BadRequestError("build procurement summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})

	se.Router.POST("/api/pim/procurement/precheck", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		request, err := readProcurementRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement request", err)
		}

		result, err := service.PrecheckProcurementItems(re.Request.Context(), re.App, request)
		if err != nil {
			return re.BadRequestError("precheck procurement failed", err)
		}

		return re.JSON(http.StatusOK, result)
	})

	se.Router.POST("/api/pim/procurement/export", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		request, err := readProcurementRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement request", err)
		}

		exported, err := service.ExportProcurement(re.Request.Context(), re.App, request)
		if err != nil {
			return re.BadRequestError("export procurement failed", err)
		}

		return re.JSON(http.StatusOK, exported)
	})

	se.Router.POST("/api/pim/procurement/submit", func(re *core.RequestEvent) error {
		if !authorized(re, cfg.Security.APIKey) {
			return re.UnauthorizedError("missing or invalid api key", nil)
		}

		request, err := readProcurementRequest(re)
		if err != nil {
			return re.BadRequestError("invalid procurement request", err)
		}

		result, err := service.SubmitProcurement(re.Request.Context(), re.App, request)
		if err != nil {
			return re.BadRequestError("submit procurement failed", err)
		}

		return re.JSON(http.StatusOK, result)
	})
}
