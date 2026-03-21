# Admin Legacy Compatibility Matrix

`mrtang-pim` 后台现在以 `/_/mrtang-admin/*` 为唯一主入口。  
旧的 `/_/source-review-workbench*` 与 `/_/procurement-workbench*` 只作为兼容入口保留。

## 主结论

- 新页面入口统一使用 SPA 壳子：
  - `/_/mrtang-admin`
  - `/_/mrtang-admin/target-sync`
  - `/_/mrtang-admin/backend-release`
  - `/_/mrtang-admin/source/*`
  - `/_/mrtang-admin/procurement*`
  - `/_/mrtang-admin/audit`
- 旧 GET 页面入口不再渲染旧 SSR 页面，只跳转到新 SPA 页面。
- 旧 POST 动作入口仍保留，但现在应复用与新后台相同的处理链，避免新旧行为漂移。

## GET 兼容入口

| 旧入口 | 当前行为 | 新入口 |
| --- | --- | --- |
| `/_/source-review-workbench` | 303 跳转 | `/_/mrtang-admin/source/products` |
| `/_/source-review-workbench/product?id=...` | 303 跳转 | `/_/mrtang-admin/source/products/detail?id=...` |
| `/_/source-review-workbench/asset?id=...` | 303 跳转 | `/_/mrtang-admin/source/assets/detail?id=...` |
| `/_/procurement-workbench` | 303 跳转 | `/_/mrtang-admin/procurement` |
| `/_/mrtang-admin/target-sync/run?id=...` | 303 跳转 | `/_/mrtang-admin/target-sync?id=...` |

## POST 兼容入口

这些旧入口仍保留，原因只有两个：

- 历史书签 / 表单 action 还可能引用旧路径
- 某些 PocketBase 管理页面或旧文档可能还在跳这些地址

当前它们都应满足：

- 最终跳回新后台页面
- 复用与新后台相同的共享 helper 或带审计服务方法

### Source Review Workbench

仍保留的旧 POST 前缀：

- `/_/source-review-workbench/product/*`
- `/_/source-review-workbench/products/*`
- `/_/source-review-workbench/assets/*`
- `/_/source-review-workbench/supplier-products/sync`

当前收口状态：

- 商品单条动作：已统一到共享 helper
- 商品批量动作：已统一到共享 helper
- 图片单条/批量动作：已统一到共享 helper
- `supplier-products/sync`：已统一到共享 helper

### Procurement Workbench

仍保留的旧 POST 前缀：

- `/_/procurement-workbench/order/status`
- `/_/procurement-workbench/order/export`
- `/_/procurement-workbench/order/review`

当前收口状态：

- 已统一走带审计的新采购链路
- 不再使用旧的无审计服务方法

## 下线边界

当前阶段不建议立即删除 legacy POST，原因是还缺一轮真实访问日志确认。

建议顺序：

1. 先继续保留 legacy GET/POST 兼容入口
2. 观察是否还有真实请求命中旧路径
3. 只有在确认无人使用后，再逐步删除对应 legacy POST

## 开发约束

后续新增后台能力时应遵循：

- 页面入口只加在 `/_/mrtang-admin/*`
- 旧 `/_/source-review-workbench*` / `/_/procurement-workbench*` 不再新增新功能
- 如必须保留旧入口，只允许做“跳转或复用新 helper”，不允许再写一套独立业务逻辑
