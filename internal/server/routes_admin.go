package server

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
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

	registerAdminSlashRedirect(se, "/_/mrtang-admin/", "/_/mrtang-admin")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/audit/", "/_/mrtang-admin/audit")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/target-sync/", "/_/mrtang-admin/target-sync")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/", "/_/mrtang-admin/source")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/products/", "/_/mrtang-admin/source/products")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/assets/", "/_/mrtang-admin/source/assets")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/source/logs/", "/_/mrtang-admin/source/logs")
	registerAdminSlashRedirect(se, "/_/mrtang-admin/procurement/", "/_/mrtang-admin/procurement")

	se.Router.GET("/_/mrtang-admin", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "dashboard") {
			return re.ForbiddenError("当前账号没有后台总览权限。", nil)
		}

		return re.HTML(http.StatusOK, admin.RenderMrtangAdminHTML(
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

	se.Router.GET("/_/mrtang-admin/audit", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "dashboard") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无后台总览权限", "当前账号没有后台总览权限。", "/_/"))
		}
		return re.HTML(http.StatusOK, admin.RenderAuditHTML(
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

	se.Router.GET("/_/mrtang-admin/target-sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无目标站同步权限", "当前账号没有目标站同步权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}
		dataset, err := miniappService.Dataset(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load target sync dataset failed", err)
		}
		summary, err := service.TargetSyncSummary(re.Request.Context(), re.App, *dataset)
		if err != nil {
			return re.InternalServerError("load target sync summary failed", err)
		}
		return re.HTML(http.StatusOK, admin.RenderTargetSyncHTML(
			summary,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/_/mrtang-admin/target-sync/run", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无目标站同步权限", "当前账号没有目标站同步权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing target sync run id", nil)
		}
		run, err := service.GetTargetSyncRun(re.App, id)
		if err != nil {
			return re.InternalServerError("load target sync run failed", err)
		}
		return re.HTML(http.StatusOK, admin.RenderTargetSyncRunDetailHTML(run, "/_/mrtang-admin/target-sync"))
	})

	se.Router.GET("/_/mrtang-admin/source", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无源数据模块权限", "当前账号没有源数据模块权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}

		summary, err := service.SourceReviewWorkbench(re.Request.Context(), re.App, 6, 6, pim.SourceReviewFilter{PageSize: 6})
		if err != nil {
			return re.InternalServerError("load source module failed", err)
		}

		return re.HTML(http.StatusOK, admin.RenderSourceModuleHTML(
			summary,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/products", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无源数据模块权限", "当前账号没有源数据模块权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}

		filter := readSourceReviewFilter(re)
		filter.AssetStatus = ""
		filter.AssetPage = 1
		summary, err := service.SourceReviewWorkbench(re.Request.Context(), re.App, 24, 1, filter)
		if err != nil {
			return re.InternalServerError("load source products failed", err)
		}

		return re.HTML(http.StatusOK, admin.RenderSourceProductsHTML(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/products/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无源数据模块权限", "当前账号没有源数据模块权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source product id", nil)
		}
		detail, err := service.SourceProductDetail(re.Request.Context(), re.App, id)
		if err != nil {
			return re.InternalServerError("load source product detail failed", err)
		}
		returnTo := strings.TrimSpace(re.Request.URL.Query().Get("returnTo"))
		if returnTo == "" {
			returnTo = "/_/mrtang-admin/source/products"
		}
		return re.HTML(http.StatusOK, admin.RenderSourceProductDetailPageHTML(
			detail,
			returnTo,
			"/_/mrtang-admin/source/products",
			returnTo,
		))
	})

	se.Router.GET("/_/mrtang-admin/source/assets", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无源数据模块权限", "当前账号没有源数据模块权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}

		filter := readSourceReviewFilter(re)
		filter.ProductStatus = ""
		filter.SyncState = ""
		filter.ProductPage = 1
		summary, err := service.SourceReviewWorkbench(re.Request.Context(), re.App, 1, 24, filter)
		if err != nil {
			return re.InternalServerError("load source assets failed", err)
		}

		return re.HTML(http.StatusOK, admin.RenderSourceAssetsHTML(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/_/mrtang-admin/source/assets/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无源数据模块权限", "当前账号没有源数据模块权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source asset id", nil)
		}
		detail, err := service.SourceAssetDetail(re.Request.Context(), re.App, id)
		if err != nil {
			return re.InternalServerError("load source asset detail failed", err)
		}
		returnTo := strings.TrimSpace(re.Request.URL.Query().Get("returnTo"))
		if returnTo == "" {
			returnTo = "/_/mrtang-admin/source/assets"
		}
		return re.HTML(http.StatusOK, admin.RenderSourceAssetDetailPageHTML(
			detail,
			returnTo,
			"/_/mrtang-admin/source/assets",
			returnTo,
		))
	})

	se.Router.GET("/_/mrtang-admin/source/logs", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无源数据模块权限", "当前账号没有源数据模块权限，请联系管理员配置 `PIM_SOURCE_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}

		data, err := buildSourceLogsPageData(re)
		if err != nil {
			return re.InternalServerError("load source logs failed", err)
		}
		return re.HTML(http.StatusOK, admin.RenderSourceLogsHTML(
			data,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/_/source-review-workbench", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		filter := readSourceReviewFilter(re)
		summary, err := service.SourceReviewWorkbench(re.Request.Context(), re.App, 24, 24, filter)
		if err != nil {
			return re.InternalServerError("load source review workbench failed", err)
		}

		return re.HTML(http.StatusOK, admin.RenderSourceReviewWorkbenchHTML(
			summary,
			filter,
			strings.TrimSpace(re.Request.URL.Query().Get("message")),
			strings.TrimSpace(re.Request.URL.Query().Get("error")),
		))
	})

	se.Router.GET("/_/source-review-workbench/product", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source product id", nil)
		}
		detail, err := service.SourceProductDetail(re.Request.Context(), re.App, id)
		if err != nil {
			return re.InternalServerError("load source product detail failed", err)
		}
		return re.HTML(http.StatusOK, admin.RenderSourceProductDetailHTML(detail))
	})

	se.Router.GET("/_/source-review-workbench/asset", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing source asset id", nil)
		}
		detail, err := service.SourceAssetDetail(re.Request.Context(), re.App, id)
		if err != nil {
			return re.InternalServerError("load source asset detail failed", err)
		}
		return re.HTML(http.StatusOK, admin.RenderSourceAssetDetailHTML(detail))
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
			return re.ForbiddenError("当前账号没有目标站同步权限。", nil)
		}
		dataset, err := miniappService.Dataset(re.Request.Context())
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?error=load+dataset+failed")
		}
		if _, err := service.EnsureTargetSyncJob(re.Request.Context(), re.App, *dataset, strings.TrimSpace(re.Request.FormValue("entityType")), strings.TrimSpace(re.Request.FormValue("scopeKey"))); err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?error=ensure+target+sync+job+failed")
		}
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?message=target+sync+job+saved")
	})

	se.Router.POST("/_/mrtang-admin/target-sync/jobs/run", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有目标站同步权限。", nil)
		}
		dataset, err := miniappService.Dataset(re.Request.Context())
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?error=load+dataset+failed")
		}
		entityType := strings.TrimSpace(re.Request.FormValue("entityType"))
		run, err := service.RunTargetSync(re.Request.Context(), re.App, *dataset, entityType, strings.TrimSpace(re.Request.FormValue("scopeKey")), targetSyncActor(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?error="+url.QueryEscape("run target sync failed: "+err.Error()))
		}
		entityLabel := "分类树"
		switch strings.TrimSpace(run.EntityType) {
		case pim.TargetSyncEntityProducts:
			entityLabel = "商品规格"
		case pim.TargetSyncEntityAssets:
			entityLabel = "图片资产"
		}
		message := fmt.Sprintf("%s同步完成: %s, 新增 %d, 更新 %d, 未变 %d", entityLabel, run.ScopeLabel, run.CreatedCount, run.UpdatedCount, run.UnchangedCount)
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/target-sync?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/source-review-workbench/product/status", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		status := strings.TrimSpace(re.Request.FormValue("status"))
		if id == "" || status == "" {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product id or status"))
		}
		if err := service.UpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, id, status, sourceActionNote(re), sourceActionActor(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("update source product status failed"))
		}
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape("updated source product review status"))
	})

	se.Router.POST("/_/mrtang-admin/source/products/status", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		status := strings.TrimSpace(re.Request.FormValue("status"))
		if id == "" || status == "" {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product id or status", true))
		}
		if err := service.UpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, id, status, sourceActionNote(re), sourceActionActor(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "update source product status failed", true))
		}
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "updated source product review status", false))
	})

	se.Router.POST("/_/source-review-workbench/products/batch-status", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		status := strings.TrimSpace(re.Request.FormValue("status"))
		ids := re.Request.Form["productIds"]
		if status == "" || len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product ids or status"))
		}
		summary, err := service.BatchUpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, ids, status, sourceActionNote(re), sourceActionActor(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("batch update source product status failed"))
		}
		message := fmt.Sprintf("updated source products: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-status", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		status := strings.TrimSpace(re.Request.FormValue("status"))
		ids := re.Request.Form["productIds"]
		if status == "" || len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product ids or status", true))
		}
		summary, err := service.BatchUpdateSourceProductReviewStatusWithAudit(re.Request.Context(), re.App, ids, status, sourceActionNote(re), sourceActionActor(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "batch update source product status failed", true))
		}
		message := fmt.Sprintf("updated source products: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, message, false))
	})

	se.Router.POST("/_/source-review-workbench/product/promote", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product id"))
		}
		if err := service.PromoteSourceProductWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("promote source product failed"))
		}
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape("promoted source product"))
	})

	se.Router.POST("/_/mrtang-admin/source/products/promote", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product id", true))
		}
		if err := service.PromoteSourceProductWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "promote source product failed", true))
		}
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "promoted source product", false))
	})

	se.Router.POST("/_/source-review-workbench/product/promote-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product id"))
		}
		if err := service.PromoteAndSyncSourceProductWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("promote and sync source product failed"))
		}
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape("promoted and synced source product"))
	})

	se.Router.POST("/_/mrtang-admin/source/products/promote-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product id", true))
		}
		if err := service.PromoteAndSyncSourceProductWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "promote and sync source product failed", true))
		}
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "promoted and synced source product", false))
	})

	se.Router.POST("/_/source-review-workbench/product/retry-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product id"))
		}
		if err := service.RetrySourceProductSyncWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("retry source product sync failed"))
		}
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape("retried source product sync"))
	})

	se.Router.POST("/_/mrtang-admin/source/products/retry-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product id", true))
		}
		if err := service.RetrySourceProductSyncWithAudit(re.Request.Context(), re.App, id, sourceActionActor(re), sourceActionNote(re)); err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "retry source product sync failed", true))
		}
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "retried source product sync", false))
	})

	se.Router.POST("/_/source-review-workbench/products/promote", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		summary, err := service.PromoteApprovedSourceProducts(re.Request.Context(), re.App, 50)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("promote approved source products failed"))
		}
		message := fmt.Sprintf("promoted approved source products: %d promoted, %d skipped, %d failed", summary.Promoted, summary.Skipped, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/source-review-workbench/products/batch-promote", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["productIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product ids"))
		}
		summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, false, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("batch promote source products failed"))
		}
		message := fmt.Sprintf("promoted source products: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-promote", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["productIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product ids", true))
		}
		summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, false, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "batch promote source products failed", true))
		}
		message := fmt.Sprintf("promoted source products: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, message, false))
	})

	se.Router.POST("/_/source-review-workbench/products/batch-promote-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["productIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product ids"))
		}
		summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, true, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("batch promote and sync source products failed"))
		}
		message := fmt.Sprintf("promoted and synced source products: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-promote-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["productIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product ids", true))
		}
		summary, err := service.BatchPromoteSourceProductsWithAudit(re.Request.Context(), re.App, ids, true, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "batch promote and sync source products failed", true))
		}
		message := fmt.Sprintf("promoted and synced source products: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, message, false))
	})

	se.Router.POST("/_/source-review-workbench/products/batch-retry-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["productIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing product ids"))
		}
		summary, err := service.BatchRetrySourceProductSyncWithAudit(re.Request.Context(), re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("batch retry source sync failed"))
		}
		message := fmt.Sprintf("retried source product sync: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/products/batch-retry-sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["productIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "missing product ids", true))
		}
		summary, err := service.BatchRetrySourceProductSyncWithAudit(re.Request.Context(), re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, "batch retry source sync failed", true))
		}
		message := fmt.Sprintf("retried source product sync: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceProductsRedirect(re, message, false))
	})

	se.Router.POST("/_/source-review-workbench/assets/process", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		assetID := strings.TrimSpace(re.Request.FormValue("id"))
		if assetID != "" {
			if err := service.ProcessSourceAssetWithAudit(re.Request.Context(), re.App, assetID, sourceActionActor(re), sourceActionNote(re)); err != nil {
				return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("process source asset failed"))
			}
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape("processed single source asset"))
		}

		summary, err := service.ProcessPendingSourceAssets(re.Request.Context(), re.App, 20)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("process source assets failed"))
		}
		message := fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/assets/process", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		assetID := strings.TrimSpace(re.Request.FormValue("id"))
		if assetID != "" {
			if err := service.ProcessSourceAssetWithAudit(re.Request.Context(), re.App, assetID, sourceActionActor(re), sourceActionNote(re)); err != nil {
				return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, "process source asset failed", true))
			}
			return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, "processed single source asset", false))
		}

		summary, err := service.ProcessPendingSourceAssets(re.Request.Context(), re.App, 20)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, "process source assets failed", true))
		}
		message := fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, message, false))
	})

	se.Router.POST("/_/mrtang-admin/source/assets/batch-process", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["assetIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, "missing asset ids", true))
		}
		summary, err := service.BatchProcessSourceAssetsWithAudit(re.Request.Context(), re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, "batch process source assets failed", true))
		}
		message := fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, message, false))
	})

	se.Router.POST("/_/mrtang-admin/source/assets/reprocess-failed", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		summary, err := service.ProcessFailedSourceAssets(re.Request.Context(), re.App, 50)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, "reprocess failed source assets failed", true))
		}
		message := fmt.Sprintf("reprocessed failed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, sourceAssetsRedirect(re, message, false))
	})

	se.Router.POST("/_/source-review-workbench/assets/batch-process", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		ids := re.Request.Form["assetIds"]
		if len(ids) == 0 {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("missing asset ids"))
		}
		summary, err := service.BatchProcessSourceAssetsWithAudit(re.Request.Context(), re.App, ids, sourceActionActor(re), sourceActionNote(re))
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("batch process source assets failed"))
		}
		message := fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/source-review-workbench/assets/reprocess-failed", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		summary, err := service.ProcessFailedSourceAssets(re.Request.Context(), re.App, 50)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("reprocess failed source assets failed"))
		}
		message := fmt.Sprintf("reprocessed failed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/source-review-workbench/supplier-products/sync", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}

		result, err := service.SyncApproved(re.Request.Context(), re.App, cfg.Workflow.SyncBatchSize)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?error="+url.QueryEscape("sync approved supplier products failed"))
		}
		message := fmt.Sprintf("sync approved supplier products: processed %d, updated %d, failed %d", result.Processed, result.Updated, result.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/source-review-workbench?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/assets/process-pending", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}

		summary, err := service.ProcessPendingSourceAssets(re.Request.Context(), re.App, 20)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?error="+url.QueryEscape("process source assets failed"))
		}
		message := fmt.Sprintf("processed source assets: %d success, %d failed", summary.Processed, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/source/products/promote-approved", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}

		summary, err := service.PromoteApprovedSourceProducts(re.Request.Context(), re.App, 50)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?error="+url.QueryEscape("promote source products failed"))
		}

		message := fmt.Sprintf("promoted approved source products: %d promoted, %d skipped, %d failed", summary.Promoted, summary.Skipped, summary.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?message="+url.QueryEscape(message))
	})

	se.Router.POST("/_/mrtang-admin/supplier-products/sync", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "source") {
			return re.ForbiddenError("当前账号没有源数据模块权限。", nil)
		}

		result, err := service.SyncApproved(re.Request.Context(), re.App, cfg.Workflow.SyncBatchSize)
		if err != nil {
			return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?error="+url.QueryEscape("sync approved supplier products failed"))
		}

		message := fmt.Sprintf("sync approved supplier products: processed %d, updated %d, failed %d", result.Processed, result.Updated, result.Failed)
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin?message="+url.QueryEscape(message))
	})

	se.Router.GET("/_/mrtang-admin/procurement", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无采购模块权限", "当前账号没有采购模块权限，请联系管理员配置 `PIM_PROCUREMENT_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}

		summary, err := service.ProcurementWorkbenchSummaryFiltered(
			re.Request.Context(),
			re.App,
			readQueryInt(re, "pageSize", 20),
			strings.TrimSpace(re.Request.URL.Query().Get("status")),
			strings.TrimSpace(re.Request.URL.Query().Get("risk")),
			strings.TrimSpace(re.Request.URL.Query().Get("q")),
			readQueryInt(re, "page", 1),
		)
		if err != nil {
			return re.InternalServerError("load procurement workbench failed", err)
		}

		return re.HTML(http.StatusOK, admin.RenderProcurementWorkbenchHTML(summary))
	})

	se.Router.GET("/_/mrtang-admin/procurement/detail", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.HTML(http.StatusForbidden, admin.RenderForbiddenPageHTML("无采购模块权限", "当前账号没有采购模块权限，请联系管理员配置 `PIM_PROCUREMENT_ADMIN_EMAILS`。", "/_/mrtang-admin"))
		}
		id := strings.TrimSpace(re.Request.URL.Query().Get("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}
		order, err := service.GetProcurementOrder(re.Request.Context(), re.App, id)
		if err != nil {
			return re.InternalServerError("load procurement order detail failed", err)
		}
		returnTo := strings.TrimSpace(re.Request.URL.Query().Get("returnTo"))
		if returnTo == "" {
			returnTo = "/_/mrtang-admin/procurement"
		}
		return re.HTML(http.StatusOK, admin.RenderProcurementDetailHTML(order, returnTo))
	})

	se.Router.GET("/_/procurement-workbench", func(re *core.RequestEvent) error {
		if !admin.AuthorizedPage(re) {
			return re.UnauthorizedError("The request requires valid superuser authorization token or localhost access.", nil)
		}
		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
	})

	se.Router.POST("/_/mrtang-admin/procurement/order/status", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
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

		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
	})

	se.Router.POST("/_/mrtang-admin/procurement/order/export", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
		}

		id := strings.TrimSpace(re.Request.FormValue("id"))
		if id == "" {
			return re.BadRequestError("missing procurement order id", nil)
		}

		if _, err := service.ExportProcurementOrderWithAudit(re.Request.Context(), re.App, id, procurementActionActor(re), strings.TrimSpace(re.Request.FormValue("note"))); err != nil {
			return re.BadRequestError("export procurement order failed", err)
		}

		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
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

		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
	})

	se.Router.POST("/_/mrtang-admin/procurement/order/review", func(re *core.RequestEvent) error {
		if !authorizedAdminModule(re, cfg, "procurement") {
			return re.ForbiddenError("当前账号没有采购模块权限。", nil)
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

		return re.Redirect(http.StatusSeeOther, "/_/mrtang-admin/procurement")
	})
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
		ProductStatus: strings.TrimSpace(re.Request.URL.Query().Get("productStatus")),
		AssetStatus:   strings.TrimSpace(re.Request.URL.Query().Get("assetStatus")),
		SyncState:     strings.TrimSpace(re.Request.URL.Query().Get("syncState")),
		Query:         strings.TrimSpace(re.Request.URL.Query().Get("q")),
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

func authorizedAdminModule(re *core.RequestEvent, cfg config.Config, module string) bool {
	if !admin.AuthorizedPage(re) {
		return false
	}
	if re.Auth == nil {
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
		allowed = append(append([]string{}, cfg.Admin.SourceAdmins...), cfg.Admin.ProcurementAdmins...)
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
