# 小程序 UI 目标模型规划

这份文档承接 [backend-miniapp-plan.md](./backend-miniapp-plan.md) 的第三部分，目标是先把“小程序最终要如何展示、筛选、下单”定清楚，再继续推进正式批量发布。

当前判断：

- 抓取入库链：已经基本可用，可继续收尾
- backend 发布链：已经基本可用，可继续收尾
- 小程序 UI 目标模型：应当成为当前主优先级

当前 P0 收口项：

- 商品列表或详情页出现价格为 `0`
- 列表页没有明确显示多规格/多单位价格
- 点击详情后提示“商品不存在或当前客户群不可见”，但缺少 `slug/id` fallback 与友好空态

## 一、为什么现在先做 UI

如果不先定 UI contract，后面会反复返工：

- backend 字段已经补了，但前端不知道如何消费
- 多单位商品已经能发布，但商品页和购物车不知道如何展示
- `targetAudience` 和 B/C 图已经有字段，但前端规则还没定
- 分类已经能发布到 backend，但分类页、商品列表和 breadcrumb 还没有最终交互约束

所以当前不是先继续深挖抓取，而是先把“前端到底要什么数据、怎么展示”定清楚。

## 二、当前输入条件

当前已经具备的输入能力：

- 已保存分类树
- 已保存分类商品来源
- 已抓取商品规格、多单位、价格、图片
- backend 已支持：
  - `targetAudience`
  - `cEndFeaturedAsset`
  - `supplierCode`
  - `supplierCostPrice`
  - `conversionRate`
  - `sourceProductId`
  - `sourceType`

这意味着现在已经足够制定 UI contract，不必等所有后台尾项全部完成。

## 三、目标拆分

### A. 分类页模型

先定：

- 一级、二级、三级分类如何展示
- 是否默认展开二级
- 商品列表切换分类时，是否保留筛选与排序
- 分类 breadcrumb 如何展示
- 分类页默认读 backend `Collection`，还是读 source 预览

当前建议：

- 导航树采用 backend `Collection`
- 顶部或侧边保留一级分类
- 分类页主体默认展示当前分类直接商品
- 商品同时属于目标分类与必要父级，但前端默认只按当前 `Collection` 查询

本批次交付物：

- 分类页 contract
- 分类对象字段清单
- 分类切换与 breadcrumb 规则

### B. 商品页与多单位模型

先定：

- 商品页默认单位
- 多单位切换位置
- 单位换算文案
- B 端是否显示“件/袋/箱”梯度价格
- C 端是否只展示默认零售单位

当前建议：

- 商品页默认展示“默认销售单位”
- 若存在多单位，显示单位切换器
- B 端展示单位换算和进货参考价
- C 端只展示零售价和可售默认单位

本批次交付物：

- 商品页 contract
- 多单位字段清单
- B/C 商品页差异规则

### C. 图片与客群分流模型

先定：

- B 端主图来源
- C 端主图来源
- 相册是否同源
- `targetAudience=ALL/B_ONLY/C_ONLY` 如何在前端过滤

当前建议：

- B 端主图：`featuredAsset`
- C 端主图：`customFields.cEndFeaturedAsset || featuredAsset`
- 相册先共用 backend product gallery
- 商品列表与详情统一按 `targetAudience` 过滤

本批次交付物：

- 图片使用优先级
- 客群过滤规则
- 列表页与详情页的过滤约束

### D. 购物车与结算模型

先定：

- 多单位加入购物车后如何展示数量
- 结算页是否展示换算说明
- 库存不足是按基础单位还是销售单位提示
- B/C 端价格与单位在结算页如何区分

当前建议：

- 购物车行项目显示：销售单位数量 + 基础单位换算提示
- 结算页保留单位说明
- 库存不足提示以销售单位为主，同时附基础单位说明

本批次交付物：

- 购物车 contract
- 结算 contract
- 库存提示规则

## 五、购物车 backlog

这份 backlog 只聚焦“购物车是否像成熟电商一样稳定经营用户意图”，按当前差距、目标能力和建议顺序整理。

| 当前差距 | 目标能力 | 落地顺序 |
| --- | --- | --- |
| 只能在结算前重新校验，购物车本身不自愈 | 购物车行级实时更新、失效项灰置、自动提示变更 | P0 |
| 没有价格快照与变更可追溯信息 | 记录加购价、预检价和当前价差异 | P0 |
| 规格变更、库存不足、已下架只在提交时集中拦截 | 行级明确标注原因，并给出处理建议 | P0 |
| 游客车、登录车、多端合并能力不完整 | 跨端同步、合并和冲突处理 | P0 |
| 结算页只做结果展示，促销重算能力有限 | 满减、券、阶梯价、赠品实时重算 | P1 |
| 缺少批量编辑购物车能力 | 批量改数量、移除、切换规格 | P1 |
| 缺少失效商品替换建议 | 自动推荐替代项或相似商品 | P1 |
| 缺少长期保价和价格保护窗口 | 指定时间内锁定可结算价格 | P2 |
| 缺少购物车操作日志 | 可追溯每次失效、改价、移除原因 | P2 |
| 缺少推荐和凑单优化 | 智能凑单、推荐排序、A/B 测试 | P2 |

## 六、实施顺序

### 批次 1：分类页 + 商品页基础 contract

一次完成：

- 分类页结构
- breadcrumb 规则
- 商品页基础字段
- 多单位切换基础规则

### 批次 2：图片与客群分流

一次完成：

- B/C 图优先级
- `targetAudience` 前端过滤规则
- 列表页与详情页统一过滤语义

本批次的初版草案见：

- [miniapp-ui-batch2-contract.md](./miniapp-ui-batch2-contract.md)

### 批次 3：购物车与结算 contract

一次完成：

- 加购单位展示
- 结算页换算提示
- 库存与价格提示规则

本批次的初版草案见：

- [miniapp-ui-batch3-contract.md](./miniapp-ui-batch3-contract.md)

### 批次 4：回写到 backend / API contract

一次完成：

- 把 UI 需要的字段清单映射回 backend 和 miniapp API
- 确定哪些字段必须补，哪些已有字段可直接复用
- 对齐正式发布链的 payload

本批次的初版草案见：

- [miniapp-ui-batch4-contract.md](./miniapp-ui-batch4-contract.md)

## 七、当前建议的执行方式

建议不要一开始就写大量前端代码，而是先按批次完成：

1. 页面级 contract
2. 字段级 contract
3. 再决定是否改 backend 查询/API

因为当前最缺的是“目标交互和目标字段定义”，不是某个具体按钮。

## 八、下一步

当前已经完成到：

- [miniapp-ui-batch1-contract.md](./miniapp-ui-batch1-contract.md)
- [miniapp-ui-batch2-contract.md](./miniapp-ui-batch2-contract.md)
- [miniapp-ui-batch3-contract.md](./miniapp-ui-batch3-contract.md)
- [miniapp-ui-batch4-contract.md](./miniapp-ui-batch4-contract.md)
- [miniapp-ui-implementation-backlog.md](./miniapp-ui-implementation-backlog.md)

当前先收这组 P0：

1. 修价格为 `0` 的 fallback
2. 列表页和详情页明确显示多规格价格
3. 统一详情页 `slug/id` fallback 与“不可见/不存在”空态

下一步建议直接进入：

### backend 查询与 API 装配实现

一次完成：

- 分类页 `Collection` 查询整理
- 商品详情页 `Product + Variant + customFields` 查询整理
- 多单位字段聚合查询
- `targetAudience` 过滤落到实际查询
