# 目标站同步

`目标站同步` 是 `mrtang-pim` 后台里承接“目标站分类、商品规格、图片资产”同步的模块。

入口：

- `/_/mrtang-admin/target-sync`
- `/_/mrtang-admin/target-sync/run?id=...`

## 作用边界

这条链路负责：

- 从当前 miniapp `Dataset` 生成目标站同步任务
- 同步分类树和子分类到 `source_categories`
- 同步商品、规格、多单位价格到 `source_products`
- 同步图片资产到 `source_assets`
- 记录每次同步运行的统计与明细
- 把变更结果重新送回 source 审核流

这条链路不负责：

- 直接跳过 source 审核，把商品直接推到 backend
- 替代 `supplier_products` 的正式同步状态机
- 保存源站敏感鉴权信息
- 在页面加载时自动执行真实写操作，例如添加地址或提交订单

## 当前同步实体

- `category_tree`
  - 分类树和子分类
- `products`
  - 商品主数据、详情、规格、多单位价格
- `assets`
  - 封面、轮播、详情图

## PocketBase 集合

- `target_sync_jobs`
  - 同步任务定义
- `target_sync_runs`
  - 每次运行的结果、统计和变更明细
- `source_categories`
  - 分类同步落库目标
- `source_products`
  - 商品和规格同步落库目标
- `source_assets`
  - 图片同步落库目标

## 后台能力

`/_/mrtang-admin/target-sync` 当前支持：

- 登记全量同步任务
- 按顶级分类登记任务
- 执行全量或按顶级分类的：
  - 分类同步
  - 商品规格同步
  - 图片同步
- 查看同步任务状态
- 查看最近运行结果
- 查看分类差异摘要
- 查看商品差异和图片差异统计
- 直接跳到：
  - 待审核商品
  - 待桥接商品
  - 待处理图片
  - 失败图片
- 查看当前模式能力矩阵：
  - `raw live`
  - `raw 只读`
  - `显式真实写入`
- 查看 checkout 来源矩阵：
  - 购物车列表、购物车详情、结算预览
  - 默认地址、地址列表、地址解析、运费试算
  - 添加地址、提交订单
  - 并显示当前实际 `contractId`
- 查看最近真实写操作结果：
  - 最近一次添加地址
  - 最近一次提交订单
  - 最近一次真实加购或改数量

## raw 模式说明

当 `MINIAPP_SOURCE_MODE=raw` 时：

- 分类树、分类商品、商品详情、购物车列表、购物车详情、结算预览会直接请求真实源站
- 默认地址、地址列表、地址解析、运费试算会走真实只读链路
- 添加地址、提交订单只会在显式调用接口并传完整 request body 时执行

这意味着：

- 后台打开 `target-sync`、`source`、`checkout summary` 不会自动真实下单
- 真实写操作必须由你主动调用对应接口
- 写操作失败时，应优先检查 body 是否完整、购物车和地址是否是同一登录上下文

## 运行详情

`/_/mrtang-admin/target-sync/run?id=...` 当前支持：

- 查看单次运行的：
  - 实体类型
  - 范围
  - 触发人
  - 新增 / 更新 / 未变 数量
  - 具体变更明细

变更明细会记录：

- `created`
  - 新增到 source 集合
- `updated`
  - 已存在记录发生变化

## 和 source 审核流的关系

目标同步不会绕过审核，而是主动回到 source 流。

### 商品与规格变更

当这些字段发生变化时，`source_products.review_status` 会自动重置为 `imported`：

- 商品标题
- 分类
- 默认单位
- 单位数量
- 默认价格
- 资产数量

这意味着：

- 同步后出现变更的商品必须重新审核
- 审核通过后，仍然要走 `approved -> promoted`

### 图片变更

当这些字段变化时，`source_assets.image_processing_status` 会自动重置为 `pending`：

- 图片地址
- 图片角色
- 排序

同时会清空已处理结果：

- `processed_image`
- `processed_image_source`
- `image_processing_error`

这意味着：

- 图片变更后必须重新进入图片处理链

## 推荐操作流程

1. 先进入 `/_/mrtang-admin/target-sync`
2. 先同步分类树
3. 再同步商品规格
4. 再同步图片资产
5. 进入 `/_/mrtang-admin/source/products?productStatus=imported` 审核变更商品
6. 进入 `/_/mrtang-admin/source/assets?assetStatus=pending` 处理变更图片
7. 审核通过后再桥接和同步到 backend

## 当前限制

- 当前差异详情页重点展示的是本次运行的新增/更新明细
- 分类差异列表仍以摘要视图为主
- 还没有“只同步缺失商品 / 只同步缺失图片 / 只同步多单位商品”这种增量策略

## 回归检查建议

每次改目标同步链路，至少检查：

1. `/_/mrtang-admin/target-sync` 可正常打开
2. 全量分类同步可执行
3. 全量商品规格同步可执行
4. 全量图片同步可执行
5. 运行详情页可打开
6. 商品变更后会回到 `imported`
7. 图片变更后会回到 `pending`
8. `go test ./...` 通过
