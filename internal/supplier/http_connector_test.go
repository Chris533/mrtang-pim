package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPConnectorSubmitAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "api-key" {
			t.Fatalf("unexpected api key header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accepted":    true,
			"mode":        "live",
			"externalRef": "SUP-ORDER-1",
		})
	}))
	defer server.Close()

	connector := NewHTTPConnector(HTTPConnectorConfig{
		BaseURL:      server.URL,
		SubmitPath:   "/submit",
		Token:        "token-123",
		APIKey:       "api-key",
		SupplierCode: "SUP_A",
		Timeout:      3 * time.Second,
	})

	result, err := connector.SubmitPurchaseOrder(context.Background(), PurchaseOrder{
		SupplierCode: "SUP_A",
		ExternalRef:  "PO-1",
	})
	if err != nil {
		t.Fatalf("submit purchase order failed: %v", err)
	}
	if !result.Accepted {
		t.Fatalf("expected accepted=true")
	}
	if result.Mode != "live" {
		t.Fatalf("unexpected mode: %s", result.Mode)
	}
	if result.ExternalRef != "SUP-ORDER-1" {
		t.Fatalf("unexpected external ref: %s", result.ExternalRef)
	}
}

func TestHTTPConnectorSubmitRejectedByPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"msg":     "out of stock",
		})
	}))
	defer server.Close()

	connector := NewHTTPConnector(HTTPConnectorConfig{
		BaseURL:      server.URL,
		SubmitPath:   "/submit",
		SupplierCode: "SUP_A",
	})

	result, err := connector.SubmitPurchaseOrder(context.Background(), PurchaseOrder{
		SupplierCode: "SUP_A",
		ExternalRef:  "PO-2",
	})
	if err != nil {
		t.Fatalf("submit purchase order failed: %v", err)
	}
	if result.Accepted {
		t.Fatalf("expected accepted=false")
	}
	if !strings.Contains(result.Message, "out of stock") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestHTTPConnectorSubmitHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	connector := NewHTTPConnector(HTTPConnectorConfig{
		BaseURL:      server.URL,
		SubmitPath:   "/submit",
		SupplierCode: "SUP_A",
	})

	_, err := connector.SubmitPurchaseOrder(context.Background(), PurchaseOrder{
		SupplierCode: "SUP_A",
		ExternalRef:  "PO-3",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
