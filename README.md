# mrtang-pim

基于 `Golang + PocketBase` 的商品信息中台，用来把供应商原始商品数据采集、清洗、图片处理、人工审核与 Vendure 同步流程拆开。

## 已实现范围

- PocketBase 单体服务，内置 Admin UI
- `supplier_products` / `category_mappings` 集合迁移
- 供应商采集入口，默认从本地 JSON 文件模拟上游供应商
- 图片处理流水线
- 默认 `mock` 图片处理器，可直接生成可预览 SVG
- 可切换 webhook 图片处理器，对接本地 OCR / IOPaint / 重绘服务
- PocketBase 审核流
- `pending -> ai_processing -> ready -> approved -> synced / offline / error`
- Vendure Admin API 同步
- 自动创建商品、变体、资产上传
- 已存在 Vendure ID 时自动走更新
- 定时采集 / 处理 / 同步任务
- 缺失 SKU 自动标记 `offline`，并尝试下架 Vendure 商品
- 自定义 HTTP 触发接口
- 脱敏后的“小程序首页/分类/商品页/购物车下单契约”接口与离线 snapshot dataset

## 目录

```text
mrtang-pim/
├── cmd/pim/main.go
├── datasets/
│   ├── miniapp/
│   │   ├── homepage/
│   │   ├── category-page/
│   │   ├── product-page/
│   │   └── cart-order/
│   └── mock_supplier_products.json
├── internal/
│   ├── config
│   ├── image
│   ├── miniapp/
│   │   ├── api
│   │   ├── importer
│   │   ├── model
│   │   ├── repository
│   │   └── service
│   ├── pim
│   ├── supplier
│   └── vendure
├── migrations/
└── docs/start.md
```

## 核心流程

1. `Harvest`
   从供应商接口拉取原始商品，写入 `supplier_products`。
2. `Process`
   下载或生成处理图，写回 PocketBase `processed_image`。
3. `Review`
   运营在 PocketBase Admin UI 中检查记录，把 `sync_status` 从 `ready` 改成 `approved`。
4. `Sync`
   同步到 Vendure，写回 `vendure_product_id`、`vendure_variant_id`，并把状态改成 `synced`。

## 开发启动

```bash
cd mrtang-pim
cp .env.example .env
go mod tidy
go run ./cmd/pim serve
```

默认服务地址:

- PocketBase Admin UI: `http://127.0.0.1:26228/_/`
- Mrtang Admin: `http://127.0.0.1:26228/_/mrtang-admin`
- Procurement Workbench: `http://127.0.0.1:26228/_/procurement-workbench`
- 健康检查: `http://127.0.0.1:26228/api/pim/healthz`

本地生成目录:

- `.cache/` 是 Go 编译缓存，可随时删除
- `pb_data/` 是 PocketBase 本地运行数据目录，重新初始化或不需要保留本地数据时可删除

## 脱敏 Miniapp 接口

`docs/rr/index`、`docs/rr/categories`、`docs/rr/product` 和 `docs/rr/cart-order` 中的小程序抓包样本已经被整理成脱敏 snapshot 目录:

- [`datasets/miniapp/homepage`](./datasets/miniapp/homepage)
- [`datasets/miniapp/category-page`](./datasets/miniapp/category-page)
- [`datasets/miniapp/product-page`](./datasets/miniapp/product-page)
- [`datasets/miniapp/cart-order`](./datasets/miniapp/cart-order)

这套数据不会重放第三方鉴权，也不保留任何敏感 token / openId / customerId。它的目标是:

- 为前端和客户端提供稳定的本地联调接口
- 保留首页相关接口之间的依赖关系
- 为后续正式授权接入保留可插拔的数据源边界

如果你需要直接看“源站 API -> rr 样本 -> dataset -> 本地接口”的总览，见 [docs/source-api.md](./docs/source-api.md)。
如果你需要直接看 checkout 摘要接口和推荐字段，见 [docs/checkout-api.md](./docs/checkout-api.md)。

环境变量:

- `PIM_HTTP_ADDR=127.0.0.1:26228`
- `MINIAPP_SOURCE_MODE=snapshot|http`
- `MINIAPP_SOURCE_URL=...`
- `MINIAPP_SOURCE_TIMEOUT=20s`
- `MINIAPP_HOMEPAGE_SNAPSHOT=./datasets/miniapp/homepage`
- `MINIAPP_CATEGORY_SNAPSHOT=./datasets/miniapp/category-page`
- `MINIAPP_PRODUCT_SNAPSHOT=./datasets/miniapp/product-page`
- `MINIAPP_CART_ORDER_SNAPSHOT=./datasets/miniapp/cart-order`
- `MINIAPP_AUTH_ACCOUNT_ID=...`
- `MINIAPP_USER_AGENT=Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.53(0x18003537) NetType/WIFI Language/zh_CN miniProgram`

数据源切换:

- `MINIAPP_SOURCE_MODE=snapshot`
  读取本地 snapshot 目录，并组装 homepage/category-page/product-page/cart-order dataset
