# Source Review Workbench

`Source Review Workbench` 是 `mrtang-pim` 里承接“目标分类 / 商品 / 图片”可视化采集后的审核工作台。

入口：

- `/_/mrtang-admin`
- `/_/mrtang-admin/source`
- `/_/mrtang-admin/source/products`
- `/_/mrtang-admin/source/assets`
- `/_/mrtang-admin/source/logs`
- `/_/source-review-workbench`
- `/_/source-review-workbench/product?id=...`
- `/_/source-review-workbench/asset?id=...`

说明：

- 新后台默认使用模块化入口
- `/_/source-review-workbench` 现在是兼容保留的统一工作台
- 推荐优先从 `/_/mrtang-admin/source` 进入

## 作用边界

这条链路负责：

- 把 miniapp source 数据导入 PocketBase
- 承接 `target-sync` 同步后的变更结果
- 在 `source_products` 中审核商品
- 在 `source_assets` 中处理图片
- 把已审核商品桥接到 `supplier_products`
- 触发同步到 backend / Vendure

这条链路不负责：

- 直接替代 `supplier_products` 的正式同步状态机
- 暴露源站鉴权 token 或原始敏感请求
- 支付、履约或真实供应商自动推单

## PocketBase 集合

- `source_categories`
  保存目标分类树和分类路径
- `source_products`
  保存目标商品、多单位价格、详情、上下文和审核状态
- `source_assets`
  保存商品封面、轮播、详情图和图片处理状态

上游同步任务集合：

- `target_sync_jobs`
- `target_sync_runs`

## 状态术语

### 商品审核状态

`source_products.review_status`

- `imported`
  已导入，待人工审核
- `approved`
  已审核通过，待桥接到 `supplier_products`
- `promoted`
  已桥接到 `supplier_products`
- `rejected`
  人工拒绝，不进入同步链

推荐流转：

`imported -> approved -> promoted`

补充流转：

- `imported -> rejected`
- `approved -> rejected`

### 图片处理状态

`source_assets.image_processing_status`

- `pending`
  待处理
- `processing`
  正在处理
- `processed`
  已生成处理图
- `failed`
  处理失败，可重试

推荐流转：

`pending -> processing -> processed`

失败重试：

`failed -> processing -> processed`

## 和目标同步的关系

`target-sync` 是 source 审核流的前置入口。

推荐顺序：

1. 在 `/_/mrtang-admin/target-sync` 执行分类、商品规格、图片同步
2. 同步结果落入：
   - `source_categories`
   - `source_products`
   - `source_assets`
3. 再在 source 模块里审核和处理

### 商品与规格变更回写

当目标同步发现这些字段变化时，商品会自动回到 `imported`：

- 标题
- 分类
- 默认单位
- 单位数量
- 默认价格
- 资产数量

含义：

- 已经审核过的商品，如果目标站有变更，也必须重新审核

### 图片变更回写

当目标同步发现这些字段变化时，图片会自动回到 `pending`：

- 图片地址
- 图片角色
- 排序

同时会清空原处理结果，再重新进入图片处理流程。

## 推荐操作流程

1. 在 `/_/mrtang-admin/target-sync` 执行目标同步
2. 在 `/_/mrtang-admin/source/products` 先筛 `imported` 商品
3. 检查标题、分类、多单位价格、缩略图
4. 审核通过后执行 `Approve`
5. 对待处理或失败图片优先在 `/_/mrtang-admin/source/assets` 执行 `Process` 或 `批量重处理失败图片`
6. 对确认可上线的商品执行 `Promote` 或 `Promote & Sync`
7. 回到列表确认 bridge / sync 状态

## 列表页能力

`/_/mrtang-admin/source/products` 当前支持：

- 商品筛选
  - `productStatus`
  - `syncState`
  - 文本检索
- 批量操作
  - `Approve`
  - `Reject`
  - `Promote`
  - `Promote & Sync`
- 分页
- 失败同步重试

`/_/mrtang-admin/source/assets` 当前支持：

- 图片筛选
  - `assetStatus`
  - 文本检索
- 批量操作
  - `Process Selected`
  - `批量处理待处理图片`
  - `批量重处理失败图片`
- 分页
- 图片失败原因聚合

`/_/mrtang-admin/source/logs` 当前支持：

- source action logs 筛选
  - `actionType`
  - `status`
  - `targetType`
  - 文本检索
- 分页

## 详情页能力

### 商品详情

`/_/mrtang-admin/source/products/detail?id=...`

可查看：

- 商品摘要
- 定价
- 多单位 `unitOptions`
- `orderUnits`
- 详情、包装、上下文
- source sections
- bridge / sync 状态

可执行：

- `Approve`
- `Reject`
- `Promote`
- `Promote & Sync`
- `Retry Sync`

### 图片详情

`/_/mrtang-admin/source/assets/detail?id=...`

可查看：

- 原图 / 处理图对比
- 当前处理状态
- 最近失败原因
- source payload

可执行：

- `Process / Reprocess`

## 和正式同步链的关系

`source_products` 是审核前台。

真正进入 backend / Vendure 的仍然是 `supplier_products`：

1. `source_products` 审核通过
2. `Promote` 写入或更新 `supplier_products`
3. `supplier_products` 进入既有同步流程
4. 同步结果回写到 source workbench 的 bridge / sync 展示区

## 回归检查建议

每次改这条链路，至少检查这几件事：

1. `/_/mrtang-admin` 能导入 source 数据
2. `/_/mrtang-admin/source`、`products`、`assets`、`logs` 能正常打开
3. products 页批量 `Approve / Promote / Retry Sync` 不报错
4. assets 页批量 `Process / Reprocess` 不报错
5. 商品详情和图片详情页可打开
6. `go test ./...` 通过
