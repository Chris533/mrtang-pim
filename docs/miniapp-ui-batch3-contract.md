# 小程序 UI 批次 3 Contract

这份文档对应 [miniapp-ui-plan.md](./miniapp-ui-plan.md) 的“批次 3：购物车与结算 contract”。

目标是把多单位商品进入购物车和结算页后的展示规则定清楚，避免后面：

- backend 有了 `conversionRate`
- 商品页支持单位切换
- 但购物车和结算页又退回单一单位视角

## 一、核心原则

### 1. 购物车与结算页必须保留销售单位视角

用户加购时选的是哪个销售单位，购物车和结算页就必须继续按这个单位展示。

不允许：

- 商品页选了“件”
- 到购物车里只剩基础单位数量

### 2. 基础单位只做说明，不抢主语义

当前建议：

- 主数量显示销售单位
- 基础单位只作为换算提示

例如：

- `3 件`
- `约 60 袋`

### 3. B/C 端共享结构，不共享强调点

当前建议：

- B/C 端购物车和结算页数据结构保持一致
- 但展示强调点不同：
  - B 端更强调单位换算、进货参考价、件装效率
  - C 端更强调零售价、默认购买单位、最终支付金额

## 二、购物车 Contract

### 1. 行项目最低字段

- `productId`
- `skuId`
- `name`
- `featuredAsset`
- `salesUnit`
- `qty`
- `unitPrice`
- `lineAmount`
- `baseUnitName`
- `unitRate`
- `hasMultiUnit`
- `stockTexts`
- `promotionTexts`

### 2. 行项目展示规则

当前建议：

- 主数量显示：
  - `qty + salesUnit`
- 若 `unitRate > 1`：
  - 追加 `约 X 基础单位`
- 若有促销：
  - 只显示简短促销标签

### 3. 数量调整规则

当前建议：

- 步进器以当前销售单位为步长
- 不允许购物车页 silently 切换单位
- 若用户想换单位，应回到商品页或弹层重新选择

### 4. B/C 端差异

#### B 端

- 可显示 `businessPrice`
- 可显示“整件更划算”或件装参考信息
- 可显示基础单位换算

#### C 端

- 主显示零售价
- 基础单位换算弱化为次级文案
- 若商品有多个单位，默认只强调当前已选单位

## 三、结算页 Contract

### 1. 结算页最低字段

- `itemCount`
- `totalQty`
- `baseUnitTotalQty`
- `totalAmount`
- `couponCount`
- `freightAmount`
- `deliveryMethodId`
- `defaultDelivery`
- `deliveries`

### 2. 金额区规则

当前建议：

- 主金额显示最终支付金额
- 运费单独显示
- 券优惠单独显示
- 不在金额区混入基础单位信息

### 3. 单位说明规则

当前建议：

- 结算页保留单位换算说明
- 但只放在商品行或说明区，不放进总计区

示例：

- `2 件，约 40 袋`

### 4. 库存不足提示规则

当前建议：

- 主提示按销售单位显示
- 若商品有换算率，则附基础单位说明

示例：

- `库存不足，仅剩 1 件`
- `约 20 袋可售`

### 5. 配送与地址

当前建议：

- 地址和配送方式与单位展示解耦
- 不因为换单位而重置地址
- 若某单位导致运费变化，只更新运费试算，不改地址状态

## 四、API 最低字段清单

这批前端需要的接口最低字段，以 [checkout-api.md](./checkout-api.md) 为基础继续收口。

### `cart/list-summary`

最低必需：

- `items[].qty`
- `items[].unitName`
- `items[].unitPrice`
- `items[].lineAmount`
- `items[].baseUnitName`
- `items[].unitRate`
- `items[].hasMultiUnit`

### `cart/detail-summary`

最低必需：

- `totalQty`
- `baseUnitTotalQty`
- `totalAmount`
- `couponCount`

### `order/freight-summary`

最低必需：

- `scenario`
- `freightAmount`
- `deliveryMethodId`

## 五、批次 3 完成标准

完成批次 3 时，应满足：

- 购物车行项目单位展示规则定稿
- 结算页金额与单位说明分层定稿
- 库存不足提示规则定稿
- B/C 端购物车与结算差异规则定稿
- API 最低字段清单定稿

## 六、下一步

批次 3 完成后，进入：

- [miniapp-ui-plan.md](./miniapp-ui-plan.md) 中的批次 4
- 回写到 backend / API contract
