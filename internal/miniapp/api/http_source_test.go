package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHTTPSourceFetchDataset(t *testing.T) {
	source := NewHTTPSource(HTTPSourceConfig{
		URL:                 "https://example.test/miniapp-dataset",
		AuthorizedAccountID: "account-123",
		UserAgent:           "ua-test",
		Timeout:             2 * time.Second,
	})
	source.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("Authorization"); got != "Bearer account-123" {
				t.Fatalf("unexpected authorization: %q", got)
			}
			if got := r.Header.Get("User-Agent"); got != "ua-test" {
				t.Fatalf("unexpected user-agent: %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"meta":{"source":"http"},"contracts":[],"homepage":{"bootstrap":{"allowSwitchBranch":false,"autoSwitchBranch":false,"appletCount":1,"contactsConfig":[],"bbAuthStatus":0,"cartSummary":[],"loginStatus":"approved","canOrder":true},"settings":{"corpName":"corp","shopName":"shop","currencySymbol":"￥","themeColor":[],"topBackgroundColor":"","topContentColor":"","pageName":"","showOnlineService":true,"showShoppingCart":true,"enableCoupon":true,"enableIntegral":true,"enableInvoice":true,"enablePrePayment":true,"enableGoodsFavorite":true,"pricePrecision":2,"qtyPrecision":2,"stockViewRange":2,"goodsListSetting":{"layoutType":1,"defaultSort":-9,"detailPageShowType":2,"listPageShowType":2,"showSkuSearchBtn":0},"goodsCategorySetting":{"layoutType":1,"defaultSort":-1,"maxCategoryLevel":3,"searchType":0,"categorySearchType":0,"categorySortType":0}},"template":{"businessId":"1","templateName":"t","pageName":"p","sharePageUrl":"","showOnlineService":true,"showShoppingCart":true,"templateType":1,"modules":[]},"categoryTabs":[],"sections":[]}}`,
				)),
				Request: r,
			}, nil
		}),
	}

	dataset, err := source.FetchDataset(context.Background())
	if err != nil {
		t.Fatalf("fetch dataset: %v", err)
	}

	if dataset.Meta.Source != "http" {
		t.Fatalf("unexpected source: %q", dataset.Meta.Source)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
