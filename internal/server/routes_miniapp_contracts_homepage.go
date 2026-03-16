package server

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappservice "mrtang-pim/internal/miniapp/service"
)

func registerMiniAppRoutes(se *core.ServeEvent, cfg config.Config, miniappService *miniappservice.Service) {
	registerMiniAppContractRoutes(se, cfg, miniappService)
	registerMiniAppHomepageRoutes(se, cfg, miniappService)
	registerMiniAppCategoryRoutes(se, cfg, miniappService)
	registerMiniAppProductRoutes(se, cfg, miniappService)
	registerMiniAppCartOrderRoutes(se, cfg, miniappService)
}

func registerMiniAppContractRoutes(se *core.ServeEvent, cfg config.Config, miniappService *miniappservice.Service) {
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
}

func registerMiniAppHomepageRoutes(se *core.ServeEvent, cfg config.Config, miniappService *miniappservice.Service) {
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
}
