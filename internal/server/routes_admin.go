package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/admin"
	"mrtang-pim/internal/adminapp"
	"mrtang-pim/internal/config"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

func registerAdminRoutes(se *core.ServeEvent, cfg config.Config, service *pim.Service, miniappService *miniappservice.Service) {
	se.Router.GET("/api/pim/healthz", func(re *core.RequestEvent) error {
		return re.JSON(http.StatusOK, map[string]any{
			"service": "mrtang-pim",
			"status":  "ok",
			"runtime": map[string]any{
				"appEnv":        strings.TrimSpace(os.Getenv("APP_ENV")),
				"sourceMode":    strings.TrimSpace(cfg.MiniApp.SourceMode),
				"sourceURL":     strings.TrimSpace(cfg.MiniApp.SourceURL),
				"connector":     strings.TrimSpace(cfg.Supplier.Connector),
				"capabilities":  service.ConnectorCapabilities(),
				"apiKeyEnabled": strings.TrimSpace(cfg.Security.APIKey) != "",
				"codeMarkers": map[string]any{
					"submitOrderProgressLogs": true,
					"resolveProductHydration": true,
					"fallbackContextGuard":    true,
					"healthSchemaVersion":     "2026-03-20.1",
				},
			},
		})
	})

	registerAdminSlashRedirect(se, "/_/mrtang-admin/", "/_/mrtang-admin")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/audit/", "/_/mrtang-admin/audit")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/target-sync/", "/_/mrtang-admin/target-sync")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/", "/_/mrtang-admin/source")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/categories/", "/_/mrtang-admin/source/categories")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/products/", "/_/mrtang-admin/source/products")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/assets/", "/_/mrtang-admin/source/assets")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/asset-jobs/", "/_/mrtang-admin/source/asset-jobs")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/product-jobs/", "/_/mrtang-admin/source/product-jobs")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/logs/", "/_/mrtang-admin/source/logs")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/procurement/", "/_/mrtang-admin/procurement")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/backend-release/", "/_/mrtang-admin/backend-release")

	se.Router.GET("/_/mrtang-admin/app.css", func(re *core.RequestEvent) error {
		return serveAdminAsset(re, "static/app.css", "text/css; charset=utf-8")
	})

	se.Router.GET("/_/mrtang-admin/app.js", func(re *core.RequestEvent) error {
		return serveAdminAsset(re, "static/app.js", "application/javascript; charset=utf-8")
	})

	se.Router.GET("/_/mrtang-admin/vendor/htm-preact-standalone.mjs", func(re *core.RequestEvent) error {
		return serveAdminAsset(re, "static/vendor/htm-preact-standalone.mjs", "application/javascript; charset=utf-8")
	})

	se.Router.GET("/api/pim/admin/dashboard", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "dashboard") {
			return re.ForbiddenError("当前账号没有后台总览权限。", nil)
		}
		return re.JSON(http.StatusOK, admin.BuildDashboardAPIData(
			re.Request.Context(),
			re.App,
			cfg,
			service,
			miniappService,
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/dashboard/miniapp-live", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "dashboard") {
			return re.ForbiddenError("当前账号没有后台总览权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 20*time.Second)
		defer cancel()
		return re.JSON(http.StatusOK, admin.BuildDashboardMiniappAPIData(loadCtx, re.App, cfg, service, miniappService))
	})

	se.Router.GET("/api/pim/admin/audit", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "dashboard") {
			return re.ForbiddenError("当前账号没有后台总览权限。", nil)
		}
		return re.JSON(http.StatusOK, admin.BuildAuditAPIData(
			re.Request.Context(),
			re.App,
			cfg,
			service,
			miniappService,
			admin.AuditFilter{
				Domain:   strings.TrimSpace(re.Request.URL.Query().Get("domain")),
				Status:   strings.TrimSpace(re.Request.URL.Query().Get("status")),
				Query:    strings.TrimSpace(re.Request.URL.Query().Get("q")),
				Page:     readQueryInt(re, "page", 1),
				PageSize: readQueryInt(re, "pageSize", 20),
			},
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/access", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "dashboard") {
			return re.ForbiddenError("当前账号没有后台总览权限。", nil)
		}
		return re.JSON(http.StatusOK, map[string]any{
			"canAccessSource":      authorizedAdminModule(re, cfg, "source"),
			"canAccessProcurement": authorizedAdminModule(re, cfg, "procurement"),
		})
	})

	se.Router.GET("/api/pim/admin/target-sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}

		summary, err := service.TargetSyncBaseSummary(re.App, strings.TrimSpace(cfg.MiniApp.SourceMode), miniappService.RawAuthStatus())
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildTargetSyncAPIData(
				cfg,
				pim.TargetSyncBaseSummary{SourceMode: strings.TrimSpace(cfg.MiniApp.SourceMode), RawAuthStatus: miniappService.RawAuthStatus()},
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"生成抓取入库基础摘要失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildTargetSyncAPIData(
			cfg,
			summary,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/target-sync/live", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 20*time.Second)
		defer cancel()

		dataset, err := miniappService.Dataset(loadCtx)
		if err != nil {
			summary, fallbackErr := service.TargetSyncStoredLiveSummary(re.App, strings.TrimSpace(cfg.MiniApp.SourceMode))
			if fallbackErr == nil {
				return re.JSON(http.StatusOK, admin.BuildTargetSyncLiveAPIData(
					summary,
					"加载源站实时摘要失败，已回退到已落库结果："+err.Error(),
				))
			}
			return re.JSON(http.StatusOK, admin.BuildTargetSyncLiveAPIData(
				pim.TargetSyncLiveSummary{SourceMode: strings.TrimSpace(cfg.MiniApp.SourceMode)},
				"加载 miniapp dataset 失败："+err.Error(),
			))
		}
		summary, err := service.TargetSyncLiveSummary(re.App, *dataset)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildTargetSyncLiveAPIData(
				pim.TargetSyncLiveSummary{SourceMode: strings.TrimSpace(dataset.Meta.Source)},
				"生成 raw 实时摘要失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildTargetSyncLiveAPIData(summary, ""))
	})

	se.Router.GET("/api/pim/admin/target-sync/checkout-live", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 20*time.Second)
		defer cancel()

		dataset, err := miniappService.Dataset(loadCtx)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildTargetSyncCheckoutLiveAPIData(pim.TargetSyncCheckoutLiveSummary{}, "加载 miniapp dataset 失败："+err.Error()))
		}
		return re.JSON(http.StatusOK, admin.BuildTargetSyncCheckoutLiveAPIData(service.TargetSyncCheckoutLiveSummary(*dataset), ""))
	})

	se.Router.POST("/api/pim/admin/source/import", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}

		scope := strings.TrimSpace(re.Request.FormValue("scope"))
		dataset, err := miniappService.Dataset(re.Request.Context())
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "加载 miniapp dataset 失败：" + err.Error(),
			})
		}

		summary, err := service.ImportMiniappSource(re.Request.Context(), re.App, *dataset, scope)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "导入源数据失败：" + err.Error(),
			})
		}

		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已导入源数据：分类 %d 新增/%d 更新，商品 %d 新增/%d 更新，图片 %d 新增/%d 更新", summary.CategoriesCreated, summary.CategoriesUpdated, summary.ProductsCreated, summary.ProductsUpdated, summary.AssetsCreated, summary.AssetsUpdated),
			"summary": summary,
		})
	})

	se.Router.POST("/api/pim/admin/target-sync/jobs/ensure", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}

		job, err := service.EnsureTargetSyncJobSpec(
			re.App,
			cfg.MiniApp.SourceMode,
			strings.TrimSpace(re.Request.FormValue("entityType")),
			strings.TrimSpace(re.Request.FormValue("scopeKey")),
			strings.TrimSpace(re.Request.FormValue("scopeLabel")),
		)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "保存抓取入库任务失败：" + err.Error(),
			})
		}

		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "已保存抓取入库任务。",
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/target-sync/jobs/run", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}

		run, err := service.StartTargetSyncAsync(
			re.App,
			func(ctx context.Context, entityType string, scopeKey string) (*miniappmodel.Dataset, error) {
				return miniappService.TargetSyncDataset(ctx, entityType, scopeKey)
			},
			func(ctx context.Context, sections []miniappmodel.CategorySection, scopeKey string) (*miniappmodel.Dataset, error) {
				return miniappService.TargetSyncProductsFromSections(ctx, sections, scopeKey)
			},
			strings.TrimSpace(re.Request.FormValue("entityType")),
			strings.TrimSpace(re.Request.FormValue("scopeKey")),
			strings.TrimSpace(re.Request.FormValue("scopeLabel")),
			targetSyncActor(re),
		)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "已在执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":      false,
				"message": "启动抓取入库失败：" + err.Error(),
				"run":     run,
			})
		}

		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": "抓取入库任务已启动。",
			"run":     run,
		})
	})

	se.Router.GET("/api/pim/admin/target-sync/run", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少抓取运行记录 ID。",
			})
		}
		run, err := service.GetTargetSyncRun(re.App, id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]any{
				"ok":      false,
				"message": "抓取运行记录不存在。",
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":  true,
			"run": run,
		})
	})

	se.Router.POST("/api/pim/admin/target-sync/run/retry-failed-branches", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}
		runID := strings.TrimSpace(re.Request.FormValue("runId"))
		if runID == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少运行记录 ID。",
			})
		}
		runs, warnings, err := service.RetryFailedTargetSyncBranches(
			re.App,
			func(ctx context.Context, entityType string, scopeKey string) (*miniappmodel.Dataset, error) {
				return miniappService.TargetSyncDataset(ctx, entityType, scopeKey)
			},
			func(ctx context.Context, sections []miniappmodel.CategorySection, scopeKey string) (*miniappmodel.Dataset, error) {
				return miniappService.TargetSyncProductsFromSections(ctx, sections, scopeKey)
			},
			runID,
			targetSyncActor(re),
		)
		if err != nil {
			message := "重跑失败分支失败：" + err.Error()
			if len(warnings) > 0 {
				message += "；" + strings.Join(warnings, "；")
			}
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":       false,
				"message":  message,
				"runs":     runs,
				"warnings": warnings,
			})
		}
		message := fmt.Sprintf("已启动 %d 个失败分支重跑任务。", len(runs))
		if len(warnings) > 0 {
			message += "；" + strings.Join(warnings, "；")
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":       true,
			"message":  message,
			"runs":     runs,
			"warnings": warnings,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/status", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		status := strings.TrimSpace(re.Request.FormValue("status"))
		if id == "" || status == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少商品记录 ID 或审核状态。",
			})
		}
		if err := service.UpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, id, status, sourceActionNote(re), sourceActionActor(re)); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "更新商品审核状态失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "商品审核状态已更新。",
		})
	})

	se.Router.POST("/api/pim/admin/source/products/promote", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少商品记录 ID。",
			})
		}
		if err := service.PromoteSourceProductWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "桥接商品失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "商品已桥接到同步链。",
		})
	})

	se.Router.POST("/api/pim/admin/source/products/promote-sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少商品记录 ID。",
			})
		}
		if err := service.PromoteAndSyncSourceProductWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "桥接并同步商品失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "商品已桥接并同步到后端。",
		})
	})

	se.Router.POST("/api/pim/admin/source/products/retry-sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少商品记录 ID。",
			})
		}
		if err := service.RetrySourceProductSyncWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "重试商品同步失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "已触发商品同步重试。",
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-status", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		status := strings.TrimSpace(re.Request.FormValue("status"))
		ids := readIDList(re.Request.FormValue("productIds"))
		if status == "" || len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "请先选择商品并指定审核状态。",
			})
		}
		summary, err := service.BatchUpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, ids, status, sourceActionNote(re), sourceActionActor(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "批量更新商品审核状态失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("批量更新完成：成功 %d，失败 %d。", summary.Processed, summary.Failed),
			"summary": summary,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-status-filtered", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		status := strings.TrimSpace(re.Request.FormValue("status"))
		filter := pim.SourceReviewFilter{
			CategoryKey:   strings.TrimSpace(re.Request.FormValue("categoryKey")),
			CategoryKeys:  strings.TrimSpace(re.Request.FormValue("categoryKeys")),
			ProductStatus: strings.TrimSpace(re.Request.FormValue("productStatus")),
			SyncState:     strings.TrimSpace(re.Request.FormValue("syncState")),
			Query:         strings.TrimSpace(re.Request.FormValue("q")),
		}
		ids, err := service.SourceProductIDsForFilter(re.App, filter)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "读取当前筛选商品失败：" + err.Error(),
			})
		}
		if status == "" || len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "当前筛选结果下没有可批量更新的商品。",
			})
		}
		summary, err := service.BatchUpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, ids, status, sourceActionNote(re), sourceActionActor(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "按当前筛选结果批量更新商品审核状态失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("按当前筛选结果批量更新完成：成功 %d，失败 %d。", summary.Processed, summary.Failed),
			"summary": summary,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-promote", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		ids := readIDList(re.Request.FormValue("productIds"))
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "请先选择要桥接的商品。",
			})
		}
		job, err := service.StartSourceProductPromoteAsyncForIDs(re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "启动批量加入发布队列失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已启动 %d 个商品的加入发布队列任务。", job.Total),
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-promote-filtered", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		filter := pim.SourceReviewFilter{
			CategoryKey:   strings.TrimSpace(re.Request.FormValue("categoryKey")),
			CategoryKeys:  strings.TrimSpace(re.Request.FormValue("categoryKeys")),
			ProductStatus: strings.TrimSpace(re.Request.FormValue("productStatus")),
			SyncState:     strings.TrimSpace(re.Request.FormValue("syncState")),
			Query:         strings.TrimSpace(re.Request.FormValue("q")),
		}
		ids, err := service.SourceProductIDsForFilter(re.App, filter)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "读取当前筛选商品失败：" + err.Error(),
			})
		}
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "当前筛选结果下没有可加入发布队列的商品。",
			})
		}
		job, err := service.StartSourceProductPromoteAsyncForFilter(re.App, filter, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "启动按当前筛选结果批量加入发布队列失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已启动当前筛选结果 %d 个商品的加入发布队列任务。", job.Total),
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-promote-sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		ids := readIDList(re.Request.FormValue("productIds"))
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "请先选择要桥接并同步的商品。",
			})
		}
		job, err := service.StartSourceProductPromoteSyncAsyncForIDs(re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "启动批量加入发布队列并发布失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已启动 %d 个商品的加入发布队列并发布任务。", job.Total),
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-promote-sync-filtered", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		filter := pim.SourceReviewFilter{
			CategoryKey:   strings.TrimSpace(re.Request.FormValue("categoryKey")),
			CategoryKeys:  strings.TrimSpace(re.Request.FormValue("categoryKeys")),
			ProductStatus: strings.TrimSpace(re.Request.FormValue("productStatus")),
			SyncState:     strings.TrimSpace(re.Request.FormValue("syncState")),
			Query:         strings.TrimSpace(re.Request.FormValue("q")),
		}
		ids, err := service.SourceProductIDsForFilter(re.App, filter)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "读取当前筛选商品失败：" + err.Error(),
			})
		}
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "当前筛选结果下没有可发布的商品。",
			})
		}
		job, err := service.StartSourceProductPromoteSyncAsyncForFilter(re.App, filter, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "启动按当前筛选结果加入发布队列并发布失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已启动当前筛选结果 %d 个商品的加入发布队列并发布任务。", job.Total),
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-retry-sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		ids := readIDList(re.Request.FormValue("productIds"))
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "请先选择要重试同步的商品。",
			})
		}
		job, err := service.StartSourceProductRetrySyncAsyncForIDs(re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "启动批量重试发布失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已启动 %d 个商品的批量重试发布任务。", job.Total),
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/source/products/batch-retry-sync-filtered", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		filter := pim.SourceReviewFilter{
			CategoryKey:   strings.TrimSpace(re.Request.FormValue("categoryKey")),
			CategoryKeys:  strings.TrimSpace(re.Request.FormValue("categoryKeys")),
			ProductStatus: strings.TrimSpace(re.Request.FormValue("productStatus")),
			SyncState:     strings.TrimSpace(re.Request.FormValue("syncState")),
			Query:         strings.TrimSpace(re.Request.FormValue("q")),
		}
		ids, err := service.SourceProductIDsForFilter(re.App, filter)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "读取当前筛选商品失败：" + err.Error(),
			})
		}
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "当前筛选结果下没有可重试发布的商品。",
			})
		}
		job, err := service.StartSourceProductRetrySyncAsyncForFilter(re.App, filter, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "启动按当前筛选结果批量重试发布失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusAccepted, map[string]any{
			"ok":      true,
			"message": fmt.Sprintf("已启动当前筛选结果 %d 个商品的重试发布任务。", job.Total),
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/source/product-jobs/retry", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "缺少商品任务 ID。"})
		}
		job, err := service.RetrySourceProductJob(re.App, id, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "重新执行商品任务失败：" + err.Error()})
		}
		return re.JSON(http.StatusAccepted, map[string]any{"ok": true, "message": "商品发布任务已重新启动。", "job": job})
	})

	se.Router.POST("/api/pim/admin/source/assets/process", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		assetID := strings.TrimSpace(re.Request.FormValue("id"))
		if assetID == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少图片记录 ID。",
			})
		}
		if err := service.ProcessSourceAssetWithAudit(re.Request.Context(), re.App, assetID, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "处理图片失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "图片已进入处理流程。",
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/download", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		assetID := strings.TrimSpace(re.Request.FormValue("id"))
		if assetID == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少图片记录 ID。",
			})
		}
		if err := service.DownloadSourceAssetOriginalWithAudit(re.Request.Context(), re.App, assetID, sourceActionActor(re), sourceActionNote(re)); err != nil {
			statusCode := http.StatusInternalServerError
			message := "下载原图失败：" + err.Error()
			if strings.Contains(strings.ToLower(err.Error()), "missing source_url") {
				statusCode = http.StatusBadRequest
				message = "下载原图失败：该图片资产缺少可用源图地址，请先检查抓取结果。"
			}
			return re.JSON(statusCode, map[string]any{
				"ok":      false,
				"message": message,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "原图已下载到本地资源。",
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/download-pending", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		progress, err := service.StartSourceAssetOriginalDownloadAsync(re.App, 50, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "批量下载原图失败：" + err.Error(),
				"progress": progress,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  "原图批量下载任务已启动。",
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/download-selected", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		ids := readIDList(re.Request.FormValue("assetIds"))
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "请先选择要下载原图的图片。",
			})
		}
		progress, err := service.StartSourceAssetOriginalDownloadSelectionAsync(re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "启动选中图片原图下载失败：" + err.Error(),
				"progress": progress,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  fmt.Sprintf("已启动 %d 张图片的原图下载任务。", len(ids)),
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/download-filtered", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		filter := pim.SourceReviewFilter{
			AssetStatus:    strings.TrimSpace(re.Request.FormValue("assetStatus")),
			OriginalStatus: strings.TrimSpace(re.Request.FormValue("originalStatus")),
			AssetIDs:       strings.TrimSpace(re.Request.FormValue("assetIds")),
			Query:          strings.TrimSpace(re.Request.FormValue("q")),
		}
		ids, err := service.SourceAssetIDsForFilter(re.App, filter)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "读取当前筛选图片失败：" + err.Error(),
			})
		}
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "当前筛选结果下没有可下载原图的图片。",
			})
		}
		progress, err := service.StartSourceAssetOriginalDownloadSelectionAsync(re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "启动当前筛选结果原图下载失败：" + err.Error(),
				"progress": progress,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  fmt.Sprintf("已启动当前筛选结果 %d 张图片的原图下载任务。", len(ids)),
			"progress": progress,
		})
	})

	se.Router.GET("/api/pim/admin/source/assets/download-progress", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少原图下载任务 ID。",
			})
		}
		progress, ok := service.SourceAssetOriginalDownloadProgress(re.App, id)
		if !ok {
			return re.JSON(http.StatusNotFound, map[string]any{
				"ok":      false,
				"message": "原图下载任务不存在。",
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/process-pending", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		progress, err := service.StartSourceAssetProcessAsync(re.App, 20, false, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "批量处理待处理图片失败：" + err.Error(),
				"progress": progress,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  "图片批量处理任务已启动。",
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/reprocess-failed", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		progress, err := service.StartSourceAssetProcessAsync(re.App, 50, true, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "批量重处理失败图片失败：" + err.Error(),
				"progress": progress,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  "失败图片重处理任务已启动。",
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/process-selected", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		ids := readIDList(re.Request.FormValue("assetIds"))
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "请先选择要处理的图片。",
			})
		}
		failedOnly := strings.EqualFold(strings.TrimSpace(re.Request.FormValue("failedOnly")), "true")
		progress, err := service.StartSourceAssetProcessSelectionAsync(re.App, ids, failedOnly, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "启动选中图片处理任务失败：" + err.Error(),
				"progress": progress,
			})
		}
		message := fmt.Sprintf("已启动 %d 张图片的处理任务。", len(ids))
		if failedOnly {
			message = fmt.Sprintf("已启动 %d 张选中失败图片的重处理任务。", len(ids))
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  message,
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/assets/process-filtered", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		filter := pim.SourceReviewFilter{
			AssetStatus:    strings.TrimSpace(re.Request.FormValue("assetStatus")),
			OriginalStatus: strings.TrimSpace(re.Request.FormValue("originalStatus")),
			AssetIDs:       strings.TrimSpace(re.Request.FormValue("assetIds")),
			Query:          strings.TrimSpace(re.Request.FormValue("q")),
		}
		ids, err := service.SourceAssetIDsForFilter(re.App, filter)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "读取当前筛选图片失败：" + err.Error(),
			})
		}
		if len(ids) == 0 {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "当前筛选结果下没有可处理的图片。",
			})
		}
		failedOnly := strings.EqualFold(strings.TrimSpace(re.Request.FormValue("failedOnly")), "true")
		progress, err := service.StartSourceAssetProcessSelectionAsync(re.App, ids, failedOnly, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":       false,
				"message":  "启动当前筛选结果图片处理失败：" + err.Error(),
				"progress": progress,
			})
		}
		message := fmt.Sprintf("已启动当前筛选结果 %d 张图片的处理任务。", len(ids))
		if failedOnly {
			message = fmt.Sprintf("已启动当前筛选结果 %d 张失败图片的重处理任务。", len(ids))
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"message":  message,
			"progress": progress,
		})
	})

	se.Router.GET("/api/pim/admin/source/assets/process-progress", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少图片处理任务 ID。",
			})
		}
		progress, ok := service.SourceAssetProcessProgressByID(re.App, id)
		if !ok {
			return re.JSON(http.StatusNotFound, map[string]any{
				"ok":      false,
				"message": "图片处理任务不存在。",
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":       true,
			"progress": progress,
		})
	})

	se.Router.POST("/api/pim/admin/source/asset-jobs/retry", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少图片任务 ID。",
			})
		}
		job, err := service.RetrySourceAssetJob(re.App, id, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			statusCode := http.StatusInternalServerError
			if strings.Contains(err.Error(), "执行中") {
				statusCode = http.StatusConflict
			}
			return re.JSON(statusCode, map[string]any{
				"ok":      false,
				"message": "重新执行图片任务失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "图片任务已重新启动。",
			"job":     job,
		})
	})

	se.Router.POST("/api/pim/admin/procurement/order/review", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		note := strings.TrimSpace(re.Request.FormValue("note"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少采购单 ID。",
			})
		}
		order, err := service.ReviewProcurementOrderWithAudit(re.Request.Context(), re.App, id, note, procurementActionActor(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "复核采购单失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "采购单已复核。",
			"order":   order,
		})
	})

	se.Router.POST("/api/pim/admin/procurement/order/export", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		note := strings.TrimSpace(re.Request.FormValue("note"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少采购单 ID。",
			})
		}
		order, err := service.ExportProcurementOrderWithAudit(re.Request.Context(), re.App, id, procurementActionActor(re), note)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "导出采购单失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "采购单已导出。",
			"order":   order,
		})
	})

	se.Router.POST("/api/pim/admin/procurement/order/submit", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		note := strings.TrimSpace(re.Request.FormValue("note"))
		if id == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少采购单 ID。",
			})
		}
		order, err := service.SubmitProcurementOrderWithAudit(re.Request.Context(), re.App, id, note, procurementActionActor(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "提交采购单失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "采购单已提交到供应商连接器。",
			"order":   order,
		})
	})

	se.Router.POST("/api/pim/admin/procurement/order/status", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}
		id := strings.TrimSpace(re.Request.FormValue("id"))
		status := strings.TrimSpace(re.Request.FormValue("status"))
		note := strings.TrimSpace(re.Request.FormValue("note"))
		if id == "" || status == "" {
			return re.JSON(http.StatusBadRequest, map[string]any{
				"ok":      false,
				"message": "缺少采购单 ID 或状态。",
			})
		}
		order, err := service.UpdateProcurementOrderStatusWithAudit(re.Request.Context(), re.App, id, status, note, procurementActionActor(re))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]any{
				"ok":      false,
				"message": "更新采购单状态失败：" + err.Error(),
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"ok":      true,
			"message": "采购单状态已更新。",
			"order":   order,
		})
	})

	se.Router.GET("/api/pim/admin/source", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		summary, err := service.SourceReviewWorkbench(loadCtx, re.App, 6, 6, pim.SourceReviewFilter{PageSize: 6})
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceModuleAPIData(
				pim.SourceReviewWorkbenchSummary{},
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载源数据模块失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceModuleAPIData(
			summary,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/categories", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		filter := pim.SourceCategoryFilter{
			Query:    strings.TrimSpace(re.Request.URL.Query().Get("q")),
			Page:     readQueryInt(re, "page", 1),
			PageSize: readQueryInt(re, "pageSize", 24),
		}
		summary, err := service.SourceCategories(loadCtx, re.App, filter)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceCategoriesAPIData(
				pim.SourceCategoriesSummary{},
				filter,
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载源数据分类失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceCategoriesAPIData(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/products", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		filter := readSourceReviewFilter(re)
		filter.AssetStatus = ""
		filter.AssetPage = 1
		summary, err := service.SourceReviewWorkbench(loadCtx, re.App, 24, 1, filter)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceProductsAPIData(
				pim.SourceReviewWorkbenchSummary{},
				filter,
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载源数据商品失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceProductsAPIData(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/products/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source product id", nil)
		}
		detail, err := service.SourceProductDetail(loadCtx, re.App, id)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceProductDetailAPIData(
				pim.SourceProductDetail{},
				strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载商品详情失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceProductDetailAPIData(
			detail,
			strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/assets", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		filter := readSourceReviewFilter(re)
		filter.ProductStatus = ""
		filter.SyncState = ""
		filter.ProductPage = 1
		summary, err := service.SourceReviewWorkbench(loadCtx, re.App, 1, 24, filter)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceAssetsAPIData(
				pim.SourceReviewWorkbenchSummary{},
				filter,
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载源数据图片失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceAssetsAPIData(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/assets/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source asset id", nil)
		}
		detail, err := service.SourceAssetDetail(loadCtx, re.App, id)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceAssetDetailAPIData(
				pim.SourceAssetDetail{},
				strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载图片详情失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceAssetDetailAPIData(
			detail,
			strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/asset-jobs", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		filter := readSourceAssetJobFilter(re)
		summary, err := service.SourceAssetJobs(loadCtx, re.App, filter)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceAssetJobsAPIData(
				pim.SourceAssetJobsSummary{},
				filter,
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载图片任务失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceAssetJobsAPIData(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/asset-jobs/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source asset job id", nil)
		}
		detail, err := service.SourceAssetJobDetail(loadCtx, re.App, id)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceAssetJobDetailAPIData(
				pim.SourceAssetJobDetail{},
				strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载图片任务详情失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceAssetJobDetailAPIData(
			detail,
			strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/product-jobs", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		filter := readSourceProductJobFilter(re)
		summary, err := service.SourceProductJobs(loadCtx, re.App, filter)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceProductJobsAPIData(
				pim.SourceProductJobsSummary{},
				filter,
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载商品发布任务失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceProductJobsAPIData(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/product-jobs/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source product job id", nil)
		}
		detail, err := service.SourceProductJobDetail(loadCtx, re.App, id)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceProductJobDetailAPIData(
				pim.SourceProductJobDetail{},
				strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载商品发布任务详情失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceProductJobDetailAPIData(
			detail,
			strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/source/logs", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}
		data, err := buildSourceLogsPageData(re)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildSourceLogsAPIData(
				admin.SourceLogsPageData{},
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载源数据日志失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildSourceLogsAPIData(
			data,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/procurement", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		summary, err := service.ProcurementWorkbenchSummaryFiltered(
			loadCtx,
			re.App,
			readQueryInt(re, "pageSize", 20),
			strings.TrimSpace(re.Request.URL.Query().Get("status")),
			strings.TrimSpace(re.Request.URL.Query().Get("risk")),
			strings.TrimSpace(re.Request.URL.Query().Get("q")),
			readQueryInt(re, "page", 1),
		)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildProcurementAPIData(
				pim.ProcurementWorkbenchSummary{},
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载采购工作台失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildProcurementAPIData(
			summary,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/procurement/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}
		loadCtx, cancel := context.WithTimeout(re.Request.Context(), 4*time.Second)
		defer cancel()
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}
		order, err := service.GetProcurementOrder(loadCtx, re.App, id)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildProcurementDetailAPIData(
				pim.ProcurementOrder{},
				strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载采购详情失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildProcurementDetailAPIData(
			order,
			strings.TrimSpace(re.Request.URL.Query().Get("returnTo")),
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/api/pim/admin/backend-release", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有发布准备权限。", nil)
		}
		summary, err := service.BackendReleaseSummary(re.Request.Context(), re.App, 12)
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildBackendReleaseAPIData(
				pim.BackendReleaseSummary{},
				strings.TrimSpace(re.Request.URL.Query().Get("message")),
				"加载 backend 发布准备失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildBackendReleaseAPIData(
			summary,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.POST("/api/pim/admin/backend-release/category-mappings", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有发布准备权限。", nil)
		}
		if err := service.SaveBackendCategoryMapping(
			re.Request.Context(),
			re.App,
			re.Request.FormValue("sourceKey"),
			re.Request.FormValue("backendCollection"),
			re.Request.FormValue("backendPath"),
			re.Request.FormValue("note"),
		); err != nil {
			return re.BadRequestError("保存分类发布映射失败："+err.Error(), nil)
		}
		return re.JSON(http.StatusOK, map[string]any{"message": "分类发布映射已保存。"})
	})

	se.Router.POST("/api/pim/admin/backend-release/category-publish", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有发布准备权限。", nil)
		}
		item, err := service.PublishBackendCategory(
			re.Request.Context(),
			re.App,
			re.Request.FormValue("sourceKey"),
			re.Request.FormValue("backendCollection"),
			re.Request.FormValue("backendPath"),
			re.Request.FormValue("note"),
		)
		if err != nil {
			return re.BadRequestError("创建 backend 分类失败："+err.Error(), nil)
		}
		return re.JSON(http.StatusOK, map[string]any{
			"message": "分类已创建到 backend。",
			"item":    item,
		})
	})

	se.Router.POST("/api/pim/admin/backend-release/category-publish-batch", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有发布准备权限。", nil)
		}
		sourceKeys := strings.Split(strings.TrimSpace(re.Request.FormValue("sourceKeys")), ",")
		result, err := service.PublishBackendCategoriesBatch(
			re.Request.Context(),
			re.App,
			sourceKeys,
		)
		if err != nil {
			return re.BadRequestError("批量创建 backend 分类失败："+err.Error(), nil)
		}
		message := fmt.Sprintf("已完成 %d 个分类的 backend 创建，其中成功 %d，失败 %d。", result.Requested, result.Published, result.Failed)
		return re.JSON(http.StatusOK, map[string]any{
			"message": message,
			"result":  result,
		})
	})

	se.Router.POST("/api/pim/admin/backend-release/cleanup-assets", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有发布准备权限。", nil)
		}
		result, err := service.CleanupBackendAssets(re.Request.Context())
		if err != nil {
			return re.BadRequestError("清理 backend 冗余图片失败："+err.Error(), nil)
		}
		message := fmt.Sprintf("已完成 backend 冗余图片清理：扫描 %d 个 PIM 图片，删除 %d 个未引用图片，失败 %d 个。", result.TaggedAssets, result.DeletedAssets, result.FailedAssets)
		return re.JSON(http.StatusOK, map[string]any{
			"message": message,
			"result":  result,
		})
	})

	se.Router.GET("/api/pim/admin/backend-release/product-preview", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有发布准备权限。", nil)
		}
		preview, err := service.PreviewBackendReleasePayload(re.Request.Context(), re.App, strings.TrimSpace(re.Request.URL.Query().Get("id")))
		if err != nil {
			return re.JSON(http.StatusOK, admin.BuildBackendReleasePreviewAPIData(
				pim.BackendReleasePayloadPreview{},
				"",
				"生成发布 payload 预览失败："+err.Error(),
			))
		}
		return re.JSON(http.StatusOK, admin.BuildBackendReleasePreviewAPIData(preview, "", ""))
	})

	se.Router.GET("/_/mrtang-admin", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"总览",
			"后台首页先秒开壳子，再异步拉 coverage、source capture 和最近动作。",
			"/_/mrtang-admin",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/audit", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/audit")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"审计",
			"统一审计入口，汇总源数据和采购动作并支持筛选。",
			"/_/mrtang-admin/audit",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/target-sync", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/target-sync")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"抓取入库",
			"先开页面，再异步拉 summary、矩阵和最近写操作；raw 慢时也只影响局部。",
			"/_/mrtang-admin/target-sync",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/target-sync/run", func(re *core.RequestEvent) error {
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing target sync run id", nil)
		}
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?id="+url.QueryEscape(id))
	})

	se.Router.GET("/_/mrtang-admin/source", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"源数据",
			"先看 source 模块概览，再分流到商品、图片和日志；数据异步加载，不阻塞整页。",
			"/_/mrtang-admin/source",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/categories", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/categories")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"源数据分类",
			"分类树同步结果和已落库分类列表都在这里查看；页面先开壳，再异步加载。",
			"/_/mrtang-admin/source/categories",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/backend-release", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/backend-release")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"发布准备",
			"先看 Vendure 字段准备度、分类映射和商品 payload 预览，再决定何时正式同步。",
			"/_/mrtang-admin/backend-release",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/products", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/products")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"源数据商品",
			"商品审核、桥接、同步重试改成前端异步列表；现有动作端点继续复用。",
			"/_/mrtang-admin/source/products",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/products/detail", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/products/detail")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"商品详情",
			"详情页也切到前端异步渲染，动作端点继续复用现有 POST 路由。",
			"/_/mrtang-admin/source/products/detail",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/assets", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/assets")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"源数据图片",
			"图片状态、失败聚合和批量处理改成前端异步列表；现有动作端点继续复用。",
			"/_/mrtang-admin/source/assets",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/assets/detail", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/assets/detail")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"图片详情",
			"详情页也切到前端异步渲染，动作端点继续复用现有 POST 路由。",
			"/_/mrtang-admin/source/assets/detail",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/asset-jobs", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/asset-jobs")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"图片任务",
			"原图下载和图片处理的历史任务、重试入口和最近日志都在这里查看。",
			"/_/mrtang-admin/source/asset-jobs",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/asset-jobs/detail", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/asset-jobs/detail")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"图片任务详情",
			"任务详情页会异步加载任务进度、错误和最近日志。",
			"/_/mrtang-admin/source/asset-jobs/detail",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/product-jobs", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/product-jobs")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"商品发布任务",
			"商品发布与重试发布的历史任务、失败项和重跑入口都在这里查看。",
			"/_/mrtang-admin/source/product-jobs",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/product-jobs/detail", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/product-jobs/detail")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"商品发布任务详情",
			"任务详情页会异步加载商品发布任务的进度、失败项和最近日志。",
			"/_/mrtang-admin/source/product-jobs/detail",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/logs", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/source/logs")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"日志",
			"源数据操作日志统一走前端异步加载，和其他模块保持一致。",
			"/_/mrtang-admin/source/logs",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/source-review-workbench", func(re *core.RequestEvent) error {
		return redirectAdminAppShell(re, "/_/mrtang-admin/source/products")
	})

	se.Router.GET("/_/source-review-workbench/product", func(re *core.RequestEvent) error {
		return redirectAdminAppShell(re, "/_/mrtang-admin/source/products/detail")
	})

	se.Router.GET("/_/source-review-workbench/asset", func(re *core.RequestEvent) error {
		return redirectAdminAppShell(re, "/_/mrtang-admin/source/assets/detail")
	})

	se.Router.POST("/_/mrtang-admin/source/import", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}

		scope := strings.TrimSpace(re.Request.FormValue("scope"))
		dataset, err := miniappService.Dataset(re.Request.Context())
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?error=load+miniapp+dataset+failed")
		}

		summary, err := service.ImportMiniappSource(re.Request.Context(), re.App, *dataset, scope)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?error=import+source+capture+failed")
		}

		message := fmt.Sprintf(
			"imported %s: categories +%d/~%d, products +%d/~%d, assets +%d/~%d",
			summary.Scope,
			summary.CategoriesCreated,
			summary.CategoriesUpdated,
			summary.ProductsCreated,
			summary.ProductsUpdated,
			summary.AssetsCreated,
			summary.AssetsUpdated,
		)
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/target-sync/jobs/ensure", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}
		if _, err := service.EnsureTargetSyncJobSpec(re.App, cfg.MiniApp.SourceMode, strings.TrimSpace(re.Request.FormValue("entityType")), strings.TrimSpace(re.Request.FormValue("scopeKey")), strings.TrimSpace(re.Request.FormValue("scopeLabel"))); err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?error=ensure+target+sync+job+failed")
		}
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?message="+url.QueryEscape("已保存抓取入库任务"))
	})

	se.Router.POST("/_/mrtang-admin/target-sync/jobs/run", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有抓取入库权限。", nil)
		}
		run, err := service.StartTargetSyncAsync(
			re.App,
			func(ctx context.Context, entityType string, scopeKey string) (*miniappmodel.Dataset, error) {
				return miniappService.TargetSyncDataset(ctx, entityType, scopeKey)
			},
			func(ctx context.Context, sections []miniappmodel.CategorySection, scopeKey string) (*miniappmodel.Dataset, error) {
				return miniappService.TargetSyncProductsFromSections(ctx, sections, scopeKey)
			},
			strings.TrimSpace(re.Request.FormValue("entityType")),
			strings.TrimSpace(re.Request.FormValue("scopeKey")),
			strings.TrimSpace(re.Request.FormValue("scopeLabel")),
			targetSyncActor(re),
		)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?error="+url.QueryEscape("执行抓取入库失败: "+err.Error()))
		}
		message := fmt.Sprintf("已启动抓取入库任务：%s / %s", run.JobName, run.ScopeLabel)
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?message="+url.QueryEscape(message))
	})

	// Legacy compatibility matrix:
	// - /_/source-review-workbench GET page entries already redirect to SPA shells.
	// - /_/source-review-workbench POST actions remain only for old bookmarks/forms and should
	//   keep redirecting to new SPA destinations after completion.
	// - /_/procurement-workbench POST actions remain only for old bookmarks/forms, but must now
	//   reuse the same audited service methods as /_/mrtang-admin/procurement/* to avoid drift.
	se.Router.POST("/_/source-review-workbench/product/status", func(re *core.RequestEvent) error {
		return handleSourceStatusPost(re, legacySourceProductRedirectRoute, func(id string, status string, actor pim.SourceActionActor, note string) error {
			return service.UpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, id, status, note, actor)
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/status", func(re *core.RequestEvent) error {
		return handleSourceStatusPost(re, sourceProductsRouteRedirect, func(id string, status string, actor pim.SourceActionActor, note string) error {
			return service.UpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, id, status, note, actor)
		})
	})

	se.Router.POST("/_/source-review-workbench/products/batch-status", func(re *core.RequestEvent) error {
		return handleSourceBatchStatusPost(re, "productIds", legacySourceProductsRouteRedirect, func(ids []string, status string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchUpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, ids, status, note, actor)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("updated source products: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-status", func(re *core.RequestEvent) error {
		return handleSourceBatchStatusPost(re, "productIds", sourceProductsListRouteRedirect, func(ids []string, status string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchUpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, ids, status, note, actor)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("updated source products: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/product/promote", func(re *core.RequestEvent) error {
		return handleSourceSinglePost(re, legacySourceProductRedirectRoute, "promote source product failed", "promoted source product", func(id string, actor pim.SourceActionActor, note string) error {
			return service.PromoteSourceProductWithAudit(re.Request.Context(), re.App, id, actor, note)
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/promote", func(re *core.RequestEvent) error {
		return handleSourceSinglePost(re, sourceProductsRouteRedirect, "promote source product failed", "promoted source product", func(id string, actor pim.SourceActionActor, note string) error {
			return service.PromoteSourceProductWithAudit(re.Request.Context(), re.App, id, actor, note)
		})
	})

	se.Router.POST("/_/source-review-workbench/product/promote-sync", func(re *core.RequestEvent) error {
		return handleSourceSinglePost(re, legacySourceProductRedirectRoute, "promote and sync source product failed", "promoted and synced source product", func(id string, actor pim.SourceActionActor, note string) error {
			return service.PromoteAndSyncSourceProductWithAudit(re.Request.Context(), re.App, id, actor, note)
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/promote-sync", func(re *core.RequestEvent) error {
		return handleSourceSinglePost(re, sourceProductsRouteRedirect, "promote and sync source product failed", "promoted and synced source product", func(id string, actor pim.SourceActionActor, note string) error {
			return service.PromoteAndSyncSourceProductWithAudit(re.Request.Context(), re.App, id, actor, note)
		})
	})

	se.Router.POST("/_/source-review-workbench/product/retry-sync", func(re *core.RequestEvent) error {
		return handleSourceSinglePost(re, legacySourceProductRedirectRoute, "retry source product sync failed", "retried source product sync", func(id string, actor pim.SourceActionActor, note string) error {
			return service.RetrySourceProductSyncWithAudit(re.Request.Context(), re.App, id, actor, note)
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/retry-sync", func(re *core.RequestEvent) error {
		return handleSourceSinglePost(re, sourceProductsRouteRedirect, "retry source product sync failed", "retried source product sync", func(id string, actor pim.SourceActionActor, note string) error {
			return service.RetrySourceProductSyncWithAudit(re.Request.Context(), re.App, id, actor, note)
		})
	})

	se.Router.POST("/_/source-review-workbench/products/promote", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, requireAuthorizedPage, "promote approved source products failed", legacySourceProductsRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.PromoteApprovedSourceProducts(re.Request.Context(), re.App, 50)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("promoted approved source products: %d promoted, %d skipped, %d failed", summary.Promoted, summary.Skipped, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/products/batch-promote", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "productIds", "missing product ids", "batch promote source products failed", legacySourceProductsRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, false, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("promoted source products: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-promote", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "productIds", "missing product ids", "batch promote source products failed", sourceProductsListRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, false, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("promoted source products: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/products/batch-promote-sync", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "productIds", "missing product ids", "batch promote and sync source products failed", legacySourceProductsRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, true, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("promoted and synced source products: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-promote-sync", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "productIds", "missing product ids", "batch promote and sync source products failed", sourceProductsListRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, true, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("promoted and synced source products: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/products/batch-retry-sync", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "productIds", "missing product ids", "batch retry source sync failed", legacySourceProductsRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchRetrySourceProductSyncWithAudit(re.Request.Context(), re.App, ids, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("retried source product sync: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-retry-sync", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "productIds", "missing product ids", "batch retry source sync failed", sourceProductsListRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchRetrySourceProductSyncWithAudit(re.Request.Context(), re.App, ids, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("retried source product sync: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/assets/process", func(re *core.RequestEvent) error {
		return handleSourceAssetProcessPost(re, legacySourceAssetRedirectRoute, legacySourceAssetsListRouteRedirect, service)
	})

	se.Router.POST("/_/mrtang-admin/source/assets/process", func(re *core.RequestEvent) error {
		return handleSourceAssetProcessPost(re, sourceAssetsRouteRedirect, sourceAssetsListRouteRedirect, service)
	})

	se.Router.POST("/_/mrtang-admin/source/assets/batch-process", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "assetIds", "missing asset ids", "batch process source assets failed", sourceAssetsListRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchProcessSourceAssetsWithAudit(re.Request.Context(), re.App, ids, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/assets/reprocess-failed", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, requireAuthorizedPage, "reprocess failed source assets failed", sourceAssetsListRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.ProcessFailedSourceAssets(re.Request.Context(), re.App, 50)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("reprocessed failed source assets: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/assets/batch-process", func(re *core.RequestEvent) error {
		return handleSourceBatchIDsPost(re, requireAuthorizedPage, "assetIds", "missing asset ids", "batch process source assets failed", legacySourceAssetsListRouteRedirect, func(ids []string, actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.BatchProcessSourceAssetsWithAudit(re.Request.Context(), re.App, ids, actor, note)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/assets/reprocess-failed", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, requireAuthorizedPage, "reprocess failed source assets failed", legacySourceAssetsListRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.ProcessFailedSourceAssets(re.Request.Context(), re.App, 50)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("reprocessed failed source assets: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/source-review-workbench/supplier-products/sync", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, requireAuthorizedPage, "sync approved supplier products failed", legacySourceProductsRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			result, err := service.SyncApproved(re.Request.Context(), re.App, cfg.Workflow.SyncBatchSize)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("sync approved supplier products: processed %d, updated %d, failed %d", result.Processed, result.Updated, result.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/assets/process-pending", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, func(re *core.RequestEvent) error {
			return requireAuthorizedModule(re, cfg, "source", "当前账号没有源数据模块权限。")
		}, "process source assets failed", adminRootRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.ProcessPendingSourceAssets(re.Request.Context(), re.App, 20)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/source/products/promote-approved", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, func(re *core.RequestEvent) error {
			return requireAuthorizedModule(re, cfg, "source", "当前账号没有源数据模块权限。")
		}, "promote source products failed", adminRootRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			summary, err := service.PromoteApprovedSourceProducts(re.Request.Context(), re.App, 50)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("promoted approved source products: %d promoted, %d skipped, %d failed", summary.Promoted, summary.Skipped, summary.Failed), nil
		})
	})

	se.Router.POST("/_/mrtang-admin/supplier-products/sync", func(re *core.RequestEvent) error {
		return handleSourceListPost(re, func(re *core.RequestEvent) error {
			return requireAuthorizedModule(re, cfg, "source", "当前账号没有源数据模块权限。")
		}, "sync approved supplier products failed", adminRootRouteRedirect, func(actor pim.SourceActionActor, note string) (string, error) {
			result, err := service.SyncApproved(re.Request.Context(), re.App, cfg.Workflow.SyncBatchSize)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("sync approved supplier products: processed %d, updated %d, failed %d", result.Processed, result.Updated, result.Failed), nil
		})
	})

	se.Router.GET("/_/mrtang-admin/procurement", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/procurement")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"采购",
			"采购列表、风险筛选和最近动作改成前端异步加载；详情页先继续复用现有服务端版本。",
			"/_/mrtang-admin/procurement",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/mrtang-admin/procurement/detail", func(re *core.RequestEvent) error {
		if adminLegacyRedirect(re) {
			return redirectAdminAppShell(re, "/_/mrtang-admin/procurement/detail")
		}
		return re.HTML(http.StatusOK, admin.RenderAdminAppShellHTML(
			"采购详情",
			"详情页也切到前端异步渲染，风险商品和原始摘要不再阻塞整页。",
			"/_/mrtang-admin/procurement/detail",
			authorizedAdminModule(re, cfg, "source"),
			authorizedAdminModule(re, cfg, "procurement"),
		))
	})

	se.Router.GET("/_/procurement-workbench", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
	})

	se.Router.POST("/_/mrtang-admin/procurement/order/status", func(re *core.RequestEvent) error {
		return handleProcurementStatusPost(re, func(re *core.RequestEvent) error {
			return requireAuthorizedModule(re, cfg, "procurement", "当前账号没有采购模块权限。")
		}, service)
	})

	se.Router.POST("/_/procurement-workbench/order/status", func(re *core.RequestEvent) error {
		return handleProcurementStatusPost(re, requireAuthorizedPage, service)
	})

	se.Router.POST("/_/mrtang-admin/procurement/order/export", func(re *core.RequestEvent) error {
		return handleProcurementExportPost(re, func(re *core.RequestEvent) error {
			return requireAuthorizedModule(re, cfg, "procurement", "当前账号没有采购模块权限。")
		}, service)
	})

	se.Router.POST("/_/procurement-workbench/order/export", func(re *core.RequestEvent) error {
		return handleProcurementExportPost(re, requireAuthorizedPage, service)
	})

	se.Router.POST("/_/mrtang-admin/procurement/order/review", func(re *core.RequestEvent) error {
		return handleProcurementReviewPost(re, func(re *core.RequestEvent) error {
			return requireAuthorizedModule(re, cfg, "procurement", "当前账号没有采购模块权限。")
		}, service)
	})

	se.Router.POST("/_/procurement-workbench/order/review", func(re *core.RequestEvent) error {
		return handleProcurementReviewPost(re, requireAuthorizedPage, service)
	})
}

func serveAdminAsset(re *core.RequestEvent, name string, contentType string) error {
	body, err := fs.ReadFile(adminapp.Static, name)
	if err != nil {
		return re.NotFoundError("File not found.", err)
	}
	re.Response.Header().Set("Content-Type", contentType)
	re.Response.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	re.Response.Header().Set("Pragma", "no-cache")
	re.Response.Header().Set("Expires", "0")
	return re.String(http.StatusOK, string(body))
}

func registerAdminSlashRedirect(se *core.ServeEvent, from string, to string) {
	se.Router.GET(from, func(re *core.RequestEvent) error {
		target := to
		if raw := strings.TrimSpace(re.Request.URL.RawQuery); raw != "" {
			target += "?" + raw
		}
		return re.Redirect(http.StatusSeeOther, target)
	})
}

func readSourceReviewFilter(re *core.RequestEvent) pim.SourceReviewFilter {
	filter := pim.SourceReviewFilter{
		CategoryKey:    strings.TrimSpace(re.Request.URL.Query().Get("categoryKey")),
		CategoryKeys:   strings.TrimSpace(re.Request.URL.Query().Get("categoryKeys")),
		ProductStatus:  strings.TrimSpace(re.Request.URL.Query().Get("productStatus")),
		AssetStatus:    strings.TrimSpace(re.Request.URL.Query().Get("assetStatus")),
		OriginalStatus: strings.TrimSpace(re.Request.URL.Query().Get("originalStatus")),
		ProductIDs:     strings.TrimSpace(re.Request.URL.Query().Get("productIds")),
		AssetIDs:       strings.TrimSpace(re.Request.URL.Query().Get("assetIds")),
		SyncState:      strings.TrimSpace(re.Request.URL.Query().Get("syncState")),
		Query:          strings.TrimSpace(re.Request.URL.Query().Get("q")),
	}
	if page, err := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("productPage"))); err == nil {
		filter.ProductPage = page
	}
	if page, err := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("assetPage"))); err == nil {
		filter.AssetPage = page
	}
	if pageSize, err := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("pageSize"))); err == nil {
		filter.PageSize = pageSize
	}
	return filter
}

func readSourceAssetJobFilter(re *core.RequestEvent) pim.SourceAssetJobFilter {
	return pim.SourceAssetJobFilter{
		JobType:  strings.TrimSpace(re.Request.URL.Query().Get("jobType")),
		Status:   strings.TrimSpace(re.Request.URL.Query().Get("status")),
		Query:    strings.TrimSpace(re.Request.URL.Query().Get("q")),
		Page:     readQueryInt(re, "page", 1),
		PageSize: readQueryInt(re, "pageSize", 20),
	}
}

func readSourceProductJobFilter(re *core.RequestEvent) pim.SourceProductJobFilter {
	return pim.SourceProductJobFilter{
		JobType:  strings.TrimSpace(re.Request.URL.Query().Get("jobType")),
		Status:   strings.TrimSpace(re.Request.URL.Query().Get("status")),
		Query:    strings.TrimSpace(re.Request.URL.Query().Get("q")),
		Page:     readQueryInt(re, "page", 1),
		PageSize: readQueryInt(re, "pageSize", 20),
	}
}

func sourceActionActor(re *core.RequestEvent) pim.SourceActionActor {
	if re.Auth == nil {
		return pim.SourceActionActor{}
	}
	actor := pim.SourceActionActor{
		Email: strings.TrimSpace(re.Auth.GetString("email")),
		Name:  strings.TrimSpace(re.Auth.GetString("name")),
	}
	if actor.Name == "" {
		actor.Name = actor.Email
	}
	return actor
}

func sourceActionNote(re *core.RequestEvent) string {
	return strings.TrimSpace(re.Request.FormValue("note"))
}

func procurementActionActor(re *core.RequestEvent) pim.ProcurementActionActor {
	sourceActor := sourceActionActor(re)
	return pim.ProcurementActionActor{
		Email: sourceActor.Email,
		Name:  sourceActor.Name,
	}
}

func targetSyncActor(re *core.RequestEvent) pim.TargetSyncActor {
	sourceActor := sourceActionActor(re)
	return pim.TargetSyncActor{
		Email: sourceActor.Email,
		Name:  sourceActor.Name,
	}
}

type adminRouteGuard func(*core.RequestEvent) error
type sourceSingleRedirect func(re *core.RequestEvent, id string, message string, isError bool) string
type sourceListRedirect func(re *core.RequestEvent, message string, isError bool) string

func requireAuthorizedPage(re *core.RequestEvent) error {
	if admin.AuthorizedPage(re) {
		return nil
	}
	return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
}

func requireAuthorizedModule(re *core.RequestEvent, cfg config.Config, module string, message string) error {
	if authorizedAdminModule(re, cfg, module) {
		return nil
	}
	return re.ForbiddenError(message, nil)
}

func handleProcurementStatusPost(re *core.RequestEvent, guard adminRouteGuard, service *pim.Service) error {
	if err := guard(re); err != nil {
		return err
	}

	id := strings.TrimSpace(re.Request.FormValue("id"))
	status := strings.TrimSpace(re.Request.FormValue("status"))
	note := strings.TrimSpace(re.Request.FormValue("note"))
	if id == "" || status == "" {
		return re.BadRequestError("missing procurement order id or status", nil)
	}

	if _, err := service.UpdateProcurementOrderStatusWithAudit(re.Request.Context(), re.App, id, status, note, procurementActionActor(re)); err != nil {
		return re.BadRequestError("update procurement order status failed", err)
	}

	return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
}

func handleProcurementExportPost(re *core.RequestEvent, guard adminRouteGuard, service *pim.Service) error {
	if err := guard(re); err != nil {
		return err
	}

	id := strings.TrimSpace(re.Request.FormValue("id"))
	if id == "" {
		return re.BadRequestError("missing procurement order id", nil)
	}

	if _, err := service.ExportProcurementOrderWithAudit(re.Request.Context(), re.App, id, procurementActionActor(re), strings.TrimSpace(re.Request.FormValue("note"))); err != nil {
		return re.BadRequestError("export procurement order failed", err)
	}

	return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
}

func handleProcurementReviewPost(re *core.RequestEvent, guard adminRouteGuard, service *pim.Service) error {
	if err := guard(re); err != nil {
		return err
	}

	id := strings.TrimSpace(re.Request.FormValue("id"))
	note := strings.TrimSpace(re.Request.FormValue("note"))
	if id == "" {
		return re.BadRequestError("missing procurement order id", nil)
	}

	if _, err := service.ReviewProcurementOrderWithAudit(re.Request.Context(), re.App, id, note, procurementActionActor(re)); err != nil {
		return re.BadRequestError("review procurement order failed", err)
	}

	return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
}

func handleSourceSinglePost(re *core.RequestEvent, redirect sourceSingleRedirect, errorMessage string, successMessage string, action func(id string, actor pim.SourceActionActor, note string) error) error {
	if err := requireAuthorizedPage(re); err != nil {
		return err
	}

	id := strings.TrimSpace(re.Request.FormValue("id"))
	if id == "" {
		return re.Redirect(http.StatusSeeOther, redirect(re, id, "missing product id", true))
	}

	if err := action(id, sourceActionActor(re), sourceActionNote(re)); err != nil {
		return re.Redirect(http.StatusSeeOther, redirect(re, id, errorMessage, true))
	}

	return re.Redirect(http.StatusSeeOther, redirect(re, id, successMessage, false))
}

func handleSourceStatusPost(re *core.RequestEvent, redirect sourceSingleRedirect, action func(id string, status string, actor pim.SourceActionActor, note string) error) error {
	if err := requireAuthorizedPage(re); err != nil {
		return err
	}

	id := strings.TrimSpace(re.Request.FormValue("id"))
	status := strings.TrimSpace(re.Request.FormValue("status"))
	if id == "" || status == "" {
		return re.Redirect(http.StatusSeeOther, redirect(re, id, "missing product id or status", true))
	}

	if err := action(id, status, sourceActionActor(re), sourceActionNote(re)); err != nil {
		return re.Redirect(http.StatusSeeOther, redirect(re, id, "update source product status failed", true))
	}

	return re.Redirect(http.StatusSeeOther, redirect(re, id, "updated source product review status", false))
}

func handleSourceBatchStatusPost(re *core.RequestEvent, idsField string, redirect sourceListRedirect, action func(ids []string, status string, actor pim.SourceActionActor, note string) (string, error)) error {
	if err := requireAuthorizedPage(re); err != nil {
		return err
	}

	status := strings.TrimSpace(re.Request.FormValue("status"))
	ids := re.Request.Form[idsField]
	if status == "" || len(ids) == 0 {
		return re.Redirect(http.StatusSeeOther, redirect(re, "missing product ids or status", true))
	}

	message, err := action(ids, status, sourceActionActor(re), sourceActionNote(re))
	if err != nil {
		return re.Redirect(http.StatusSeeOther, redirect(re, "batch update source product status failed", true))
	}

	return re.Redirect(http.StatusSeeOther, redirect(re, message, false))
}

func handleSourceBatchIDsPost(re *core.RequestEvent, guard adminRouteGuard, idsField string, missingMessage string, errorMessage string, redirect sourceListRedirect, action func(ids []string, actor pim.SourceActionActor, note string) (string, error)) error {
	if err := guard(re); err != nil {
		return err
	}

	ids := re.Request.Form[idsField]
	if len(ids) == 0 {
		return re.Redirect(http.StatusSeeOther, redirect(re, missingMessage, true))
	}

	message, err := action(ids, sourceActionActor(re), sourceActionNote(re))
	if err != nil {
		return re.Redirect(http.StatusSeeOther, redirect(re, errorMessage, true))
	}

	return re.Redirect(http.StatusSeeOther, redirect(re, message, false))
}

func handleSourceListPost(re *core.RequestEvent, guard adminRouteGuard, errorMessage string, redirect sourceListRedirect, action func(actor pim.SourceActionActor, note string) (string, error)) error {
	if err := guard(re); err != nil {
		return err
	}

	message, err := action(sourceActionActor(re), sourceActionNote(re))
	if err != nil {
		return re.Redirect(http.StatusSeeOther, redirect(re, errorMessage, true))
	}

	return re.Redirect(http.StatusSeeOther, redirect(re, message, false))
}

func handleSourceAssetProcessPost(re *core.RequestEvent, singleRedirect sourceSingleRedirect, listRedirect sourceListRedirect, service *pim.Service) error {
	if err := requireAuthorizedPage(re); err != nil {
		return err
	}

	assetID := strings.TrimSpace(re.Request.FormValue("id"))
	if assetID != "" {
		if err := service.ProcessSourceAssetWithAudit(re.Request.Context(), re.App, assetID, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, singleRedirect(re, assetID, "process source asset failed", true))
		}
		return re.Redirect(http.StatusSeeOther, singleRedirect(re, assetID, "processed single source asset", false))
	}

	summary, err := service.ProcessPendingSourceAssets(re.Request.Context(), re.App, 20)
	if err != nil {
		return re.Redirect(http.StatusSeeOther, listRedirect(re, "process source assets failed", true))
	}
	message := fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed)
	return re.Redirect(http.StatusSeeOther, listRedirect(re, message, false))
}

