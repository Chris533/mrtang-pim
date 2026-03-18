# 小程序 UI 批次 1 Contract

这份文档对应 [miniapp-ui-plan.md](./miniapp-ui-plan.md) 的“批次 1：分类页 + 商品页基础 contract”。

目标不是立刻写前端，而是先把：

- 分类页结构
- 分类切换规则
- 商品页基础字段
- 多单位基础交互

定成统一约束。

## 一、分类页 Contract

### 1. 页面结构

分类页拆成三块：

1. 一级分类导航
2. 当前分类 breadcrumb
3. 商品列表区

当前建议：

- 一级分类作为稳定主导航
- 二级、三级分类在内容区内切换
- breadcrumb 显示当前分类链

### 2. 分类数据来源

优先来源：

- backend `Collection`

字段最低要求：

- `id`
- `name`
- `slug`
- `breadcrumbs`
- `parentId`
- `featuredAsset`
- `customFields.sourceCategoryKey`
- `customFields.sourceCategoryPath`
- `customFields.sourceCategoryLevel`

### 3. 分类切换规则

切换分类时：

- 保留当前排序
- 保留当前客群上下文
- 清空不再适用的局部筛选

当前建议：

- 一级分类切换时重置分页
- 同级分类切换时只重置分页，不重置排序

### 4. 分类商品列表

最低字段：

- `productId`
- `productName`
- `slug`
- `featuredAsset`
- `targetAudience`
- `defaultUnit`
- `defaultPrice`
- `hasMultiUnit`

最低交互：

- 分页
- 排序
- 当前分类 breadcrumb
- 进入商品详情

## 二、商品页 Contract

### 1. 页面结构

商品页拆成五块：

1. 主图与相册
2. 标题与标签
3. 价格与单位区
4. 规格/单位切换区
5. 详情与说明区

### 2. 商品基础字段

最低字段：

- `productId`
- `name`
- `slug`
- `description`
- `featuredAsset`
- `assetGallery`
- `targetAudience`
- `salesUnit`
- `conversionRate`
- `consumerPrice`
- `businessPrice`
- `defaultStock`

### 3. 多单位字段

最低字段：

- `defaultUnit`
- `unitOptions`
- `orderUnits`
- `conversionRate`

每个 `unitOption` 最低包含：

- `unitId`
- `unitName`
- `price`
- `displayName`
- `isDefault`

### 4. 多单位交互

当前建议：

- 页面默认选中默认销售单位
- 若存在多个可售单位，显示单位切换器
- 切换单位后：
  - 更新价格
  - 更新购买数量步进
  - 更新库存提示文案

### 5. B/C 展示差异

当前建议：

- B 端：
  - 展示原图主图
  - 展示多单位和进货参考信息
  - 可显示 `businessPrice`
- C 端：
  - 优先展示 `cEndFeaturedAsset`
  - 展示零售价
  - 多单位展示收敛到用户可购买单位

## 三、第一批需要 backend/API 明确提供的字段

### 分类页

- `Collection.id`
- `Collection.name`
- `Collection.slug`
- `Collection.breadcrumbs`
- `Collection.featuredAsset`
- `Collection.customFields.sourceCategoryKey`
- `Collection.customFields.sourceCategoryPath`
- `Collection.customFields.sourceCategoryLevel`

### 商品页

- `Product.id`
- `Product.name`
- `Product.slug`
- `Product.featuredAsset`
- `Product.assets`
- `Product.customFields.targetAudience`
- `Product.customFields.cEndFeaturedAsset`
- `Variant.customFields.salesUnit`
- `Variant.customFields.bPrice`
- `Variant.customFields.conversionRate`
- `Variant.customFields.sourceProductId`
- `Variant.customFields.sourceType`

## 四、批次 1 完成标准

完成批次 1 时，应满足：

- 分类页结构定稿
- 分类切换规则定稿
- 商品页基础字段清单定稿
- 多单位最小交互定稿
- 明确哪些字段 backend 已有，哪些还要补 API 装配

## 五、下一步

批次 1 完成后，直接进入：

- [miniapp-ui-plan.md](./miniapp-ui-plan.md) 中的批次 2
- 图片与客群分流模型
