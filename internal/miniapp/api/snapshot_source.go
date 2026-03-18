package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"mrtang-pim/internal/miniapp/model"
)

type SnapshotSource struct {
	homepagePath  string
	categoryPath  string
	productPath   string
	cartOrderPath string

	mu     sync.RWMutex
	loaded bool
	data   model.Dataset
}

func NewSnapshotSource(homepagePath string, categoryPath string, productPath string, cartOrderPath string) *SnapshotSource {
	return &SnapshotSource{
		homepagePath:  homepagePath,
		categoryPath:  categoryPath,
		productPath:   productPath,
		cartOrderPath: cartOrderPath,
	}
}

func (s *SnapshotSource) FetchDataset(ctx context.Context) (*model.Dataset, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	if s.loaded {
		data := s.data
		s.mu.RUnlock()
		return &data, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded {
		data := s.data
		return &data, nil
	}

	dataset, err := s.loadHomepageSnapshot()
	if err != nil {
		return nil, err
	}
	if err := s.mergeCategorySnapshot(&dataset); err != nil {
		return nil, err
	}
	if err := s.mergeProductSnapshot(&dataset); err != nil {
		return nil, err
	}
	if err := s.mergeCartOrderSnapshot(&dataset); err != nil {
		return nil, err
	}

	s.data = dataset
	s.loaded = true

	data := s.data
	return &data, nil
}

func (s *SnapshotSource) FetchTargetSyncDataset(ctx context.Context, entityType string, scopeKey string) (*model.Dataset, error) {
	return s.FetchDataset(ctx)
}

func (s *SnapshotSource) loadHomepageSnapshot() (model.Dataset, error) {
	info, err := os.Stat(s.homepagePath)
	if err != nil {
		return model.Dataset{}, fmt.Errorf("stat miniapp homepage snapshot: %w", err)
	}

	if !info.IsDir() {
		return loadDatasetFile(s.homepagePath, "homepage")
	}

	return loadHomepageSnapshotDir(s.homepagePath)
}

func (s *SnapshotSource) mergeCategorySnapshot(dataset *model.Dataset) error {
	if dataset == nil || s.categoryPath == "" {
		return nil
	}

	info, err := os.Stat(s.categoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat miniapp category snapshot: %w", err)
	}

	var categoryDataset model.Dataset
	if info.IsDir() {
		categoryDataset, err = loadCategorySnapshotDir(s.categoryPath)
	} else {
		categoryDataset, err = loadDatasetFile(s.categoryPath, "category")
	}
	if err != nil {
		return err
	}

	dataset.Contracts = append(dataset.Contracts, categoryDataset.Contracts...)
	if len(categoryDataset.Meta.Notes) > 0 {
		dataset.Meta.Notes = append(dataset.Meta.Notes, categoryDataset.Meta.Notes...)
	}
	if len(categoryDataset.CategoryPage.Tree) > 0 || len(categoryDataset.CategoryPage.Sections) > 0 {
		dataset.CategoryPage = categoryDataset.CategoryPage
	}

	return nil
}

func (s *SnapshotSource) mergeProductSnapshot(dataset *model.Dataset) error {
	if dataset == nil || s.productPath == "" {
		return nil
	}

	info, err := os.Stat(s.productPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat miniapp product snapshot: %w", err)
	}

	var productDataset model.Dataset
	if info.IsDir() {
		productDataset, err = loadProductSnapshotDir(s.productPath)
	} else {
		productDataset, err = loadDatasetFile(s.productPath, "product")
	}
	if err != nil {
		return err
	}

	dataset.Contracts = append(dataset.Contracts, productDataset.Contracts...)
	if len(productDataset.Meta.Notes) > 0 {
		dataset.Meta.Notes = append(dataset.Meta.Notes, productDataset.Meta.Notes...)
	}
	if len(productDataset.ProductPage.Products) > 0 {
		dataset.ProductPage = productDataset.ProductPage
	}

	return nil
}

func (s *SnapshotSource) mergeCartOrderSnapshot(dataset *model.Dataset) error {
	if dataset == nil || s.cartOrderPath == "" {
		return nil
	}

	info, err := os.Stat(s.cartOrderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat miniapp cart-order snapshot: %w", err)
	}

	var cartOrderDataset model.Dataset
	if info.IsDir() {
		cartOrderDataset, err = loadCartOrderSnapshotDir(s.cartOrderPath)
	} else {
		cartOrderDataset, err = loadDatasetFile(s.cartOrderPath, "cart-order")
	}
	if err != nil {
		return err
	}

	dataset.Contracts = append(dataset.Contracts, cartOrderDataset.Contracts...)
	if len(cartOrderDataset.Meta.Notes) > 0 {
		dataset.Meta.Notes = append(dataset.Meta.Notes, cartOrderDataset.Meta.Notes...)
	}
	dataset.CartOrder = cartOrderDataset.CartOrder

	return nil
}

func loadDatasetFile(path string, label string) (model.Dataset, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return model.Dataset{}, fmt.Errorf("read miniapp %s snapshot: %w", label, err)
	}

	var dataset model.Dataset
	if err := json.Unmarshal(body, &dataset); err != nil {
		return model.Dataset{}, fmt.Errorf("decode miniapp %s snapshot: %w", label, err)
	}

	return dataset, nil
}

