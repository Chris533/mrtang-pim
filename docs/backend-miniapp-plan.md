# Backend 与小程序发布前规划

这份文档基于 [external-supplier-strategy.md](./external-supplier-strategy.md) 和当前 `mrtang-pim` 已实现能力，目标不是立刻同步到 backend，而是先明确：

- backend 还缺哪些能力
- 小程序 UI 还缺哪些能力
- `source -> backend` 正式发布链还缺哪些环节
- 哪些必须先做，哪些可以延后

## 当前结论

当前系统已经能完成：

- 从目标站抓取分类、商品规格、图片资产到 PocketBase
- 在 `source_products / source_assets` 中审核、处理、加入发布队列
- 把基础商品同步到 Vendure backend

但这更接近“基础商品同步能力”，还不等于文档里那套完整的“外部供应商商品经营体系”。

如果现在直接正式同步到 backend，风险主要有：

- backend 字段不够，后续会重复同步和返工
- 多单位商品在 backend 的库存和销售语义还不完整
- B/C 端图片和可见性还没真正分流
- 采购、履约、风控信息没有完整沉淀到 backend

所以推荐顺序是：

1. 先完善 backend 目标模型
2. 再完善小程序 UI 目标交互
3. 再定义正式发布链路
4. 最后才做正式同步

## 当前联调建议

在真正开始同步到 `mrtang-backend` 前，当前建议先做一轮最小联调：

1. 先在 `/_/mrtang-admin/backend-release` 查看 `最小分类映射样例`
2. 先保存 1 到 3 个 source category -> backend collection/path 映射
3. 再从同一页的 `联调候选商品` 里挑 1 到 3 个商品看 payload 预览
4. 确认以下字段都符合预期后，再做真实同步：
   - `supplierCode`
   - `supplierCostPrice`
   - `conversionRate`
   - `sourceProductId`
   - `sourceType`
   - `targetAudience`
   - `cEndFeaturedAsset`

## 一、当前已具备能力

### 1. 抓取入库

当前已经可以抓取并落库：

- 分类树到 `source_categories`
- 商品与规格到 `source_products`
- 图片资产到 `source_assets`

### 2. source 审核与加入发布队列

当前已经可以：

- 审核商品
- 下载原图
- 处理图片
- 加入发布队列，写入 `supplier_products`
- 同步基础商品到 Vendure

### 3. 当前 Vendure 同步范围

当前实际同步的核心字段仍然偏基础：

- 名称
- slug
- 描述
- SKU
- C 端价格
- B 端价格
- 默认库存
- 销售单位
- 主图

当前已经进入 Vendure `customFields` 的只有：

- `salesUnit`
- `bPrice`

这意味着很多策略文档要求的字段，当前还停留在 PocketBase 或根本未建模。

## 二、Backend 必补能力

这一组是正式同步前最优先要补的。

### 1. 商品来源与成本字段

必须补齐：

- `supplierCode`
- `supplierCostPrice`
- `sourceProductId`
- `sourceType`

目的：

- 区分外部供应商商品和其他商品
- 支持毛利计算与异常预警
- 为采购汇总和履约追踪提供基础数据

### 2. 多单位与库存换算

必须补齐：

- `conversionRate`
- 基础库存单位
- 销售单位
- 默认下单单位

目的：

- 支撑“袋/件”等多单位真实售卖
- 避免只展示多单位，但库存仍按单一 SKU 粗暴扣减

如果 backend 仍然只有单一库存语义，现在同步过去后，多单位商品会在库存与下单逻辑上埋雷。

### 3. B/C 端展示分流

必须补齐：

- `targetAudience`
- `cEndFeaturedAsset`

目的：

- 实现 `ALL / B_ONLY / C_ONLY`
- B 端保留箱图/进货参考图
- C 端使用精修图或重绘图

### 4. 履约与仓位

建议尽早补齐：

- `StockLocation`
- 自有仓与供应商虚拟仓区分
- 商品履约来源标记

目的：

- 为后续“自有库存 / 外部供应商库存”混合销售打基础
- 避免商品同步后看起来都一样，但履约逻辑无法区分

### 5. 分类映射

必须补齐：

- source category -> backend collection/category 映射
- 顶级分类和叶子分类的映射策略
- 分类变更后的更新策略

目的：

