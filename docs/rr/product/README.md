# 商品页抓包与数据整理建议

`docs/rr/product` 下的样本对应的是商品详情页链路，不是首页 section，也不是分类页 section。

但这三者不能分开治理。原因很直接：

- 首页和分类页里的商品卡片，本质上都是商品详情页的入口。
- 同一个 `spuId/skuId` 会同时出现在首页、分类页、商品页。
- 如果首页、分类页、商品页各自维护一份商品实体，价格、库存、单位、促销会很快漂移。

结论：

- 商品页要单独建聚合
- 首页商品和分类商品也要一并纳入同一套商品标准化方案
- 但首页/分类页只保留“列表视图需要的字段”和商品引用
- 商品详情、价格库存、单位切换、加购上下文应落到独立商品数据里

## 当前样本链路

`docs/rr/product/1` 现在是一条完整的商品页访问链路，核心请求如下：

| 编号 | 源站 API | 作用 |
| --- | --- | --- |
| `[2776]` | `/gateway/goodsservice/api/v1/wx/goods/info` | 商品详情基础信息 |
| `[2777]` | `/gateway/goodsservice/api/v1/wx/price/list` | 当前默认单位价格列表 |
| `[2778]` | `/gateway/goodsservice/api/v1/wx/goods/sku/price_stock` | 当前单位价格库存补充 |
| `[2779]` | `/gateway/goodsservice/api/v1/goods/get_sku_relation_discounts_package` | 关联优惠/套餐信息 |
| `[2780]` | `/gateway/goodsservice/api/v1/wx/cart/get_cart_tot_num` | 购物车总数 |
| `[2781]` | `/gateway/goodsservice/api/v1/wx/cart/choose/{skuId}` | 商品下单单位、配置和上下文 |
| `[2782]` | `/gateway/goodsservice/api/v1/wx/goods/sku/price_stock` | 多单位价格库存补充 |
| `[2783]` | `/gateway/goodsservice/api/v1/wx/cart/get_add_cart_tot` | 当前商品已加购数量 |

从职责看，这条链路至少要拆成 4 块：

- `detail`
  商品基础信息、轮播图、表单字段、展示配置
- `pricing`
  默认价格、多单位价格、库存、促销
- `package`
  关联优惠和套餐
- `context`
  购物车总数、商品已加购数量、下单配置、单位切换上下文

## 对首页和分类页的影响

现有 `datasets/miniapp/homepage/sections/*.json` 和 `datasets/miniapp/category-page/sections/*.json` 已经带了商品卡片数据，但它们更适合继续承担“列表结果”职责，而不是商品主数据职责。

建议后续这样收敛：

1. 首页 section 保留列表排序、卡片字段、商品引用
2. 分类 section 保留筛选条件、列表结果、商品引用
3. 商品页 dataset 维护商品详情和价格库存等完整链路
4. 三者都以 `spuId + skuId` 作为稳定关联键

这样做的好处：

- 首页、分类、商品详情的数据职责清楚
- 后续补样本时不会重复搬运同一份商品数据
- 一旦要接 `http` 模式，也只需要让上游返回统一商品实体

## 当前 dataset 方向

当前已经接入运行时，目录结构如下：

```text
datasets/miniapp/
├── homepage/
├── category-page/
└── product-page/
    ├── meta.json
    ├── contracts.json
    └── products/
        └── <spuId>_<skuId>.json
```

其中单商品文件内部再拆成：

- `detail`
  商品详情主数据
- `pricing`
  单位、价格、库存、促销
- `package`
  关联套餐和优惠
- `context`
  商品页购物车和下单上下文

当前文件分两类：

- 完整 rr 样本
  来自 `docs/rr/product` 的真实商品详情链路，字段最完整
- list-derived skeleton
  从首页或分类 section 的商品卡片回填，先保证列表里的商品都能落到独立 `product-page` 文件

当前还提供一个优先级视图：

- `GET /api/miniapp/product-page/coverage`
  按 `homepage_dual_unit -> category_dual_unit -> visible_single_unit -> done_rr_detail` 排序，直接告诉你下一批该先补哪些商品 rr 样本
- `GET /api/miniapp/product-page/coverage-summary`
  直接返回分桶统计和 `firstBatch`

## 当前建议

下一步不是单独“补商品页”，而是一起做这三件事：

1. 继续把 `docs/rr/product` 样本增量整理进 `datasets/miniapp/product-page/products/*.json`，逐步替换已有 skeleton
2. 给首页和分类页商品补更多稳定引用关系
3. 再决定是否要把首页/分类当前重复字段逐步瘦身成引用 + 列表展示字段

这比继续往 `homepage` 或 `category-page` 的 section 里堆更多商品详情字段要稳。
