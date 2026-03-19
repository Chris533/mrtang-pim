package vendure

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mrtang-pim/internal/config"
)

type ProductPayload struct {
	Name              string
	Slug              string
	Description       string
	SKU               string
	CurrencyCode      string
	ConsumerPrice     int
	AssetURL          string
	AssetName         string
	AssetURLs         []string
	AssetNames        []string
	CEndAssetURL      string
	CEndAssetName     string
	BusinessPrice     int
	SupplierCode      string
	SupplierCostPrice int
	ConversionRate    float64
	SourceProductID   string
	SourceType        string
	TargetAudience    string
	DefaultStock      int
	SalesUnit         string
	VendureProduct    string
	VendureVariant    string
	NeedColdChain     bool
}

type SyncResult struct {
	ProductID string
	VariantID string
	AssetID   string
}

type AssetCleanupResult struct {
	TaggedAssets     int      `json:"taggedAssets"`
	ReferencedAssets int      `json:"referencedAssets"`
	DeletedAssets    int      `json:"deletedAssets"`
	FailedAssets     int      `json:"failedAssets"`
	DeletedIDs       []string `json:"deletedIds,omitempty"`
	FailedIDs        []string `json:"failedIds,omitempty"`
}

type CollectionPayload struct {
	SourceCategoryKey   string
	SourceCategoryPath  string
	SourceCategoryLevel int
	Name                string
	Slug                string
	Description         string
	ParentCollectionID  string
}

type CollectionSyncResult struct {
	CollectionID string
	Created      bool
}

type MiniappAsset struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Preview string `json:"preview"`
	Source  string `json:"source"`
}

type MiniappCollectionBreadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type MiniappCollectionNode struct {
	ID                  string                        `json:"id"`
	Name                string                        `json:"name"`
	Slug                string                        `json:"slug"`
	ParentID            string                        `json:"parentId"`
	FeaturedAsset       *MiniappAsset                 `json:"featuredAsset,omitempty"`
	Breadcrumbs         []MiniappCollectionBreadcrumb `json:"breadcrumbs"`
	SourceCategoryKey   string                        `json:"sourceCategoryKey"`
	SourceCategoryPath  string                        `json:"sourceCategoryPath"`
	SourceCategoryLevel int                           `json:"sourceCategoryLevel"`
	Children            []MiniappCollectionNode       `json:"children"`
}

type MiniappUnitOption struct {
	UnitID    string  `json:"unitId"`
	UnitName  string  `json:"unitName"`
	Price     int     `json:"price"`
	BaseUnit  string  `json:"baseUnit"`
	Rate      float64 `json:"rate"`
	IsDefault bool    `json:"isDefault"`
	StockQty  int     `json:"stockQty"`
	StockText string  `json:"stockText"`
}

type MiniappOrderUnit struct {
	UnitID      string  `json:"unitId"`
	UnitName    string  `json:"unitName"`
	Rate        float64 `json:"rate"`
	IsBase      bool    `json:"isBase"`
	IsDefault   bool    `json:"isDefault"`
	AllowOrder  bool    `json:"allowOrder"`
	MinOrderQty int     `json:"minOrderQty"`
	MaxOrderQty int     `json:"maxOrderQty"`
}

type MiniappProductCard struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Slug              string              `json:"slug"`
	FeaturedAsset     *MiniappAsset       `json:"featuredAsset,omitempty"`
	CEndFeaturedAsset *MiniappAsset       `json:"cEndFeaturedAsset,omitempty"`
	TargetAudience    string              `json:"targetAudience"`
	DefaultPrice      int                 `json:"defaultPrice"`
	BusinessPrice     int                 `json:"businessPrice"`
	DefaultUnit       string              `json:"defaultUnit"`
	HasMultiUnit      bool                `json:"hasMultiUnit"`
	UnitOptions       []MiniappUnitOption `json:"unitOptions"`
}

type MiniappProductList struct {
	Items      []MiniappProductCard `json:"items"`
	TotalItems int                  `json:"totalItems"`
}

type MiniappProductVariant struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	SKU             string        `json:"sku"`
	FeaturedAsset   *MiniappAsset `json:"featuredAsset,omitempty"`
	Price           int           `json:"price"`
	StockOnHand     int           `json:"stockOnHand"`
	SalesUnit       string        `json:"salesUnit"`
	BPrice          int           `json:"bPrice"`
	ConversionRate  float64       `json:"conversionRate"`
	SourceProductID string        `json:"sourceProductId"`
	SourceType      string        `json:"sourceType"`
}

type MiniappProductDetail struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Slug              string                  `json:"slug"`
	Description       string                  `json:"description"`
	FeaturedAsset     *MiniappAsset           `json:"featuredAsset,omitempty"`
	CEndFeaturedAsset *MiniappAsset           `json:"cEndFeaturedAsset,omitempty"`
	Assets            []MiniappAsset          `json:"assets"`
	TargetAudience    string                  `json:"targetAudience"`
	Variants          []MiniappProductVariant `json:"variants"`
	DefaultVariant    *MiniappProductVariant  `json:"defaultVariant,omitempty"`
	DefaultUnit       string                  `json:"defaultUnit"`
	UnitOptions       []MiniappUnitOption     `json:"unitOptions"`
	OrderUnits        []MiniappOrderUnit      `json:"orderUnits"`
}

