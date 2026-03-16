package vendure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mrtang-pim/internal/config"
)

type ProductPayload struct {
	Name           string
	Slug           string
	Description    string
	SKU            string
	CurrencyCode   string
	ConsumerPrice  int
	AssetURL       string
	AssetName      string
	BusinessPrice  int
	DefaultStock   int
	SalesUnit      string
	VendureProduct string
	VendureVariant string
	NeedColdChain  bool
}

type SyncResult struct {
	ProductID string
	VariantID string
	AssetID   string
}

type Client struct {
	cfg        config.VendureConfig
	httpClient *http.Client
	mu         sync.Mutex
	loggedIn   bool
}

func NewClient(cfg config.VendureConfig) *Client {
	jar, _ := cookiejar.New(nil)

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
			Jar:     jar,
		},
	}
}

func (c *Client) SyncProduct(ctx context.Context, payload ProductPayload) (SyncResult, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return SyncResult{}, err
	}

	var result SyncResult

	if strings.TrimSpace(payload.AssetURL) != "" {
		assetID, err := c.uploadAsset(ctx, payload.AssetURL, payload.AssetName)
		if err != nil {
			return SyncResult{}, err
		}
		result.AssetID = assetID
	}

	if strings.TrimSpace(payload.VendureProduct) == "" {
		productID, err := c.createProduct(ctx, payload, result.AssetID)
		if err != nil {
			return SyncResult{}, err
		}
		result.ProductID = productID

		variantID, err := c.createVariant(ctx, payload, productID, result.AssetID)
		if err != nil {
			return SyncResult{}, err
		}
		result.VariantID = variantID
		return result, nil
	}

	result.ProductID = payload.VendureProduct
	result.VariantID = payload.VendureVariant

	if err := c.updateProduct(ctx, payload, result.AssetID); err != nil {
		return SyncResult{}, err
	}

	if strings.TrimSpace(payload.VendureVariant) != "" {
		if err := c.updateVariant(ctx, payload, result.AssetID); err != nil {
			return SyncResult{}, err
		}
	}

	return result, nil
}

func (c *Client) DisableProduct(ctx context.Context, productID string) error {
	if strings.TrimSpace(productID) == "" {
		return nil
	}

	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}

	mutation := `
mutation UpdateProduct($input: UpdateProductInput!) {
  updateProduct(input: $input) {
    ... on Product {
      id
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`

	var response struct {
		UpdateProduct graphQLErrorResult `json:"updateProduct"`
	}

	_, err := c.graphQL(ctx, mutation, map[string]any{
		"input": map[string]any{
			"id":      productID,
			"enabled": false,
		},
	}, &response)
	return err
}

func (c *Client) ensureAuthenticated(ctx context.Context) error {
	if strings.TrimSpace(c.cfg.Token) != "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loggedIn {
		return nil
	}

	if strings.TrimSpace(c.cfg.Username) == "" || strings.TrimSpace(c.cfg.Password) == "" {
		return fmt.Errorf("vendure credentials are not configured")
	}

	mutation := `
mutation Login($username: String!, $password: String!) {
  login(username: $username, password: $password) {
    ... on CurrentUser {
      id
      identifier
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`

	var response struct {
		Login graphQLErrorResult `json:"login"`
	}

	if _, err := c.graphQL(ctx, mutation, map[string]any{
		"username": c.cfg.Username,
		"password": c.cfg.Password,
	}, &response); err != nil {
		return err
	}

	c.loggedIn = true
	return nil
}

func (c *Client) createProduct(ctx context.Context, payload ProductPayload, assetID string) (string, error) {
	mutation := `
mutation CreateProduct($input: CreateProductInput!) {
  createProduct(input: $input) {
    ... on Product {
      id
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`

	input := map[string]any{
		"enabled": true,
		"translations": []map[string]any{
			{
				"languageCode": c.cfg.LanguageCode,
				"name":         payload.Name,
				"slug":         payload.Slug,
				"description":  payload.Description,
			},
		},
	}

	if assetID != "" {
		input["featuredAssetId"] = assetID
		input["assetIds"] = []string{assetID}
	}

	var response struct {
		CreateProduct graphQLNode `json:"createProduct"`
	}

	if _, err := c.graphQL(ctx, mutation, map[string]any{"input": input}, &response); err != nil {
		return "", err
	}

	return response.CreateProduct.ID, nil
}

func (c *Client) createVariant(ctx context.Context, payload ProductPayload, productID string, assetID string) (string, error) {
	mutation := `
mutation CreateProductVariants($input: [CreateProductVariantInput!]!) {
  createProductVariants(input: $input) {
    ... on ProductVariant {
      id
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`

	input := map[string]any{
		"productId": productID,
		"enabled":   true,
		"translations": []map[string]any{
			{
				"languageCode": c.cfg.LanguageCode,
				"name":         payload.Name,
			},
		},
		"sku":         payload.SKU,
		"stockOnHand": payload.DefaultStock,
		"prices": []map[string]any{
			{
				"currencyCode": payload.CurrencyCode,
				"price":        payload.ConsumerPrice,
			},
		},
		"customFields": map[string]any{
			"salesUnit": payload.SalesUnit,
			"bPrice":    payload.BusinessPrice,
		},
	}

	if assetID != "" {
		input["featuredAssetId"] = assetID
		input["assetIds"] = []string{assetID}
	}

	var response struct {
		CreateProductVariants []graphQLNode `json:"createProductVariants"`
	}

	if _, err := c.graphQL(ctx, mutation, map[string]any{"input": []map[string]any{input}}, &response); err != nil {
		return "", err
	}

	if len(response.CreateProductVariants) == 0 {
		return "", fmt.Errorf("vendure returned empty variant list")
	}

	return response.CreateProductVariants[0].ID, nil
}