func loadHomepageSnapshotDir(root string) (model.Dataset, error) {
	var dataset model.Dataset

	if err := readJSONFile(filepath.Join(root, "meta.json"), &dataset.Meta, "homepage meta"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "contracts.json"), &dataset.Contracts, "homepage contracts"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "bootstrap.json"), &dataset.Homepage.Bootstrap, "homepage bootstrap"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "settings.json"), &dataset.Homepage.Settings, "homepage settings"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "template.json"), &dataset.Homepage.Template, "homepage template"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "category-tabs.json"), &dataset.Homepage.CategoryTabs, "homepage category tabs"); err != nil {
		return model.Dataset{}, err
	}

	sections, err := readSectionFiles[model.HomepageSection](filepath.Join(root, "sections"), "homepage sections")
	if err != nil {
		return model.Dataset{}, err
	}
	dataset.Homepage.Sections = sections

	return dataset, nil
}

func loadCategorySnapshotDir(root string) (model.Dataset, error) {
	var dataset model.Dataset

	if err := readJSONFile(filepath.Join(root, "meta.json"), &dataset.Meta, "category meta"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "contracts.json"), &dataset.Contracts, "category contracts"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "context.json"), &dataset.CategoryPage.Context, "category context"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "tree.json"), &dataset.CategoryPage.Tree, "category tree"); err != nil {
		return model.Dataset{}, err
	}

	sections, err := readSectionFiles[model.CategorySection](filepath.Join(root, "sections"), "category sections")
	if err != nil {
		return model.Dataset{}, err
	}
	dataset.CategoryPage.Sections = sections

	return dataset, nil
}

func loadProductSnapshotDir(root string) (model.Dataset, error) {
	var dataset model.Dataset

	if err := readJSONFile(filepath.Join(root, "meta.json"), &dataset.Meta, "product meta"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "contracts.json"), &dataset.Contracts, "product contracts"); err != nil {
		return model.Dataset{}, err
	}

	products, err := readSectionFiles[model.ProductPage](filepath.Join(root, "products"), "product pages")
	if err != nil {
		return model.Dataset{}, err
	}
	dataset.ProductPage.Products = products

	return dataset, nil
}

func loadCartOrderSnapshotDir(root string) (model.Dataset, error) {
	var dataset model.Dataset

	if err := readJSONFile(filepath.Join(root, "meta.json"), &dataset.Meta, "cart-order meta"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "contracts.json"), &dataset.Contracts, "cart-order contracts"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "cart.json"), &dataset.CartOrder.Cart, "cart-order cart"); err != nil {
		return model.Dataset{}, err
	}
	if err := readJSONFile(filepath.Join(root, "order.json"), &dataset.CartOrder.Order, "cart-order order"); err != nil {
		return model.Dataset{}, err
	}

	return dataset, nil
}

func readJSONFile(path string, target any, label string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read miniapp %s: %w", label, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode miniapp %s: %w", label, err)
	}
	return nil
}

func readSectionFiles[T any](root string, label string) ([]T, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read miniapp %s dir: %w", label, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	sections := make([]T, 0, len(names))
	for _, name := range names {
		var item T
		if err := readJSONFile(filepath.Join(root, name), &item, label+" file "+name); err != nil {
			return nil, err
		}
		sections = append(sections, item)
	}

	return sections, nil
}
