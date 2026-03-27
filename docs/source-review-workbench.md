# Source Review Workbench

`Source Review Workbench` 是 `mrtang-pim` 里承接“目标分类 / 商品 / 图片”可视化采集后的审核工作台。

入口：

- `/_/mrtang-admin`
- `/_/mrtang-admin/source`
- `/_/mrtang-admin/source/products`
- `/_/mrtang-admin/source/assets`
- `/_/mrtang-admin/source/asset-jobs`
- `/_/mrtang-admin/source/logs`
- `/_/source-review-workbench`
- `/_/source-review-workbench/product?id=...`
- `/_/source-review-workbench/asset?id=...`

说明：

- 新后台默认使用模块化入口
- `/_/source-review-workbench` 现在是兼容保留的统一工作台
- 推荐优先从 `/_/mrtang-admin/source` 进入
- 兼容入口矩阵见 [admin-legacy-compat-matrix.md](./admin-legacy-compat-matrix.md)

## 作用边界

这条链路负责：

- 把 miniapp source 数据导入 PocketBase
- 承接 `target-sync` 抓取入库后的变更结果
- 在 `source_products` 中审核商品
- 在 `source_assets` 中处理图片

这条链路不负责：

- 直接替代 `supplier_products` 的正式同步状态机
- 作为供应商正式商品发布入口
- 暴露源站鉴权 token 或原始敏感请求
- 支付、履约或真实供应商自动推单

## PocketBase 集合

- `source_categories`
  保存目标分类树和分类路径
- `source_products`
  保存目标商品、多单位价格、详情、上下文和审核状态
- `source_assets`
  保存商品封面、轮播、详情图、原图下载状态和图片处理状态

上游同步任务集合：

- `target_sync_jobs`
- `target_sync_runs`
- `source_asset_jobs`

## 状态术语

### 商品审核状态

`source_products.review_status`

- `imported`
  已导入，待人工审核
- `approved`
  已审核通过
- `promoted`
  历史已发布链处理，仅用于兼容历史数据查看
- `rejected`
  人工拒绝，不进入发布链

推荐流转：

`imported -> approved`

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

### 原图下载状态

`source_assets.original_image_status`

- `pending`
  待下载原图
- `downloading`
  正在下载原图
- `downloaded`
  原图已保存到 PocketBase
- `failed`
  原图下载失败，可重试

## 和抓取入库的关系

`target-sync` 是 source 审核流的前置抓取入库入口。

推荐顺序：

1. 在 `/_/mrtang-admin/target-sync` 执行分类来源和图片抓取入库
2. 同步结果落入：
   - `source_categories`
   - `source_products`
   - `source_assets`
3. 再在 source 模块里审核和处理

### 商品与规格变更回写

当抓取入库发现这些字段变化时，商品会自动回到 `imported`：

- 标题
- 分类
- 默认单位
- 单位数量
- 默认价格
- 资产数量

含义：

- 已经审核过的商品，如果目标站有变更，也必须重新审核

### 图片变更回写

当抓取入库发现这些字段变化时，图片会自动回到 `pending`：

- 图片地址
- 图片角色
- 排序

同时会清空原处理结果，再重新进入图片处理流程。

## 推荐操作流程

完整当前 SOP 见 [product-capture-release-sop.md](./product-capture-release-sop.md)。

1. 在 `/_/mrtang-admin/target-sync` 执行抓取入库
2. 在 `/_/mrtang-admin/source/products` 先筛 `imported` 商品
3. 检查标题、分类、多单位价格、缩略图
4. 审核通过后执行 `Approve`
5. 对待处理或失败图片优先在 `/_/mrtang-admin/source/assets` 执行 `Process` 或 `批量重处理失败图片`
6. 如果需要先把原图固定保存到本地，再在 `/_/mrtang-admin/source/assets` 执行 `下载原图` 或 `批量下载待下载原图`
7. 回到列表继续查看 bridge / sync 历史状态

补充说明：

- 当前 UI 已不再提供 source 商品发布按钮
- 当前 `source/products` 的定位是审核区、对照区和历史状态查看区
- 正式商品同步应改由“供应商同步”承担

## 列表页能力

`/_/mrtang-admin/source/products` 当前支持：

- 商品筛选
  - `productStatus`
  - `syncState`
  - 文本检索
- 批量操作
  - `选中项批量通过`
  - `选中项批量拒绝`
- 分页

`/_/mrtang-admin/source/assets` 当前支持：

- 图片筛选
  - `assetStatus`
  - `originalStatus`
  - `assetIds`
  - 文本检索
- 批量操作
  - `批量下载待下载原图`
  - `仅对选中图片下载原图`
  - `仅对选中图片处理`
  - `仅对选中失败图片重处理`
  - `批量处理待处理图片`
  - `批量重处理失败图片`
- 分页
- 图片失败原因聚合
- 原图批量下载进度与最近日志
- 图片批量处理进度与最近日志

`/_/mrtang-admin/source/asset-jobs` 当前支持：

- 图片任务筛选
  - `jobType`
  - `status`
  - 文本检索
- 分页
- 任务详情
- `重新执行`
- 失败项会只重跑失败图片
- 跳转到本任务图片 / 成功项 / 原图失败 / 处理失败图片
- 从活动任务直接跳详情
- 同类任务执行中时会直接接管现有进度，而不是重复启动
- 统一审计里会按“原图下载任务 / 选中图片处理任务 / 失败图片重处理任务”等中文动作显示

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

### 图片详情

`/_/mrtang-admin/source/assets/detail?id=...`

可查看：

- 源地址 / 原图文件 / 处理图对比
- 原图下载状态
- 当前处理状态
- 最近失败原因
- source payload

可执行：

- `下载原图`
- `Process / Reprocess`

### 图片任务详情

`/_/mrtang-admin/source/asset-jobs/detail?id=...`

可查看：

- 任务类型
- 当前状态
- 已处理 / 总数 / 失败数
- 当前项
- 最近任务日志
- 失败图片明细

可执行：

- `重新执行`

## 和正式发布链的关系

`source_products` 是审核前台。

`source_products` 当前主要承担审核与对照。

真正进入 backend / Vendure 的正式商品主链应优先以 `supplier_products` 为准，而当前 `supplier_products` 的正式同步入口是“供应商同步”。

这意味着：

- `source/products` 适合做人工确认和历史对照
- 图片处理结果适合做素材准备
- 日常正式商品同步不要再从这里发起

## 回归检查建议

每次改这条链路，至少检查这几件事：

1. `/_/mrtang-admin` 能导入 source 数据
2. `/_/mrtang-admin/source`、`products`、`assets`、`asset-jobs`、`logs` 能正常打开
3. products 页批量“通过 / 拒绝”不报错
4. assets 页批量 `Process / Reprocess` 不报错
5. 商品详情和图片详情页可打开
6. `go test ./...` 通过
