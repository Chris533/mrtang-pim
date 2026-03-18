# Mrtang Admin

`Mrtang Admin` 是 `mrtang-pim` 在 PocketBase 原生 Admin 之外提供的扩展后台。

当前无构建前端壳子的依赖已经收进项目本地静态资源，不再依赖外部 CDN。

当前后台结构已经从单页工作台调整为模块化入口：

- `/_/mrtang-admin`
  - 后台总览
  - 总览、待办、异常和高频入口
  - 默认使用无构建前端壳子异步加载数据，基础摘要和 miniapp raw 实时摘要分块加载，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/target-sync`：抓取入库
  - 源站抓取入库
  - 分类树、商品规格、图片资产的统一抓取入库入口
  - 默认使用无构建前端壳子异步加载基础摘要、raw 实时摘要、checkout 矩阵、最近写操作和当前运行进度；raw 超时只影响局部区块，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/target-sync/run?id=...`：抓取运行详情
  - 抓取入库运行详情
  - 查看单次抓取入库到底改了什么，以及阶段日志和进度
- `/_/mrtang-admin/source`
  - 源数据首页
  - source 模块首页，负责 products / assets / logs 分流
  - 默认使用无构建前端壳子异步加载数据，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/source/products`
  - 源数据商品
  - 商品审核、选中项批量审核、桥接、桥接并同步、同步重试
  - 默认使用无构建前端壳子异步加载列表，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/source/products/detail?id=...`
  - 商品详情
  - 默认使用无构建前端壳子异步加载详情，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/source/assets`
  - 源数据图片
  - 原图下载、图片处理、选中图片批量任务、失败重试、批量进度、单图详情、原图失败/处理失败筛选
  - 默认使用无构建前端壳子异步加载列表，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/source/asset-jobs`
  - 图片任务
  - 原图下载和图片处理的历史任务、选中图片任务、失败重试、任务详情
  - 默认使用无构建前端壳子异步加载列表
- `/_/mrtang-admin/source/asset-jobs/detail?id=...`
  - 图片任务详情
  - 查看任务进度、失败信息和最近日志
  - 默认使用无构建前端壳子异步加载详情
- `/_/mrtang-admin/source/assets/detail?id=...`
  - 图片详情
  - 默认使用无构建前端壳子异步加载详情，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/source/logs`
  - 源数据日志
  - source action logs 查询与失败追踪
- `/_/mrtang-admin/procurement`
  - 采购首页
  - 手动采购工作台，支持筛选、分页和详情页
  - 默认使用无构建前端壳子异步加载列表和最近动作，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/procurement/detail?id=...`
  - 采购单详情
  - 查看采购单摘要与风险信息
  - 默认使用无构建前端壳子异步加载详情，可用 `?legacy=1` 临时回退旧服务端模板页
- `/_/mrtang-admin/audit`
  - 统一审计
  - 汇总 source、图片任务与 procurement 最近动作，可按模块、状态、关键词筛选

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

原图下载状态：

- `pending`
  - 待下载原图
- `downloading`
  - 下载中
- `downloaded`
  - 原图已保存
- `failed`
  - 原图下载失败

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
2. 进入 `/_/mrtang-admin/target-sync` 执行分类、商品规格、图片抓取入库
3. 再进入 `/_/mrtang-admin/source` 查看 source 模块摘要
4. 到 `/_/mrtang-admin/source/products` 处理待审核、待桥接、同步失败商品
5. 到 `/_/mrtang-admin/source/assets` 下载原图、处理失败图片和批量重试
6. 到 `/_/mrtang-admin/source/asset-jobs` 查看批量任务进度和历史
7. 到 `/_/mrtang-admin/source/logs` 追踪失败动作和最近操作

## raw 读取边界

- `总览` 和 `抓取入库` 页面会读取 raw 实时摘要，但已经拆成局部异步区块
- `源数据`、`商品`、`图片`、`任务`、`采购` 页面本身主要读取 PocketBase 已落库数据
- 会真正触发 raw 读取的，主要是这些显式动作：
  - `抓取入库`
  - `抓取并导入 source 数据`
- 因此 raw 超时通常只会影响总览/抓取入库的 live 区块，或上述显式动作，不应该再拖死 source 列表页

## 抓取入库和 source 的关系

`target-sync` 负责把目标站数据抓取并写入 source 集合：

- `source_categories`
- `source_products`
- `source_assets`

但它不会跳过审核。

抓取入库后如果发生变化：

- 商品/规格变更
  - 自动回到 `source_products.review_status=imported`
- 图片变更
  - 自动回到 `source_assets.image_processing_status=pending`

也就是说，推荐链路是：

1. `抓取入库`
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
- `/_/mrtang-admin/source/asset-jobs`
- `/_/mrtang-admin/source/logs`
