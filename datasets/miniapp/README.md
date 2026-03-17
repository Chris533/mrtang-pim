# miniapp datasets

`datasets/miniapp` 是本地 `snapshot` 模式使用的标准化数据目录。

结构约定：

- `homepage/`
  首页聚合数据，按 `meta/contracts/bootstrap/settings/template/category-tabs/sections` 拆分。
- `category-page/`
  分类页聚合数据，按 `meta/contracts/context/tree/sections` 拆分。
- `product-page/`
  商品页聚合数据，按 `meta/contracts/products` 拆分；每个商品文件内部再拆 detail/pricing/package/context。
- `cart-order/`
  购物车和下单聚合数据，按 `meta/contracts/cart/order` 拆分；`order.json` 内继续按地址、运费、提交订单场景分组。
- `*/sections/*.json`
  商品分组单独存文件，便于按分类或模块增量维护。
- `product-page/products/*.json`
  商品页按 `spuId_skuId` 单独存文件，便于增量补样本。

约束：

- `docs/rr/**` 只保留原始抓包样本，不参与运行时读取。
- `snapshot` 模式只从 `datasets/miniapp/**` 装配 dataset。
- `raw` 模式直接请求目标站原始 API，并在本项目内标准化成与本目录同构的数据结构。
