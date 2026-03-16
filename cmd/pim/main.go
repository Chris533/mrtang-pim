package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

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
	app := pocketbase.New()
	service := pim.NewService(cfg)
	miniappService := miniappservice.New(
		newMiniAppSource(cfg),
		nil,
	)

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
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

func newMiniAppSource(cfg config.Config) miniappapi.Source {
	if strings.EqualFold(strings.TrimSpace(cfg.MiniApp.SourceMode), "http") {
		return miniappapi.NewHTTPSource(miniappapi.HTTPSourceConfig{
			URL:                 cfg.MiniApp.SourceURL,
			AuthorizedAccountID: cfg.MiniApp.AuthorizedAccountID,
			UserAgent:           cfg.MiniApp.UserAgent,
			Timeout:             cfg.MiniApp.SourceTimeout,
		})
	}

	return miniappapi.NewSnapshotSource(
		cfg.MiniApp.HomepageSnapshotFile,
		cfg.MiniApp.CategorySnapshotFile,
		cfg.MiniApp.ProductSnapshotFile,
	)
}

func init() {
	if os.Getenv("PB_ENCRYPTION_ENV") == "" {
		_ = os.Setenv("PB_ENCRYPTION_ENV", "MRTANG_PIM_ENCRYPTION_KEY")
	}
}