func (c *Client) updateProduct(ctx context.Context, payload ProductPayload, assetID string) error {
	mutation := `
mutation UpdateProduct($input: UpdateProductInput!) {
  updateProduct(input: $input) {
    ... on Product {
      id
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`

	input := map[string]any{
		"id":      payload.VendureProduct,
		"enabled": true,
		"translations": []map[string]any{
			{
				"languageCode": c.cfg.LanguageCode,
				"name":         payload.Name,
				"slug":         payload.Slug,
				"description":  payload.Description,
			},
		},
	}

	if assetID != "" {
		input["featuredAssetId"] = assetID
		input["assetIds"] = []string{assetID}
	}

	var response struct {
		UpdateProduct graphQLNode `json:"updateProduct"`
	}

	_, err := c.graphQL(ctx, mutation, map[string]any{"input": input}, &response)
	return err
}

func (c *Client) updateVariant(ctx context.Context, payload ProductPayload, assetID string) error {
	mutation := `
mutation UpdateProductVariant($input: UpdateProductVariantInput!) {
  updateProductVariant(input: $input) {
    ... on ProductVariant {
      id
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`

	input := map[string]any{
		"id":          payload.VendureVariant,
		"enabled":     true,
		"sku":         payload.SKU,
		"stockOnHand": payload.DefaultStock,
		"translations": []map[string]any{
			{
				"languageCode": c.cfg.LanguageCode,
				"name":         payload.Name,
			},
		},
		"prices": []map[string]any{
			{
				"currencyCode": payload.CurrencyCode,
				"price":        payload.ConsumerPrice,
			},
		},
		"customFields": map[string]any{
			"salesUnit": payload.SalesUnit,
			"bPrice":    payload.BusinessPrice,
		},
	}

	if assetID != "" {
		input["featuredAssetId"] = assetID
		input["assetIds"] = []string{assetID}
	}

	var response struct {
		UpdateProductVariant graphQLNode `json:"updateProductVariant"`
	}

	_, err := c.graphQL(ctx, mutation, map[string]any{"input": input}, &response)
	return err
}

func (c *Client) uploadAsset(ctx context.Context, fileURL string, filename string) (string, error) {
	body, contentType, err := c.buildUploadBody(ctx, fileURL, filename)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", contentType)
	c.attachHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vendure asset upload failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded struct {
		Data struct {
			CreateAssets []graphQLNode `json:"createAssets"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}

	if len(decoded.Errors) > 0 {
		return "", fmt.Errorf("vendure asset upload error: %s", decoded.Errors[0].Message)
	}

	if len(decoded.Data.CreateAssets) == 0 {
		return "", fmt.Errorf("vendure returned no assets")
	}

	return decoded.Data.CreateAssets[0].ID, nil
}

func (c *Client) buildUploadBody(ctx context.Context, fileURL string, filename string) (*bytes.Buffer, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, "", fmt.Errorf("download asset failed with status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if strings.TrimSpace(filename) == "" {
		parsed, err := url.Parse(fileURL)
		if err == nil {
			filename = filepath.Base(parsed.Path)
		}
	}

	if strings.TrimSpace(filename) == "" {
		filename = "asset-upload"
	}

	operations := map[string]any{
		"query": `mutation CreateAssets($input: [CreateAssetInput!]!) {
  createAssets(input: $input) {
    ... on Asset {
      id
      name
      fileSize
    }
    ... on ErrorResult {
      errorCode
      message
    }
  }
}`,
		"variables": map[string]any{
			"input": []map[string]any{
				{
					"file": nil,
					"tags": c.cfg.AssetTags,
				},
			},
		},
	}

	mapping := map[string][]string{
		"0": {"variables.input.0.file"},
	}

	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)

	operationsBody, err := json.Marshal(operations)
	if err != nil {
		return nil, "", err
	}

	mapBody, err := json.Marshal(mapping)
	if err != nil {
		return nil, "", err
	}

	if err := writer.WriteField("operations", string(operationsBody)); err != nil {
		return nil, "", err
	}

	if err := writer.WriteField("map", string(mapBody)); err != nil {
		return nil, "", err
	}

	part, err := writer.CreateFormFile("0", filename)
	if err != nil {
		return nil, "", err
	}

	if _, err := part.Write(content); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return buffer, writer.FormDataContentType(), nil
}

func (c *Client) graphQL(ctx context.Context, query string, variables map[string]any, target any) ([]graphQLError, error) {
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	c.attachHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vendure request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}

	if len(envelope.Errors) > 0 {
		return envelope.Errors, fmt.Errorf("vendure graphql error: %s", envelope.Errors[0].Message)
	}

	if target != nil {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (c *Client) attachHeaders(req *http.Request) {
	if strings.TrimSpace(c.cfg.Token) != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}

	if strings.TrimSpace(c.cfg.ChannelToken) != "" {
		req.Header.Set("vendure-token", c.cfg.ChannelToken)
	}
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLErrorResult struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

type graphQLNode struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
