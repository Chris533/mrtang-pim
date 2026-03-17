# mrtang-pim 启动说明

`mrtang-pim` 当前已经不是概念方案，而是一套可运行的 `Golang + PocketBase` PIM 服务。它负责承接供应商商品、图片处理、人工审核、Vendure 同步，以及小程序首页数据接入。

## 当前架构

项目分成两条主链路：

1. PIM 商品链路
   `Supplier -> PocketBase -> Image Processor -> Admin Review -> Vendure`
2. Miniapp 契约链路
   `Snapshot/HTTP Source -> miniapp importer/service -> API -> 后续入库`

当前目录重点如下：

```text
mrtang-pim/
├── cmd/pim/main.go
├── datasets/
│   ├── miniapp/
│   │   ├── category-page/
│   │   │   ├── context.json
│   │   │   ├── contracts.json
│   │   │   ├── meta.json
│   │   │   ├── tree.json
│   │   │   └── sections/
│   │   ├── product-page/
│   │   │   ├── contracts.json
│   │   │   ├── meta.json
│   │   │   └── products/
│   │   ├── cart-order/
│   │   │   ├── cart.json
│   │   │   ├── contracts.json
│   │   │   ├── meta.json
│   │   │   └── order.json
│   │   └── homepage/
│   │       ├── bootstrap.json
│   │       ├── category-tabs.json
│   │       ├── contracts.json
│   │       ├── meta.json
│   │       ├── settings.json
│   │       ├── template.json
│   │       └── sections/
│   └── mock_supplier_products.json
├── docs/
│   ├── rr/
│   ├── rr.md
│   └── start.md
├── internal/
│   ├── config/
│   ├── image/
│   ├── miniapp/
│   │   ├── api/
│   │   ├── importer/
│   │   ├── model/
│   │   ├── repository/
│   │   └── service/
│   ├── pim/
│   ├── supplier/
│   └── vendure/
└── migrations/
```

## PIM 商品流程

1. `Harvest`
   从供应商连接器拉取商品，写入 `supplier_products`。
2. `Process`
   处理图片并更新 `processed_image`、`sync_status`。
3. `Review`
   运营在 PocketBase Admin UI 中确认标题、分类、文案和售价。
4. `Sync`
   将 `approved` 记录同步到 Vendure，并回写 Vendure ID。
5. `Offline`
   上游缺失 SKU 会标记为 `offline`，并尝试下架 Vendure 商品。

状态流转：

`pending -> ai_processing -> ready -> approved -> synced / offline / error`

## Miniapp 首页流程

Miniapp 模块已经拆成明确分层：

- `internal/miniapp/api`
  数据源层。支持 `snapshot` 和 `http` 两种 source。
- `internal/miniapp/importer`
  把上游 `Dataset` 整理成标准首页模型。
- `internal/miniapp/model`
  首页领域模型和契约模型。
- `internal/miniapp/repository`
  预留入库接口，后续用于保存首页数据。
- `internal/miniapp/service`
  编排 `load -> transform -> expose`。

当前支持两种数据源模式：

- `MINIAPP_SOURCE_MODE=snapshot`
  从 `MINIAPP_HOMEPAGE_SNAPSHOT` 读取本地脱敏快照。
- `MINIAPP_SOURCE_MODE=http`
  从 `MINIAPP_SOURCE_URL` 拉取标准化后的首页 `Dataset` JSON。

当使用 `http` source 时，会自动带上：

- `Authorization: Bearer <MINIAPP_AUTH_ACCOUNT_ID>`
- `User-Agent: <MINIAPP_USER_AGENT>`

默认 `User-Agent` 是较新的 iPhone 微信小程序模板。

如果要快速理解“源站 API、抓包归档、dataset 和本地接口”之间的关系，直接看 [source-api.md](./source-api.md)。
如果要直接对接结算页，优先看 [checkout-api.md](./checkout-api.md)。
如果要直接看目标站同步模块、运行记录和变更详情，见 [target-sync.md](./target-sync.md)。
如果要直接操作 source 商品审核、图片处理和桥接同步，见 [source-review-workbench.md](./source-review-workbench.md)。
如果要理解后台模块结构和页面入口，见 [mrtang-admin.md](./mrtang-admin.md)。

## 环境变量

至少关注这些配置：

- `PIM_HTTP_ADDR=127.0.0.1:26228`
- `PIM_PUBLIC_URL`
- `PIM_API_KEY`
- `PIM_SUPERUSER_EMAIL`
- `PIM_SUPERUSER_PASSWORD`
- `MINIAPP_SOURCE_MODE`
- `MINIAPP_SOURCE_URL`
- `MINIAPP_SOURCE_TIMEOUT`
- `MINIAPP_HOMEPAGE_SNAPSHOT=./datasets/miniapp/homepage`
- `MINIAPP_CATEGORY_SNAPSHOT=./datasets/miniapp/category-page`
- `MINIAPP_PRODUCT_SNAPSHOT=./datasets/miniapp/product-page`
- `MINIAPP_CART_ORDER_SNAPSHOT=./datasets/miniapp/cart-order`
- `MINIAPP_AUTH_ACCOUNT_ID`
- `MINIAPP_USER_AGENT`
- `SUPPLIER_CONNECTOR=file`
- `SUPPLIER_FILE=./datasets/mock_supplier_products.json`
- `IMAGE_PROCESSOR=mock|webhook`
- `VENDURE_ADMIN_API`
- `VENDURE_ADMIN_TOKEN`

