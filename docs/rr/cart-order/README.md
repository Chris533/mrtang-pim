# cart-order rr 样本说明

`docs/rr/cart-order` 保留原始购物车、地址和下单抓包样本；运行时不直接读取这里，而是先脱敏整理到 `datasets/miniapp/cart-order`。

当前样本目录：

- `1/`
  一条从“加入购物车 -> 购物车结算 -> 添加收货地址 -> 提交订单待支付”的完整链路

当前已整理进 dataset 的核心样本映射：

- `[5195]` `wx/cart/addCart`
- `[5197]` `wx/cart/list`
- `[5215]` `wx/cart/change_cart_num`
- `[5216]` `wx/cart/settle`
- `[5221]` `order/get_default_delivery`
- `[5224]` `wx/cart/detail`
- `[5225]` `freight/cost` 预估场景
- `[5232]` `order/get_deliverys`
- `[5265]` `address/analyse_address`
- `[5266]` `order/add_delivery`
- `[5269]` `freight/cost` 已选配送方式场景
- `[5276]` `wx/sale_bill/save`

整理原则：

- `docs/rr/**` 只保留原始报文文本
- `datasets/miniapp/cart-order` 才是 `snapshot` 模式读取的数据源
- `customerId`、`cartId`、`addressId`、`phone`、`openId` 等敏感值在 dataset 中必须脱敏

关于重复请求：

- 当前 `docs/rr/cart-order/1` 中重复的 `request_*.txt` 已去重
- 保留了无对应 request 的少量 `response_*.txt` 归档，便于后续补链路时对照
