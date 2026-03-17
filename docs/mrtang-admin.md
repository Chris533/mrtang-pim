# Mrtang Admin

`Mrtang Admin` 是 `mrtang-pim` 在 PocketBase 原生 Admin 之外提供的扩展后台。

当前后台结构已经从单页工作台调整为模块化入口：

- `/_/mrtang-admin`
  - 后台总览
  - 总览、待办、异常和高频入口
- `/_/mrtang-admin/target-sync`
  - 目标站同步
  - 分类树、商品规格、图片资产的统一同步入口
- `/_/mrtang-admin/target-sync/run?id=...`
  - 同步运行详情
  - 查看单次同步到底改了什么
- `/_/mrtang-admin/source`
  - 源数据首页
  - source 模块首页，负责 products / assets / logs 分流
- `/_/mrtang-admin/source/products`
  - 源数据商品
  - 商品审核、桥接、同步重试
- `/_/mrtang-admin/source/assets`
  - 源数据图片
  - 图片处理、失败重试、单图详情
- `/_/mrtang-admin/source/logs`
  - 源数据日志
  - source action logs 查询与失败追踪
- `/_/mrtang-admin/procurement`
  - 采购首页
  - 手动采购工作台，支持筛选、分页和详情页
- `/_/mrtang-admin/procurement/detail?id=...`
  - 采购单详情
  - 查看采购单摘要与风险信息
- `/_/mrtang-admin/audit`
  - 统一审计
  - 汇总 source 与 procurement 最近动作，可按模块、状态、关键词筛选

## Source 统一术语

商品审核状态：

- `imported`
  - 待审核
- `approved`
  - 待桥接
- `promoted`
  - 已桥接
- `rejected`
  - 已拒绝

图片处理状态：

- `pending`
  - 待处理
- `processing`
  - 处理中
- `processed`
  - 已处理
- `failed`
  - 处理失败

桥接 / 同步状态：

- `unlinked`
  - 未桥接
- `approved` / `ready`
  - 待同步
- `synced`
  - 已同步
- `error`
  - 同步失败

## 推荐使用方式

1. 先打开 `/_/mrtang-admin` 查看总览和高频待办
2. 进入 `/_/mrtang-admin/target-sync` 执行分类、商品规格、图片同步
3. 再进入 `/_/mrtang-admin/source` 查看 source 模块摘要
4. 到 `/_/mrtang-admin/source/products` 处理待审核、待桥接、同步失败商品
5. 到 `/_/mrtang-admin/source/assets` 处理失败图片和批量重试
6. 到 `/_/mrtang-admin/source/logs` 追踪失败动作和最近操作

## 目标同步和 source 的关系

`target-sync` 负责把目标站数据同步到 source 集合：

- `source_categories`
- `source_products`
- `source_assets`

但它不会跳过审核。

同步后如果发生变化：

- 商品/规格变更
  - 自动回到 `source_products.review_status=imported`
- 图片变更
  - 自动回到 `source_assets.image_processing_status=pending`

也就是说，推荐链路是：

1. `target-sync`
2. `source products / assets`
3. `promote`
4. `supplier_products`
5. backend / Vendure

## 兼容页面

`/_/source-review-workbench` 仍然保留，用于兼容原有统一工作台入口。

但新的默认入口应当是：

- `/_/mrtang-admin/target-sync`
- `/_/mrtang-admin/source/products`
- `/_/mrtang-admin/source/assets`
- `/_/mrtang-admin/source/logs`
