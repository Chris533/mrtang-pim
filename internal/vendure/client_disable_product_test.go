package vendure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mrtang-pim/internal/config"
)

func TestDisableProductUsesPlainUpdateProductSelection(t *testing.T) {
	var query string
	var variables map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		query = payload.Query
		variables = payload.Variables

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"updateProduct":{"id":"27"}}}`))
	}))
	defer server.Close()

	client := NewClient(config.VendureConfig{
		Endpoint:       server.URL,
		Token:          "test-token",
		RequestTimeout: 2 * time.Second,
	})

	if err := client.DisableProduct(context.Background(), "27"); err != nil {
		t.Fatalf("DisableProduct returned error: %v", err)
	}

	if strings.Contains(query, "... on ErrorResult") {
		t.Fatalf("DisableProduct mutation must not spread ErrorResult: %s", query)
	}
	if !strings.Contains(query, "updateProduct(input: $input)") || !strings.Contains(query, "id") {
		t.Fatalf("unexpected DisableProduct mutation: %s", query)
	}

	input, ok := variables["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input variables map, got %T", variables["input"])
	}
	if got := input["id"]; got != "27" {
		t.Fatalf("expected input.id=27, got %v", got)
	}
	if got := input["enabled"]; got != false {
		t.Fatalf("expected input.enabled=false, got %v", got)
	}
}
