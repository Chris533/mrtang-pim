package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"mrtang-pim/internal/config"
	miniappmodel "mrtang-pim/internal/miniapp/model"
	miniappservice "mrtang-pim/internal/miniapp/service"
	"mrtang-pim/internal/pim"
)

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
