# Miniapp Source API 总览

这份文档只回答四个问题：

1. 源站原始 API 是什么
2. 原始抓包样本归档在哪里
3. 脱敏后的 dataset 落到哪里
4. 运行时最终暴露哪个本地接口

它是 `README.md`、`docs/start.md` 和 `docs/rr/categories/README.md` 的总览补充，不替代启动说明。

如果你只关心结算页该怎么接，直接看 [checkout-api.md](./checkout-api.md)。

## 一眼看懂

```text
源站 API / 抓包样本
    -> 脱敏整理
    -> datasets/miniapp/homepage | datasets/miniapp/category-page | datasets/miniapp/product-page | datasets/miniapp/cart-order
    -> snapshot/raw source
    -> internal/miniapp/service
    -> /api/miniapp/**
```

关键约束：

- `docs/rr/**` 只保留原始抓包归档，不参与运行时读取。
- `snapshot` 模式只从 `datasets/miniapp/**` 组装标准化 `Dataset`。
- `raw` 模式直接请求目标站原始 API，并在本项目内标准化成 `Dataset`。
- 本地接口不重放源站敏感鉴权，不保留 token、openId、customerId 等敏感值。

## 运行模式

| 模式 | 读取位置 | 用途 |
| --- | --- | --- |
| `snapshot` | `datasets/miniapp/homepage` + `datasets/miniapp/category-page` + `datasets/miniapp/product-page` + `datasets/miniapp/cart-order` | 本地联调、脱网开发、固定样本回放 |
| `raw` | 目标站原始 API | 由 `mrtang-pim` 直接请求真实源站并本地标准化 |

### 三种模式的职责差异

| 模式 | 谁负责理解目标站原始协议 | 是否实时 | 典型用途 |
| --- | --- | --- | --- |
| `snapshot` | 无 | 否 | 本地稳定联调 |
| `raw` | `mrtang-pim` 自己 | 是 | 直接接目标站原始 API |

相关配置：

- `MINIAPP_SOURCE_MODE=snapshot|raw`
- `MINIAPP_SOURCE_URL=...`
- `MINIAPP_HOMEPAGE_SNAPSHOT=./datasets/miniapp/homepage`
- `MINIAPP_CATEGORY_SNAPSHOT=./datasets/miniapp/category-page`
- `MINIAPP_PRODUCT_SNAPSHOT=./datasets/miniapp/product-page`
- `MINIAPP_CART_ORDER_SNAPSHOT=./datasets/miniapp/cart-order`
- `MINIAPP_AUTH_ACCOUNT_ID=...`
- `MINIAPP_USER_AGENT=...`
- `MINIAPP_RAW_TEMPLATE_ID=962`
- `MINIAPP_RAW_REFERER=https://servicewechat.com/.../page-frame.html`
- `MINIAPP_RAW_CONCURRENCY=4`
- `MINIAPP_RAW_MIN_INTERVAL=300ms`
- `MINIAPP_RAW_RETRY_MAX=2`

### raw 模式当前边界

- 已接入：分类树、分类商品、商品详情、价格、多单位、套餐、商品上下文、购物车列表、购物车详情、结算预览、默认地址、地址列表、地址解析、运费试算
- 显式写操作：加入购物车、改数量、添加地址、提交订单
- 安全策略：`FetchDataset()` 自动抓取阶段保持只读优先，不自动触发真实添加地址和真实下单

### 关于旧 http 模式

`http` 模式已经废弃，不再作为正式运行模式保留。当前统一只保留：

- `snapshot`
- `raw`

如果历史上曾将 `MINIAPP_SOURCE_URL` 指向标准化 Dataset 服务，请直接改为：

- 开发与回归时使用 `snapshot`
- 真实接源站时使用 `raw`

## Dataset 目录

### 首页

目录：[datasets/miniapp/homepage](../datasets/miniapp/homepage)