type collectionFilterArg struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type collectionFilter struct {
	Code string                `json:"code"`
	Args []collectionFilterArg `json:"args"`
}

type assetNode struct {
	ID string `json:"id"`
}

type assetListItem struct {
	ID string `json:"id"`
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
	productAssetIDs, primaryAssetID, err := c.uploadProductAssets(ctx, payload)
	if err != nil {
		return SyncResult{}, err
	}
	result.AssetID = primaryAssetID

	cEndAssetID := ""
	if strings.TrimSpace(payload.CEndAssetURL) != "" {
		if strings.TrimSpace(payload.CEndAssetURL) == strings.TrimSpace(payload.AssetURL) && result.AssetID != "" {
			cEndAssetID = result.AssetID
		} else {
			assetID, err := c.uploadAsset(ctx, payload.CEndAssetURL, payload.CEndAssetName)
			if err != nil {
				return SyncResult{}, err
			}
			cEndAssetID = assetID
		}
	}

	if strings.TrimSpace(payload.VendureProduct) == "" {
		productID, err := c.createProduct(ctx, payload, primaryAssetID, productAssetIDs, cEndAssetID)
		if err != nil {
			return SyncResult{}, err
		}
		result.ProductID = productID

		variantID, err := c.createVariant(ctx, payload, productID, primaryAssetID)
		if err != nil {
			return SyncResult{}, err
		}
		result.VariantID = variantID
		return result, nil
	}

	result.ProductID = payload.VendureProduct
	result.VariantID = payload.VendureVariant

	if err := c.updateProduct(ctx, payload, primaryAssetID, productAssetIDs, cEndAssetID); err != nil {
		return SyncResult{}, err
	}

	if strings.TrimSpace(payload.VendureVariant) != "" {
		if err := c.updateVariant(ctx, payload, result.AssetID); err != nil {
			return SyncResult{}, err
		}
	}

	return result, nil
}

func (c *Client) MiniappCollectionsTree(ctx context.Context) ([]MiniappCollectionNode, error) {
	query := `
query MiniappCollectionsTree {
  miniappCollectionsTree {
    id
    name
    slug
    parentId
    sourceCategoryKey
    sourceCategoryPath
    sourceCategoryLevel
    breadcrumbs { id name slug }
    featuredAsset { id name preview source }
    children {
      id
      name
      slug
      parentId
      sourceCategoryKey
      sourceCategoryPath
      sourceCategoryLevel
      breadcrumbs { id name slug }
      featuredAsset { id name preview source }
      children {
        id
        name
        slug
        parentId
        sourceCategoryKey
        sourceCategoryPath
        sourceCategoryLevel
        breadcrumbs { id name slug }
        featuredAsset { id name preview source }
        children {
          id
          name
          slug
          parentId
          sourceCategoryKey
          sourceCategoryPath
          sourceCategoryLevel
          breadcrumbs { id name slug }
          featuredAsset { id name preview source }
        }
      }
    }
  }
}`

	var response struct {
		MiniappCollectionsTree []MiniappCollectionNode `json:"miniappCollectionsTree"`
	}
	if _, err := c.shopGraphQL(ctx, query, map[string]any{}, &response); err != nil {
		return nil, err
	}
	return response.MiniappCollectionsTree, nil
}

func (c *Client) MiniappCollectionProducts(ctx context.Context, slug string, audience string, skip int, take int) (MiniappProductList, error) {
	query := `
query MiniappCollectionProducts($slug: String!, $audience: String!, $skip: Int!, $take: Int!) {
  miniappCollectionProducts(slug: $slug, audience: $audience, skip: $skip, take: $take) {
    totalItems
    items {
      id
      name
      slug
      targetAudience
      defaultPrice
      businessPrice
      defaultUnit
      hasMultiUnit
      featuredAsset { id name preview source }
      cEndFeaturedAsset { id name preview source }
      unitOptions {
        unitId
        unitName
        price
        baseUnit
        rate
        isDefault
        stockQty
        stockText
      }
    }
  }
}`

	var response struct {
		MiniappCollectionProducts MiniappProductList `json:"miniappCollectionProducts"`
	}
	_, err := c.shopGraphQL(ctx, query, map[string]any{
		"slug":     slug,
		"audience": audience,
		"skip":     skip,
		"take":     take,
	}, &response)
	return response.MiniappCollectionProducts, err
}