- `MINIAPP_SOURCE_MODE=http`
  从 `MINIAPP_SOURCE_URL` 拉取标准化后的 `Dataset` JSON

当使用 `http` source 时，请求会自动带上:

- `Authorization: Bearer <MINIAPP_AUTH_ACCOUNT_ID>`
- `User-Agent: <MINIAPP_USER_AGENT>`

脱敏 demo 覆盖:

- `customerId`、`customerName`、`phone`、`openId`、`addressId`、`cartIdList`、`deliveryMethodId`、`dueMoney` 都会从前序 cart/address/order 样本自动串联
- `datasets/miniapp/**` 仍保持稳定的脱敏样本，不因为本地配置被改写

可用接口:

```bash
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/contracts/homepage
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/homepage
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/homepage/bootstrap
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/homepage/settings
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/homepage/template
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/homepage/categories
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/homepage/sections
curl -H 'Authorization: Bearer your-account-id' 'http://127.0.0.1:26228/api/miniapp/homepage/section?id=new'
curl -H 'Authorization: Bearer your-account-id' 'http://127.0.0.1:26228/api/miniapp/homepage/section?id=hot'
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/contracts/category-page
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/category-page/tree
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/category-page/sections
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/contracts/product-page
curl -H 'Authorization: Bearer your-account-id' 'http://127.0.0.1:26228/api/miniapp/product-page/product?id=670168385396461568_670168388273754112'
curl -H 'Authorization: Bearer your-account-id' 'http://127.0.0.1:26228/api/miniapp/product-page/detail?id=670168385396461568_670168388273754112'
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/product-page/coverage
curl -H 'Authorization: Bearer your-account-id' 'http://127.0.0.1:26228/api/miniapp/product-page/coverage?priority=homepage_dual_unit'
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/product-page/coverage-summary
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/contracts/cart-order
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order
curl -X POST -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/cart/add
curl -X POST -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/cart/change-num
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/cart/list
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/cart/list-summary
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/cart/detail
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/cart/detail-summary
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/default-delivery-summary
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/deliveries-summary
curl -X POST -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/address/analyse
curl -X POST -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/address/add
curl -H 'Authorization: Bearer your-account-id' 'http://127.0.0.1:26228/api/miniapp/cart-order/order/freight-cost?scenario=selected_delivery'
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/freight-summary
curl -X POST -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/submit
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/order/submit-summary
curl -H 'Authorization: Bearer your-account-id' http://127.0.0.1:26228/api/miniapp/cart-order/checkout-summary
```

接口说明:

- `/api/miniapp/contracts/homepage`
  返回脱敏后的“原始接口清单”，包含方法、原始路径、你自己的本地路径、请求字段、响应字段，以及当前本地客户端配置。
- `/api/miniapp/homepage`
  返回首页聚合数据，适合前端一次性拉取。
- `/api/miniapp/homepage/section?id=new|hot`
  返回某个首页商品分组及其标准化商品列表。

鉴权说明:

- miniapp 契约层只保留 `Authorization: Bearer <authorized-account-id>` 这一种本地配置方式
- `MINIAPP_AUTH_ACCOUNT_ID` 为空时，`/api/miniapp/*` 默认公开
- 你提到的 IP 白名单视为“上游源站已经授予我们出口 IP 权限”，这里作为接入前提，不在本地接口层重复校验

`User-Agent` 说明:

- `MINIAPP_USER_AGENT` 用于保存你未来正式授权数据源客户端的默认 UA 配置
- 默认值已经切到“较新的 iPhone 微信小程序”模板，便于后续正式授权数据源接入时直接复用
- 当前会在 `/api/miniapp/contracts/homepage` 的 `clientConfig.userAgent` 中返回，方便前后端或后续 connector 统一读取

## 管理员初始化

如果设置了以下环境变量，首次迁移时会自动创建 PocketBase superuser:

- `MRTANG_PIM_ENCRYPTION_KEY`
- `PIM_SUPERUSER_EMAIL`
- `PIM_SUPERUSER_PASSWORD`

也可以手动创建:

```bash
go run ./cmd/pim superuser create admin@example.com change-me
```

## 自定义接口

如果设置了 `PIM_API_KEY`，以下接口需要请求头 `X-PIM-API-Key: <key>` 或 `Authorization: Bearer <key>`。

```bash
curl -X POST http://127.0.0.1:26228/api/pim/harvest
curl -X POST http://127.0.0.1:26228/api/pim/process
curl -X POST http://127.0.0.1:26228/api/pim/sync
curl http://127.0.0.1:26228/api/pim/procurement/capabilities
curl http://127.0.0.1:26228/api/pim/procurement/workbench-summary
```

采购汇总与导出接口也已经接入，同样受 `PIM_API_KEY` 保护：

