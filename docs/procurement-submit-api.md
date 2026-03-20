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

## 常见错误

- `missing procurement order id`
- `submit procurement order failed`
- `procurement order in status received/canceled cannot be submitted`
- 供应商 connector 返回错误（写入 `results_json.message`）