func (c *Client) MiniappProductDetail(ctx context.Context, slug string, audience string) (*MiniappProductDetail, error) {
	query := `
query MiniappProductDetail($slug: String!, $audience: String!) {
  miniappProductDetail(slug: $slug, audience: $audience) {
    id
    name
    slug
    description
    targetAudience
    defaultUnit
    featuredAsset { id name preview source }
    cEndFeaturedAsset { id name preview source }
    assets { id name preview source }
    defaultVariant {
      id
      name
      sku
      price
      stockOnHand
      salesUnit
      bPrice
      conversionRate
      sourceProductId
      sourceType
      featuredAsset { id name preview source }
    }
    variants {
      id
      name
      sku
      price
      stockOnHand
      salesUnit
      bPrice
      conversionRate
      sourceProductId
      sourceType
      featuredAsset { id name preview source }
    }
    unitOptions {
      unitId
      unitName
      price
      baseUnit
      rate
      isDefault
      stockQty
      stockText
    }
    orderUnits {
      unitId
      unitName
      rate
      isBase
      isDefault
      allowOrder
      minOrderQty
      maxOrderQty
    }
  }
}`

	var response struct {
		MiniappProductDetail *MiniappProductDetail `json:"miniappProductDetail"`
	}
	if _, err := c.shopGraphQL(ctx, query, map[string]any{
		"slug":     slug,
		"audience": audience,
	}, &response); err != nil {
		return nil, err
	}
	return response.MiniappProductDetail, nil
}

func (c *Client) uploadProductAssets(ctx context.Context, payload ProductPayload) ([]string, string, error) {
	urls := make([]string, 0, 1+len(payload.AssetURLs))
	names := make([]string, 0, 1+len(payload.AssetNames))

	if value := strings.TrimSpace(payload.AssetURL); value != "" {
		urls = append(urls, value)
		names = append(names, payload.AssetName)
	}
	for index, rawURL := range payload.AssetURLs {
		urls = append(urls, rawURL)
		name := ""
		if index < len(payload.AssetNames) {
			name = payload.AssetNames[index]
		}
		names = append(names, name)
	}

	assetIDs := make([]string, 0, len(urls))
	uploadedByURL := make(map[string]string, len(urls))
	primaryAssetID := ""

	for index, rawURL := range urls {
		value := strings.TrimSpace(rawURL)
		if value == "" {
			continue
		}
		if existing := strings.TrimSpace(uploadedByURL[value]); existing != "" {
			if index == 0 && primaryAssetID == "" {
				primaryAssetID = existing
			}
			assetIDs = append(assetIDs, existing)
			continue
		}
		name := ""
		if index < len(names) {
			name = names[index]
		}
		assetID, err := c.uploadAsset(ctx, value, name)
		if err != nil {
			return nil, "", err
		}
		uploadedByURL[value] = assetID
		assetIDs = append(assetIDs, assetID)
		if index == 0 && primaryAssetID == "" {
			primaryAssetID = assetID
		}
	}

	assetIDs = uniqueNonEmptyStrings(assetIDs)
	if primaryAssetID == "" && len(assetIDs) > 0 {
		primaryAssetID = assetIDs[0]
	}
	return assetIDs, primaryAssetID, nil
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

func (c *Client) CleanupOrphanedPIMAssets(ctx context.Context) (AssetCleanupResult, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return AssetCleanupResult{}, err
	}

	assets, err := c.listTaggedAssets(ctx)
	if err != nil {
		return AssetCleanupResult{}, err
	}
	referenced, err := c.referencedAssetIDs(ctx)
	if err != nil {
		return AssetCleanupResult{}, err
	}

	result := AssetCleanupResult{
		TaggedAssets:     len(assets),
		ReferencedAssets: len(referenced),
		DeletedIDs:       []string{},
		FailedIDs:        []string{},
	}

	orphanIDs := make([]string, 0, len(assets))
	for _, item := range assets {
		if _, ok := referenced[item.ID]; ok {
			continue
		}
		orphanIDs = append(orphanIDs, item.ID)
	}

	for _, chunk := range chunkStrings(orphanIDs, 20) {
		if len(chunk) == 0 {
			continue
		}
		if err := c.deleteAssets(ctx, chunk); err != nil {
			result.FailedAssets += len(chunk)
			result.FailedIDs = append(result.FailedIDs, chunk...)
			continue
		}
		result.DeletedAssets += len(chunk)
		result.DeletedIDs = append(result.DeletedIDs, chunk...)
	}

	return result, nil
}

func (c *Client) EnsureCollection(ctx context.Context, payload CollectionPayload) (CollectionSyncResult, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return CollectionSyncResult{}, err
	}
	if strings.TrimSpace(payload.SourceCategoryKey) == "" {
		return CollectionSyncResult{}, fmt.Errorf("source category key is required")
	}
	if strings.TrimSpace(payload.Name) == "" {
		return CollectionSyncResult{}, fmt.Errorf("collection name is required")
	}
	if strings.TrimSpace(payload.Slug) == "" {
		return CollectionSyncResult{}, fmt.Errorf("collection slug is required")
	}

	existing, err := c.findCollectionBySourceKey(ctx, payload.SourceCategoryKey)
	if err != nil {
		return CollectionSyncResult{}, err
	}
	if existing != nil {
		if err := c.updateCollection(ctx, existing.ID, payload); err != nil {
			return CollectionSyncResult{}, err
		}
		return CollectionSyncResult{CollectionID: existing.ID, Created: false}, nil
	}

	id, err := c.createCollection(ctx, payload)
	if err != nil {
		return CollectionSyncResult{}, err
	}
	return CollectionSyncResult{CollectionID: id, Created: true}, nil
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

