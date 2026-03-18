# 小程序 UI 批次 4 Contract

这份文档对应 [miniapp-ui-plan.md](./miniapp-ui-plan.md) 的“批次 4：回写到 backend / API contract”。

目标是把前 3 批 UI contract 需要的字段，逐项映射回：

- `mrtang-backend / Vendure`
- `mrtang-pim / miniapp API`

并明确：

- 哪些字段已经有
- 哪些字段需要 API 装配
- 哪些字段仍是缺口

## 一、总原则

当前建议采用两层来源：

### 1. 上架后正式前端

优先读取：

- `mrtang-backend / Vendure`

原因：

- 这是正式发布后的商品真相来源
- 分类、商品、图片、客群都应以 backend 为准

### 2. PIM / Miniapp API

优先用于：

- 联调
- 发布前预览
- contract 验证
- 未正式发布前的灰度验证

也就是说：

- 最终小程序正式页应尽量读 backend
- `mrtang-pim` 更像预演层和接入层

## 二、批次 1 到 3 字段回写清单

### A. 分类页

#### UI 需要

- `id`
- `name`
- `slug`
- `breadcrumbs`
- `parentId`
- `featuredAsset`
- `sourceCategoryKey`
- `sourceCategoryPath`
- `sourceCategoryLevel`

#### backend 现状

当前已具备：

- `Collection.id`
- `Collection.name`
- `Collection.slug`
- `Collection.breadcrumbs`
- `Collection.parentId`
- `Collection.featuredAsset`
- `Collection.customFields.sourceCategoryKey`
- `Collection.customFields.sourceCategoryPath`
- `Collection.customFields.sourceCategoryLevel`

结论：

- backend 分类页基础字段已基本齐
- 下一步主要是查询装配，不是 schema 缺字段

#### pim / miniapp API 现状

当前已具备：

- source 分类树
- source 分类路径
- source 分类商品来源

结论：

- 作为预览层足够
- 但正式前端分类页不建议长期直接读 `source_*`

### B. 商品页

#### UI 需要

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
- `unitOptions`
- `orderUnits`

#### backend 现状

当前已具备：

- `Product.id`
- `Product.name`
- `Product.slug`
- `Product.featuredAsset`
- `Product.assets`
- `Product.customFields.targetAudience`
- `Product.customFields.cEndFeaturedAsset`
- `Variant.price`
- `Variant.customFields.bPrice`
- `Variant.customFields.salesUnit`
- `Variant.customFields.conversionRate`
- `Variant.customFields.sourceProductId`
- `Variant.customFields.sourceType`

当前仍缺或未明确：

- 一个清晰的 `unitOptions/orderUnits` 对外查询装配
- 多单位商品在 backend 查询层如何聚合展示

结论：

- schema 基本够
- 真正缺的是“商品详情查询 contract”

#### pim / miniapp API 现状

当前已具备：

- raw/snapshot 商品详情
- `unitOptions`
- `orderUnits`
- `conversionRate`
- `carousel/detail`

结论：

- PIM 侧已足够作为多单位 contract 参考
- backend 端要按这个 contract 补查询装配

### C. 图片与客群分流

#### UI 需要

- B 端主图：`featuredAsset`
- C 端主图：`cEndFeaturedAsset || featuredAsset`
- 相册：`assets`
- 客群：`targetAudience`

#### backend 现状

当前已具备：

- `featuredAsset`
- `assets`
- `customFields.cEndFeaturedAsset`
- `customFields.targetAudience`

结论：

- schema 已够
- 主要缺“前端查询与过滤规则”的最终装配

### D. 购物车与结算

#### UI 需要

- `qty`
- `salesUnit`
- `unitPrice`
- `lineAmount`
- `baseUnitName`
- `unitRate`
- `hasMultiUnit`
- `totalQty`
- `baseUnitTotalQty`
- `totalAmount`
- `couponCount`
- `freightAmount`

#### backend 现状

当前结论：

- 这部分还没有明确落到 `mrtang-backend`
- 目前更接近 `mrtang-pim/miniapp API` contract

结论：

- 购物车与结算目前仍以 `miniapp API` 作为 contract 参考
- 若后续正式接 backend 购物车/结算，需要单独设计查询与聚合结构

## 三、缺口分级

### 已有 schema，缺查询装配

- 分类页 `Collection` 查询
- 商品页 `Product + Variant + customFields` 聚合查询
- 多单位商品详情聚合
- 列表页 `targetAudience` 过滤查询

### 已有 PIM contract，backend 未完全承接

- `unitOptions`
- `orderUnits`
- 多单位切换用的完整 price/unit 结构

### 仍属未来扩展

- backend 原生购物车/结算多单位模型
- B/C 双相册
- 更复杂的价格梯度和件装优惠查询

## 四、下一步代码任务建议

### 批次 A：backend 查询装配

一次完成：

- 分类页 `Collection` 查询字段整理
- 商品详情页 `Product + Variant + customFields` 查询整理
- `targetAudience` 查询过滤约定

### 批次 B：多单位查询模型

一次完成：

- 定义 backend 如何返回 `unitOptions`
- 定义默认销售单位
- 定义 B/C 端价格字段

### 批次 C：miniapp API 对齐

一次完成：

- 把当前 `miniapp` 侧字段整理成和正式前端一致的 contract
- 明确预演层与正式 backend 查询的对应关系

## 五、批次 4 完成标准

完成批次 4 时，应满足：

- UI 需求字段与 backend 字段有一一对应表
- UI 需求字段与 miniapp API 字段有一一对应表
- 明确哪些缺口是查询装配，哪些缺口是 schema
- 能直接进入“补 backend 查询/API”的实际开发阶段
