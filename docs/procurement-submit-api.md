# Procurement Submit API

## 鉴权

- Header `X-PIM-API-Key: <PIM_API_KEY>`
- 若未配置 `PIM_API_KEY`，接口默认放开（不建议生产）

## 1) 创建采购单

`POST /api/pim/procurement/orders`

请求体示例：

```json
{
  "externalRef": "ORDER-0001",
  "deliveryAddress": "四川省 攀枝花市 ...",
  "notes": "vendure-order:ORDER-0001",
  "items": [
    { "supplierCode": "SUP_A", "originalSku": "683215792313163776", "quantity": 2 }
  ]
}
```

返回：`ProcurementOrder`（含 `id`, `status`）。

## 2) 提交采购单到供应商连接器

`POST /api/pim/procurement/order/submit?id=<procurementOrderId>`

请求体（可选）：

```json
{
  "note": "auto submit from backend order ORDER-0001"
}
```

返回：更新后的 `ProcurementOrder`。

## 状态流转（提交相关）

- `draft/reviewed/exported -> ordered`：至少一个供应商结果 `accepted=true`
- `draft/reviewed -> exported`：本次提交无任何 accepted（通常表示仅生成人工导出链路）
- `received/canceled`：禁止 submit

## 结果字段

- `results_json`：供应商 submit 结果数组
- `last_action_note`：提交备注
- `ordered_at/exported_at`：按状态自动更新时间

`results_json[*]` 当前推荐关注：

- `accepted`
- `externalRef`
- `message`
- `verificationStatus`
- `verificationMessage`
- `details`

其中：

- `accepted=true` 表示供应商接口已受理并返回成功结果
- `verificationStatus=verified` 表示基于 submit 返回的强校验已通过
- `verificationStatus=warning` 表示供应商已受理，但 submit 返回中的关键字段存在不一致，需人工复核，避免盲目重提

`details`（当前 `miniapp_cart_order`）会沉淀：

- `request.expectedDueAmount`
- `request.goodsAmount`
- `request.freightAmount`
- `request.lines`
- `submit.billId`
- `submit.dueAmount`
- `submit.paymentOptions`
- `detail.billId`
- `detail.billNo`
- `detail.goodsTypeCount`
- `detail.goodsCount`
- `detail.orderGoods`
- `detail.receiveAddress`
- `verification.issues`

说明：

- 当前 `miniapp_cart_order` 已在 submit 成功后继续调用 supplier `POST /gateway/billservice/api/v1/wx/sale_bill/detail` 做二次回查。
- 强校验会同时比对 submit 返回与 detail 返回；若 supplier 已受理但 detail 校验不通过，会落为 `accepted=true + verificationStatus=warning`。

## 常见错误

- `missing procurement order id`
- `submit procurement order failed`
- `procurement order in status received/canceled cannot be submitted`
- 供应商 connector 返回错误（写入 `results_json.message`）
