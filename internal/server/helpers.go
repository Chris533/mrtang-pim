package server

import (
	"encoding/json"
	"errors"
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

func readOptionalJSONBody(re *core.RequestEvent) (any, error) {
	defer re.Request.Body.Close()

	var payload any
	if err := json.NewDecoder(re.Request.Body).Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	return payload, nil
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

	requestBody, err := readOptionalJSONBody(re)
	if err != nil {
		return re.BadRequestError("invalid request body", err)
	}

	operation, err := service.ExecuteCartOperation(re.Request.Context(), id, requestBody)
	if err != nil {
		return re.InternalServerError(label+" failed", err)
	}

	if operation == nil {
		return re.NotFoundError("cart operation not found", nil)
	}

	logMiniappWriteAction(re.App, cfg, id, label, requestBody, operation)

	return re.JSON(http.StatusOK, operation.Response)
}

func serveMiniAppOrderOperation(re *core.RequestEvent, cfg config.Config, service *miniappservice.Service, id string, label string) error {
	if !authorizedMiniApp(re, cfg) {
		return re.UnauthorizedError("missing or invalid miniapp authorization", nil)
	}

	requestBody, err := readOptionalJSONBody(re)
	if err != nil {
		return re.BadRequestError("invalid request body", err)
	}

	operation, err := service.ExecuteOrderOperation(re.Request.Context(), id, requestBody)
	if err != nil {
		return re.InternalServerError(label+" failed", err)
	}

	if operation == nil {
		return re.NotFoundError("order operation not found", nil)
	}

	logMiniappWriteAction(re.App, cfg, id, label, requestBody, operation)

	return re.JSON(http.StatusOK, operation.Response)
}

func logMiniappWriteAction(app core.App, cfg config.Config, operationID string, operationLabel string, requestBody any, operation *miniappmodel.OperationSnapshot) {
	if operation == nil || !strings.EqualFold(strings.TrimSpace(cfg.MiniApp.SourceMode), "raw") {
		return
	}
	if !isMiniappWriteOperation(operationID) || !strings.HasPrefix(strings.TrimSpace(operation.ContractID), "raw_") {
		return
	}

	collection, err := app.FindCollectionByNameOrId(pim.CollectionMiniappActionLogs)
	if err != nil {
		return
	}
	record := core.NewRecord(collection)
	record.Set("source_mode", strings.TrimSpace(cfg.MiniApp.SourceMode))
	record.Set("operation_id", operationID)
	record.Set("operation_label", operationLabel)
	record.Set("contract_id", strings.TrimSpace(operation.ContractID))
	record.Set("status", "success")
	record.Set("message", miniappActionMessage(operation.Response))
	record.Set("request_json", requestBody)
	record.Set("response_json", operation.Response)
	_ = app.Save(record)
}

func isMiniappWriteOperation(id string) bool {
	switch strings.TrimSpace(id) {
	case "add", "change-num", "add-delivery", "submit":
		return true
	default:
		return false
	}
}

func miniappActionMessage(response any) string {
	if mapped, ok := response.(map[string]any); ok {
		if message, ok := mapped["message"].(string); ok {
			return strings.TrimSpace(message)
		}
	}
	return ""
}
