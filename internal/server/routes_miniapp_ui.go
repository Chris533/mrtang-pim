package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/pim"
)

func registerMiniAppUiRoutes(se *core.ServeEvent, cfg config.Config, service *pim.Service) {
	se.Router.GET("/api/miniapp-ui/collections/tree", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}
		tree, err := service.MiniappCollectionsTree(re.Request.Context())
		if err != nil {
			return re.InternalServerError("load miniapp ui collections tree failed", err)
		}
		return re.JSON(http.StatusOK, tree)
	})

	se.Router.GET("/api/miniapp-ui/collections/products", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}
		slug := strings.TrimSpace(re.Request.URL.Query().Get("slug"))
		if slug == "" {
			return re.BadRequestError("missing collection slug", nil)
		}
		audience := strings.TrimSpace(re.Request.URL.Query().Get("audience"))
		if audience == "" {
			audience = "C"
		}
		skip, _ := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("skip")))
		take, _ := strconv.Atoi(strings.TrimSpace(re.Request.URL.Query().Get("take")))
		if take <= 0 {
			take = 24
		}
		result, err := service.MiniappCollectionProducts(re.Request.Context(), slug, audience, skip, take)
		if err != nil {
			return re.InternalServerError("load miniapp ui collection products failed", err)
		}
		return re.JSON(http.StatusOK, result)
	})

	se.Router.GET("/api/miniapp-ui/products/detail", func(re *core.RequestEvent) error {
		if !authorizedMiniApp(re, cfg) {
			return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
		}
		slug := strings.TrimSpace(re.Request.URL.Query().Get("slug"))
		if slug == "" {
			return re.BadRequestError("missing product slug", nil)
		}
		audience := strings.TrimSpace(re.Request.URL.Query().Get("audience"))
		if audience == "" {
			audience = "C"
		}
		product, err := service.MiniappProductDetail(re.Request.Context(), slug, audience)
		if err != nil {
			return re.InternalServerError("load miniapp ui product detail failed", err)
		}
		if product == nil {
			return re.NotFoundError("product not found", nil)
		}
		return re.JSON(http.StatusOK, product)
	})
}