| 文件 | 含义 |
| --- | --- |
| `meta.json` | 数据来源说明、脱敏说明、补充备注 |
| `contracts.json` | 首页相关原始 API 到本地接口的契约映射 |
| `bootstrap.json` | 启动和登录态汇总 |
| `settings.json` | 店铺配置、首页视觉和分类页默认配置 |
| `template.json` | 首页装修模块和页面布局 |
| `category-tabs.json` | 首页快捷分类 tab |
| `sections/*.json` | 首页商品分组，当前有 `new.json`、`hot.json` |

### 分类页

目录：[datasets/miniapp/category-page](../datasets/miniapp/category-page)

| 文件 | 含义 |
| --- | --- |
| `meta.json` | 数据来源说明、脱敏说明、补充备注 |
| `contracts.json` | 分类页相关原始 API 到本地接口的契约映射 |
| `context.json` | 分类页登录态、购物车角标等上下文 |
| `tree.json` | 分类树，当前已固化 18 个顶级类目和递归子分类 |
| `sections/*.json` | 分类页商品分组，按分类单独维护 |

当前分类页 section 状态：

- 已有 18 个顶级类目对应的 `sections/*.json`
- 只有 `chicken.json` 带真实商品样本
- 其余 section 先保留标准请求骨架和空商品列表

### 商品页

目录：[datasets/miniapp/product-page](../datasets/miniapp/product-page)

| 文件 | 含义 |
| --- | --- |
| `meta.json` | 数据来源说明、脱敏说明、补充备注 |
| `contracts.json` | 商品页相关原始 API 到本地接口的契约映射 |
| `products/*.json` | 单商品聚合记录，内部再分 `detail/pricing/package/context` |

当前商品页状态：

- 已落 1 条完整 rr 商品页样本
- 已为当前首页 `new/hot` 和分类 `chicken` 中出现的商品补齐 list-derived skeleton 商品记录
- 样本主键使用 `spuId_skuId`
- 首页和分类页继续以 `spuId + skuId` 关联到商品页

### 购物车与下单

目录：[datasets/miniapp/cart-order](../datasets/miniapp/cart-order)

| 文件 | 含义 |
| --- | --- |
| `meta.json` | 数据来源说明、脱敏说明、补充备注 |
| `contracts.json` | 购物车和下单链路原始 API 到本地接口的契约映射 |
| `cart.json` | 加购、改数量、购物车列表、购物车详情、结算前校验 |
| `order.json` | 默认地址、地址列表、地址解析、添加地址、运费试算、提交订单 |

当前 cart-order 状态：

- 已接入加入购物车、更新购物车、购物车列表、购物车详情、结算前校验
- 已接入地址解析、添加地址、运费试算、提交订单
- 当前 `submit` 只保留“提交订单到待支付结果”的响应，不处理支付动作本身
- `customerId`、`cartId`、`addressId`、`openId` 等敏感值已替换成 demo 占位值
- 同时提供 `detail-summary` 和 `submit-summary` 两个派生接口，给前端读取稳定摘要字段
- `default-delivery-summary` 和 `deliveries-summary` 会在原始接口为空时回退到 `add_delivery` 响应，避免 checkout 前置页面拿到空地址
- `checkout-summary` 会把购物车、地址、运费和提交预览摘要聚到一个响应里，适合前端一次性拉取结算页数据

## 源站到本地接口映射

### 首页链路

