# 分类页抓包与本地 Source 映射

`docs/rr/categories` 下的抓包样本不对应首页 `categoryTabs`，而是对应分类页独立链路。

当前仓库已经按首页 source 的实现方式，把这组样本整理进统一 dataset 的 `categoryPage` 聚合，供以下接口直接消费：

- `GET /api/miniapp/contracts/category-page`
- `GET /api/miniapp/category-page`
- `GET /api/miniapp/category-page/context`
- `GET /api/miniapp/category-page/tree`
- `GET /api/miniapp/category-page/sections`
- `GET /api/miniapp/category-page/section?id=chicken`

如果要连同首页链路一起看完整映射，见 [docs/source-api.md](../../source-api.md)。

说明：

- 运行时不直接读取 `docs/rr/categories`，这里的文件只作为原始抓包归档。
- `snapshot` 模式读取两个独立目录：
- [homepage](../../../datasets/miniapp/homepage)
- [category-page](../../../datasets/miniapp/category-page)
- `http` 模式读取 `MINIAPP_SOURCE_URL` 返回的标准化 dataset JSON。
- 分类页商品 section 单独放在 `datasets/miniapp/category-page/sections/` 下，便于按分类增量维护。
- 当前 dataset 中已经固化了全量分类树；18 个顶级分类都已有对应 section 文件。
- 目前只有 `chicken.json` 带真实商品样本，其余 section 先保留标准请求骨架和空商品列表，等待后续 rr 补样本。

样本与本地字段映射如下：

- `[294]` `wx_login_send` -> `categoryPage.context` 的登录前置动作说明
- `[295]` `get_login_status` -> `categoryPage.context`
- `[296]` `get_have_goods_category` -> `categoryPage.tree`
- `[298]` `get_cart_tot_num` -> `categoryPage.context.cartItemCount`
- `[299]` `wx/goods/list` -> `categoryPage.sections[*].requestBody` 与商品基础列表
- `[300]` `sku/list/price_stock` -> `categoryPage.sections[*].products[*].price/stock/unitOptions/promotionTexts`

设计约束：

- 首页 `/api/miniapp/homepage/categories` 仍然只返回首页快捷分类 tab，不承载多级分类树。
- 分类树使用 `pathCode` 作为稳定 key，避免只用单级 `id` 后丢失层级语义。
- 商品列表和价格库存拆成两段保留，和原始小程序链路一致。
