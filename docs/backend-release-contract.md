# Backend 发布字段与分类模型

这份文档是 [backend-miniapp-plan.md](./backend-miniapp-plan.md) 的落地补充，专门说明：

- `source_products / supplier_products` 到 backend 的字段映射
- 主图 / C 端图的发布规则
- 分类发布前的映射模型

## 一、当前发布链路

当前正式发布链路仍然是：

1. `source_products`
2. 审核通过后加入发布队列，写入 `supplier_products`
3. `SyncApproved` 推送到 Vendure backend

也就是说，backend 读取的直接来源仍然是 `supplier_products`，不是直接读 `source_products`。

## 二、商品字段映射

### 1. 已有基础字段

当前仍然会同步这些基础字段：

- `name`
- `slug`
- `description`
- `sku`
- `consumerPrice`
- `businessPrice`
- `defaultStock`
- `salesUnit`
- `featuredAsset`

### 2. 这次新增的 backend 最小必要字段

这批代码已经补了以下映射准备：

- `supplierCode`
- `supplierCostPrice`
- `conversionRate`
- `sourceProductId`
- `sourceType`
- `targetAudience`
- `cEndFeaturedAsset`

### 3. 来源映射表

#### source -> supplier_products

- `source_products.product_id` -> `supplier_products.source_product_id`
- `source_products.source_type` -> `supplier_products.source_type`
- `source_products.default_unit` -> `supplier_payload.sales_unit`
- `source_products.unit_options_json` -> `supplier_payload.unit_options`
- `source_products.order_units_json` -> `supplier_payload.order_units`
- `source_products.category_key` -> `supplier_payload.category_key`
- 默认换算率 -> `supplier_products.conversion_rate`
- 默认客群 -> `supplier_products.target_audience=ALL`

#### supplier_products -> Vendure Variant

- `supplier_products.supplier_code` -> `supplierCode`
- `supplier_products.cost_price` -> `supplierCostPrice`
- `supplier_products.conversion_rate` -> `conversionRate`
- `supplier_products.source_product_id` -> `sourceProductId`
- `supplier_products.source_type` -> `sourceType`
- `supplier_payload.sales_unit` -> `salesUnit`
- `supplier_products.b_price` -> `bPrice`

#### supplier_products -> Vendure Product

- `supplier_products.target_audience` -> `targetAudience`
- C 端图 -> `cEndFeaturedAsset`

## 三、图片发布规则

### 1. 当前默认规则

当前同步时使用两张图的语义已经收出来了：

- B 端主图 / `featuredAsset`
  - 优先 `raw_image_url`
  - 没有时回退到处理图
- C 端主图 / `cEndFeaturedAsset`
  - 优先处理图
  - 没有时回退到 B 端主图

这意味着：

- 如果你“不需要处理图片”，商品仍可先用原图同步
- 如果后面补了处理图，C 端图会比 B 端图更容易独立出来

### 2. 当前限制

`cEndFeaturedAsset` 是否真正写入 backend，取决于 Vendure 端是否已配置对应 custom field。

因此这次实现做成了“配置式启用”，不会强行假设 backend 已经准备好。

## 四、Vendure 配置项

这次新增的是“可选 custom field 映射配置”。只有配置了字段名，发布时才会写入。

### Variant 级字段

- `VENDURE_CF_VARIANT_SUPPLIER_CODE`
- `VENDURE_CF_VARIANT_SUPPLIER_COST_PRICE`
- `VENDURE_CF_VARIANT_CONVERSION_RATE`
- `VENDURE_CF_VARIANT_SOURCE_PRODUCT_ID`
- `VENDURE_CF_VARIANT_SOURCE_TYPE`

### Product 级字段

- `VENDURE_CF_PRODUCT_TARGET_AUDIENCE`
- `VENDURE_CF_PRODUCT_C_END_FEATURED_ASSET`

## 五、分类发布模型

这次先补了最小映射存储，不直接强推发布逻辑。

新增集合：

- `backend_category_mappings`

字段包括：

- `source_key`
- `source_path`
- `backend_collection`
- `backend_path`
- `publish_status`
- `last_error`
- `note`

### 当前推荐用法

1. 先把 `source_categories` 跑全
2. 在 `/_/mrtang-admin/backend-release` 直接执行：
   - `按建议创建`
   - `创建到 Backend`
   - `按建议批量创建`
3. 创建成功后，系统会自动回写：
   - `backend_collection`
   - `backend_path`
   - `backend_collection_id`
   - `publish_status`
   - `published_at`
4. 只有分类已创建到 backend 后，再进入商品小批量联调同步

### publish_status 语义

- `pending`
  - 待创建
- `mapped`
  - 已保存本地路径，但还未创建到 backend
- `published`
  - backend 分类已创建并可用
- `error`
  - 创建失败，需要人工重试

## 六、下一步建议

下一步最合理的是先做 backend 配置和联调，不急着全量发布：

1. 在 Vendure 配置上述 custom fields
2. 确认 Product / Variant 哪些字段已经存在
3. 先在 `/_/mrtang-admin/backend-release` 直接创建最小分类样例
4. 小批量同步几条商品验证：
   - 多单位
   - B/C 端图
   - supplierCode / costPrice
5. 再推进分类映射和正式发布