| 源站 API | Dataset 落点 | 本地接口 | 作用 |
| --- | --- | --- | --- |
| `/gateway/shop-decoration-service/api/v1/open/get_loading_setting` | `homepage/bootstrap.json` | `GET /api/miniapp/homepage/bootstrap` | 小程序启动配置 |
| `/gateway/customer-service/api/v1/config/get_contacts_config` | `homepage/bootstrap.json` | `GET /api/miniapp/homepage/bootstrap` | 联系人配置开关 |
| `/gateway/customer-service/api/v1/order/app/get_bb_auth_status` | `homepage/bootstrap.json` | `GET /api/miniapp/homepage/bootstrap` | 访问权限状态 |
| `/gateway/customer-service/api/v1/order/app/get_login_status` | `homepage/bootstrap.json` | `GET /api/miniapp/homepage/bootstrap` | 登录和下单资格摘要 |
| `/gateway/goodsservice/api/v1/wx/cart/get_cart_tot_num` | `homepage/bootstrap.json` | `GET /api/miniapp/homepage/bootstrap` | 购物车数量 |
| `/gateway/shop-decoration-service/api/v1/shop/config/setting` | `homepage/settings.json` | `GET /api/miniapp/homepage/settings` | 店铺配置和视觉设置 |
| `/gateway/shop-decoration-service/api/v1/goods/category/setting/detail` | `homepage/settings.json` | `GET /api/miniapp/homepage/settings` | 分类页默认布局配置 |
| `/gateway/gateway-mall-service/api/v1/wx/goods/category_list` | `homepage/category-tabs.json` | `GET /api/miniapp/homepage/categories` | 首页快捷分类 tab |
| `/gateway/shop-decoration-service/api/v1/shop/template/using` | `homepage/template.json` | `GET /api/miniapp/homepage/template` | 首页装修模板 |
| `/gateway/gateway-mall-service/api/v1/wx/goods/list` | `homepage/sections/new.json` | `GET /api/miniapp/homepage/section?id=new` | 新品区商品列表 |
| `/gateway/gateway-mall-service/api/v1/wx/goods/list` | `homepage/sections/hot.json` | `GET /api/miniapp/homepage/section?id=hot` | 热销区商品列表 |

首页聚合接口：

- `GET /api/miniapp/contracts/homepage`
- `GET /api/miniapp/homepage`
- `GET /api/miniapp/homepage/sections`

### 分类页链路

分类页原始样本归档说明见 [docs/rr/categories/README.md](./rr/categories/README.md)。

| 源站 API | rr 样本编号 | Dataset 落点 | 本地接口 | 作用 |
| --- | --- | --- | --- | --- |
| `/gateway/marketing-service/api/v1/integral/wx_login_send` | `[294]` | `category-page/context.json` | `GET /api/miniapp/category-page/context` | 分类页登录前置动作说明 |
| `/gateway/customer-service/api/v1/order/app/get_login_status` | `[295]` | `category-page/context.json` | `GET /api/miniapp/category-page/context` | 分类页登录和下单状态 |
| `/gateway/goodsservice/api/v1/wx/category/get_have_goods_category` | `[296]` | `category-page/tree.json` | `GET /api/miniapp/category-page/tree` | 分类树 |
| `/gateway/goodsservice/api/v1/wx/cart/get_cart_tot_num` | `[298]` | `category-page/context.json` | `GET /api/miniapp/category-page/context` | 分类页购物车角标 |
| `/gateway/gateway-mall-service/api/v1/wx/goods/list` | `[299]` | `category-page/sections/*.json` | `GET /api/miniapp/category-page/section?id=<section-id>` | 分类商品列表 |
| `/gateway/goodsservice/api/v1/wx/goods/sku/list/price_stock` | `[300]` | `category-page/sections/*.json` | `GET /api/miniapp/category-page/section?id=<section-id>` | 价格、库存、单位和促销补充 |

分类页聚合接口：

- `GET /api/miniapp/contracts/category-page`
- `GET /api/miniapp/category-page`
- `GET /api/miniapp/category-page/sections`

### 商品页链路

商品页原始样本归档说明见 [docs/rr/product/README.md](./rr/product/README.md)。