func sourceProductsRouteRedirect(re *core.RequestEvent, id string, message string, isError bool) string {
	return sourceProductsRedirect(re, message, isError)
}

func legacySourceProductRedirectRoute(re *core.RequestEvent, id string, message string, isError bool) string {
	return legacySourceProductRedirect(id, message, isError)
}

func sourceProductsListRouteRedirect(re *core.RequestEvent, message string, isError bool) string {
	return sourceProductsRedirect(re, message, isError)
}

func legacySourceProductsRouteRedirect(re *core.RequestEvent, message string, isError bool) string {
	return legacySourceProductsRedirect(message, isError)
}

func sourceAssetsRouteRedirect(re *core.RequestEvent, id string, message string, isError bool) string {
	return sourceAssetsRedirect(re, message, isError)
}

func legacySourceAssetRedirectRoute(re *core.RequestEvent, id string, message string, isError bool) string {
	return legacySourceAssetRedirect(id, message, isError)
}

func sourceAssetsListRouteRedirect(re *core.RequestEvent, message string, isError bool) string {
	return sourceAssetsRedirect(re, message, isError)
}

func legacySourceAssetsListRouteRedirect(re *core.RequestEvent, message string, isError bool) string {
	return legacySourceAssetsRedirect(message, isError)
}

func adminRootRouteRedirect(re *core.RequestEvent, message string, isError bool) string {
	return adminMessageRedirect("/_/mrtang-admin", message, isError)
}

