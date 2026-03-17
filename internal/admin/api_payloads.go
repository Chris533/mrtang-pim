package admin

import (
	"context"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

type DashboardAPIData = mrtangAdminPageData

type TargetSyncAPIData struct {
	Summary      pim.TargetSyncSummary `json:"summary"`
	FlashMessage string                `json:"flashMessage"`
	FlashError   string                `json:"flashError"`
	RequiresAuth bool                  `json:"requiresAuth"`
	SourceURL    string                `json:"sourceURL"`
}

type SourceModuleAPIData struct {
	Summary      pim.SourceReviewWorkbenchSummary `json:"summary"`
	FlashMessage string                           `json:"flashMessage"`
	FlashError   string                           `json:"flashError"`
}

type SourceProductsAPIData struct {
	Summary      pim.SourceReviewWorkbenchSummary `json:"summary"`
	Filter       pim.SourceReviewFilter           `json:"filter"`
	FlashMessage string                           `json:"flashMessage"`
	FlashError   string                           `json:"flashError"`
}

type SourceCategoriesAPIData struct {
	Summary      pim.SourceCategoriesSummary `json:"summary"`
	Filter       pim.SourceCategoryFilter    `json:"filter"`
	FlashMessage string                      `json:"flashMessage"`
	FlashError   string                      `json:"flashError"`
}

type SourceAssetsAPIData struct {
	Summary      pim.SourceReviewWorkbenchSummary `json:"summary"`
	Filter       pim.SourceReviewFilter           `json:"filter"`
	FlashMessage string                           `json:"flashMessage"`
	FlashError   string                           `json:"flashError"`
}

type SourceAssetJobsAPIData struct {
	Summary      pim.SourceAssetJobsSummary `json:"summary"`
	Filter       pim.SourceAssetJobFilter   `json:"filter"`
	FlashMessage string                     `json:"flashMessage"`
	FlashError   string                     `json:"flashError"`
}

type ProcurementAPIData struct {
	Summary      pim.ProcurementWorkbenchSummary `json:"summary"`
	FlashMessage string                          `json:"flashMessage"`
	FlashError   string                          `json:"flashError"`
}

type SourceProductDetailAPIData struct {
	Detail       pim.SourceProductDetail `json:"detail"`
	ReturnTo     string                  `json:"returnTo"`
	FlashMessage string                  `json:"flashMessage"`
	FlashError   string                  `json:"flashError"`
}

type SourceAssetDetailAPIData struct {
	Detail       pim.SourceAssetDetail `json:"detail"`
	ReturnTo     string                `json:"returnTo"`
	FlashMessage string                `json:"flashMessage"`
	FlashError   string                `json:"flashError"`
}

type SourceAssetJobDetailAPIData struct {
	Detail       pim.SourceAssetJobDetail `json:"detail"`
	ReturnTo     string                   `json:"returnTo"`
	FlashMessage string                   `json:"flashMessage"`
	FlashError   string                   `json:"flashError"`
}

type ProcurementDetailAPIData struct {
	Order        pim.ProcurementOrder `json:"order"`
	ReturnTo     string               `json:"returnTo"`
	FlashMessage string               `json:"flashMessage"`
	FlashError   string               `json:"flashError"`
}

func BuildDashboardAPIData(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
	canAccessSource bool,
	canAccessProcurement bool,
	flashMessage string,
	flashError string,
) DashboardAPIData {
	data := buildMrtangAdminPageData(ctx, app, cfg, pimService, miniappService)
	data.CanAccessSource = canAccessSource
	data.CanAccessProcurement = canAccessProcurement
	data.FlashMessage = strings.TrimSpace(flashMessage)
	data.FlashError = strings.TrimSpace(flashError)
	return data
}

func BuildTargetSyncAPIData(
	cfg config.Config,
	summary pim.TargetSyncSummary,
	flashMessage string,
	flashError string,
) TargetSyncAPIData {
	return TargetSyncAPIData{
		Summary:      summary,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
		RequiresAuth: strings.TrimSpace(cfg.MiniApp.AuthorizedAccountID) != "",
		SourceURL:    strings.TrimSpace(cfg.MiniApp.SourceURL),
	}
}

func BuildSourceModuleAPIData(summary pim.SourceReviewWorkbenchSummary, flashMessage string, flashError string) SourceModuleAPIData {
	return SourceModuleAPIData{
		Summary:      summary,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceProductsAPIData(summary pim.SourceReviewWorkbenchSummary, filter pim.SourceReviewFilter, flashMessage string, flashError string) SourceProductsAPIData {
	return SourceProductsAPIData{
		Summary:      summary,
		Filter:       filter,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceCategoriesAPIData(summary pim.SourceCategoriesSummary, filter pim.SourceCategoryFilter, flashMessage string, flashError string) SourceCategoriesAPIData {
	return SourceCategoriesAPIData{
		Summary:      summary,
		Filter:       filter,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceAssetsAPIData(summary pim.SourceReviewWorkbenchSummary, filter pim.SourceReviewFilter, flashMessage string, flashError string) SourceAssetsAPIData {
	return SourceAssetsAPIData{
		Summary:      summary,
		Filter:       filter,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceAssetJobsAPIData(summary pim.SourceAssetJobsSummary, filter pim.SourceAssetJobFilter, flashMessage string, flashError string) SourceAssetJobsAPIData {
	return SourceAssetJobsAPIData{
		Summary:      summary,
		Filter:       filter,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildProcurementAPIData(summary pim.ProcurementWorkbenchSummary, flashMessage string, flashError string) ProcurementAPIData {
	return ProcurementAPIData{
		Summary:      summary,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceProductDetailAPIData(detail pim.SourceProductDetail, returnTo string, flashMessage string, flashError string) SourceProductDetailAPIData {
	return SourceProductDetailAPIData{
		Detail:       detail,
		ReturnTo:     strings.TrimSpace(returnTo),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceAssetDetailAPIData(detail pim.SourceAssetDetail, returnTo string, flashMessage string, flashError string) SourceAssetDetailAPIData {
	return SourceAssetDetailAPIData{
		Detail:       detail,
		ReturnTo:     strings.TrimSpace(returnTo),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildSourceAssetJobDetailAPIData(detail pim.SourceAssetJobDetail, returnTo string, flashMessage string, flashError string) SourceAssetJobDetailAPIData {
	return SourceAssetJobDetailAPIData{
		Detail:       detail,
		ReturnTo:     strings.TrimSpace(returnTo),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildProcurementDetailAPIData(order pim.ProcurementOrder, returnTo string, flashMessage string, flashError string) ProcurementDetailAPIData {
	return ProcurementDetailAPIData{
		Order:        order,
		ReturnTo:     strings.TrimSpace(returnTo),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}