| 源站 API | rr 样本编号 | Dataset 落点 | 本地接口 | 作用 |
| --- | --- | --- | --- | --- |
| `/gateway/goodsservice/api/v1/wx/goods/info` | `[2776]` | `product-page/products/*.json.detail` | `GET /api/miniapp/product-page/detail?id=<spuId>_<skuId>` | 商品详情基础信息 |
| `/gateway/goodsservice/api/v1/wx/price/list` | `[2777]` | `product-page/products/*.json.pricing` | `GET /api/miniapp/product-page/pricing?id=<spuId>_<skuId>` | 默认单位价格 |
| `/gateway/goodsservice/api/v1/wx/goods/sku/price_stock` | `[2778]` | `product-page/products/*.json.pricing` | `GET /api/miniapp/product-page/pricing?id=<spuId>_<skuId>` | 默认单位库存补充 |
| `/gateway/goodsservice/api/v1/goods/get_sku_relation_discounts_package` | `[2779]` | `product-page/products/*.json.package` | `GET /api/miniapp/product-page/package?id=<spuId>_<skuId>` | 套餐和关联优惠 |
| `/gateway/goodsservice/api/v1/wx/cart/get_cart_tot_num` | `[2780]` | `product-page/products/*.json.context` | `GET /api/miniapp/product-page/context?id=<spuId>_<skuId>` | 购物车角标 |
| `/gateway/goodsservice/api/v1/wx/cart/choose/{skuId}` | `[2781]` | `product-page/products/*.json.context` | `GET /api/miniapp/product-page/context?id=<spuId>_<skuId>` | 单位切换和下单上下文 |
| `/gateway/goodsservice/api/v1/wx/goods/sku/price_stock` | `[2782]` | `product-page/products/*.json.pricing` | `GET /api/miniapp/product-page/pricing?id=<spuId>_<skuId>` | 多单位价格库存补充 |
| `/gateway/goodsservice/api/v1/wx/cart/get_add_cart_tot` | `[2783]` | `product-page/products/*.json.context` | `GET /api/miniapp/product-page/context?id=<spuId>_<skuId>` | 当前商品已加购数量 |

商品页聚合接口：

- `GET /api/miniapp/contracts/product-page`
- `GET /api/miniapp/product-page`
- `GET /api/miniapp/product-page/product?id=<spuId>_<skuId>`
- `GET /api/miniapp/product-page/coverage`
- `GET /api/miniapp/product-page/coverage-summary`

当前状态：

- `docs/rr/product` 已经有完整商品页访问链路样本
- 运行时现在从 `datasets/miniapp/product-page` 读取 `product-page` dataset
- 当前 `product-page` 目录中同时包含完整 rr 商品样本和列表回填的 skeleton 商品记录
- 首页和分类页商品仍保留列表职责，不把商品主数据继续塞回 section 文件
- `/api/miniapp/product-page/coverage` 会把当前可见商品按 `homepage_dual_unit -> category_dual_unit -> visible_single_unit -> done_rr_detail` 排序，直接给出下一批应优先替换的目标
- `/api/miniapp/product-page/coverage?priority=homepage_dual_unit` 可以直接只看“首页双单位优先批次”
- `/api/miniapp/product-page/coverage-summary` 会返回总数、分桶统计和 `firstBatch`

### 购物车与下单链路

购物车与下单原始样本归档说明见 [docs/rr/cart-order/README.md](./rr/cart-order/README.md)。

| 源站 API | rr 样本编号 | Dataset 落点 | 本地接口 | 作用 |
| --- | --- | --- | --- | --- |
| `/gateway/goodsservice/api/v1/wx/cart/addCart` | `[5195]` | `cart-order/cart.json.add` | `POST /api/miniapp/cart-order/cart/add` | 加入购物车 |
| `/gateway/goodsservice/api/v1/wx/cart/change_cart_num` | `[5215]` | `cart-order/cart.json.changeNum` | `POST /api/miniapp/cart-order/cart/change-num` | 更新购物车数量 |
| `/gateway/goodsservice/api/v1/wx/cart/list` | `[5197]` | `cart-order/cart.json.list` | `GET /api/miniapp/cart-order/cart/list` | 购物车列表和合计 |
| `/gateway/goodsservice/api/v1/wx/cart/detail` | `[5224]` | `cart-order/cart.json.detail` | `GET /api/miniapp/cart-order/cart/detail` | 结算页购物车详情 |
| `/gateway/goodsservice/api/v1/wx/cart/settle` | `[5216]` | `cart-order/cart.json.settle` | `POST /api/miniapp/cart-order/cart/settle` | 结算前校验 |
| `/gateway/customer-service/api/v1/order/get_default_delivery` | `[5221]` | `cart-order/order.json.defaultDelivery` | `GET /api/miniapp/cart-order/order/default-delivery` | 默认收货地址 |
| `/gateway/customer-service/api/v1/order/get_deliverys` | `[5232]` | `cart-order/order.json.deliveries` | `GET /api/miniapp/cart-order/order/deliveries` | 收货地址列表 |
| `/gateway/saas-platform-service/api/v1/address/analyse_address` | `[5265]` | `cart-order/order.json.analyseAddress` | `POST /api/miniapp/cart-order/order/address/analyse` | 文本地址解析 |
| `/gateway/customer-service/api/v1/order/add_delivery` | `[5266]` | `cart-order/order.json.addDelivery` | `POST /api/miniapp/cart-order/order/address/add` | 添加收货地址 |
| `/gateway/logisticsservice/api/v1/freight/cost` | `[5225]` | `cart-order/order.json.freightCosts[preview]` | `GET /api/miniapp/cart-order/order/freight-cost?scenario=preview` | 未选配送方式时运费试算 |
| `/gateway/logisticsservice/api/v1/freight/cost` | `[5269]` | `cart-order/order.json.freightCosts[selected_delivery]` | `GET /api/miniapp/cart-order/order/freight-cost?scenario=selected_delivery` | 选定配送方式后的运费试算 |
| `/gateway/billservice/api/v1/wx/sale_bill/save` | `[5276]` | `cart-order/order.json.submit` | `POST /api/miniapp/cart-order/order/submit` | 提交订单到待支付结果 |