func authorizedAdminModule(re *core.RequestEvent, cfg config.Config, module string) bool {
	if !admin.AuthorizedPage(re) {
		return false
	}
	if re.Auth == nil {
		return true
	}
	if re.Auth.IsSuperuser() {
		return true
	}
	email := strings.ToLower(strings.TrimSpace(re.Auth.GetString("email")))
	if email == "" {
		return true
	}

	var allowed []string
	switch strings.ToLower(strings.TrimSpace(module)) {
	case "source":
		allowed = cfg.Admin.SourceAdmins
	case "procurement":
		allowed = cfg.Admin.ProcurementAdmins
	case "dashboard":
		// 总览页作为登录入口，不再强制邮箱白名单。
		// 具体模块权限（source/procurement）仍由对应白名单控制。
		return true
	default:
		return true
	}
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if strings.EqualFold(strings.TrimSpace(item), email) {
			return true
		}
	}
	return false
}

func adminLegacyRedirect(re *core.RequestEvent) bool {
	return strings.TrimSpace(re.Request.URL.Query().Get("legacy")) == "1"
}

func redirectAdminAppShell(re *core.RequestEvent, path string) error {
	query := re.Request.URL.Query()
	query.Del("legacy")
	target := strings.TrimSpace(path)
	if target == "" {
		target = "/_/mrtang-admin"
	}
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	return re.Redirect(http.StatusSeeOther, target)
}

