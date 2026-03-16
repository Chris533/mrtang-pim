package api

import (
	"context"
	"testing"

	"mrtang-pim/internal/miniapp/model"
)

func TestOverlaySourceNormalizesCartOrderFlow(t *testing.T) {
	base := &stubSource{
		dataset: &model.Dataset{
			CartOrder: model.CartOrderAggregate{
				Cart: model.CartAggregate{
					ChangeNum: model.OperationSnapshot{
						RequestBody: map[string]any{
							"id": "demo-cart-0",
						},
					},
					Detail: model.OperationSnapshot{
						Response: map[string]any{
							"data": map[string]any{
								"spuDetail": map[string]any{
									"spuList": []any{
										map[string]any{"id": "demo-cart-9"},
										map[string]any{"id": "demo-cart-10"},
									},
								},
								"totMoney": "88.50",
							},
						},
					},
				},
				Order: model.OrderAggregate{
					AddDelivery: model.OperationSnapshot{
						RequestBody: map[string]any{
							"customerId":   "demo-customer-001",
							"customerName": "演示客户",
							"phone":        "13800000000",
						},
						Response: map[string]any{
							"data": map[string]any{
								"businessId":   "demo-address-99",
								"customerId":   "demo-customer-001",
								"customerName": "演示客户",
								"phone":        "13800000000",
								"deliveryId":   "delivery-123",
								"deliveryName": "装车",
								"province":     510000,
								"city":         510400,
								"district":     510411,
							},
						},
					},
					Deliveries: model.OperationSnapshot{
						RequestBody: map[string]any{
							"businessId": "demo-customer-001",
						},
					},
					FreightCosts: []model.ScenarioAction{
						{
							Scenario: "selected_delivery",
							RequestBody: map[string]any{
								"deliveryMethodId": "old-delivery",
							},
							Response: map[string]any{
								"data": 12.5,
							},
						},
					},
					Submit: model.OperationSnapshot{
						RequestBody: map[string]any{
							"cartIdList": []any{"demo-cart-0"},
							"freight":    0,
							"dueMoney":   "0",
							"receiveAddressInfo": map[string]any{
								"addressId": "demo-address-001",
							},
							"wappEnv": map[string]any{
								"openId": "demo-open-id",
							},
						},
						Response: map[string]any{
							"data": map[string]any{
								"billId": "demo-bill-001",
							},
						},
					},
				},
			},
		},
	}

	source := NewOverlaySource(base)

	dataset, err := source.FetchDataset(context.Background())
	if err != nil {
		t.Fatalf("fetch dataset: %v", err)
	}

	changeBody := dataset.CartOrder.Cart.ChangeNum.RequestBody.(map[string]any)
	if changeBody["id"] != "demo-cart-9" {
		t.Fatalf("unexpected cart id: %#v", changeBody["id"])
	}

	addDeliveryBody := dataset.CartOrder.Order.AddDelivery.RequestBody.(map[string]any)
	if addDeliveryBody["customerId"] != "demo-customer-001" {
		t.Fatalf("unexpected customer id: %#v", addDeliveryBody["customerId"])
	}
	if addDeliveryBody["customerName"] != "演示客户" {
		t.Fatalf("unexpected customer name: %#v", addDeliveryBody["customerName"])
	}
	if addDeliveryBody["phone"] != "13800000000" {
		t.Fatalf("unexpected phone: %#v", addDeliveryBody["phone"])
	}

	submitBody := dataset.CartOrder.Order.Submit.RequestBody.(map[string]any)
	cartIDs := submitBody["cartIdList"].([]any)
	if len(cartIDs) != 2 || cartIDs[0] != "demo-cart-9" {
		t.Fatalf("unexpected cart ids: %#v", cartIDs)
	}
	if submitBody["dueMoney"] != "88.50" {
		t.Fatalf("unexpected due money: %#v", submitBody["dueMoney"])
	}
	if submitBody["freight"] != 12.5 {
		t.Fatalf("unexpected freight: %#v", submitBody["freight"])
	}
	address := submitBody["receiveAddressInfo"].(map[string]any)
	if address["addressId"] != "demo-address-99" {
		t.Fatalf("unexpected address id: %#v", address["addressId"])
	}
	if address["deliveryMethodId"] != "delivery-123" {
		t.Fatalf("unexpected delivery method id: %#v", address["deliveryMethodId"])
	}

	wappEnv := submitBody["wappEnv"].(map[string]any)
	if wappEnv["openId"] != "demo-open-id" {
		t.Fatalf("unexpected open id: %#v", wappEnv["openId"])
	}

	deliveriesBody := dataset.CartOrder.Order.Deliveries.RequestBody.(map[string]any)
	if deliveriesBody["businessId"] != "demo-customer-001" {
		t.Fatalf("unexpected deliveries business id: %#v", deliveriesBody["businessId"])
	}

	freightBody := dataset.CartOrder.Order.FreightCosts[0].RequestBody.(map[string]any)
	if freightBody["deliveryMethodId"] != "delivery-123" {
		t.Fatalf("unexpected freight delivery method id: %#v", freightBody["deliveryMethodId"])
	}
	if freightBody["customerId"] != "demo-customer-001" {
		t.Fatalf("unexpected freight customer id: %#v", freightBody["customerId"])
	}

	submitResponse := dataset.CartOrder.Order.Submit.Response.(map[string]any)
	data := submitResponse["data"].(map[string]any)
	if data["billId"] != "demo-bill-001" {
		t.Fatalf("unexpected bill id: %#v", data["billId"])
	}
}

type stubSource struct {
	dataset *model.Dataset
}

func (s *stubSource) FetchDataset(ctx context.Context) (*model.Dataset, error) {
	return s.dataset, nil
}
