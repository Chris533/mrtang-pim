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

func TestEnsureAuthenticatedReturnsErrorOnLoginErrorResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"login":{"errorCode":"INVALID_CREDENTIALS","message":"Invalid credentials"}}}`))
	}))
	defer server.Close()

	client := NewClient(config.VendureConfig{
		Endpoint:       server.URL,
		Username:       "superadmin",
		Password:       "wrong-password",
		RequestTimeout: 2 * time.Second,
	})

	err := client.ensureAuthenticated(context.Background())
	if err == nil {
		t.Fatal("expected login error, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid credentials") {
		t.Fatalf("expected Invalid credentials error, got %v", err)
	}
	if client.loggedIn {
		t.Fatal("client must not be marked logged in after ErrorResult")
	}
}

func TestEnsureAuthenticatedCapturesBearerSessionToken(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var payload struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")

		if requestCount == 1 {
			if got := r.Header.Get("Authorization"); got != "" {
				t.Fatalf("login request must not send Authorization header before session token is captured, got %q", got)
			}
			w.Header().Set("vendure-auth-token", "session-token-123")
			_, _ = w.Write([]byte(`{"data":{"login":{"id":"1","identifier":"superadmin"}}}`))
			return
		}

		if got := r.Header.Get("Authorization"); got != "Bearer session-token-123" {
			t.Fatalf("expected Authorization header on authenticated request, got %q", got)
		}
		_, _ = w.Write([]byte(`{"data":{"updateProduct":{"id":"27"}}}`))
	}))
	defer server.Close()

	client := NewClient(config.VendureConfig{
		Endpoint:       server.URL,
		Username:       "superadmin",
		Password:       "superadmin",
		RequestTimeout: 2 * time.Second,
	})

	if err := client.DisableProduct(context.Background(), "27"); err != nil {
		t.Fatalf("DisableProduct returned error: %v", err)
	}
	if got := client.sessionToken; got != "session-token-123" {
		t.Fatalf("expected captured session token, got %q", got)
	}
}
