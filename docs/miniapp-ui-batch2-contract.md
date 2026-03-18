# 小程序 UI 批次 2 Contract

这份文档对应 [miniapp-ui-plan.md](./miniapp-ui-plan.md) 的“批次 2：图片与客群分流”。

目标是把两件最容易返工的规则定死：

- B/C 端图片使用优先级
- `targetAudience` 在列表页、详情页和查询层的过滤语义

## 一、设计目标

这批不是为了增加更多字段，而是为了让已经补进 backend 的字段真正有明确用法：

- `Product.featuredAsset`
- `Product.customFields.cEndFeaturedAsset`
- `Product.customFields.targetAudience`
- `Product.assets`

如果这部分不先定稿，后面会出现：

- backend 已经有图，但前端不知道 B/C 各看哪张
- `targetAudience` 已有，但列表和详情过滤不一致
- 商品相册已同步，但不同端口径不一致

## 二、B/C 图片使用规则

### 1. B 端主图

当前定义：

- B 端主图默认使用 `featuredAsset`

原因：

- `featuredAsset` 当前发布策略更接近供应商原图
- 对 B 端进货参考更直观

### 2. C 端主图

当前定义：

- C 端主图优先使用 `customFields.cEndFeaturedAsset`
- 若为空，则回退到 `featuredAsset`

这条规则与 [external-supplier-strategy.md](./external-supplier-strategy.md) 一致。

### 3. 相册规则

当前定义：

- Product gallery 先共用 backend `assets`
- 批次 2 不再拆 B/C 双相册
- B/C 都读同一套 `assets`
- 主图区分只发生在首图

原因：

- 当前 backend 已支持 product gallery
- 先保证主图分流，能覆盖主要业务差异
- 双相册不是当前最小必要能力

### 4. 无处理图时的回退

当前定义：

- 如果 `cEndFeaturedAsset` 为空：
  - C 端详情页回退到 `featuredAsset`
  - 不阻塞商品展示

也就是说：

- 是否允许“无处理图直接发布”
  当前答案是：允许
- 但 C 端会优先吃精修图，只在没有精修图时回退

## 三、客群过滤规则

### 1. 枚举语义

- `ALL`
  - B/C 都可见
- `B_ONLY`
  - 仅 B 端可见
- `C_ONLY`
  - 仅 C 端可见

### 2. 列表页过滤规则

#### B 端用户

列表页显示：

- `ALL`
- `B_ONLY`

默认隐藏：

- `C_ONLY`

#### C 端用户或未登录用户

列表页显示：

- `ALL`
- `C_ONLY`

默认隐藏：

- `B_ONLY`

### 3. 商品详情页过滤规则

当前定义：

- 详情页必须复用与列表页一致的 `targetAudience` 规则
- 不允许列表可见、详情页不可见
- 也不允许列表不可见、详情页直接透出

也就是说：

- 详情页不是“宽松模式”
- 而是沿用同一份访问口径

### 4. 分类页统计规则

当前定义：

- 分类商品数应按当前客群过滤后的可见商品数统计
- 不应把 `B_ONLY` 商品数量直接暴露给 C 端

## 四、前端页面级 contract

### 分类列表卡片

最低显示：

- 主图
- 标题
- 零售价
- 默认销售单位
- 若是 B 端：
  - 可额外显示进货参考信息

### 商品详情页

#### B 端

- 首图：`featuredAsset`
- 相册：`assets`
- 价格区：
  - 可显示 `businessPrice`
  - 可显示单位换算说明

#### C 端

- 首图：`cEndFeaturedAsset || featuredAsset`
- 相册：`assets`
- 价格区：
  - 优先零售价
  - 单位信息收敛到可售默认单位

## 五、API / backend 最低字段清单

### Product 级

- `id`
- `name`
- `slug`
- `featuredAsset`
- `assets`
- `customFields.targetAudience`
- `customFields.cEndFeaturedAsset`

### Variant 级

- `price`
- `customFields.bPrice`
- `customFields.salesUnit`
- `customFields.conversionRate`

### Session / user 上下文

前端还需要一个明确上下文：

- 当前是否 B 端用户
- 当前是否 C 端用户/游客

否则 `targetAudience` 无法正确消费。

## 六、批次 2 完成标准

完成批次 2 时，应满足：

- B 端主图规则定稿
- C 端主图规则定稿
- 相册是否共用定稿
- `targetAudience` 在列表页和详情页的过滤规则定稿
- 分类页数量口径定稿

## 七、下一步

批次 2 完成后，进入：

- [miniapp-ui-plan.md](./miniapp-ui-plan.md) 中的批次 3
- 购物车与结算 contract
