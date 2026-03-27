# Checkout API

> 说明：本文件是 `mrtang-pim` 的 checkout 契约/样本适配接口说明。  
> 小程序线上真实购物车与下单主链路走 `mrtang-backend` 的 shop-api GraphQL。

这份文档只描述结算页应直接依赖的本地接口，不重复讲源站抓包映射。

目标：

- 给前端一个稳定的 checkout 读取面
- 明确哪些接口是推荐依赖的 summary
- 明确哪些接口只是原始回放，主要用于排查和补样本

## 推荐读取顺序

最推荐：

1. `GET /api/miniapp/cart-order/checkout-summary`

按需拆分：

1. `GET /api/miniapp/cart-order/cart/list-summary`
2. `GET /api/miniapp/cart-order/cart/detail-summary`
3. `GET /api/miniapp/cart-order/order/default-delivery-summary`
4. `GET /api/miniapp/cart-order/order/deliveries-summary`
5. `GET /api/miniapp/cart-order/order/freight-summary`
6. `GET /api/miniapp/cart-order/order/submit-summary`

仅调试或对照原始结构时再使用：

- `GET /api/miniapp/cart-order/cart/list`
- `GET /api/miniapp/cart-order/cart/detail`
- `GET /api/miniapp/cart-order/order/default-delivery`
- `GET /api/miniapp/cart-order/order/deliveries`
- `GET /api/miniapp/cart-order/order/freight-cost`
- `POST /api/miniapp/cart-order/order/submit`

## 结算前预检

结算页在真正提交前，还会走一次 `precheckActiveOrder`。

- 如果商品价格变化，前端会提示刷新后重新确认
- 如果规格变化、库存不足或商品已下架，前端会拦截结算
- 购物车本身不会被立即清空，用户需要根据最新状态调整后再提交

这也是大多数电商的常见做法：购物车保存意向，结算时才以最新状态为准。

## 聚合接口

### `GET /api/miniapp/cart-order/checkout-summary`

用途：

- 结算页首次加载
- 一次性拿齐购物车、默认地址、地址列表、运费场景和提交预览

响应结构：

```json
{
  "cartList": {},
  "cartDetail": {},
  "defaultDelivery": {},
  "deliveries": {},
  "freight": {},
  "submit": {}
}
```

字段含义：

- `cartList`
  购物车列表摘要，适合商品清单区域
- `cartDetail`
  结算态购物车摘要，适合金额和券信息区域
- `defaultDelivery`
  当前默认地址摘要；如果原始样本为空，会自动回退到 `add_delivery`
- `deliveries`
  地址列表摘要；如果原始样本为空，会自动回退到 `add_delivery`
- `freight`
  运费试算摘要，包含 `preview` 和 `selected_delivery` 两个场景
- `submit`
  提交订单预览摘要，表示“提交后待支付”的结果包

## Summary 契约

### `cart/list-summary`

```json
{
  "varietyNum": 3,
  "itemCount": 3,
  "totalQty": 3.04,
  "totalAmount": 946.24,
  "taxAmount": 946.24,
  "items": [
    {
      "cartId": "demo-cart-2",
      "productId": "646995016921055232_646995017386622976",
      "spuId": "646995016921055232",
      "skuId": "646995017386622976",
      "name": "天府小厨娘香卤肥肠",
      "skuName": "500g*20袋",
      "unitName": "件",
      "qty": 2,
      "unitPrice": 460,
      "lineAmount": 920,
      "baseUnitName": "袋",
      "unitRate": 20,
      "hasMultiUnit": true,
      "stockTexts": ["有货"],
      "promotionTexts": ["满299-5", "满1199-10"]
    }
  ]
}
```

推荐前端依赖字段：

- `itemCount`
- `totalQty`
- `totalAmount`
- `items[].cartId`
- `items[].productId`
- `items[].qty`
- `items[].lineAmount`
- `items[].hasMultiUnit`