func (c *Client) createProduct(ctx context.Context, payload ProductPayload, assetID string, assetIDs []string, cEndAssetID string) (string, error) {
	mutation := `
mutation CreateProduct($input: CreateProductInput!) {
  createProduct(input: $input) {
    id
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
	}
	if len(assetIDs) > 0 {
		input["assetIds"] = assetIDs
	}

	if customFields := c.buildProductCustomFields(payload, cEndAssetID); len(customFields) > 0 {
		input["customFields"] = customFields
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
    id
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
		"price":       payload.ConsumerPrice,
		"stockOnHand": payload.DefaultStock,
		"prices": []map[string]any{
			{
				"currencyCode": payload.CurrencyCode,
				"price":        payload.ConsumerPrice,
			},
		},
		"customFields": c.buildVariantCustomFields(payload),
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

func (c *Client) updateProduct(ctx context.Context, payload ProductPayload, assetID string, assetIDs []string, cEndAssetID string) error {
	mutation := `
mutation UpdateProduct($input: UpdateProductInput!) {
  updateProduct(input: $input) {
    id
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
	}
	if len(assetIDs) > 0 {
		input["assetIds"] = assetIDs
	}

	if customFields := c.buildProductCustomFields(payload, cEndAssetID); len(customFields) > 0 {
		input["customFields"] = customFields
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
    id
  }
}`

	input := map[string]any{
		"id":          payload.VendureVariant,
		"enabled":     true,
		"sku":         payload.SKU,
		"price":       payload.ConsumerPrice,
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
		"customFields": c.buildVariantCustomFields(payload),
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

type collectionNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Parent *struct {
		ID string `json:"id"`
	} `json:"parent"`
}

func (c *Client) findCollectionBySourceKey(ctx context.Context, sourceKey string) (*collectionNode, error) {
	query := `
query Collections($options: CollectionListOptions) {
  collections(options: $options) {
    items {
      id
      name
      slug
      parent { id }
    }
  }
}`

	var response struct {
		Collections struct {
			Items []collectionNode `json:"items"`
		} `json:"collections"`
	}

	_, err := c.graphQL(ctx, query, map[string]any{
		"options": map[string]any{
			"take": 1,
			"filter": map[string]any{
				"sourceCategoryKey": map[string]any{
					"eq": sourceKey,
				},
			},
		},
	}, &response)
	if err != nil {
		return nil, err
	}
	if len(response.Collections.Items) == 0 {
		return nil, nil
	}
	return &response.Collections.Items[0], nil
}

func (c *Client) createCollection(ctx context.Context, payload CollectionPayload) (string, error) {
	mutation := `
mutation CreateCollection($input: CreateCollectionInput!) {
  createCollection(input: $input) {
    id
  }
}`

	input := c.collectionInput(payload, true)
	var response struct {
		CreateCollection graphQLNode `json:"createCollection"`
	}
	if _, err := c.graphQL(ctx, mutation, map[string]any{"input": input}, &response); err != nil {
		return "", err
	}
	return response.CreateCollection.ID, nil
}

func (c *Client) updateCollection(ctx context.Context, collectionID string, payload CollectionPayload) error {
	mutation := `
mutation UpdateCollection($input: UpdateCollectionInput!) {
  updateCollection(input: $input) {
    id
  }
}`

	input := c.collectionInput(payload, false)
	input["id"] = collectionID

	var response struct {
		UpdateCollection graphQLNode `json:"updateCollection"`
	}
	_, err := c.graphQL(ctx, mutation, map[string]any{"input": input}, &response)
	return err
}

func (c *Client) EnsureProductInCollections(ctx context.Context, productID string, collectionIDs []string) error {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}
	for _, collectionID := range uniqueNonEmptyStrings(collectionIDs) {
		if err := c.ensureProductInCollection(ctx, collectionID, productID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) SyncProductCollectionsExact(ctx context.Context, productID string, desiredCollectionIDs []string, candidateCollectionIDs []string) error {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return err
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return fmt.Errorf("product id is required")
	}

	desired := uniqueNonEmptyStrings(desiredCollectionIDs)
	candidates := uniqueNonEmptyStrings(candidateCollectionIDs)
	current, err := c.productCollectionIDs(ctx, productID)
	if err != nil {
		return err
	}
	candidates = uniqueNonEmptyStrings(append(candidates, current...))
	desiredSet := make(map[string]struct{}, len(desired))
	for _, id := range desired {
		desiredSet[id] = struct{}{}
	}
	for _, id := range desired {
		if err := c.ensureProductCollectionMembership(ctx, id, productID, true); err != nil {
			return err
		}
	}
	for _, id := range candidates {
		if _, keep := desiredSet[id]; keep {
			continue
		}
		if err := c.ensureProductCollectionMembership(ctx, id, productID, false); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) productCollectionIDs(ctx context.Context, productID string) ([]string, error) {
	query := `
query ProductCollections($id: ID!) {
  product(id: $id) {
    id
    collections {
      id
    }
  }
}`

	var response struct {
		Product struct {
			ID          string `json:"id"`
			Collections []struct {
				ID string `json:"id"`
			} `json:"collections"`
		} `json:"product"`
	}

	if _, err := c.graphQL(ctx, query, map[string]any{"id": productID}, &response); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(response.Product.Collections))
	for _, collection := range response.Product.Collections {
		if id := strings.TrimSpace(collection.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return uniqueNonEmptyStrings(ids), nil
}

func (c *Client) ensureProductInCollection(ctx context.Context, collectionID string, productID string) error {
	return c.ensureProductCollectionMembership(ctx, collectionID, productID, true)
}

func (c *Client) ensureProductCollectionMembership(ctx context.Context, collectionID string, productID string, shouldExist bool) error {
	collectionID = strings.TrimSpace(collectionID)
	if collectionID == "" {
		return nil
	}

	filters, err := c.collectionFilters(ctx, collectionID)
	if err != nil {
		return err
	}
	if shouldExist {
		filters = upsertProductCollectionFilter(filters, productID)
	} else {
		filters = removeProductCollectionFilter(filters, productID)
	}

	mutation := `
mutation UpdateCollectionFilters($input: UpdateCollectionInput!) {
  updateCollection(input: $input) {
    id
  }
}`

	input := map[string]any{
		"id":             collectionID,
		"inheritFilters": false,
		"filters":        encodeCollectionFilters(filters),
	}

	var response struct {
		UpdateCollection graphQLNode `json:"updateCollection"`
	}

	_, err = c.graphQL(ctx, mutation, map[string]any{"input": input}, &response)
	return err
}

func (c *Client) collectionFilters(ctx context.Context, collectionID string) ([]collectionFilter, error) {
	query := `
query CollectionFilters($id: ID!) {
  collection(id: $id) {
    id
    filters {
      code
      args {
        name
        value
      }
    }
  }
}`

	var response struct {
		Collection struct {
			ID      string             `json:"id"`
			Filters []collectionFilter `json:"filters"`
		} `json:"collection"`
	}

	if _, err := c.graphQL(ctx, query, map[string]any{"id": collectionID}, &response); err != nil {
		return nil, err
	}
	return response.Collection.Filters, nil
}

func (c *Client) collectionInput(payload CollectionPayload, includeFilters bool) map[string]any {
	input := map[string]any{
		"isPrivate":      false,
		"inheritFilters": false,
		"translations": []map[string]any{
			{
				"languageCode": c.cfg.LanguageCode,
				"name":         payload.Name,
				"slug":         payload.Slug,
				"description":  payload.Description,
			},
		},
		"customFields": map[string]any{
			"sourceCategoryKey":   payload.SourceCategoryKey,
			"sourceCategoryPath":  payload.SourceCategoryPath,
			"sourceCategoryLevel": payload.SourceCategoryLevel,
		},
	}
	if strings.TrimSpace(payload.ParentCollectionID) != "" {
		input["parentId"] = payload.ParentCollectionID
	}
	if includeFilters {
		input["filters"] = []map[string]any{}
	}
	return input
}

func (c *Client) buildProductCustomFields(payload ProductPayload, cEndAssetID string) map[string]any {
	customFields := map[string]any{}

	if field := strings.TrimSpace(c.cfg.ProductTargetAudienceField); field != "" && strings.TrimSpace(payload.TargetAudience) != "" {
		customFields[field] = payload.TargetAudience
	}

	if field := strings.TrimSpace(c.cfg.ProductCEndAssetField); field != "" {
		if trimmedID := strings.TrimSpace(cEndAssetID); trimmedID != "" {
			relationField := relationInputFieldName(field)
			customFields[relationField] = trimmedID
		}
	}

	return customFields
}

func (c *Client) buildVariantCustomFields(payload ProductPayload) map[string]any {
	customFields := map[string]any{
		"salesUnit": payload.SalesUnit,
		"bPrice":    payload.BusinessPrice,
	}

	if field := strings.TrimSpace(c.cfg.VariantSupplierCodeField); field != "" && strings.TrimSpace(payload.SupplierCode) != "" {
		customFields[field] = payload.SupplierCode
	}
	if field := strings.TrimSpace(c.cfg.VariantSupplierCostField); field != "" && payload.SupplierCostPrice > 0 {
		customFields[field] = payload.SupplierCostPrice
	}
	if field := strings.TrimSpace(c.cfg.VariantConversionRateField); field != "" && payload.ConversionRate > 0 {
		customFields[field] = payload.ConversionRate
	}
	if field := strings.TrimSpace(c.cfg.VariantSourceProductField); field != "" && strings.TrimSpace(payload.SourceProductID) != "" {
		customFields[field] = payload.SourceProductID
	}
	if field := strings.TrimSpace(c.cfg.VariantSourceTypeField); field != "" && strings.TrimSpace(payload.SourceType) != "" {
		customFields[field] = payload.SourceType
	}

	return customFields
}

func (c *Client) uploadAsset(ctx context.Context, fileURL string, filename string) (string, error) {
	fileURL = strings.TrimSpace(fileURL)
	filename = strings.TrimSpace(filename)
	keyTag := vendureAssetKeyTag(fileURL)
	assetTags := appendVendureAssetTags(c.cfg.AssetTags, keyTag)
	if existingID, err := c.findExistingAssetByKey(ctx, assetTags, filename); err != nil {
		return "", err
	} else if existingID != "" {
		return existingID, nil
	}

	body, contentType, err := c.buildUploadBody(ctx, fileURL, filename, assetTags)
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
			CreateAssets []struct {
				Typename  string `json:"__typename"`
				ID        string `json:"id"`
				Name      string `json:"name"`
				ErrorCode string `json:"errorCode"`
				Message   string `json:"message"`
			} `json:"createAssets"`
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
	first := decoded.Data.CreateAssets[0]
	if strings.EqualFold(first.Typename, "ErrorResult") || strings.TrimSpace(first.ID) == "" {
		if strings.TrimSpace(first.Message) != "" {
			return "", fmt.Errorf("vendure asset upload error: %s", first.Message)
		}
		return "", fmt.Errorf("vendure asset upload returned empty asset id")
	}

	return first.ID, nil
}

func (c *Client) buildUploadBody(ctx context.Context, fileURL string, filename string, assetTags []string) (*bytes.Buffer, string, error) {
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

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))

	if strings.TrimSpace(filename) == "" {
		parsed, err := url.Parse(fileURL)
		if err == nil {
			filename = filepath.Base(parsed.Path)
		}
	}

	if strings.TrimSpace(filename) == "" {
		filename = "asset-upload"
	}

	contentType = detectUploadContentType(filename, contentType, content)

	operations := map[string]any{
		"query": `mutation CreateAssets($input: [CreateAssetInput!]!) {
  createAssets(input: $input) {
    __typename
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
					"tags": uniqueNonEmptyStrings(assetTags),
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

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "0", escapeMultipartFilename(filename)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
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

func (c *Client) findExistingAssetByKey(ctx context.Context, assetTags []string, filename string) (string, error) {
	tags := uniqueNonEmptyStrings(assetTags)
	if len(tags) == 0 {
		return "", nil
	}

	query := `
query Assets($options: AssetListOptions) {
  assets(options: $options) {
    items {
      id
      name
    }
  }
}`

	options := map[string]any{
		"take":         10,
		"tags":         tags,
		"tagsOperator": "AND",
		"sort": map[string]any{
			"createdAt": "DESC",
		},
	}
	if strings.TrimSpace(filename) != "" {
		options["filter"] = map[string]any{
			"name": map[string]any{
				"eq": strings.TrimSpace(filename),
			},
		}
	}

	var response struct {
		Assets struct {
			Items []assetListItem `json:"items"`
		} `json:"assets"`
	}
	if _, err := c.graphQL(ctx, query, map[string]any{"options": options}, &response); err != nil {
		return "", err
	}
	if len(response.Assets.Items) == 0 && strings.TrimSpace(filename) != "" {
		delete(options, "filter")
		if _, err := c.graphQL(ctx, query, map[string]any{"options": options}, &response); err != nil {
			return "", err
		}
	}
	if len(response.Assets.Items) == 0 {
		return "", nil
	}
	return strings.TrimSpace(response.Assets.Items[0].ID), nil
}

func (c *Client) listTaggedAssets(ctx context.Context) ([]assetListItem, error) {
	query := `
query Assets($options: AssetListOptions) {
  assets(options: $options) {
    items {
      id
      name
    }
    totalItems
  }
}`

	items := []assetListItem{}
	skip := 0
	for {
		var response struct {
			Assets struct {
				Items      []assetListItem `json:"items"`
				TotalItems int             `json:"totalItems"`
			} `json:"assets"`
		}
		_, err := c.graphQL(ctx, query, map[string]any{
			"options": map[string]any{
				"skip":         skip,
				"take":         100,
				"tags":         uniqueNonEmptyStrings(c.cfg.AssetTags),
				"tagsOperator": "AND",
				"sort": map[string]any{
					"createdAt": "DESC",
				},
			},
		}, &response)
		if err != nil {
			return nil, err
		}
		items = append(items, response.Assets.Items...)
		skip += len(response.Assets.Items)
		if skip >= response.Assets.TotalItems || len(response.Assets.Items) == 0 {
			break
		}
	}
	return items, nil
}

func (c *Client) referencedAssetIDs(ctx context.Context) (map[string]struct{}, error) {
	referenced := make(map[string]struct{})
	if err := c.collectReferencedProductAssets(ctx, referenced); err != nil {
		return nil, err
	}
	if err := c.collectReferencedVariantAssets(ctx, referenced); err != nil {
		return nil, err
	}
	if err := c.collectReferencedCollectionAssets(ctx, referenced); err != nil {
		return nil, err
	}
	return referenced, nil
}

func (c *Client) collectReferencedProductAssets(ctx context.Context, target map[string]struct{}) error {
	consumerField := ""
	if field := strings.TrimSpace(c.cfg.ProductCEndAssetField); field != "" {
		consumerField = fmt.Sprintf("\n      customFields {\n        pimConsumerAsset: %s { id }\n      }", field)
	}
	query := fmt.Sprintf(`
query Products($options: ProductListOptions) {
  products(options: $options) {
    items {
      id
      featuredAsset { id }
      assets { id }%s
    }
    totalItems
  }
}`, consumerField)

	skip := 0
	for {
		var response struct {
			Products struct {
				Items []struct {
					ID            string      `json:"id"`
					FeaturedAsset *assetNode  `json:"featuredAsset"`
					Assets        []assetNode `json:"assets"`
					CustomFields  struct {
						ConsumerAsset *assetNode `json:"pimConsumerAsset"`
					} `json:"customFields"`
				} `json:"items"`
				TotalItems int `json:"totalItems"`
			} `json:"products"`
		}
		_, err := c.graphQL(ctx, query, map[string]any{
			"options": map[string]any{"skip": skip, "take": 100},
		}, &response)
		if err != nil {
			return err
		}
		for _, item := range response.Products.Items {
			if item.FeaturedAsset != nil {
				target[item.FeaturedAsset.ID] = struct{}{}
			}
			for _, asset := range item.Assets {
				target[asset.ID] = struct{}{}
			}
			if item.CustomFields.ConsumerAsset != nil {
				target[item.CustomFields.ConsumerAsset.ID] = struct{}{}
			}
		}
		skip += len(response.Products.Items)
		if skip >= response.Products.TotalItems || len(response.Products.Items) == 0 {
			break
		}
	}
	return nil
}

func (c *Client) collectReferencedVariantAssets(ctx context.Context, target map[string]struct{}) error {
	query := `
query ProductVariants($options: ProductVariantListOptions) {
  productVariants(options: $options) {
    items {
      id
      featuredAsset { id }
      assets { id }
    }
    totalItems
  }
}`

	skip := 0
	for {
		var response struct {
			ProductVariants struct {
				Items []struct {
					ID            string      `json:"id"`
					FeaturedAsset *assetNode  `json:"featuredAsset"`
					Assets        []assetNode `json:"assets"`
				} `json:"items"`
				TotalItems int `json:"totalItems"`
			} `json:"productVariants"`
		}
		_, err := c.graphQL(ctx, query, map[string]any{
			"options": map[string]any{"skip": skip, "take": 100},
		}, &response)
		if err != nil {
			return err
		}
		for _, item := range response.ProductVariants.Items {
			if item.FeaturedAsset != nil {
				target[item.FeaturedAsset.ID] = struct{}{}
			}
			for _, asset := range item.Assets {
				target[asset.ID] = struct{}{}
			}
		}
		skip += len(response.ProductVariants.Items)
		if skip >= response.ProductVariants.TotalItems || len(response.ProductVariants.Items) == 0 {
			break
		}
	}
	return nil
}

func (c *Client) collectReferencedCollectionAssets(ctx context.Context, target map[string]struct{}) error {
	query := `
query Collections($options: CollectionListOptions) {
  collections(options: $options) {
    items {
      id
      featuredAsset { id }
      assets { id }
    }
    totalItems
  }
}`

	skip := 0
	for {
		var response struct {
			Collections struct {
				Items []struct {
					ID            string      `json:"id"`
					FeaturedAsset *assetNode  `json:"featuredAsset"`
					Assets        []assetNode `json:"assets"`
				} `json:"items"`
				TotalItems int `json:"totalItems"`
			} `json:"collections"`
		}
		_, err := c.graphQL(ctx, query, map[string]any{
			"options": map[string]any{"skip": skip, "take": 100},
		}, &response)
		if err != nil {
			return err
		}
		for _, item := range response.Collections.Items {
			if item.FeaturedAsset != nil {
				target[item.FeaturedAsset.ID] = struct{}{}
			}
			for _, asset := range item.Assets {
				target[asset.ID] = struct{}{}
			}
		}
		skip += len(response.Collections.Items)
		if skip >= response.Collections.TotalItems || len(response.Collections.Items) == 0 {
			break
		}
	}
	return nil
}

func (c *Client) deleteAssets(ctx context.Context, assetIDs []string) error {
	assetIDs = uniqueNonEmptyStrings(assetIDs)
	if len(assetIDs) == 0 {
		return nil
	}
	mutation := `
mutation DeleteAssets($input: DeleteAssetsInput!) {
  deleteAssets(input: $input) {
    result
    message
  }
}`
	var response struct {
		DeleteAssets struct {
			Result  string `json:"result"`
			Message string `json:"message"`
		} `json:"deleteAssets"`
	}
	if _, err := c.graphQL(ctx, mutation, map[string]any{
		"input": map[string]any{
			"assetIds": assetIDs,
			"force":    true,
		},
	}, &response); err != nil {
		return err
	}
	if result := strings.TrimSpace(response.DeleteAssets.Result); result != "" && !strings.EqualFold(result, "DELETED") {
		message := strings.TrimSpace(response.DeleteAssets.Message)
		if message == "" {
			message = result
		}
		return fmt.Errorf("delete assets failed: %s", message)
	}
	return nil
}

func vendureAssetKeyTag(fileURL string) string {
	value := strings.TrimSpace(fileURL)
	if value == "" {
		return ""
	}
	sum := sha1.Sum([]byte(value))
	return "pim-asset-key-" + hex.EncodeToString(sum[:10])
}

func appendVendureAssetTags(base []string, extras ...string) []string {
	tags := make([]string, 0, len(base)+len(extras))
	tags = append(tags, base...)
	tags = append(tags, extras...)
	return uniqueNonEmptyStrings(tags)
}

func chunkStrings(items []string, size int) [][]string {
	if size <= 0 || len(items) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

func detectUploadContentType(filename string, responseContentType string, content []byte) string {
	contentType := strings.TrimSpace(responseContentType)
	if contentType != "" {
		if idx := strings.Index(contentType, ";"); idx >= 0 {
			contentType = strings.TrimSpace(contentType[:idx])
		}
	}
	if isPermittedUploadContentType(contentType) {
		return contentType
	}

	if extType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); isPermittedUploadContentType(extType) {
		return extType
	}

	if sniffed := strings.TrimSpace(http.DetectContentType(content)); isPermittedUploadContentType(sniffed) {
		return sniffed
	}

	return "application/octet-stream"
}

func isPermittedUploadContentType(contentType string) bool {
	value := strings.TrimSpace(strings.ToLower(contentType))
	if value == "" || value == "application/octet-stream" {
		return false
	}
	return true
}

func escapeMultipartFilename(filename string) string {
	replacer := strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
	return replacer.Replace(filename)
}

func (c *Client) graphQL(ctx context.Context, query string, variables map[string]any, target any) ([]graphQLError, error) {
	return c.graphQLToEndpoint(ctx, c.cfg.Endpoint, query, variables, target)
}

func (c *Client) shopGraphQL(ctx context.Context, query string, variables map[string]any, target any) ([]graphQLError, error) {
	return c.graphQLToEndpoint(ctx, c.cfg.ShopEndpoint, query, variables, target)
}

func (c *Client) graphQLToEndpoint(ctx context.Context, endpoint string, query string, variables map[string]any, target any) ([]graphQLError, error) {
	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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

func upsertProductCollectionFilter(filters []collectionFilter, productID string) []collectionFilter {
	const filterCode = "product-id-filter"
	normalizedID := strings.TrimSpace(productID)
	if normalizedID == "" {
		return filters
	}

	for idx, filter := range filters {
		if !strings.EqualFold(strings.TrimSpace(filter.Code), filterCode) {
			continue
		}
		ids := parseConfigArgIDList(filter.Args, "productIds")
		if !containsString(ids, normalizedID) {
			ids = append(ids, normalizedID)
		}
		filters[idx].Args = []collectionFilterArg{
			{Name: "productIds", Value: encodeIDList(ids)},
			{Name: "combineWithAnd", Value: "false"},
		}
		return filters
	}

	return append(filters, collectionFilter{
		Code: filterCode,
		Args: []collectionFilterArg{
			{Name: "productIds", Value: encodeIDList([]string{normalizedID})},
			{Name: "combineWithAnd", Value: "false"},
		},
	})
}

func removeProductCollectionFilter(filters []collectionFilter, productID string) []collectionFilter {
	const filterCode = "product-id-filter"
	normalizedID := strings.TrimSpace(productID)
	if normalizedID == "" {
		return filters
	}

	result := make([]collectionFilter, 0, len(filters))
	for _, filter := range filters {
		if !strings.EqualFold(strings.TrimSpace(filter.Code), filterCode) {
			result = append(result, filter)
			continue
		}
		ids := parseConfigArgIDList(filter.Args, "productIds")
		next := make([]string, 0, len(ids))
		for _, id := range ids {
			if strings.EqualFold(strings.TrimSpace(id), normalizedID) {
				continue
			}
			next = append(next, id)
		}
		if len(next) == 0 {
			continue
		}
		filter.Args = []collectionFilterArg{
			{Name: "productIds", Value: encodeIDList(next)},
			{Name: "combineWithAnd", Value: "false"},
		}
		result = append(result, filter)
	}
	return result
}

func parseConfigArgIDList(args []collectionFilterArg, name string) []string {
	target := strings.TrimSpace(name)
	for _, arg := range args {
		if !strings.EqualFold(strings.TrimSpace(arg.Name), target) {
			continue
		}
		raw := strings.TrimSpace(arg.Value)
		if raw == "" {
			return nil
		}
		var items []string
		if err := json.Unmarshal([]byte(raw), &items); err == nil {
			return uniqueNonEmptyStrings(items)
		}
		return uniqueNonEmptyStrings(strings.Split(raw, ","))
	}
	return nil
}

func encodeCollectionFilters(filters []collectionFilter) []map[string]any {
	encoded := make([]map[string]any, 0, len(filters))
	for _, filter := range filters {
		args := make([]map[string]any, 0, len(filter.Args))
		for _, arg := range filter.Args {
			args = append(args, map[string]any{
				"name":  arg.Name,
				"value": arg.Value,
			})
		}
		encoded = append(encoded, map[string]any{
			"code":      filter.Code,
			"arguments": args,
		})
	}
	return encoded
}

func encodeIDList(ids []string) string {
	encoded, err := json.Marshal(uniqueNonEmptyStrings(ids))
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func uniqueNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func relationInputFieldName(field string) string {
	value := strings.TrimSpace(field)
	if value == "" {
		return value
	}
	if strings.HasSuffix(value, "Id") {
		return value
	}
	return value + "Id"
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
