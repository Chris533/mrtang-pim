package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type FileConnector struct {
	path string
	code string
}

func NewFileConnector(path string, code string) *FileConnector {
	return &FileConnector{
		path: path,
		code: code,
	}
}

func (c *FileConnector) Fetch(ctx context.Context) ([]Product, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	body, err := os.ReadFile(c.path)
	if err != nil {
		return nil, fmt.Errorf("read supplier file: %w", err)
	}

	var products []Product
	if err := json.Unmarshal(body, &products); err != nil {
		return nil, fmt.Errorf("decode supplier file: %w", err)
	}

	for i := range products {
		if products[i].SupplierCode == "" {
			products[i].SupplierCode = c.code
		}

		if products[i].CurrencyCode == "" {
			products[i].CurrencyCode = "CNY"
		}
	}

	return products, nil
}