func sourceProductsRedirect(re *core.RequestEvent, message string, isError bool) string {
	target := strings.TrimSpace(re.Request.FormValue("returnTo"))
	if !strings.HasPrefix(target, "/_/mrtang-admin/source/products") {
		target = "/_/mrtang-admin/source/products"
	}
	if message == "" {
		return target
	}
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	key := "message"
	if isError {
		key = "error"
	}
	return target + sep + key + "=" + url.QueryEscape(message)
}

func legacySourceProductsRedirect(message string, isError bool) string {
	return adminMessageRedirect("/_/mrtang-admin/source/products", message, isError)
}

func legacySourceProductRedirect(id string, message string, isError bool) string {
	target := "/_/mrtang-admin/source/products"
	if strings.TrimSpace(id) != "" {
		target = "/_/mrtang-admin/source/products/detail?id=" + url.QueryEscape(strings.TrimSpace(id))
	}
	return adminMessageRedirect(target, message, isError)
}

func sourceAssetsRedirect(re *core.RequestEvent, message string, isError bool) string {
	target := strings.TrimSpace(re.Request.FormValue("returnTo"))
	if !strings.HasPrefix(target, "/_/mrtang-admin/source/assets") {
		target = "/_/mrtang-admin/source/assets"
	}
	if message == "" {
		return target
	}
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	key := "message"
	if isError {
		key = "error"
	}
	return target + sep + key + "=" + url.QueryEscape(message)
}

