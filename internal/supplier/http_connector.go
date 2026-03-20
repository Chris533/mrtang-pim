package supplier

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPConnectorConfig struct {
	BaseURL       string
	SubmitPath    string
	FetchPath     string
	Token         string
	APIKey        string
	SupplierCode  string
	Timeout       time.Duration
	SkipTLSVerify bool
}

type HTTPConnector struct {
	baseURL      string
	submitPath   string
	fetchPath    string
	token        string
	apiKey       string
	supplierCode string
	client       *http.Client
}

func NewHTTPConnector(cfg HTTPConnectorConfig) *HTTPConnector {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	transport := &http.Transport{}
	if cfg.SkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit opt-in via env
	}

	return &HTTPConnector{
		baseURL:      strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		submitPath:   defaultPath(cfg.SubmitPath, "/purchase-orders"),
		fetchPath:    strings.TrimSpace(cfg.FetchPath),
		token:        strings.TrimSpace(cfg.Token),
		apiKey:       strings.TrimSpace(cfg.APIKey),
		supplierCode: strings.TrimSpace(cfg.SupplierCode),
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *HTTPConnector) Capabilities() ConnectorCapabilities {
	return ConnectorCapabilities{
		FetchProducts:       c.fetchPath != "",
		SubmitPurchaseOrder: true,
		ExportPurchaseOrder: false,
	}
}

func (c *HTTPConnector) Fetch(ctx context.Context) ([]Product, error) {
	if c.fetchPath == "" {
		return nil, fmt.Errorf("http connector fetch endpoint is not configured")
	}
	endpoint, err := c.resolveURL(c.fetchPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		return nil, fmt.Errorf("supplier fetch failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var products []Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, fmt.Errorf("decode supplier fetch response: %w", err)
	}
	for i := range products {
		if strings.TrimSpace(products[i].SupplierCode) == "" {
			products[i].SupplierCode = c.supplierCode
		}
	}
	return products, nil
}

func (c *HTTPConnector) SubmitPurchaseOrder(ctx context.Context, order PurchaseOrder) (PurchaseOrderResult, error) {
	endpoint, err := c.resolveURL(c.submitPath)
	if err != nil {
		return PurchaseOrderResult{}, err
	}

	body, err := json.Marshal(order)
	if err != nil {
		return PurchaseOrderResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return PurchaseOrderResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return PurchaseOrderResult{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return PurchaseOrderResult{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PurchaseOrderResult{}, fmt.Errorf("supplier submit failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	result := PurchaseOrderResult{
		SupplierCode: defaultStringFrom(strings.TrimSpace(order.SupplierCode), c.supplierCode),
		ExternalRef:  strings.TrimSpace(order.ExternalRef),
		Mode:         "http_live",
		Accepted:     true,
	}

	if len(strings.TrimSpace(string(respBody))) == 0 {
		return result, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return result, nil
	}

	flat := flattenData(payload)
	if v, ok := readBool(flat, "accepted"); ok {
		result.Accepted = v
	} else if v, ok := readBool(flat, "success"); ok {
		result.Accepted = v
	}

	if v := readString(flat, "mode"); v != "" {
		result.Mode = v
	}
	if v := readString(flat, "message", "msg", "error"); v != "" {
		result.Message = v
	}
	if v := readString(flat, "supplierCode", "supplier_code"); v != "" {
		result.SupplierCode = v
	}
	if v := readString(flat, "externalRef", "external_ref", "orderNo", "order_no", "id"); v != "" {
		result.ExternalRef = v
	}

	if code, ok := readFloat(flat, "code"); ok && code != 0 && code != 200 {
		result.Accepted = false
		if result.Message == "" {
			result.Message = fmt.Sprintf("supplier returned code %.0f", code)
		}
	}

	return result, nil
}

func (c *HTTPConnector) resolveURL(path string) (string, error) {
	base := strings.TrimSpace(c.baseURL)
	if base == "" {
		return "", fmt.Errorf("supplier http base url is empty")
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid supplier http base url: %w", err)
	}
	next := strings.TrimSpace(path)
	if next == "" {
		next = "/"
	}
	if !strings.HasPrefix(next, "/") {
		next = "/" + next
	}
	u.Path = strings.TrimRight(u.Path, "/") + next
	return u.String(), nil
}

func (c *HTTPConnector) setHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

func defaultPath(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func defaultStringFrom(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func flattenData(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		merged := make(map[string]any, len(payload)+len(data))
		for k, v := range payload {
			merged[k] = v
		}
		for k, v := range data {
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
		}
		return merged
	}
	return payload
}

func readString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch casted := value.(type) {
		case string:
			if strings.TrimSpace(casted) != "" {
				return strings.TrimSpace(casted)
			}
		}
	}
	return ""
}

func readBool(payload map[string]any, key string) (bool, bool) {
	value, ok := payload[key]
	if !ok || value == nil {
		return false, false
	}
	switch casted := value.(type) {
	case bool:
		return casted, true
	case string:
		raw := strings.ToLower(strings.TrimSpace(casted))
		if raw == "" {
			return false, false
		}
		if raw == "true" || raw == "1" || raw == "yes" || raw == "on" {
			return true, true
		}
		if raw == "false" || raw == "0" || raw == "no" || raw == "off" {
			return false, true
		}
	}
	return false, false
}

func readFloat(payload map[string]any, key string) (float64, bool) {
	value, ok := payload[key]
	if !ok || value == nil {
		return 0, false
	}
	switch casted := value.(type) {
	case float64:
		return casted, true
	case float32:
		return float64(casted), true
	case int:
		return float64(casted), true
	case int64:
		return float64(casted), true
	case json.Number:
		v, err := casted.Float64()
		if err == nil {
			return v, true
		}
	}
	return 0, false
}
