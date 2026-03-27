package admin

import (
	"context"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

type DashboardAPIData = mrtangAdminPageData
type DashboardMiniappLiveAPIData = DashboardMiniappAPIData

type TargetSyncAPIData struct {
	Summary      pim.TargetSyncBaseSummary `json:"summary"`
	Harvest      HarvestAdminData          `json:"harvest"`
	HarvestError string                    `json:"harvestError"`
	FlashMessage string                    `json:"flashMessage"`
	FlashError   string                    `json:"flashError"`
	RequiresAuth bool                      `json:"requiresAuth"`
	SourceURL    string                    `json:"sourceURL"`
}

type TargetSyncLiveAPIData struct {
	Summary       pim.TargetSyncLiveSummary  `json:"summary"`
	RawAuthStatus miniappmodel.RawAuthStatus `json:"rawAuthStatus"`
	FlashError    string                     `json:"flashError"`
}

type TargetSyncCheckoutLiveAPIData struct {
	Summary    pim.TargetSyncCheckoutLiveSummary `json:"summary"`
	FlashError string                            `json:"flashError"`
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

type SourceProductJobsAPIData struct {
	Summary      pim.SourceProductJobsSummary `json:"summary"`
	Filter       pim.SourceProductJobFilter   `json:"filter"`
	FlashMessage string                       `json:"flashMessage"`
	FlashError   string                       `json:"flashError"`
}

type ProcurementAPIData struct {
	Summary      pim.ProcurementWorkbenchSummary `json:"summary"`
	FlashMessage string                          `json:"flashMessage"`
	FlashError   string                          `json:"flashError"`
}

type BackendReleaseAPIData struct {
	Summary      pim.BackendReleaseSummary `json:"summary"`
	FlashMessage string                    `json:"flashMessage"`
	FlashError   string                    `json:"flashError"`
}

type BackendReleasePreviewAPIData struct {
	Preview      pim.BackendReleasePayloadPreview `json:"preview"`
	FlashMessage string                           `json:"flashMessage"`
	FlashError   string                           `json:"flashError"`
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

type SourceProductJobDetailAPIData struct {
	Detail       pim.SourceProductJobDetail `json:"detail"`
	ReturnTo     string                     `json:"returnTo"`
	FlashMessage string                     `json:"flashMessage"`
	FlashError   string                     `json:"flashError"`
}

type HarvestDetailAPIData struct {
	Detail       pim.HarvestRun `json:"detail"`
	ReturnTo     string         `json:"returnTo"`
	FlashMessage string         `json:"flashMessage"`
	FlashError   string         `json:"flashError"`
}

type HarvestSummaryAPIData struct {
	Harvest    HarvestAdminData `json:"harvest"`
	FlashError string           `json:"flashError"`
}

type SourceLogsAPIData struct {
	Data         SourceLogsPageData `json:"data"`
	FlashMessage string             `json:"flashMessage"`
	FlashError   string             `json:"flashError"`
}

type AuditAPIData struct {
	Data         AuditPageData `json:"data"`
	FlashMessage string        `json:"flashMessage"`
	FlashError   string        `json:"flashError"`
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
	data := buildMrtangAdminBaseData(ctx, app, cfg, pimService)
	if app != nil && pimService != nil {
		if storedData, err := buildStoredDashboardMiniappData(app, cfg, miniappService, pimService); err == nil {
			data.Miniapp = storedData
		}
	}
	data.CanAccessSource = canAccessSource
	data.CanAccessProcurement = canAccessProcurement
	data.FlashMessage = strings.TrimSpace(flashMessage)
	data.FlashError = strings.TrimSpace(flashError)
	return data
}

func BuildDashboardMiniappAPIData(ctx context.Context, app core.App, cfg config.Config, pimService *pim.Service, miniappService *miniappservice.Service) DashboardMiniappLiveAPIData {
	data := buildDashboardMiniappAPIData(ctx, cfg, miniappService)
	if data.MiniappError == "" || app == nil || pimService == nil {
		return data
	}
	storedData, err := buildStoredDashboardMiniappData(app, cfg, miniappService, pimService)
	if err != nil {
		return data
	}
	data.Miniapp = storedData
	data.MiniappError = "加载实时摘要失败，已回退到已落库结果：" + data.MiniappError
	return data
}

func BuildTargetSyncAPIData(
	cfg config.Config,
	summary pim.TargetSyncBaseSummary,
	harvest HarvestAdminData,
	harvestError string,
	flashMessage string,
	flashError string,
) TargetSyncAPIData {
	return TargetSyncAPIData{
		Summary:      summary,
		Harvest:      harvest,
		HarvestError: strings.TrimSpace(harvestError),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
		RequiresAuth: strings.TrimSpace(cfg.MiniApp.AuthorizedAccountID) != "",
		SourceURL:    strings.TrimSpace(cfg.MiniApp.SourceURL),
	}
}

func BuildTargetSyncLiveAPIData(summary pim.TargetSyncLiveSummary, rawAuthStatus miniappmodel.RawAuthStatus, flashError string) TargetSyncLiveAPIData {
	return TargetSyncLiveAPIData{
		Summary:       summary,
		RawAuthStatus: rawAuthStatus,
		FlashError:    strings.TrimSpace(flashError),
	}
}

func BuildTargetSyncCheckoutLiveAPIData(summary pim.TargetSyncCheckoutLiveSummary, flashError string) TargetSyncCheckoutLiveAPIData {
	return TargetSyncCheckoutLiveAPIData{
		Summary:    summary,
		FlashError: strings.TrimSpace(flashError),
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

func BuildSourceProductJobsAPIData(summary pim.SourceProductJobsSummary, filter pim.SourceProductJobFilter, flashMessage string, flashError string) SourceProductJobsAPIData {
	return SourceProductJobsAPIData{
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

func BuildBackendReleaseAPIData(summary pim.BackendReleaseSummary, flashMessage string, flashError string) BackendReleaseAPIData {
	return BackendReleaseAPIData{
		Summary:      summary,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildBackendReleasePreviewAPIData(preview pim.BackendReleasePayloadPreview, flashMessage string, flashError string) BackendReleasePreviewAPIData {
	return BackendReleasePreviewAPIData{
		Preview:      preview,
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

func BuildSourceProductJobDetailAPIData(detail pim.SourceProductJobDetail, returnTo string, flashMessage string, flashError string) SourceProductJobDetailAPIData {
	return SourceProductJobDetailAPIData{
		Detail:       detail,
		ReturnTo:     strings.TrimSpace(returnTo),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildHarvestDetailAPIData(detail pim.HarvestRun, returnTo string, flashMessage string, flashError string) HarvestDetailAPIData {
	return HarvestDetailAPIData{
		Detail:       detail,
		ReturnTo:     strings.TrimSpace(returnTo),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildHarvestSummaryAPIData(harvest HarvestAdminData, flashError string) HarvestSummaryAPIData {
	return HarvestSummaryAPIData{
		Harvest:    harvest,
		FlashError: strings.TrimSpace(flashError),
	}
}

func BuildSourceLogsAPIData(data SourceLogsPageData, flashMessage string, flashError string) SourceLogsAPIData {
	return SourceLogsAPIData{
		Data:         data,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}
}

func BuildAuditAPIData(
	ctx context.Context,
	app core.App,
	cfg config.Config,
	pimService *pim.Service,
	miniappService *miniappservice.Service,
	filter AuditFilter,
	flashMessage string,
	flashError string,
) AuditAPIData {
	pageData := buildMrtangAdminPageData(ctx, app, cfg, pimService, miniappService)
	return AuditAPIData{
		Data:         filterAuditActions(pageData.RecentActions, filter),
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
