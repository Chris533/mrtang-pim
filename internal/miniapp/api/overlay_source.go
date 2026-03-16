package api

import (
	"context"
	"encoding/json"
	"strings"

	"mrtang-pim/internal/miniapp/model"
)

type OverlaySource struct {
	base Source
}

func NewOverlaySource(base Source) *OverlaySource {
	return &OverlaySource{
		base: base,
	}
}

func (s *OverlaySource) FetchDataset(ctx context.Context) (*model.Dataset, error) {
	dataset, err := s.base.FetchDataset(ctx)
	if err != nil || dataset == nil {
		return dataset, err
	}

	body, err := json.Marshal(dataset)
	if err != nil {
		return dataset, nil
	}

	var copied model.Dataset
	if err := json.Unmarshal(body, &copied); err != nil {
		return dataset, nil
	}

	normalizeCartOrderFlow(&copied)

	return &copied, nil
}

func normalizeCartOrderFlow(dataset *model.Dataset) {
	if dataset == nil {
		return
	}

	customer := extractCustomer(dataset.CartOrder.Order.AddDelivery)
	propagateCustomerIdentity(&dataset.CartOrder.Order, customer)

	cartIDs := collectCartIDs(dataset.CartOrder.Cart.Detail.Response)
	if len(cartIDs) > 0 {
		setNestedValue(dataset.CartOrder.Cart.ChangeNum.RequestBody, cartIDs[0], "id")
		setNestedValue(dataset.CartOrder.Order.Submit.RequestBody, cartIDs, "cartIdList")
	}

	address := extractAddressResponse(dataset.CartOrder.Order.AddDelivery.Response)
	if len(address) > 0 {
		copyAddressToSubmit(&dataset.CartOrder.Order.Submit, address)
		propagateDeliverySelection(&dataset.CartOrder.Order, address)
	}

	if total, ok := extractCartTotal(dataset.CartOrder.Cart.Detail.Response); ok {
		setNestedValue(dataset.CartOrder.Order.Submit.RequestBody, total, "dueMoney")
	}

	if freight, ok := extractFreight(dataset.CartOrder.Order.FreightCosts, "selected_delivery"); ok {
		setNestedValue(dataset.CartOrder.Order.Submit.RequestBody, freight, "freight")
	}
}

func collectCartIDs(payload any) []any {
	items, ok := getNestedSlice(payload, "data", "spuDetail", "spuList")
	if !ok {
		return nil
	}

	ids := make([]any, 0, len(items))
	for _, item := range items {
		if id, ok := getNestedString(item, "id"); ok && strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}

	return ids
}

func extractAddressResponse(payload any) map[string]any {
	address, ok := getNestedMap(payload, "data")
	if !ok {
		return nil
	}

	return address
}

func extractCustomer(addDelivery model.OperationSnapshot) map[string]any {
	requestBody, _ := addDelivery.RequestBody.(map[string]any)
	responseData, _ := getNestedMap(addDelivery.Response, "data")

	customer := map[string]any{}
	for _, key := range []string{"customerId", "customerName", "phone"} {
		if responseData != nil {
			if value, ok := responseData[key]; ok {
				customer[key] = value
				continue
			}
		}
		if requestBody != nil {
			if value, ok := requestBody[key]; ok {
				customer[key] = value
			}
		}
	}

	if len(customer) == 0 {
		return nil
	}

	return customer
}

func copyAddressToSubmit(submit *model.OperationSnapshot, address map[string]any) {
	if submit == nil || submit.RequestBody == nil || len(address) == 0 {
		return
	}

	request, ok := submit.RequestBody.(map[string]any)
	if !ok {
		return
	}

	cloned := cloneMap(address)
	addressID, _ := getNestedString(address, "businessId")
	if addressID != "" {
		cloned["addressId"] = addressID
	}
	deliveryID, _ := getNestedString(address, "deliveryId")
	if deliveryID != "" {
		cloned["deliveryMethodId"] = deliveryID
	}
	deliveryName, _ := getNestedString(address, "deliveryName")
	if deliveryName != "" {
		cloned["deliveryMethodName"] = deliveryName
	}
	if _, ok := cloned["contentType"]; !ok {
		cloned["contentType"] = 1
	}

	request["receiveAddressInfo"] = cloned
}