cart-order 聚合接口：

- `GET /api/miniapp/contracts/cart-order`
- `GET /api/miniapp/cart-order`
- `GET /api/miniapp/cart-order/cart`
- `GET /api/miniapp/cart-order/order`
- `GET /api/miniapp/cart-order/cart/list-summary`
- `GET /api/miniapp/cart-order/cart/detail-summary`
- `GET /api/miniapp/cart-order/order/default-delivery-summary`
- `GET /api/miniapp/cart-order/order/deliveries-summary`
- `GET /api/miniapp/cart-order/order/freight-summary`
- `GET /api/miniapp/cart-order/order/submit-summary`
- `GET /api/miniapp/cart-order/checkout-summary`

## 运行时读取规则

实现位置：

- [snapshot_source.go](../internal/miniapp/api/snapshot_source.go)
- [raw_source.go](../internal/miniapp/api/raw_source.go)
- [config.go](../internal/config/config.go)

规则如下：

1. `snapshot` 模式先加载首页 dataset，再合并分类页、商品页和 cart-order dataset。
2. `snapshot` 支持“目录模式”和“单文件兼容模式”，但当前推荐目录模式。
3. `raw` 模式直接请求目标站原始 API，并在本项目内部标准化成 `Dataset`。
4. `raw` 模式会自动附带：
   - `Authorization: Bearer <MINIAPP_AUTH_ACCOUNT_ID>`
   - `User-Agent: <MINIAPP_USER_AGENT>`
5. `raw` 模式当前支持通过配置限制抓取压力：
   - `MINIAPP_RAW_CONCURRENCY`
   - `MINIAPP_RAW_MIN_INTERVAL`
   - `MINIAPP_RAW_RETRY_MAX`
6. cart-order 中的 `customerId`、`customerName`、`phone`、`openId`、`addressId`、`cartIdList`、`deliveryMethodId`、`dueMoney` 都会从前序样本自动串联，但不会改写 `datasets/miniapp/**` 原始脱敏样本。

## 维护建议

- 新增首页商品模块时，优先补 `datasets/miniapp/homepage/sections/<id>.json`。
- 新增分类商品样本时，优先补 `datasets/miniapp/category-page/sections/<id>.json`。
- 新增商品详情样本时，优先补 `datasets/miniapp/product-page/products/<spuId>_<skuId>.json`。
- 新增购物车或下单样本时，优先补 `datasets/miniapp/cart-order/cart.json` 或 `datasets/miniapp/cart-order/order.json`。
- 原始抓包只追加到 `docs/rr/**`，不要让运行时代码直接依赖 rr 目录。
- 如果接入真实源站，优先通过 `raw` 模式在本项目内部标准化，不要把源站字段直接泄漏到本地接口层。