- 不让 source 分类树落库后停在 PocketBase，而无法稳定发布到 backend 分类体系

## 三、小程序 UI 必补能力

这部分需要在同步前先定形，否则 backend 补了字段，前端也接不住。

### 1. 分类页

需要明确：

- 一级、二级、三级分类如何展示
- 分类商品列表是否分页
- 分类切换是否保留筛选状态
- B/C 端是否共用同一分类体系

### 2. 商品页

需要明确：

- 多单位切换方式
- 默认单位展示规则
- 袋/件价格对比方式
- 整件更优惠是否突出展示
- B 端是否使用平铺规格下单
- C 端是否使用弹层切规格

### 3. 图片策略

需要明确：

- 哪张图是 B 端主图
- 哪张图是 C 端主图
- 是否允许无处理直接发布
- 原图、处理图、C 端图的优先级

### 4. 可见性与客群分流

需要明确：

- 哪些商品仅 B 端可见
- 哪些商品仅 C 端可见
- 未登录用户默认按哪类客群看
- backend 与小程序如何共用这套过滤规则

### 5. 下单与库存提示

需要明确：

- 多单位加入购物车时如何展示单位与数量
- 结算页是否保留单位换算说明
- 库存不足时按基础单位还是销售单位提示

## 四、正式发布链路必补能力

这部分决定“什么时候才算真正同步到 backend”。

### 1. 发布前状态机

建议明确分层：

- `imported`
- `approved`
- `promoted`
- `publish_ready`
- `published`
- `publish_error`

当前 `source -> supplier_products -> synced` 对基础同步够用，但对“已加入发布队列但未正式发布”还不够清楚。

### 2. 分类先行还是商品先行

建议顺序：

1. 先发布分类映射
2. 再发布商品
3. 最后补图片和展示字段

否则商品先上 backend 后，分类和展示位可能还是错的。

### 3. 图片发布策略

需要明确：

- 主图是否默认用处理图
- B 端图和 C 端图是否分开写
- 无处理图时是否允许先用原图发布
- 图片失败是否阻塞商品发布

### 4. 失败与回写

需要明确：

- backend 发布失败如何重试
- 是否保留上一次成功状态
- backend product/variant/asset id 如何回写
- 分类发布失败是否阻塞商品发布

## 五、风控与运营必补能力

这是策略文档里最容易后补，但实际最容易出事故的一组。

### 1. 价格波动预警

建议 backend 或 PIM 里明确支持：

- 当采购价高于售价阈值时预警
- 阈值可配置，例如 `80%`
- 预警商品禁止自动发布或禁止自动履约

### 2. 断货与异常履约

建议支持：

- 外部供应商无货时的人工处理状态
- 退款/替换/延迟处理标记
- 客服兜底备注

### 3. 审计与追责

当前 source 侧已有部分审计，但正式发布前还应补：

- backend 发布日志
- 发布人
- 驳回原因
- 重试记录

## 六、建议实施顺序

### 批次 A：Backend 最小必要字段

先做：

- `supplierCode`
- `supplierCostPrice`
- `conversionRate`
- `targetAudience`
- `cEndFeaturedAsset`
- source category -> backend 分类映射

这是正式同步前的最低门槛。

### 批次 B：小程序目标交互定稿

先定清楚：

- 分类页结构
- 多单位商品页交互
- B/C 端图片和可见性策略
- 下单与库存展示规则

### 批次 C：正式发布链路

再做：

- source 到 backend 的正式发布 payload
- 分类先发、商品后发、图片补发
- 发布状态机与失败重试

### 批次 D：风控与运营

最后做：

- 价格波动预警
- 断货异常处理
- 发布审计
- 采购与履约联动

## 七、是否现在就同步到 backend

当前建议是：

- 可以做“小范围基础试同步”
- 不建议立即做“正式全量同步”

更合理的是：

1. 先按本规划补 backend 和 UI 缺口
2. 再选一小批商品做联调同步
3. 联调通过后再正式推进全量同步

## 八、下一步建议

下一步最合理的是直接进入“批次 A：Backend 最小必要字段”，一次完成：

- backend/Vendure 扩展字段清单
- `supplier_products -> Vendure` 字段映射表
- 分类映射方案
- 图片字段发布方案

做完这一步，再进入小程序 UI 定稿会更顺。