func propagateDeliverySelection(order *model.OrderAggregate, address map[string]any) {
	if order == nil || len(address) == 0 {
		return
	}

	if deliveryID, ok := getNestedString(address, "deliveryId"); ok && deliveryID != "" {
		setNestedValue(order.Submit.RequestBody, deliveryID, "deliveryMethodId")
		for idx := range order.FreightCosts {
			if strings.EqualFold(order.FreightCosts[idx].Scenario, "selected_delivery") {
				setNestedValue(order.FreightCosts[idx].RequestBody, deliveryID, "deliveryMethodId")
			}
		}
	}
	if province, ok := getNestedValue(address, "province"); ok {
		for idx := range order.FreightCosts {
			if strings.EqualFold(order.FreightCosts[idx].Scenario, "selected_delivery") {
				setNestedValue(order.FreightCosts[idx].RequestBody, province, "province")
			}
		}
	}
	if city, ok := getNestedValue(address, "city"); ok {
		for idx := range order.FreightCosts {
			if strings.EqualFold(order.FreightCosts[idx].Scenario, "selected_delivery") {
				setNestedValue(order.FreightCosts[idx].RequestBody, city, "city")
			}
		}
	}
	if district, ok := getNestedValue(address, "district"); ok {
		for idx := range order.FreightCosts {
			if strings.EqualFold(order.FreightCosts[idx].Scenario, "selected_delivery") {
				setNestedValue(order.FreightCosts[idx].RequestBody, district, "district")
			}
		}
	}
}

func propagateCustomerIdentity(order *model.OrderAggregate, customer map[string]any) {
	if order == nil || len(customer) == 0 {
		return
	}

	if customerID, ok := customer["customerId"]; ok {
		setNestedValue(order.DefaultDelivery.RequestBody, customerID, "businessId")
		setNestedValue(order.Deliveries.RequestBody, customerID, "businessId")
		setNestedValue(order.Submit.RequestBody, customerID, "customerId")
		for idx := range order.FreightCosts {
			setNestedValue(order.FreightCosts[idx].RequestBody, customerID, "customerId")
		}
	}
}

func extractCartTotal(payload any) (any, bool) {
	if total, ok := getNestedValue(payload, "data", "totMoney"); ok {
		return total, true
	}
	return nil, false
}

func extractFreight(items []model.ScenarioAction, scenario string) (any, bool) {
	for _, item := range items {
		if !strings.EqualFold(item.Scenario, scenario) {
			continue
		}
		if cost, ok := getNestedValue(item.Response, "data"); ok {
			return cost, true
		}
	}
	return nil, false
}

func setNestedValue(target any, value any, path ...string) {
	current, ok := target.(map[string]any)
	if !ok || len(path) == 0 {
		return
	}

	for idx := 0; idx < len(path)-1; idx++ {
		next, ok := current[path[idx]].(map[string]any)
		if !ok {
			return
		}
		current = next
	}
	current[path[len(path)-1]] = value
}

func getNestedValue(target any, path ...string) (any, bool) {
	current := target
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = next[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func getNestedString(target any, path ...string) (string, bool) {
	value, ok := getNestedValue(target, path...)
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func getNestedMap(target any, path ...string) (map[string]any, bool) {
	value, ok := getNestedValue(target, path...)
	if !ok {
		return nil, false
	}
	result, ok := value.(map[string]any)
	return result, ok
}

func getNestedSlice(target any, path ...string) ([]any, bool) {
	value, ok := getNestedValue(target, path...)
	if !ok {
		return nil, false
	}
	result, ok := value.([]any)
	return result, ok
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = cloneValue(value)
	}

	return copied
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		copied := make([]any, len(typed))
		for idx, item := range typed {
			copied[idx] = cloneValue(item)
		}
		return copied
	default:
		return typed
	}
}