完整示例见项目根目录 `.env.example`。

## 启动方式

```bash
cd mrtang-pim
cp .env.example .env
go mod tidy
go run ./cmd/pim serve
```

默认地址：

- Admin UI: `http://127.0.0.1:26228/_/`
- Mrtang Admin: `http://127.0.0.1:26228/_/mrtang-admin`
- Target Sync: `http://127.0.0.1:26228/_/mrtang-admin/target-sync`
- Source Home: `http://127.0.0.1:26228/_/mrtang-admin/source`
- Source Products: `http://127.0.0.1:26228/_/mrtang-admin/source/products`
- Source Assets: `http://127.0.0.1:26228/_/mrtang-admin/source/assets`
- Source Logs: `http://127.0.0.1:26228/_/mrtang-admin/source/logs`
- Procurement: `http://127.0.0.1:26228/_/mrtang-admin/procurement`
- Source Review Workbench: `http://127.0.0.1:26228/_/source-review-workbench`（兼容保留）
- Procurement Workbench: `http://127.0.0.1:26228/_/procurement-workbench`（兼容保留）
- Health: `http://127.0.0.1:26228/api/pim/healthz`

## Source Workbench 状态流

推荐操作顺序：

1. 先在 `/_/mrtang-admin/target-sync` 同步目标站分类、商品规格和图片。
2. 再进入 `/_/mrtang-admin/source/products` 和 `/_/mrtang-admin/source/assets` 处理待审核商品与待处理图片。
3. 商品桥接到 `supplier_products` 后，再进入既有 backend 同步链。

商品审核状态：

- `imported -> approved -> promoted`
- 允许人工转为 `rejected`

图片处理状态：

- `pending -> processing -> processed`
- `failed` 可重试并重新进入处理链

## 当前可用接口

PIM：

- `POST /api/pim/harvest`
- `POST /api/pim/process`
- `POST /api/pim/sync`
- `GET /api/pim/procurement/capabilities`
- `GET /api/pim/procurement/workbench-summary`
- `POST /api/pim/procurement/summary`
- `POST /api/pim/procurement/export`
- `POST /api/pim/procurement/submit`
- `POST /api/pim/procurement/orders`
- `GET /api/pim/procurement/orders`
- `GET /api/pim/procurement/order?id=<id>`
- `POST /api/pim/procurement/order/review?id=<id>`
- `POST /api/pim/procurement/order/export?id=<id>`
- `POST /api/pim/procurement/order/status?id=<id>`

Miniapp：

- `GET /api/miniapp/contracts/homepage`
- `GET /api/miniapp/contracts/category-page`
- `GET /api/miniapp/homepage`
- `GET /api/miniapp/homepage/bootstrap`
- `GET /api/miniapp/homepage/settings`
- `GET /api/miniapp/homepage/template`
- `GET /api/miniapp/homepage/categories`
- `GET /api/miniapp/homepage/sections`
- `GET /api/miniapp/homepage/section?id=<section-id>`
- `GET /api/miniapp/category-page`
- `GET /api/miniapp/category-page/context`
- `GET /api/miniapp/category-page/tree`
- `GET /api/miniapp/category-page/sections`
- `GET /api/miniapp/category-page/section?id=<section-id>`
- `GET /api/miniapp/contracts/product-page`
- `GET /api/miniapp/contracts/cart-order`
- `GET /api/miniapp/product-page`
- `GET /api/miniapp/product-page/product?id=<spuId>_<skuId>`
- `GET /api/miniapp/product-page/detail?id=<spuId>_<skuId>`
- `GET /api/miniapp/product-page/pricing?id=<spuId>_<skuId>`
- `GET /api/miniapp/product-page/package?id=<spuId>_<skuId>`
- `GET /api/miniapp/product-page/context?id=<spuId>_<skuId>`
- `GET /api/miniapp/product-page/coverage`
- `GET /api/miniapp/product-page/coverage-summary`
- `GET /api/miniapp/cart-order`
- `GET /api/miniapp/cart-order/cart`
- `GET /api/miniapp/cart-order/order`
- `POST /api/miniapp/cart-order/cart/add`
- `POST /api/miniapp/cart-order/cart/change-num`
- `GET /api/miniapp/cart-order/cart/list`
- `GET /api/miniapp/cart-order/cart/list-summary`
- `GET /api/miniapp/cart-order/cart/detail`
- `GET /api/miniapp/cart-order/cart/detail-summary`
- `POST /api/miniapp/cart-order/cart/settle`
- `GET /api/miniapp/cart-order/order/default-delivery`
- `GET /api/miniapp/cart-order/order/default-delivery-summary`
- `GET /api/miniapp/cart-order/order/deliveries`
- `GET /api/miniapp/cart-order/order/deliveries-summary`
- `POST /api/miniapp/cart-order/order/address/analyse`
- `POST /api/miniapp/cart-order/order/address/add`
- `GET /api/miniapp/cart-order/order/freight-cost?scenario=preview|selected_delivery`
- `GET /api/miniapp/cart-order/order/freight-summary`
- `POST /api/miniapp/cart-order/order/submit`
- `GET /api/miniapp/cart-order/order/submit-summary`
- `GET /api/miniapp/cart-order/checkout-summary`

## 下一步

当前还缺两块正式能力：

1. `internal/miniapp/repository` 的真实入库实现
2. 真实上游 miniapp source 返回结果到本地数据库的同步任务

