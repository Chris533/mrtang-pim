# 小程序 UI 实现 Backlog

这份文档承接：

- [miniapp-ui-batch1-contract.md](./miniapp-ui-batch1-contract.md)
- [miniapp-ui-batch2-contract.md](./miniapp-ui-batch2-contract.md)
- [miniapp-ui-batch3-contract.md](./miniapp-ui-batch3-contract.md)
- [miniapp-ui-batch4-contract.md](./miniapp-ui-batch4-contract.md)

目标不是再写原则，而是把下一阶段真正要改的代码任务列成实现 backlog。

## 一、当前判断

按前 4 批 contract 对照下来：

- backend schema：大部分已经够
- 真正的缺口主要在：
  - backend 查询装配
  - miniapp API 对齐
  - `targetAudience` 查询过滤落地
  - 多单位聚合查询

所以现在最合理的不是继续补 schema，而是开始补“查询与 API 装配”。

## P0 收口项

这组问题优先级高于外围体验优化：

- 商品列表或详情页价格显示为 `0`
- 商品列表没有明确显示多规格价格
- 点击详情后提示“商品不存在或当前客户群不可见”，但缺少 `slug/id` fallback 与友好空态

需要在前端装配层一次收掉：

- price fallback 优先取非零默认价、单位价、业务价
- 列表页和详情页统一显示多规格价格摘要
- 详情页在 `slug` 查询失败但存在 `id` 时，回退旧商品查询

## 二、代码任务拆分

### 批次 A：backend 分类页查询

目标：

- 给正式小程序分类页提供可直接消费的 `Collection` 查询

需要改：

- `mrtang-backend`
  - shop API 查询装配
  - 返回：
    - `id`
    - `name`
    - `slug`
    - `breadcrumbs`
    - `parentId`
    - `featuredAsset`
    - `customFields.sourceCategoryKey`
    - `customFields.sourceCategoryPath`
    - `customFields.sourceCategoryLevel`

完成标准：

- 分类页不再需要猜字段
- 能按 `Collection` 直接渲染一级/二级/三级分类

### 批次 B：backend 商品详情查询

目标：

- 给正式小程序商品页提供可直接消费的 `Product + Variant + customFields` 查询

需要改：

- `mrtang-backend`
  - shop API 商品详情查询装配
  - 返回：
    - `featuredAsset`
    - `assets`
    - `customFields.targetAudience`
    - `customFields.cEndFeaturedAsset`
    - `Variant.customFields.salesUnit`
    - `Variant.customFields.bPrice`
    - `Variant.customFields.conversionRate`
    - `Variant.customFields.sourceProductId`
    - `Variant.customFields.sourceType`

完成标准：

- 商品页不再只拿原生 Vendure 裸字段
- B/C 图规则可直接落地

### 批次 C：`targetAudience` 查询过滤

目标：

- 让列表页和详情页真正按客群过滤

需要改：

- `mrtang-backend`
  - shop API 列表查询增加客群上下文
  - 详情查询复用同样过滤口径

当前目标规则：

- B 端：
  - `ALL + B_ONLY`
- C 端/游客：
  - `ALL + C_ONLY`

完成标准：

- 列表页和详情页过滤一致
- 不再只是文档约定

### 批次 D：多单位聚合查询

目标：

- 让商品页、购物车、结算页真正拿到可用的多单位结构

需要改：

- 优先在 `mrtang-pim / miniapp API` 侧整理统一结构
- 再决定 `mrtang-backend` 是否直接承接完整 `unitOptions/orderUnits`

当前重点：

- `defaultUnit`
- `unitOptions`
- `orderUnits`
- `conversionRate`
- 价格字段映射

完成标准：

- 商品页和购物车都能按同一份多单位 contract 渲染

## 三、推荐执行顺序

### 1. 先做 backend 分类页查询

原因：

- 分类页结构已定
- backend `Collection` 字段已齐
- 查询装配成本低，收益高

### 2. 再做 backend 商品详情查询

原因：

- 商品详情比分类页复杂
- 但图片、价格、customFields 已基本齐

### 3. 再做 `targetAudience` 过滤

原因：

- 需要和用户上下文一起定
- 但 schema 已具备

### 4. 最后做多单位聚合查询

原因：

- 这是最复杂的一块
- 需要同时考虑 backend 与 miniapp API 的对齐

## 四、下一步建议

下一步最合理的是直接进入：

### 批次 A：backend 分类页查询

一次完成：

- 分类页最小 shop API 查询
- breadcrumb/featuredAsset/customFields 一起对齐
- 明确返回结构，作为正式小程序分类页第一版接口
