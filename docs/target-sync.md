# 源站抓取入库

`源站抓取入库` 是 `mrtang-pim` 后台里承接“目标站分类、分类商品来源、图片资产”抓取入库的模块。

入口：

- `/_/mrtang-admin/target-sync`
- `/_/mrtang-admin/target-sync?id=...`

## 作用边界

这条链路负责：

- 从当前 miniapp `Dataset` 生成源站抓取入库任务
- 抓取分类树和子分类到 `source_categories`
- 刷新分类商品来源关系
- 抓取图片资产到 `source_assets`
- 记录每次抓取运行的统计与明细
- 为 source 审核区和图片处理链准备最新来源数据

这里的“当前源站结果”指本次 raw 或 snapshot 实际成功读取到的数据，不等于历史已经入库的全部商品总量。

这条链路不负责：

- 直接跳过 source 审核，把商品直接推到 backend
- 替代 `supplier_products` 的正式同步状态机
- 保存源站敏感鉴权信息
- 在页面加载时自动执行真实写操作，例如添加地址或提交订单

## 当前抓取实体

- `category_tree`
  - 分类树和子分类
- `category_product_sources`
  - 分类下商品来源关系
- `assets`
  - 封面、轮播、详情图
  - 按当前源站结果抓取图片资产时，会优先复用已落库 `source_products` 生成图片资源，避免每次都重新等待完整 raw 分类树

## PocketBase 集合

- `target_sync_jobs`
  - 抓取任务定义
- `target_sync_runs`
  - 每次运行的结果、统计和变更明细
- `source_categories`
  - 分类抓取落库目标
- `source_products`
  - 商品审核区与图片抓取复用的商品上下文
- `source_assets`
  - 图片抓取落库目标

## 后台能力

`/_/mrtang-admin/target-sync` 当前支持：

- 页面先加载基础摘要，再分块加载 raw 实时摘要
- raw 分类树或商品摘要超时时，只影响对应区块，不再整页失败
- 保存当前源站结果抓取任务
- 按顶级分类保存任务
- 异步启动按当前源站结果或按顶级分类的：
  - 分类抓取入库
  - 分类商品来源刷新
  - 图片抓取入库
- 实时查看当前运行中的：
  - 阶段
  - 进度
  - 当前项
  - 阶段日志
- 查看抓取任务状态
- 查看最近运行结果
- 查看分类差异摘要
- 查看商品差异和图片差异统计
- 直接跳到：
  - 待审核商品
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
- 在抓数据和执行显式 cart/order 动作前，会先做 raw 登录续活
- 如果配置了 `MINIAPP_RAW_OPEN_ID`，续活会额外补做 `get_bb_auth_status`

这意味着：

- 后台打开 `target-sync`、`source`、`checkout summary` 不会自动真实下单
- 真实写操作必须由你主动调用对应接口
- 写操作失败时，应优先检查 body 是否完整、购物车和地址是否是同一登录上下文

### raw 续活相关配置

- `MINIAPP_AUTH_ACCOUNT_ID`
  - 当前 raw 请求使用的 Bearer
- `MINIAPP_RAW_OPEN_ID`
  - 可选；配置后可补做预授权状态校验
- `MINIAPP_RAW_CONTACTS_ID`
  - 建议配置；用于 raw 续活时调用 `get_login_status`
- `MINIAPP_RAW_CUSTOMER_ID`
  - 建议配置；用于 raw 续活时调用 `get_login_status`
- `MINIAPP_RAW_IS_DISTRIBUTOR`
  - 默认 `true`；应尽量和真实小程序请求一致
- `MINIAPP_RAW_WARMUP_MIN_INTERVAL`
  - raw 登录续活成功后的最小缓存窗口，默认 `30m`
- `MINIAPP_RAW_WARMUP_MAX_INTERVAL`
  - raw 登录续活成功后的最大缓存窗口，默认 `60m`

系统会在这个区间内随机选择下一次续活窗口，避免所有实例固定同一时间续活。

如果后台提示“续活登录状态失败：返回空登录数据”，通常优先检查：

- `MINIAPP_RAW_CONTACTS_ID`
- `MINIAPP_RAW_CUSTOMER_ID`
- `MINIAPP_RAW_IS_DISTRIBUTOR`

这三个值是否和当前真实小程序会话一致。

## 运行详情

`/_/mrtang-admin/target-sync?id=...` 当前支持：

- 在同一 SPA 页面内直接展开运行详情
- 查看这次运行的范围、状态、统计和最近日志

- 查看单次运行的：
  - 实体类型
  - 范围
  - 触发人
  - 当前阶段
  - 进度
  - 阶段日志
  - 新增 / 更新 / 未变 数量
  - 具体变更明细

变更明细会记录：

- `created`
  - 新增到 source 集合
- `updated`
  - 已存在记录发生变化

## 和 source 审核流的关系

抓取入库不会绕过审核，而是主动回到 source 流。

### 商品与规格变更

当这些字段发生变化时，`source_products.review_status` 会自动重置为 `imported`：

- 商品标题
- 分类
- 默认单位
- 单位数量
- 默认价格
- 资产数量

这意味着：

- 抓取入库后出现变更的商品必须重新审核
- 审核通过后状态会停在 `approved`
- 当前不再从这里继续走正式商品发布

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

完整当前 SOP 见 [product-capture-release-sop.md](./product-capture-release-sop.md)。

1. 先进入 `/_/mrtang-admin/target-sync`
2. 先抓取入库分类树
3. 再刷新分类商品来源
4. 需要时执行全量重建分类商品归属
5. 再抓取入库图片资产
6. 进入 `/_/mrtang-admin/source/products?productStatus=imported` 审核变更商品
7. 进入 `/_/mrtang-admin/source/assets?assetStatus=pending` 处理变更图片

补充说明：

- 当前正式商品价格、规格、库存、上下架同步主链是“供应商同步”
- `target-sync` 当前不再提供商品规格抓取按钮，避免和正式同步主链混用
- `target-sync` 当前主要承担分类来源核对、图片抓取和图片处理准备

## 当前限制

- 当前差异详情页重点展示的是本次运行的新增/更新明细
- 分类差异列表仍以摘要视图为主
- 还没有“只同步缺失商品 / 只同步缺失图片 / 只同步多单位商品”这种增量策略

## 回归检查建议

每次改抓取入库链路，至少检查：

1. `/_/mrtang-admin/target-sync` 可正常打开
2. 按当前源站结果抓分类可启动，并出现进度和阶段日志
3. 刷新分类商品来源可启动，并出现进度和阶段日志
4. 按当前源站结果抓图片可启动，并出现进度和阶段日志
5. 运行详情页可打开
6. 商品变更后会回到 `imported`
7. 图片变更后会回到 `pending`
8. `go test ./...` 通过
