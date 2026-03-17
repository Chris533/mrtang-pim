package server

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappservice "mrtang-pim/internal/miniapp/service"
)

func registerMiniAppCartOrderRoutes(se *core.ServeEvent, cfg config.Config, miniappService *miniappservice.Service) {
	se.Router.GET("/api/miniapp/cart-order", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		cartOrder, err := miniappService.CartOrder(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp cart-order failed", err)
		}

		return re.JSON(http.StatusOK, cartOrder)
	})

	se.Router.GET("/api/miniapp/cart-order/cart", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		cart, err := miniappService.Cart(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp cart failed", err)
		}

		return re.JSON(http.StatusOK, cart)
	})

	se.Router.GET("/api/miniapp/cart-order/order", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		order, err := miniappService.Order(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp order failed", err)
		}

		return re.JSON(http.StatusOK, order)
	})

	se.Router.POST("/api/miniapp/cart-order/cart/add", func(re *core.RequestEvent) error {
		return serveMiniAppCartOperation(re, cfg, miniappService, "add", "add cart")
	})
	se.Router.POST("/api/miniapp/cart-order/cart/change-num", func(re *core.RequestEvent) error {
		return serveMiniAppCartOperation(re, cfg, miniappService, "change-num", "change cart quantity")
	})
	se.Router.GET("/api/miniapp/cart-order/cart/list", func(re *core.RequestEvent) error {
		return serveMiniAppCartOperation(re, cfg, miniappService, "list", "load cart list")
	})
	se.Router.POST("/api/miniapp/cart-order/cart/list", func(re *core.RequestEvent) error {
		return serveMiniAppCartOperation(re, cfg, miniappService, "list", "load cart list")
	})
	se.Router.GET("/api/miniapp/cart-order/cart/list-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.CartListSummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp cart list summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})

	se.Router.GET("/api/miniapp/cart-order/cart/detail", func(re *core.RequestEvent) error {
		return serveMiniAppCartOperation(re, cfg, miniappService, "detail", "load cart detail")
	})
	se.Router.GET("/api/miniapp/cart-order/cart/detail-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.CartDetailSummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp cart detail summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})
	se.Router.POST("/api/miniapp/cart-order/cart/settle", func(re *core.RequestEvent) error {
		return serveMiniAppCartOperation(re, cfg, miniappService, "settle", "settle cart")
	})

	se.Router.GET("/api/miniapp/cart-order/order/default-delivery", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "default-delivery", "load default delivery")
	})
	se.Router.POST("/api/miniapp/cart-order/order/default-delivery", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "default-delivery", "load default delivery")
	})
	se.Router.GET("/api/miniapp/cart-order/order/default-delivery-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.DefaultDeliverySummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp default delivery summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})

	se.Router.GET("/api/miniapp/cart-order/order/deliveries", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "deliveries", "load delivery list")
	})
	se.Router.POST("/api/miniapp/cart-order/order/deliveries", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "deliveries", "load delivery list")
	})
	se.Router.GET("/api/miniapp/cart-order/order/deliveries-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.DeliveriesSummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp deliveries summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})

	se.Router.POST("/api/miniapp/cart-order/order/address/analyse", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "analyse-address", "analyse address")
	})
	se.Router.POST("/api/miniapp/cart-order/order/address/add", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "add-delivery", "add delivery address")
	})

	se.Router.GET("/api/miniapp/cart-order/order/freight-cost", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		freight, err := miniappService.ExecuteFreightCost(re.Request.Context(), miniAppFreightScenario(re), nil)
		if err != nil {
			return re.InternalServerError("load miniapp freight cost failed", err)
		}

		if freight == nil {
			return re.NotFoundError("freight cost scenario not found", nil)
		}

		return re.JSON(http.StatusOK, freight.Response)
	})
	se.Router.POST("/api/miniapp/cart-order/order/freight-cost", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		requestBody, err := readOptionalJSONBody(re)
		if err != nil {
			return re.BadRequestError("invalid request body", err)
		}

		freight, err := miniappService.ExecuteFreightCost(re.Request.Context(), miniAppFreightScenario(re), requestBody)
		if err != nil {
			return re.InternalServerError("load miniapp freight cost failed", err)
		}

		if freight == nil {
			return re.NotFoundError("freight cost scenario not found", nil)
		}

		return re.JSON(http.StatusOK, freight.Response)
	})
	se.Router.GET("/api/miniapp/cart-order/order/freight-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.FreightSummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp freight summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})

	se.Router.POST("/api/miniapp/cart-order/order/submit", func(re *core.RequestEvent) error {
		return serveMiniAppOrderOperation(re, cfg, miniappService, "submit", "submit order")
	})
	se.Router.GET("/api/miniapp/cart-order/order/submit-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.OrderSubmitSummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp order submit summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})
	se.Router.GET("/api/miniapp/cart-order/checkout-summary", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}

		summary, err := miniappService.CheckoutSummary(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp checkout summary failed", err)
		}

		return re.JSON(http.StatusOK, summary)
	})
}
