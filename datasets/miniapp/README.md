# miniapp datasets

`datasets/miniapp` 是本地 `snapshot` 模式使用的标准化数据目录。

结构约定：

- `homepage/`
  首页聚合数据，按 `meta/contracts/bootstrap/settings/template/category-tabs/sections` 拆分。
- `category-page/`
  分类页聚合数据，按 `meta/contracts/context/tree/sections` 拆分。
- `*/sections/*.json`
  商品分组单独存文件，便于按分类或模块增量维护。

约束：

- `docs/rr/**` 只保留原始抓包样本，不参与运行时读取。
- `snapshot` 模式只从 `datasets/miniapp/**` 装配 dataset。
- `http` 模式应返回与本目录同构的标准化聚合结果。
