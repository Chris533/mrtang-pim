# mrtang-pim 启动说明

`mrtang-pim` 当前已经不是概念方案，而是一套可运行的 `Golang + PocketBase` PIM 服务。它负责承接供应商商品、图片处理、人工审核、Vendure 同步，以及小程序首页数据接入。

## 当前架构

项目分成两条主链路：

1. PIM 商品链路
   `Supplier -> PocketBase -> Image Processor -> Admin Review -> Vendure`
2. Miniapp 首页链路
   `Snapshot/HTTP Source -> miniapp importer/service -> API -> 后续入库`

当前目录重点如下：

```text
mrtang-pim/
├── cmd/pim/main.go
├── datasets/
│   ├── miniapp_homepage_snapshot.json
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

## 环境变量

至少关注这些配置：

- `PIM_PUBLIC_URL`
- `PIM_API_KEY`
- `PIM_SUPERUSER_EMAIL`
- `PIM_SUPERUSER_PASSWORD`
- `MINIAPP_SOURCE_MODE`
- `MINIAPP_SOURCE_URL`
- `MINIAPP_SOURCE_TIMEOUT`
- `MINIAPP_HOMEPAGE_SNAPSHOT=./datasets/miniapp_homepage_snapshot.json`
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

- Admin UI: `http://127.0.0.1:8090/_/`
- Health: `http://127.0.0.1:8090/api/pim/healthz`

## 当前可用接口

PIM：

- `POST /api/pim/harvest`
- `POST /api/pim/process`
- `POST /api/pim/sync`

Miniapp：

- `GET /api/miniapp/contracts/homepage`
- `GET /api/miniapp/homepage`
- `GET /api/miniapp/homepage/bootstrap`
- `GET /api/miniapp/homepage/settings`
- `GET /api/miniapp/homepage/template`
- `GET /api/miniapp/homepage/categories`
- `GET /api/miniapp/homepage/sections`
- `GET /api/miniapp/homepage/section?id=<section-id>`

## 下一步

当前还缺两块正式能力：

1. `internal/miniapp/repository` 的真实入库实现
2. 真实上游 miniapp source 返回结果到本地数据库的同步任务