func legacySourceAssetsRedirect(message string, isError bool) string {
	return adminMessageRedirect("/_/mrtang-admin/source/assets", message, isError)
}

func legacySourceAssetRedirect(id string, message string, isError bool) string {
	target := "/_/mrtang-admin/source/assets"
	if strings.TrimSpace(id) != "" {
		target = "/_/mrtang-admin/source/assets/detail?id=" + url.QueryEscape(strings.TrimSpace(id))
	}
	return adminMessageRedirect(target, message, isError)
}

func adminMessageRedirect(target string, message string, isError bool) string {
	target = strings.TrimSpace(target)
	if target == "" {
		target = "/_/mrtang-admin"
	}
	if message == "" {
		return target
	}
	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}
	key := "message"
	if isError {
		key = "error"
	}
	return target + sep + key + "=" + url.QueryEscape(message)
}

func buildSourceLogsPageData(re *core.RequestEvent) (admin.SourceLogsPageData, error) {
	filter := admin.SourceLogFilter{
		ActionType: strings.TrimSpace(re.Request.URL.Query().Get("actionType")),
		Status:     strings.TrimSpace(re.Request.URL.Query().Get("status")),
		TargetType: strings.TrimSpace(re.Request.URL.Query().Get("targetType")),
		Actor:      strings.TrimSpace(re.Request.URL.Query().Get("actor")),
		Query:      strings.TrimSpace(re.Request.URL.Query().Get("q")),
	}
	if page, err := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("page"))); err == nil {
		filter.Page = page
	}
	if pageSize, err := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("pageSize"))); err == nil {
		filter.PageSize = pageSize
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	records, err := re.App.FindAllRecords(pim.CollectionSourceActionLogs)
	if err != nil {
		return admin.SourceLogsPageData{}, err
	}

	items := make([]pim.SourceActionLog, 0, len(records))
	successCount := 0
	failedCount := 0
	query := strings.ToLower(filter.Query)
	for _, record := range records {
		item := pim.SourceActionLog{
			ID:          record.Id,
			TargetType:  record.GetString("target_type"),
			TargetID:    record.GetString("target_id"),
			TargetLabel: record.GetString("target_label"),
			ActionType:  record.GetString("action_type"),
			Status:      record.GetString("status"),
			Message:     record.GetString("message"),
			ActorEmail:  record.GetString("actor_email"),
			ActorName:   record.GetString("actor_name"),
			Note:        record.GetString("note"),
			Created:     record.GetString("created"),
		}
		if filter.ActionType != "" && !strings.EqualFold(item.ActionType, filter.ActionType) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(item.Status, filter.Status) {
			continue
		}
		if filter.TargetType != "" && !strings.EqualFold(item.TargetType, filter.TargetType) {
			continue
		}
		if filter.Actor != "" {
			actorSearch := strings.ToLower(strings.Join([]string{item.ActorName, item.ActorEmail}, " "))
			if !strings.Contains(actorSearch, strings.ToLower(filter.Actor)) {
				continue
			}
		}
		if query != "" {
			search := strings.ToLower(strings.Join([]string{item.TargetLabel, item.TargetID, item.Message, item.ActionType, item.ActorName, item.ActorEmail, item.Note}, " "))
			if !strings.Contains(search, query) {
				continue
			}
		}
		if strings.EqualFold(item.Status, "success") {
			successCount++
		}
		if strings.EqualFold(item.Status, "failed") {
			failedCount++
		}
		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b pim.SourceActionLog) int {
		return strings.Compare(b.Created, a.Created)
	})

	total := len(items)
	pages := sourceLogTotalPages(total, filter.PageSize)
	start, end := sourceLogPaginateBounds(total, filter.Page, filter.PageSize)
	pagedItems := items[start:end]

	return admin.SourceLogsPageData{
		Items:        pagedItems,
		Filter:       filter,
		Total:        total,
		Page:         filter.Page,
		Pages:        pages,
		PageSize:     filter.PageSize,
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

func sourceLogTotalPages(total int, pageSize int) int {
	if total == 0 {
		return 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	if pages <= 0 {
		pages = 1
	}
	return pages
}

func sourceLogPaginateBounds(total int, page int, pageSize int) (int, int) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func readQueryInt(re *core.RequestEvent, key string, fallback int) int {
	value := strings.TrimSpace(re.Request.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func readIDList(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	ids := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		ids = append(ids, item)
	}
	return ids
}
