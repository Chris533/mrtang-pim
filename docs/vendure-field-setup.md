# Vendure 字段配置建议

这份文档用于配合 [backend-release-contract.md](./backend-release-contract.md) 使用，目标是给出一份“可直接照着配置”的 Vendure custom fields 清单。

## 目标

为了让 `mrtang-pim` 推送到 Vendure 的商品具备外部供应商经营能力，建议至少补齐下面这组字段。

## 一、Variant 级 custom fields

建议在 `ProductVariant` 上增加：

### 1. `supplierCode`

- 类型：`string`
- 用途：标记供应商来源，例如 `SUP_A`

### 2. `supplierCostPrice`

- 类型：`int` 或 `float`
- 用途：保存供应商底价，用于毛利和预警

### 3. `conversionRate`

- 类型：`float`
- 用途：保存多单位换算率，例如：
  - 袋：`1`
  - 件：`20`

### 4. `sourceProductId`

- 类型：`string`
- 用途：回溯到 `source_products.product_id`

### 5. `sourceType`

- 类型：`string`
- 用途：区分 `raw / rr_detail / list_skeleton` 等来源

## 二、Product 级 custom fields

建议在 `Product` 上增加：

### 1. `targetAudience`

- 类型：`string` 或 `enum`
- 建议值：
  - `ALL`
  - `B_ONLY`
  - `C_ONLY`

### 2. `cEndFeaturedAsset`

- 类型：`relation -> Asset`
- 用途：单独给 C 端展示图

## 三、推荐命名

如果你希望和 `mrtang-pim` 默认建议保持一致，推荐直接使用下面这些字段名：

### Variant

- `supplierCode`
- `supplierCostPrice`
- `conversionRate`
- `sourceProductId`
- `sourceType`

### Product

- `targetAudience`
- `cEndFeaturedAsset`

## 四、环境变量映射

在 `mrtang-pim` 里，把这些字段名填进环境变量即可：

```env
VENDURE_CF_VARIANT_SUPPLIER_CODE=supplierCode
VENDURE_CF_VARIANT_SUPPLIER_COST_PRICE=supplierCostPrice
VENDURE_CF_VARIANT_CONVERSION_RATE=conversionRate
VENDURE_CF_VARIANT_SOURCE_PRODUCT_ID=sourceProductId
VENDURE_CF_VARIANT_SOURCE_TYPE=sourceType
VENDURE_CF_PRODUCT_TARGET_AUDIENCE=targetAudience
VENDURE_CF_PRODUCT_C_END_FEATURED_ASSET=cEndFeaturedAsset
```

如果这些变量留空，`mrtang-pim` 不会强行写入这些 custom fields。

## 五、建议联调顺序

1. 先在 Vendure 建好上述 custom fields
2. 把环境变量配置到 `mrtang-pim`
3. 先挑 1 到 3 个商品做试同步
4. 重点检查：
   - Variant 是否带上 `supplierCode / supplierCostPrice / conversionRate`
   - Product 是否带上 `targetAudience`
   - 主图和 `cEndFeaturedAsset` 是否按预期分流

## 六、当前限制

当前这批代码已经支持把上述字段装进同步 payload，但还没有自动验证 Vendure schema 是否存在这些字段。

因此如果：

- Vendure 没建字段
- 环境变量字段名写错

同步可能会因为 GraphQL 输入校验失败而报错。

建议先小批量联调，不要一开始就全量同步。
