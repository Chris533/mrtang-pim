# Miniapp Source API 总览

这份文档只回答四个问题：

1. 源站原始 API 是什么
2. 原始抓包样本归档在哪里
3. 脱敏后的 dataset 落到哪里
4. 运行时最终暴露哪个本地接口

它是 `README.md`、`docs/start.md` 和 `docs/rr/categories/README.md` 的总览补充，不替代启动说明。

## 一眼看懂

```text
源站 API / 抓包样本
    -> 脱敏整理
    -> datasets/miniapp/homepage | datasets/miniapp/category-page
    -> snapshot/http source
    -> internal/miniapp/service
    -> /api/miniapp/**
```

关键约束：

- `docs/rr/**` 只保留原始抓包归档，不参与运行时读取。
- `snapshot` 模式只从 `datasets/miniapp/**` 组装标准化 `Dataset`。
- `http` 模式只从 `MINIAPP_SOURCE_URL` 拉取同构的标准化 `Dataset` JSON。
- 本地接口不重放源站敏感鉴权，不保留 token、openId、customerId 等敏感值。

## 运行模式

| 模式 | 读取位置 | 用途 |
| --- | --- | --- |
| `snapshot` | `datasets/miniapp/homepage` + `datasets/miniapp/category-page` | 本地联调、脱网开发、固定样本回放 |
| `http` | `MINIAPP_SOURCE_URL` | 接入你自己的上游 connector 或标准化聚合服务 |

相关配置：

- `MINIAPP_SOURCE_MODE=snapshot|http`
- `MINIAPP_SOURCE_URL=...`
- `MINIAPP_HOMEPAGE_SNAPSHOT=./datasets/miniapp/homepage`
- `MINIAPP_CATEGORY_SNAPSHOT=./datasets/miniapp/category-page`
- `MINIAPP_AUTH_ACCOUNT_ID=...`
- `MINIAPP_USER_AGENT=...`

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

## 运行时读取规则

实现位置：

- [snapshot_source.go](../internal/miniapp/api/snapshot_source.go)
- [http_source.go](../internal/miniapp/api/http_source.go)
- [config.go](../internal/config/config.go)

规则如下：

1. `snapshot` 模式先加载首页 dataset，再合并分类页 dataset。
2. `snapshot` 支持“目录模式”和“单文件兼容模式”，但当前推荐目录模式。
3. `http` 模式只请求一次 `MINIAPP_SOURCE_URL`，要求对方直接返回标准化后的 `Dataset` JSON。
4. `http` 模式会自动附带：
   - `Authorization: Bearer <MINIAPP_AUTH_ACCOUNT_ID>`
   - `User-Agent: <MINIAPP_USER_AGENT>`

## 维护建议

- 新增首页商品模块时，优先补 `datasets/miniapp/homepage/sections/<id>.json`。
- 新增分类商品样本时，优先补 `datasets/miniapp/category-page/sections/<id>.json`。
- 原始抓包只追加到 `docs/rr/**`，不要让运行时代码直接依赖 rr 目录。
- 如果接入真实源站，先把源站返回整理成标准化 `Dataset`，再接到 `http` 模式，不要把源站字段直接泄漏到本地接口层。