### `cart/detail-summary`

```json
{
  "varietyNum": 4,
  "itemCount": 4,
  "cartIds": ["demo-cart-1", "demo-cart-2"],
  "totalQty": 4.04,
  "baseUnitTotalQty": 42.04,
  "totalAmount": 959.24,
  "taxRate": 0,
  "exemptionFreight": 0,
  "couponCount": 1,
  "items": []
}
```

推荐前端依赖字段：

- `cartIds`
- `totalAmount`
- `couponCount`
- `baseUnitTotalQty`

### `order/default-delivery-summary`

```json
{
  "found": true,
  "source": "add_delivery_fallback",
  "address": {
    "addressId": "demo-address-001",
    "customerId": "demo-customer-001",
    "customerName": "演示客户",
    "phone": "13800000000",
    "fullAddress": "四川省-攀枝花市-仁和区演示地址",
    "detailAddress": "演示地址",
    "provinceName": "四川省",
    "cityName": "攀枝花市",
    "districtName": "仁和区",
    "deliveryId": "592750618322829313",
    "deliveryName": "装车",
    "isDefault": true,
    "longitude": 101.744656257685,
    "latitude": 26.523963047463
  }
}
```

说明：

- `source=default_delivery`
  表示原始默认地址接口有值
- `source=add_delivery_fallback`
  表示从 `add_delivery` 回退生成
- `source=none`
  表示当前样本里没有可用地址

### `order/deliveries-summary`

```json
{
  "count": 1,
  "defaultAddressId": "demo-address-001",
  "items": []
}
```

推荐前端依赖字段：

- `count`
- `defaultAddressId`
- `items[]`

### `order/freight-summary`

```json
{
  "scenarios": [
    {
      "scenario": "preview",
      "label": "Before delivery method selection",
      "deliveryMethodId": "",
      "customerId": "demo-customer-001",
      "qty": 42.04,
      "totalAmount": 959.24,
      "freightAmount": 0,
      "skuCount": 4
    }
  ]
}
```

推荐前端依赖字段：

- `scenarios[].scenario`
- `scenarios[].freightAmount`
- `scenarios[].deliveryMethodId`

约定：

- `preview`
  未选择配送方式的试算结果
- `selected_delivery`
  已选择配送方式后的试算结果

### `order/submit-summary`

```json
{
  "message": "请选择支付方式进行支付",
  "billId": "demo-bill-001",
  "customerId": "demo-customer-001",
  "customerName": "演示客户",
  "addressId": "demo-address-001",
  "deliveryMethodId": "592750618322829313",
  "cartIds": ["demo-cart-1", "demo-cart-2"],
  "dueAmount": 959.24,
  "freightAmount": 0,
  "requiresPayment": true,
  "deadlineTime": 1773654185111,
  "billType": 3,
  "paymentOptions": [
    {
      "name": "微信支付",
      "type": 13,
      "payRecommend": 0
    }
  ],
  "receiveAddress": {}
}
```

推荐前端依赖字段：

- `billId`
- `dueAmount`
- `requiresPayment`
- `paymentOptions`
- `receiveAddress`

## 稳定性约束

前端应优先依赖这些字段，不直接依赖原始回放结构里的深层字段：

- `cartList.items[].productId`
- `cartDetail.cartIds`
- `defaultDelivery.address`
- `deliveries.items`
- `freight.scenarios`
- `submit.billId`
- `submit.dueAmount`
- `submit.paymentOptions`

当前这些 summary 字段的目标是稳定；后面如果补更多 rr 样本，优先保持这些字段不破坏性变化。

## 已知边界

- 这套接口当前仍基于 snapshot 样本，不是实时购物车
- 支付动作本身还没纳入本地 API
- 地址摘要当前可能来自 `add_delivery` 回退，不代表真实线上“用户已有地址列表”
- `submit-summary` 表示“提交后待支付结果”，不是支付完成结果