```bash
curl -X POST http://127.0.0.1:26228/api/pim/procurement/summary \
  -H 'Content-Type: application/json' \
  -d '{
    "externalRef": "PO-20260316-001",
    "deliveryAddress": "攀枝花一号冻库",
    "notes": "今晚汇总采购",
    "items": [
      {"supplierCode": "SUP_A", "originalSku": "SKU-1001", "quantity": 5},
      {"supplierCode": "SUP_A", "originalSku": "SKU-1002", "quantity": 8}
    ]
  }'

curl -X POST http://127.0.0.1:26228/api/pim/procurement/export \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"supplierCode":"SUP_A","originalSku":"SKU-1001","quantity":5}]}'

curl -X POST http://127.0.0.1:26228/api/pim/procurement/submit \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"supplierCode":"SUP_A","originalSku":"SKU-1001","quantity":5}]}'

curl -X POST http://127.0.0.1:26228/api/pim/procurement/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"supplierCode":"SUP_A","originalSku":"SKU-1001","quantity":5}]}'

curl http://127.0.0.1:26228/api/pim/procurement/orders
curl 'http://127.0.0.1:26228/api/pim/procurement/order?id=<procurement-order-id>'
curl -X POST 'http://127.0.0.1:26228/api/pim/procurement/order/review?id=<procurement-order-id>'
curl -X POST 'http://127.0.0.1:26228/api/pim/procurement/order/export?id=<procurement-order-id>'
curl -X POST 'http://127.0.0.1:26228/api/pim/procurement/order/status?id=<procurement-order-id>' \
  -H 'Content-Type: application/json' \
  -d '{"status":"ordered","note":"已在供应商后台手工下单"}'
```

采购接口说明：

- `GET /api/pim/procurement/capabilities`
  返回当前 supplier connector 是否支持真实推单，还是只支持手工导出。
- `GET /api/pim/procurement/workbench-summary`
  返回手动采购工作台摘要，包括各状态计数、未完成风险单数和最近采购单。
- `POST /api/pim/procurement/summary`
  按 `supplier_products` 中已采集商品生成采购汇总，并带出成本、B/C 价、风险等级。
- `POST /api/pim/procurement/export`
  返回标准 CSV 内容，适合走“半自动汇单 + 人工发给供应商”的模式。
- `POST /api/pim/procurement/submit`
  走 connector 的统一下单边界。当前 `file` connector 会明确返回 `manual_export`，不给出伪成功。
- `POST /api/pim/procurement/orders`
  创建一张可追踪的手动采购单，初始状态为 `draft`。
- `GET /api/pim/procurement/orders`
  查询采购单列表，可按 `status` 过滤。
- `GET /api/pim/procurement/order?id=...`
  查看单张采购单详情，含汇总、导出内容和状态时间戳。
- `POST /api/pim/procurement/order/review?id=...`
  将采购单标记为 `reviewed`，表示人工已经确认过价格、数量和地址。
- `POST /api/pim/procurement/order/export?id=...`
  生成并保存 CSV，同时将采购单状态推进到 `exported`。
- `POST /api/pim/procurement/order/status?id=...`
  手动推进状态，支持 `reviewed`、`exported`、`ordered`、`received`、`canceled`。

手动采购状态流转：

- `draft -> reviewed -> exported -> ordered -> received`
- 任一未完成阶段都允许转 `canceled`

## PocketBase 审核操作

在 `supplier_products` 中，运营主要维护这些字段:

- `normalized_title`
- `marketing_description`
- `normalized_category`
- `b_price`
- `c_price`
- `sync_status`

推荐流程:

1. 采集后等待系统把记录处理到 `ready`
2. 人工确认图片和文案
3. 设置 `sync_status = approved`
4. 等待定时同步，或调用 `/api/pim/sync`

## Vendure 对接约束

当前项目默认对接本仓库的 Vendure:

- Admin API: `http://127.0.0.1:26227/admin-api`
- 语言: `zh_Hans`
- 货币: `CNY`
- 变体自定义字段:
  - `salesUnit`
  - `bPrice`

同步逻辑会使用这些字段。如果 Vendure 侧字段变化，需要同步调整 [`internal/vendure/client.go`](./internal/vendure/client.go)。

## 图片处理模式

### `IMAGE_PROCESSOR=mock`

不依赖 AI 服务，直接生成一张 SVG 预览图，用于跑通全链路。

### `IMAGE_PROCESSOR=webhook`

向 `IMAGE_WEBHOOK_URL` 发送 JSON:

```json
{
  "supplierCode": "SUP_A",
  "sku": "SKU-1001",
  "title": "谷饲肥牛卷 500g",
  "sourceURL": "https://example.com/images/sku-1001.jpg"
}
```

期望返回两种格式之一:

```json
{
  "output_url": "http://127.0.0.1:9000/result/sku-1001.png"
}
```

或

```json
{
  "filename": "sku-1001.png",
  "base64_data": "..."
}
```

## 后续建议

- 接入真实供应商 API / 爬虫
- 用真实 OCR + IOPaint 替换 mock 图片处理器
- 增加多供应商 connector
- 增加价格波动报警与审批
- 补充集成测试，验证 PocketBase 和 Vendure 实际联调

