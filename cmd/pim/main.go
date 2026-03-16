package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"mrtang-pim/internal/config"
	miniappapi "mrtang-pim/internal/miniapp/api"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
	_ "mrtang-pim/migrations"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	applyDefaultServeHTTPAddr(cfg.App.HTTPAddr)
	app := pocketbase.New()
	service := pim.NewService(cfg)
	miniappService := miniappservice.New(
		newMiniAppSource(cfg),
		nil,
	)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	registerHooks(app, cfg, service)
	registerCrons(app, cfg, service)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/pim/healthz", func(re *core.RequestEvent) error {
			return re.JSON(http.StatusOK, map[string]any{
				"service": "mrtang-pim",
				"status":  "ok",
			})
		})
		se.Router.GET("/_/mrtang-admin", func(re *core.RequestEvent) error {
			if !authorizedAdminPage(re) {
				return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
			}

			summary, err := service.ProcurementWorkbenchSummary(re.Request.Context(), re.App, 12)
			if err != nil {
				return re.InternalServerError("load mrtang admin failed", err)
			}

			return re.HTML(http.StatusOK, renderMrtangAdminHTML(summary))
		})
		se.Router.GET("/_/procurement-workbench", func(re *core.RequestEvent) error {
			if !authorizedAdminPage(re) {
				return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
			}

			summary, err := service.ProcurementWorkbenchSummary(re.Request.Context(), re.App, 20)
			if err != nil {
				return re.InternalServerError("load procurement workbench failed", err)
			}

			return re.HTML(http.StatusOK, renderProcurementWorkbenchHTML(summary))
		})
		se.Router.POST("/_/procurement-workbench/order/status", func(re *core.RequestEvent) error {
			if !authorizedAdminPage(re) {
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
			if !authorizedAdminPage(re) {
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
			if !authorizedAdminPage(re) {
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

		se.Router.GET("/api/miniapp/contracts/homepage", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			dataset, err := miniappService.Dataset(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp contracts failed", err)
			}

			return re.JSON(http.StatusOK, map[string]any{
				"meta":      dataset.Meta,
				"contracts": filterContractsByPrefix(dataset.Contracts, "/api/miniapp/homepage"),
				"clientConfig": map[string]any{
					"sourceMode":        cfg.MiniApp.SourceMode,
					"sourceURL":         cfg.MiniApp.SourceURL,
					"userAgent":         cfg.MiniApp.UserAgent,
					"authHeader":        "Authorization: Bearer <authorized-account-id>",
					"authorizationMode": miniAppAuthorizationMode(cfg),
				},
			})
		})

		se.Router.GET("/api/miniapp/contracts/category-page", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			dataset, err := miniappService.Dataset(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp category-page contracts failed", err)
			}

			return re.JSON(http.StatusOK, map[string]any{
				"meta":      dataset.Meta,
				"contracts": filterContractsByPrefix(dataset.Contracts, "/api/miniapp/category-page"),
				"clientConfig": map[string]any{
					"sourceMode":        cfg.MiniApp.SourceMode,
					"sourceURL":         cfg.MiniApp.SourceURL,
					"userAgent":         cfg.MiniApp.UserAgent,
					"authHeader":        "Authorization: Bearer <authorized-account-id>",
					"authorizationMode": miniAppAuthorizationMode(cfg),
				},
			})
		})

		se.Router.GET("/api/miniapp/contracts/product-page", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			dataset, err := miniappService.Dataset(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp product-page contracts failed", err)
			}

			return re.JSON(http.StatusOK, map[string]any{
				"meta":      dataset.Meta,
				"contracts": filterContractsByPrefix(dataset.Contracts, "/api/miniapp/product-page"),
				"clientConfig": map[string]any{
					"sourceMode":        cfg.MiniApp.SourceMode,
					"sourceURL":         cfg.MiniApp.SourceURL,
					"userAgent":         cfg.MiniApp.UserAgent,
					"authHeader":        "Authorization: Bearer <authorized-account-id>",
					"authorizationMode": miniAppAuthorizationMode(cfg),
				},
			})
		})

		se.Router.GET("/api/miniapp/contracts/cart-order", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			dataset, err := miniappService.Dataset(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp cart-order contracts failed", err)
			}

			return re.JSON(http.StatusOK, map[string]any{
				"meta":      dataset.Meta,
				"contracts": filterContractsByPrefix(dataset.Contracts, "/api/miniapp/cart-order"),
				"clientConfig": map[string]any{
					"sourceMode":        cfg.MiniApp.SourceMode,
					"sourceURL":         cfg.MiniApp.SourceURL,
					"userAgent":         cfg.MiniApp.UserAgent,
					"authHeader":        "Authorization: Bearer <authorized-account-id>",
					"authorizationMode": miniAppAuthorizationMode(cfg),
				},
			})
		})

		se.Router.GET("/api/miniapp/homepage", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			homepage, err := miniappService.Homepage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp homepage failed", err)
			}

			return re.JSON(http.StatusOK, homepage)
		})

		se.Router.GET("/api/miniapp/homepage/bootstrap", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			homepage, err := miniappService.Homepage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp bootstrap failed", err)
			}

			return re.JSON(http.StatusOK, homepage.Bootstrap)
		})

		se.Router.GET("/api/miniapp/homepage/settings", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			homepage, err := miniappService.Homepage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp settings failed", err)
			}

			return re.JSON(http.StatusOK, homepage.Settings)
		})

		se.Router.GET("/api/miniapp/homepage/template", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			homepage, err := miniappService.Homepage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp template failed", err)
			}

			return re.JSON(http.StatusOK, homepage.Template)
		})

		se.Router.GET("/api/miniapp/homepage/categories", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			homepage, err := miniappService.Homepage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp categories failed", err)
			}

			return re.JSON(http.StatusOK, homepage.CategoryTabs)
		})

		se.Router.GET("/api/miniapp/homepage/sections", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			homepage, err := miniappService.Homepage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp sections failed", err)
			}

			return re.JSON(http.StatusOK, homepage.Sections)
		})

		se.Router.GET("/api/miniapp/homepage/section", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			sectionID := strings.TrimSpace(re.Request.URL.Query().Get("id"))
			if sectionID == "" {
				return re.BadRequestError("missing section id", nil)
			}

			section, err := miniappService.Section(re.Request.Context(), sectionID)
			if err != nil {
				return re.InternalServerError("load miniapp section failed", err)
			}

			if section == nil {
				return re.NotFoundError("section not found", nil)
			}

			return re.JSON(http.StatusOK, section)
		})

		se.Router.GET("/api/miniapp/category-page", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			categoryPage, err := miniappService.CategoryPage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp category page failed", err)
			}

			return re.JSON(http.StatusOK, categoryPage)
		})

		se.Router.GET("/api/miniapp/category-page/context", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			categoryPage, err := miniappService.CategoryPage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp category context failed", err)
			}

			return re.JSON(http.StatusOK, categoryPage.Context)
		})

		se.Router.GET("/api/miniapp/category-page/tree", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			categoryPage, err := miniappService.CategoryPage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp category tree failed", err)
			}

			return re.JSON(http.StatusOK, categoryPage.Tree)
		})

		se.Router.GET("/api/miniapp/category-page/sections", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			categoryPage, err := miniappService.CategoryPage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp category sections failed", err)
			}

			return re.JSON(http.StatusOK, categoryPage.Sections)
		})

		se.Router.GET("/api/miniapp/category-page/section", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			sectionID := strings.TrimSpace(re.Request.URL.Query().Get("id"))
			if sectionID == "" {
				return re.BadRequestError("missing section id", nil)
			}

			section, err := miniappService.CategorySection(re.Request.Context(), sectionID)
			if err != nil {
				return re.InternalServerError("load miniapp category section failed", err)
			}

			if section == nil {
				return re.NotFoundError("section not found", nil)
			}

			return re.JSON(http.StatusOK, section)
		})

		se.Router.GET("/api/miniapp/product-page", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			productPage, err := miniappService.ProductPage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp product page failed", err)
			}

			return re.JSON(http.StatusOK, productPage)
		})

		se.Router.GET("/api/miniapp/product-page/product", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			productID := miniAppProductID(re)
			if productID == "" {
				return re.BadRequestError("missing product id", nil)
			}

			product, err := miniappService.Product(re.Request.Context(), productID)
			if err != nil {
				return re.InternalServerError("load miniapp product failed", err)
			}

			if product == nil {
				return re.NotFoundError("product not found", nil)
			}

			return re.JSON(http.StatusOK, product)
		})

		se.Router.GET("/api/miniapp/product-page/detail", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			productID := miniAppProductID(re)
			if productID == "" {
				return re.BadRequestError("missing product id", nil)
			}

			product, err := miniappService.Product(re.Request.Context(), productID)
			if err != nil {
				return re.InternalServerError("load miniapp product detail failed", err)
			}

			if product == nil {
				return re.NotFoundError("product not found", nil)
			}

			return re.JSON(http.StatusOK, product.Detail)
		})

		se.Router.GET("/api/miniapp/product-page/pricing", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			productID := miniAppProductID(re)
			if productID == "" {
				return re.BadRequestError("missing product id", nil)
			}

			product, err := miniappService.Product(re.Request.Context(), productID)
			if err != nil {
				return re.InternalServerError("load miniapp product pricing failed", err)
			}

			if product == nil {
				return re.NotFoundError("product not found", nil)
			}

			return re.JSON(http.StatusOK, product.Pricing)
		})

		se.Router.GET("/api/miniapp/product-page/package", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			productID := miniAppProductID(re)
			if productID == "" {
				return re.BadRequestError("missing product id", nil)
			}

			product, err := miniappService.Product(re.Request.Context(), productID)
			if err != nil {
				return re.InternalServerError("load miniapp product package failed", err)
			}

			if product == nil {
				return re.NotFoundError("product not found", nil)
			}

			return re.JSON(http.StatusOK, product.Package)
		})

		se.Router.GET("/api/miniapp/product-page/context", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			productID := miniAppProductID(re)
			if productID == "" {
				return re.BadRequestError("missing product id", nil)
			}

			product, err := miniappService.Product(re.Request.Context(), productID)
			if err != nil {
				return re.InternalServerError("load miniapp product context failed", err)
			}

			if product == nil {
				return re.NotFoundError("product not found", nil)
			}

			return re.JSON(http.StatusOK, product.Context)
		})

		se.Router.GET("/api/miniapp/product-page/coverage", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			coverage, err := miniappService.ProductCoverage(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp product coverage failed", err)
			}

			priority := strings.TrimSpace(re.Request.URL.Query().Get("priority"))
			if priority != "" {
				filtered := make([]miniappmodel.ProductCoverage, 0, len(coverage))
				for _, item := range coverage {
					if strings.EqualFold(item.Priority, priority) {
						filtered = append(filtered, item)
					}
				}
				coverage = filtered
			}

			return re.JSON(http.StatusOK, coverage)
		})

		se.Router.GET("/api/miniapp/product-page/coverage-summary", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			summary, err := miniappService.ProductCoverageSummary(re.Request.Context())
			if err != nil {
				return re.InternalServerError("load miniapp product coverage summary failed", err)
			}

			priority := strings.TrimSpace(re.Request.URL.Query().Get("priority"))
			if priority != "" {
				filteredBuckets := make([]miniappmodel.ProductCoverageBucket, 0, len(summary.ByPriority))
				for _, bucket := range summary.ByPriority {
					if strings.EqualFold(bucket.Priority, priority) {
						filteredBuckets = append(filteredBuckets, bucket)
					}
				}
				summary.ByPriority = filteredBuckets
				if !slices.ContainsFunc(summary.FirstBatch, func(item miniappmodel.ProductCoverage) bool {
					return strings.EqualFold(item.Priority, priority)
				}) {
					summary.FirstBatch = nil
				}
			}

			return re.JSON(http.StatusOK, summary)
		})

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

			freight, err := miniappService.FreightCost(re.Request.Context(), miniAppFreightScenario(re))
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
		se.Router.POST("/api/miniapp/cart-order/order/freight-cost", func(re *core.RequestEvent) error {
			if !authorizedMiniApp(re, cfg) {
				return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
			}

			freight, err := miniappService.FreightCost(re.Request.Context(), miniAppFreightScenario(re))
			if err != nil {
				return re.InternalServerError("load miniapp freight cost failed", err)
			}

			if freight == nil {
				return re.NotFoundError("freight cost scenario not found", nil)
			}

			return re.JSON(http.StatusOK, freight.Response)
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

		se.Router.POST("/api/pim/harvest", func(re *core.RequestEvent) error {
			if !authorized(re, cfg.Security.APIKey) {
				return re.UnauthorizedError("missing or invalid api key", nil)
			}

			result, err := service.Harvest(re.Request.Context(), re.App)
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

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func applyDefaultServeHTTPAddr(defaultAddr string) {
	defaultAddr = strings.TrimSpace(defaultAddr)
	if defaultAddr == "" || len(os.Args) < 2 || os.Args[1] != "serve" {
		return
	}

	for _, arg := range os.Args[2:] {
		if arg == "--http" || strings.HasPrefix(arg, "--http=") {
			return
		}
	}

	os.Args = append(os.Args, "--http="+defaultAddr)
}

func registerHooks(app *pocketbase.PocketBase, cfg config.Config, service *pim.Service) {
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

func registerCrons(app *pocketbase.PocketBase, cfg config.Config, service *pim.Service) {
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

func readProcurementRequest(re *core.RequestEvent) (pim.ProcurementRequest, error) {
	var request pim.ProcurementRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&request); err != nil {
		return pim.ProcurementRequest{}, err
	}

	return request, nil
}

func readProcurementStatusUpdateRequest(re *core.RequestEvent) (pim.ProcurementStatusUpdateRequest, error) {
	var request pim.ProcurementStatusUpdateRequest
	if err := json.NewDecoder(re.Request.Body).Decode(&request); err != nil {
		if err == io.EOF {
			return pim.ProcurementStatusUpdateRequest{}, nil
		}
		return pim.ProcurementStatusUpdateRequest{}, err
	}

	return request, nil
}

func readIntQuery(re *core.RequestEvent, key string, fallback int) int {
	value := strings.TrimSpace(re.Request.URL.Query().Get(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func renderProcurementWorkbenchHTML(summary pim.ProcurementWorkbenchSummary) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Procurement Workbench</title>
  <style>
    :root {
      --bg: #f5f1e8;
      --card: #fffaf0;
      --ink: #1d1d1b;
      --muted: #6e6259;
      --line: #d9cdbf;
      --accent: #8b3d16;
      --warning: #c27c0e;
      --danger: #b42318;
      --ok: #216e39;
    }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: "Segoe UI", "PingFang SC", sans-serif; background: linear-gradient(180deg, #efe6d7 0%, var(--bg) 100%); color: var(--ink); }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 24px; }
    .hero { display: flex; justify-content: space-between; align-items: end; gap: 16px; margin-bottom: 20px; }
    .hero h1 { margin: 0; font-size: 28px; }
    .hero p { margin: 6px 0 0; color: var(--muted); }
    .hero a { color: var(--accent); text-decoration: none; font-weight: 600; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 20px; }
    .card { background: var(--card); border: 1px solid var(--line); border-radius: 16px; padding: 14px 16px; box-shadow: 0 12px 24px rgba(29,29,27,0.04); }
    .metric { font-size: 26px; font-weight: 700; margin-top: 6px; }
    .label { color: var(--muted); font-size: 13px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; padding: 10px 8px; border-bottom: 1px solid var(--line); vertical-align: top; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    .status { display: inline-block; padding: 4px 8px; border-radius: 999px; font-size: 12px; font-weight: 700; background: #eee3d4; }
    .status.warning { background: #fff0d0; color: var(--warning); }
    .status.danger { background: #fde7e7; color: var(--danger); }
    .status.ok { background: #deefe1; color: var(--ok); }
    .actions { display: flex; flex-wrap: wrap; gap: 8px; }
    form { margin: 0; }
    button, select, input {
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 8px 10px;
      background: white;
      color: var(--ink);
      font: inherit;
    }
    button { cursor: pointer; background: var(--accent); color: white; border-color: var(--accent); }
    button.secondary { background: white; color: var(--accent); }
    .toolbar { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 14px; }
    .note { width: 180px; }
    .muted { color: var(--muted); font-size: 12px; }
    .risk { font-weight: 700; color: var(--danger); }
    .small { font-size: 12px; color: var(--muted); }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1>采购工作台</h1>
        <p>在 PocketBase Admin 内完成 review、导出、手工下单和收货推进。</p>
      </div>
      <div>
        <a href="/_/mrtang-admin">Mrtang Admin</a>
        <span class="small"> | </span>
        <a href="/_/">返回 Admin</a>
        <span class="small"> | </span>
        <a href="/_/#/collections/procurement_orders">打开 procurement_orders 集合</a>
      </div>
    </div>

    <div class="stats">
      <div class="card"><div class="label">总采购单</div><div class="metric">{{.TotalOrders}}</div></div>
      <div class="card"><div class="label">草稿</div><div class="metric">{{.DraftOrders}}</div></div>
      <div class="card"><div class="label">已复核</div><div class="metric">{{.ReviewedOrders}}</div></div>
      <div class="card"><div class="label">已导出</div><div class="metric">{{.ExportedOrders}}</div></div>
      <div class="card"><div class="label">已下单</div><div class="metric">{{.OrderedOrders}}</div></div>
      <div class="card"><div class="label">已收货</div><div class="metric">{{.ReceivedOrders}}</div></div>
      <div class="card"><div class="label">未完成风险单</div><div class="metric">{{.OpenRiskyOrders}}</div></div>
    </div>

    <div class="card">
      <div class="toolbar">
        <div class="muted">最近采购单：{{len .RecentOrders}} 条</div>
      </div>
      <table>
        <thead>
          <tr>
            <th>外部单号</th>
            <th>状态</th>
            <th>商品</th>
            <th>金额</th>
            <th>风险</th>
            <th>说明</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
        {{range .RecentOrders}}
          <tr>
            <td>
              <div><strong>{{.ExternalRef}}</strong></div>
              <div class="small">{{.ID}}</div>
            </td>
            <td><span class="status {{statusClass .Status .RiskyItemCount}}">{{.Status}}</span></td>
            <td>
              <div>{{.ItemCount}} items / {{printf "%.2f" .TotalQty}}</div>
              <div class="small">{{.SupplierCount}} suppliers</div>
            </td>
            <td>
              <div>成本 {{printf "%.2f" .TotalCostAmount}}</div>
            </td>
            <td>
              {{if gt .RiskyItemCount 0}}
              <span class="risk">{{.RiskyItemCount}} risky</span>
              {{else}}
              <span class="small">normal</span>
              {{end}}
            </td>
            <td>
              <div>{{.LastActionNote}}</div>
              <div class="small">{{.Updated}}</div>
            </td>
            <td>
              <div class="actions">
                <form method="post" action="/_/procurement-workbench/order/review">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <input class="note" type="text" name="note" placeholder="review note">
                  <button class="secondary" type="submit">Review</button>
                </form>
                <form method="post" action="/_/procurement-workbench/order/export">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <button class="secondary" type="submit">Export CSV</button>
                </form>
                <form method="post" action="/_/procurement-workbench/order/status">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <select name="status">
                    <option value="reviewed">reviewed</option>
                    <option value="exported">exported</option>
                    <option value="ordered">ordered</option>
                    <option value="received">received</option>
                    <option value="canceled">canceled</option>
                  </select>
                  <input class="note" type="text" name="note" placeholder="status note">
                  <button type="submit">Update</button>
                </form>
              </div>
            </td>
          </tr>
        {{else}}
          <tr><td colspan="7" class="muted">暂无采购单。先调用 procurement order create 接口生成草稿单。</td></tr>
        {{end}}
        </tbody>
      </table>
    </div>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("procurement-workbench").Funcs(template.FuncMap{
		"statusClass": func(status string, risky int) string {
			switch status {
			case pim.ProcurementStatusReceived:
				return "ok"
			case pim.ProcurementStatusCanceled:
				return "danger"
			default:
				if risky > 0 {
					return "warning"
				}
				return ""
			}
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, summary); err != nil {
		return fmt.Sprintf("<pre>render procurement workbench failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return builder.String()
}

func renderMrtangAdminHTML(summary pim.ProcurementWorkbenchSummary) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mrtang Admin</title>
  <style>
    :root {
      --paper: #f4efe7;
      --ink: #1f1f1a;
      --muted: #6f665e;
      --line: #d8ccbd;
      --card: #fffaf3;
      --accent: #8d3e16;
      --accent-soft: #f2e1d3;
      --danger: #b42318;
      --ok: #216e39;
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background:
      radial-gradient(circle at top right, #efe2d3 0, transparent 28%),
      linear-gradient(180deg, #f8f4ee 0%, var(--paper) 100%);
      font-family: "Segoe UI", "PingFang SC", sans-serif; }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 28px 24px 40px; }
    .hero { display: grid; grid-template-columns: 1.4fr .8fr; gap: 16px; margin-bottom: 18px; }
    .panel { background: var(--card); border: 1px solid var(--line); border-radius: 18px; padding: 18px; box-shadow: 0 18px 36px rgba(31,31,26,0.05); }
    h1, h2, h3, p { margin: 0; }
    h1 { font-size: 30px; }
    h2 { font-size: 18px; margin-bottom: 12px; }
    p.lead { margin-top: 8px; color: var(--muted); line-height: 1.55; }
    .actions, .grid, .mini-grid { display: grid; gap: 12px; }
    .actions { grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); margin-top: 16px; }
    .grid { grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); margin-top: 18px; }
    .mini-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
    .link-card, .stat { display: block; background: white; border: 1px solid var(--line); border-radius: 14px; padding: 14px 16px; text-decoration: none; color: inherit; }
    .link-card:hover { border-color: var(--accent); transform: translateY(-1px); transition: .16s ease; }
    .eyebrow { font-size: 12px; letter-spacing: .06em; text-transform: uppercase; color: var(--muted); }
    .title { margin-top: 6px; font-weight: 700; }
    .desc { margin-top: 6px; color: var(--muted); font-size: 13px; line-height: 1.45; }
    .metric { font-size: 28px; font-weight: 800; margin-top: 8px; }
    .accent { color: var(--accent); }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 8px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    .badge { display: inline-block; border-radius: 999px; padding: 4px 8px; font-size: 12px; font-weight: 700; background: var(--accent-soft); color: var(--accent); }
    .badge.ok { background: #deefe1; color: var(--ok); }
    .badge.danger { background: #fde7e7; color: var(--danger); }
    .muted { color: var(--muted); }
    .small { font-size: 12px; }
    code { font-family: Consolas, monospace; background: #f7efe5; padding: 1px 5px; border-radius: 6px; }
    @media (max-width: 860px) {
      .hero { grid-template-columns: 1fr; }
      .mini-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <section class="panel">
        <div class="eyebrow">Mrtang Admin</div>
        <h1>统一后台入口</h1>
        <p class="lead">这里不改 PocketBase 原生前端，只把 PIM、采购工作台、数据集合和常用联调入口集中放在一个 Admin 扩展页里。</p>
        <div class="actions">
          <a class="link-card" href="/_/procurement-workbench">
            <div class="eyebrow">Procurement</div>
            <div class="title">采购工作台</div>
            <div class="desc">Review、导出 CSV、手工推进到 ordered / received。</div>
          </a>
          <a class="link-card" href="/_/#/collections/procurement_orders">
            <div class="eyebrow">PocketBase</div>
            <div class="title">采购单集合</div>
            <div class="desc">直接查看 <code>procurement_orders</code> 记录。</div>
          </a>
          <a class="link-card" href="/_/#/collections/supplier_products">
            <div class="eyebrow">PocketBase</div>
            <div class="title">供应商商品</div>
            <div class="desc">审核标题、分类、价格和图片处理结果。</div>
          </a>
          <a class="link-card" href="/_/#/collections/category_mappings">
            <div class="eyebrow">PocketBase</div>
            <div class="title">分类映射</div>
            <div class="desc">维护上游原始类目到业务类目的映射。</div>
          </a>
        </div>
      </section>
      <aside class="panel">
        <h2>采购概览</h2>
        <div class="mini-grid">
          <div class="stat"><div class="eyebrow">Open</div><div class="metric">{{.OpenOrderCount}}</div></div>
          <div class="stat"><div class="eyebrow">Risky</div><div class="metric accent">{{.OpenRiskyOrders}}</div></div>
          <div class="stat"><div class="eyebrow">Recent</div><div class="metric">{{len .RecentOrders}}</div></div>
        </div>
        <div class="grid">
          <a class="link-card" href="/api/pim/healthz">
            <div class="eyebrow">Health</div>
            <div class="title">PIM 健康检查</div>
            <div class="desc">快速确认服务已正常启动。</div>
          </a>
          <a class="link-card" href="/docs/start.md">
            <div class="eyebrow">Docs</div>
            <div class="title">启动说明</div>
            <div class="desc">查看环境变量、接口和启动方式。</div>
          </a>
        </div>
      </aside>
    </div>

    <section class="panel">
      <h2>最近采购单</h2>
      <table>
        <thead>
          <tr>
            <th>外部单号</th>
            <th>状态</th>
            <th>商品数</th>
            <th>成本</th>
            <th>风险</th>
            <th>最近说明</th>
          </tr>
        </thead>
        <tbody>
        {{range .RecentOrders}}
          <tr>
            <td>
              <div><strong>{{.ExternalRef}}</strong></div>
              <div class="small muted">{{.ID}}</div>
            </td>
            <td>
              <span class="badge {{statusClass .Status .RiskyItemCount}}">{{.Status}}</span>
            </td>
            <td>{{.ItemCount}} / {{printf "%.2f" .TotalQty}}</td>
            <td>{{printf "%.2f" .TotalCostAmount}}</td>
            <td>
              {{if gt .RiskyItemCount 0}}
              <span class="badge danger">{{.RiskyItemCount}} risky</span>
              {{else}}
              <span class="badge ok">normal</span>
              {{end}}
            </td>
            <td class="muted">{{.LastActionNote}}</td>
          </tr>
        {{else}}
          <tr><td colspan="6" class="muted">暂无采购单。可先调用 <code>POST /api/pim/procurement/orders</code> 创建草稿单。</td></tr>
        {{end}}
        </tbody>
      </table>
    </section>

    <section class="grid">
      <div class="panel">
        <h2>常用 PIM API</h2>
        <div class="desc"><code>POST /api/pim/harvest</code></div>
        <div class="desc"><code>POST /api/pim/process</code></div>
        <div class="desc"><code>POST /api/pim/sync</code></div>
        <div class="desc"><code>GET /api/pim/procurement/workbench-summary</code></div>
      </div>
      <div class="panel">
        <h2>常用 Miniapp API</h2>
        <div class="desc"><code>GET /api/miniapp/homepage</code></div>
        <div class="desc"><code>GET /api/miniapp/category-page/tree</code></div>
        <div class="desc"><code>GET /api/miniapp/product-page/coverage-summary</code></div>
        <div class="desc"><code>GET /api/miniapp/cart-order/checkout-summary</code></div>
      </div>
    </section>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("mrtang-admin").Funcs(template.FuncMap{
		"statusClass": func(status string, risky int) string {
			switch status {
			case pim.ProcurementStatusReceived:
				return "ok"
			case pim.ProcurementStatusCanceled:
				return "danger"
			default:
				if risky > 0 {
					return "danger"
				}
				return ""
			}
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, summary); err != nil {
		return fmt.Sprintf("<pre>render mrtang admin failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return builder.String()
}

func authorized(re *core.RequestEvent, apiKey string) bool {
	if strings.TrimSpace(apiKey) == "" {
		return true
	}

	if re.Request.Header.Get("X-PIM-API-Key") == apiKey {
		return true
	}

	if bearer := strings.TrimSpace(strings.TrimPrefix(re.Request.Header.Get("Authorization"), "Bearer")); bearer == apiKey {
		return true
	}

	return false
}

func authorizedAdminPage(re *core.RequestEvent) bool {
	if re.Auth != nil && re.Auth.IsSuperuser() {
		return true
	}

	host := strings.TrimSpace(re.Request.Host)
	if host != "" {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
	}

	if isLoopbackHost(host) {
		return true
	}

	remoteAddr := strings.TrimSpace(re.Request.RemoteAddr)
	if remoteAddr == "" {
		return false
	}

	remoteHost := remoteAddr
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteHost = parsedHost
	}

	return isLoopbackHost(remoteHost)
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return false
	}

	if strings.EqualFold(host, "localhost") {
		return true
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	return addr.IsLoopback()
}

func authorizedMiniApp(re *core.RequestEvent, cfg config.Config) bool {
	accountID := strings.TrimSpace(cfg.MiniApp.AuthorizedAccountID)
	if accountID == "" {
		return true
	}

	return miniAppBearer(re) == accountID
}

func miniAppAuthorizationMode(cfg config.Config) string {
	if strings.TrimSpace(cfg.MiniApp.AuthorizedAccountID) != "" {
		return "upstream_ip_whitelist_and_bearer_account"
	}

	return "public"
}

func miniAppBearer(re *core.RequestEvent) string {
	header := strings.TrimSpace(re.Request.Header.Get("Authorization"))
	if header == "" {
		return ""
	}

	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}

	return ""
}

func filterContractsByPrefix(contracts []miniappmodel.Contract, prefix string) []miniappmodel.Contract {
	filtered := make([]miniappmodel.Contract, 0, len(contracts))
	for _, contract := range contracts {
		if strings.HasPrefix(contract.LocalPath, prefix) {
			filtered = append(filtered, contract)
		}
	}

	return filtered
}

func miniAppProductID(re *core.RequestEvent) string {
	query := re.Request.URL.Query()
	if id := strings.TrimSpace(query.Get("id")); id != "" {
		return id
	}

	spuID := strings.TrimSpace(query.Get("spuId"))
	skuID := strings.TrimSpace(query.Get("skuId"))
	if spuID == "" || skuID == "" {
		return ""
	}

	return spuID + "_" + skuID
}

func miniAppFreightScenario(re *core.RequestEvent) string {
	scenario := strings.TrimSpace(re.Request.URL.Query().Get("scenario"))
	if scenario == "" {
		return "preview"
	}

	return scenario
}

func serveMiniAppCartOperation(re *core.RequestEvent, cfg config.Config, service *miniappservice.Service, id string, label string) error {
	if !authorizedMiniApp(re, cfg) {
		return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
	}

	operation, err := service.CartOperation(re.Request.Context(), id)
	if err != nil {
		return re.InternalServerError(label+" failed", err)
	}

	if operation == nil {
		return re.NotFoundError("cart operation not found", nil)
	}

	return re.JSON(http.StatusOK, operation.Response)
}

func serveMiniAppOrderOperation(re *core.RequestEvent, cfg config.Config, service *miniappservice.Service, id string, label string) error {
	if !authorizedMiniApp(re, cfg) {
		return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
	}

	operation, err := service.OrderOperation(re.Request.Context(), id)
	if err != nil {
		return re.InternalServerError(label+" failed", err)
	}

	if operation == nil {
		return re.NotFoundError("order operation not found", nil)
	}

	return re.JSON(http.StatusOK, operation.Response)
}

func newMiniAppSource(cfg config.Config) miniappapi.Source {
	var base miniappapi.Source
	if strings.EqualFold(strings.TrimSpace(cfg.MiniApp.SourceMode), "http") {
		base = miniappapi.NewHTTPSource(miniappapi.HTTPSourceConfig{
			URL:                 cfg.MiniApp.SourceURL,
			AuthorizedAccountID: cfg.MiniApp.AuthorizedAccountID,
			UserAgent:           cfg.MiniApp.UserAgent,
			Timeout:             cfg.MiniApp.SourceTimeout,
		})
	} else {
		base = miniappapi.NewSnapshotSource(
			cfg.MiniApp.HomepageSnapshotFile,
			cfg.MiniApp.CategorySnapshotFile,
			cfg.MiniApp.ProductSnapshotFile,
			cfg.MiniApp.CartOrderSnapshotFile,
		)
	}

	return miniappapi.NewOverlaySource(base)
}

func init() {
	if os.Getenv("PB_ENCRYPTION_ENV") == "" {
		_ = os.Setenv("PB_ENCRYPTION_ENV", "MRTANG_PIM_ENCRYPTION_KEY")
	}
}
