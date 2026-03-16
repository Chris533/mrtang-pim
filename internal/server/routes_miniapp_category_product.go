package server

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
)

func registerMiniAppCategoryRoutes(se *core.ServeEvent, cfg config.Config, miniappService *miniappservice.Service) {
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
}

func registerMiniAppProductRoutes(se *core.ServeEvent, cfg config.Config, miniappService *miniappservice.Service) {
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
			matchedFirstBatch := false
			for _, item := range summary.FirstBatch {
				if strings.EqualFold(item.Priority, priority) {
					matchedFirstBatch = true
					break
				}
			}
			if !matchedFirstBatch {
				summary.FirstBatch = nil
			}
		}

		return re.JSON(http.StatusOK, summary)
	})
}
