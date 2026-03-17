package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

func renderMrtangAdminHTML(pageData mrtangAdminPageData) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mrtang Admin</title>
  <style>
    :root {
      --bg: #06131f;
      --bg-soft: #0d1c2b;
      --card: rgba(10, 24, 38, 0.78);
      --card-strong: rgba(8, 20, 32, 0.92);
      --ink: #ecf6ff;
      --muted: #8aa3bb;
      --line: rgba(123, 168, 203, 0.16);
      --accent: #5ee6ff;
      --accent-strong: #22b8cf;
      --accent-soft: rgba(94, 230, 255, 0.14);
      --violet: #8c7bff;
      --danger: #ff6b8a;
      --ok: #6ef2b4;
      --warning: #ffd166;
      --shadow: 0 24px 60px rgba(0, 0, 0, 0.32);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top right, rgba(94,230,255,.18) 0, transparent 24%),
        radial-gradient(circle at left top, rgba(140,123,255,.16) 0, transparent 28%),
        linear-gradient(180deg, #07111b 0%, var(--bg) 54%, #04101b 100%);
      font-family: "Segoe UI", "PingFang SC", sans-serif;
    }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 28px 24px 40px; }
    .hero { display: grid; grid-template-columns: 1.4fr .8fr; gap: 16px; margin-bottom: 18px; }
    .panel {
      background: linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--card-strong) 100%);
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 18px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(16px);
    }
    h1, h2, h3, p { margin: 0; }
    h1 { font-size: 34px; letter-spacing: -.04em; }
    h2 { font-size: 18px; margin-bottom: 12px; letter-spacing: -.02em; }
    h3 { font-size: 15px; }
    p.lead { margin-top: 8px; color: var(--muted); line-height: 1.6; max-width: 72ch; }
    .actions, .grid, .mini-grid { display: grid; gap: 12px; }
    .actions { grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); margin-top: 16px; }
    .grid { grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); margin-top: 18px; }
    .mini-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
    .metric-grid { display:grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; }
    .ops-grid { display:grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px; margin-top: 16px; }
    .link-card, .stat {
      display: block;
      background: linear-gradient(180deg, rgba(16,35,53,.88) 0%, rgba(9,23,36,.94) 100%);
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 14px 16px;
      text-decoration: none;
      color: inherit;
      box-shadow: inset 0 1px 0 rgba(255,255,255,.04);
      position: relative;
      overflow: hidden;
    }
    .link-card::before, .stat::before {
      content: "";
      position: absolute;
      inset: 0 auto auto 0;
      width: 100%;
      height: 2px;
      background: linear-gradient(90deg, transparent 0%, rgba(94,230,255,.8) 22%, rgba(140,123,255,.7) 100%);
      opacity: .9;
    }
    .link-card:hover {
      border-color: rgba(94,230,255,.42);
      transform: translateY(-2px);
      transition: .18s ease;
      box-shadow: 0 18px 34px rgba(5,12,20,.32);
    }
    .eyebrow { font-size: 11px; letter-spacing: .14em; text-transform: uppercase; color: var(--accent); }
    .title { margin-top: 8px; font-weight: 700; letter-spacing: -.01em; }
    .desc { margin-top: 7px; color: var(--muted); font-size: 13px; line-height: 1.5; }
    .metric { font-size: 30px; font-weight: 800; margin-top: 10px; letter-spacing: -.04em; }
    .accent { color: var(--accent); }
    .table-wrap { overflow: auto; border-radius: 16px; }
    table { width: 100%; border-collapse: separate; border-spacing: 0; min-width: 720px; }
    th, td { padding: 12px 8px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-size: 11px; text-transform: uppercase; letter-spacing: .12em; position: sticky; top: 0; background: rgba(7,17,27,.96); backdrop-filter: blur(12px); z-index: 1; }
    .badge {
      display: inline-block;
      border-radius: 999px;
      padding: 5px 10px;
      font-size: 11px;
      font-weight: 700;
      background: var(--accent-soft);
      color: var(--accent);
      border: 1px solid rgba(94,230,255,.16);
    }
    .badge.ok { background: rgba(110,242,180,.12); color: var(--ok); border-color: rgba(110,242,180,.18); }
    .badge.danger { background: rgba(255,107,138,.12); color: var(--danger); border-color: rgba(255,107,138,.18); }
    .badge.warning { background: rgba(255,209,102,.12); color: var(--warning); border-color: rgba(255,209,102,.18); }
    .muted { color: var(--muted); }
    .small { font-size: 12px; }
    .error {
      margin-top: 12px;
      padding: 12px 14px;
      border-radius: 14px;
      background: rgba(255,107,138,.1);
      color: var(--danger);
      border: 1px solid rgba(255,107,138,.2);
      font-size: 13px;
    }
    .inline-list { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
    .pill {
      display:inline-block;
      border:1px solid var(--line);
      background:rgba(10,26,40,.82);
      border-radius:999px;
      padding:6px 10px;
      font-size:12px;
      color:var(--muted);
    }
    .section-title { display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:12px; }
    code { font-family: Consolas, monospace; background: rgba(255,255,255,.06); padding: 1px 5px; border-radius: 6px; color: var(--accent); }
    @media (max-width: 860px) {
      .hero { grid-template-columns: 1fr; }
      .mini-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <section class="panel">
        <div class="eyebrow">Mrtang Admin</div>
        <h1>统一后台入口</h1>
        <p class="lead">这里不改 PocketBase 原生前端，把 PIM、采购工作台、Miniapp source coverage、目标 API backlog 和常用联调入口集中放在一个 Admin 扩展页里。</p>
        <div class="actions">
          {{if .CanAccessSource}}<a class="link-card" href="/_/mrtang-admin/target-sync"><div class="eyebrow">目标同步</div><div class="title">目标站同步</div><div class="desc">分类树、子分类与后续商品规格同步入口。</div></a>{{end}}
          {{if .CanAccessProcurement}}<a class="link-card" href="/_/mrtang-admin/procurement"><div class="eyebrow">采购模块</div><div class="title">采购工作台</div><div class="desc">复核、导出 CSV、手工推进到已下单 / 已收货。</div></a>{{end}}
          {{if .CanAccessSource}}<a class="link-card" href="/_/mrtang-admin/source"><div class="eyebrow">源数据模块</div><div class="title">源数据管理模块</div><div class="desc">进入源数据模块首页，再分流到商品、图片和日志。</div></a>{{end}}
          <a class="link-card" href="/_/mrtang-admin/audit"><div class="eyebrow">统一审计</div><div class="title">审计与追踪</div><div class="desc">汇总源数据与采购动作，统一回查。</div></a>
          <a class="link-card" href="/_/#/collections/procurement_orders"><div class="eyebrow">PocketBase</div><div class="title">采购单集合</div><div class="desc">直接查看 <code>procurement_orders</code> 记录。</div></a>
          <a class="link-card" href="/_/#/collections/supplier_products"><div class="eyebrow">PocketBase</div><div class="title">供应商商品</div><div class="desc">审核标题、分类、价格和图片处理结果。</div></a>
          <a class="link-card" href="/_/#/collections/category_mappings"><div class="eyebrow">PocketBase</div><div class="title">分类映射</div><div class="desc">维护上游原始类目到业务类目的映射。</div></a>
        </div>
        {{if .ProcurementError}}<div class="error">采购数据加载失败：{{.ProcurementError}}</div>{{end}}
      </section>
      <aside class="panel">
        <h2>采购概览</h2>
        <div class="mini-grid">
          <div class="stat"><div class="eyebrow">未完成</div><div class="metric">{{.Procurement.OpenOrderCount}}</div></div>
          <div class="stat"><div class="eyebrow">风险</div><div class="metric accent">{{.Procurement.OpenRiskyOrders}}</div></div>
          <div class="stat"><div class="eyebrow">最近</div><div class="metric">{{len .Procurement.RecentOrders}}</div></div>
        </div>
        <div class="grid">
          <a class="link-card" href="/api/pim/healthz"><div class="eyebrow">健康检查</div><div class="title">PIM 健康检查</div><div class="desc">快速确认服务已正常启动。</div></a>
          <a class="link-card" href="/api/miniapp/contracts/homepage"><div class="eyebrow">源接口</div><div class="title">Miniapp 契约总览</div><div class="desc">查看源站 API 到本地接口映射。</div></a>
        </div>
      </aside>
    </div>

    {{if .FlashMessage}}<div class="panel"><div class="badge ok">Success</div><p class="lead">{{.FlashMessage}}</p></div>{{end}}
    {{if .FlashError}}<div class="panel"><div class="badge danger">Error</div><p class="lead">{{.FlashError}}</p></div>{{end}}

    <section class="panel">
      <div class="section-title">
        <h2>Miniapp Source Coverage</h2>
        <span class="badge {{sourceModeClass .Miniapp.SourceMode}}">{{sourceModeLabel .Miniapp.SourceMode}}</span>
      </div>
      {{if .MiniappError}}
      <div class="error">Miniapp 数据加载失败：{{.MiniappError}}</div>
      {{else}}
      <div class="metric-grid">
        <div class="stat"><div class="eyebrow">Contracts</div><div class="metric">{{.Miniapp.ContractCount}}</div><div class="desc">Dataset source: {{blank .Miniapp.DatasetSource "-"}}</div></div>
        <div class="stat"><div class="eyebrow">Homepage</div><div class="metric">{{.Miniapp.HomepageSectionCount}}</div><div class="desc">{{.Miniapp.HomepageProductCount}} 个首页商品</div></div>
        <div class="stat"><div class="eyebrow">Category Tree</div><div class="metric">{{.Miniapp.CategoryTopLevelCount}}</div><div class="desc">{{.Miniapp.CategoryNodeCount}} 个分类节点</div></div>
        <div class="stat"><div class="eyebrow">Category Sections</div><div class="metric">{{.Miniapp.CategorySectionCount}}</div><div class="desc">{{.Miniapp.CategorySectionWithProducts}} 个带商品</div></div>
        <div class="stat"><div class="eyebrow">Products</div><div class="metric">{{.Miniapp.ProductTotal}}</div><div class="desc">{{.Miniapp.ProductRRDetailCount}} rr_detail / {{.Miniapp.ProductSkeletonCount}} skeleton</div></div>
        <div class="stat"><div class="eyebrow">Checkout</div><div class="metric">{{.Miniapp.OrderOperationCount}}</div><div class="desc">{{.Miniapp.CartOperationCount}} cart / {{.Miniapp.FreightScenarioCount}} freight</div></div>
      </div>
      <div class="inline-list">
        <span class="pill">sourceMode: <code>{{blank .Miniapp.SourceMode "snapshot"}}</code></span>
        <span class="pill">sourceURL: <code>{{blank .Miniapp.SourceURL "-"}}</code></span>
        <span class="pill">multiUnitVisible: <code>{{.Miniapp.MultiUnitTotal}}</code></span>
        <span class="pill">categoryProducts: <code>{{.Miniapp.CategoryProductCount}}</code></span>
      </div>
      {{if .Miniapp.FirstBatch}}
      <div class="grid">
        {{range .Miniapp.FirstBatch}}
        <a class="link-card" href="/api/miniapp/product-page/product?id={{.ProductID}}">
          <div class="eyebrow">{{priorityLabel .Priority}}</div>
          <div class="title">{{.Name}}</div>
          <div class="desc">{{.ProductID}} · {{.UnitCount}} 个单位 · {{.SourceType}}</div>
        </a>
        {{end}}
      </div>
      {{end}}
      {{end}}
    </section>

    <section class="panel">
      <div class="section-title">
        <h2>Source Capture</h2>
        <span class="badge">PocketBase Collections</span>
      </div>
      {{if .SourceError}}
      <div class="error">Source capture 数据加载失败：{{.SourceError}}</div>
      {{else}}
      <div class="metric-grid">
        <div class="stat"><div class="eyebrow">Categories</div><div class="metric">{{.SourceCapture.CategoryCount}}</div></div>
        <div class="stat"><div class="eyebrow">Products</div><div class="metric">{{.SourceCapture.ProductCount}}</div><div class="desc">{{.SourceCapture.ImportedCount}} imported / {{.SourceCapture.ApprovedCount}} approved / {{.SourceCapture.PromotedCount}} promoted</div></div>
        <div class="stat"><div class="eyebrow">Assets</div><div class="metric">{{.SourceCapture.AssetCount}}</div><div class="desc">{{.SourceCapture.ProcessedAssetCount}} processed / {{.SourceCapture.FailedAssetCount}} failed</div></div>
        <div class="stat"><div class="eyebrow">Bridge</div><div class="metric">{{.SourceCapture.LinkedCount}}</div><div class="desc">{{.SourceCapture.SyncedCount}} synced / {{.SourceCapture.SyncErrorCount}} error</div></div>
      </div>
      {{if .CanAccessSource}}<div class="section-title" style="margin-top:16px;">
        <h3>Source Operations</h3>
        <span class="badge warning">Workbench Entry</span>
      </div>
      <div class="ops-grid">
        <a class="link-card" href="/_/mrtang-admin/target-sync">
          <div class="eyebrow">Target Sync</div>
          <div class="title">目标站同步</div>
          <div class="desc">登记同步任务、执行分类树同步并查看差异。</div>
        </a>
        <a class="link-card" href="/_/mrtang-admin/source/products?productStatus=imported">
          <div class="eyebrow">Review</div>
          <div class="title">待审批商品</div>
          <div class="desc">直接打开 imported 商品列表，执行 Approve / Reject。</div>
        </a>
        <a class="link-card" href="/_/mrtang-admin/source/assets?assetStatus=failed">
          <div class="eyebrow">Assets</div>
          <div class="title">失败图片处理</div>
          <div class="desc">查看失败图片、单图详情和批量重处理入口。</div>
        </a>
        <a class="link-card" href="/_/mrtang-admin/source/products?syncState=error">
          <div class="eyebrow">Sync</div>
          <div class="title">同步失败重试</div>
          <div class="desc">查看 linked sync error 商品并执行 Retry Sync。</div>
        </a>
        <a class="link-card" href="/_/mrtang-admin/source">
          <div class="eyebrow">Workbench</div>
          <div class="title">Source 模块页</div>
          <div class="desc">进入 source 模块首页，再分流到 products、assets 和 logs。</div>
        </a>
      </div>
      <div class="actions" style="margin-top:16px;">
        <form method="post" action="/_/mrtang-admin/source/import"><input type="hidden" name="scope" value="all"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">Import</div><div class="title">抓取并保存分类、商品、图片</div><div class="desc">从当前 miniapp dataset 导入到 <code>source_categories</code>、<code>source_products</code>、<code>source_assets</code>。</div></button></form>
        <form method="post" action="/_/mrtang-admin/source/import"><input type="hidden" name="scope" value="categories"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">Import</div><div class="title">仅刷新分类</div><div class="desc">更新目标分类树到 <code>source_categories</code>。</div></button></form>
        <form method="post" action="/_/mrtang-admin/source/import"><input type="hidden" name="scope" value="products"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">Import</div><div class="title">仅刷新商品与规格</div><div class="desc">更新商品、规格、多单位价格到 <code>source_products</code>。</div></button></form>
        <form method="post" action="/_/mrtang-admin/source/import"><input type="hidden" name="scope" value="assets"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">Import</div><div class="title">仅刷新图片资产</div><div class="desc">更新封面、轮播、详情图到 <code>source_assets</code>。</div></button></form>
        <form method="post" action="/_/mrtang-admin/source/assets/process-pending"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">AI Assets</div><div class="title">批量处理待处理图片</div><div class="desc">对 <code>source_assets</code> 中 pending/failed 的图片执行 AI 处理。</div></button></form>
        <form method="post" action="/_/mrtang-admin/source/products/promote-approved"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">Approve Bridge</div><div class="title">推送已审批商品到同步链</div><div class="desc">把 <code>source_products.review_status=approved</code> 的商品桥接到 <code>supplier_products</code>。</div></button></form>
        <form method="post" action="/_/mrtang-admin/supplier-products/sync"><button type="submit" class="link-card" style="width:100%; text-align:left; cursor:pointer;"><div class="eyebrow">Sync</div><div class="title">同步已批准商品到 Backend</div><div class="desc">触发现有 <code>supplier_products -> synced</code> 同步流程。</div></button></form>
      </div>
      <div class="grid">
        <a class="link-card" href="/_/#/collections/source_categories"><div class="eyebrow">PocketBase</div><div class="title">目标分类集合</div><div class="desc">可直接审阅目标分类树落库结果。</div></a>
        <a class="link-card" href="/_/#/collections/source_products"><div class="eyebrow">PocketBase</div><div class="title">目标商品集合</div><div class="desc">包含商品详情、默认价格、单位选项和多单位规格 JSON。</div></a>
        <a class="link-card" href="/_/#/collections/source_assets"><div class="eyebrow">PocketBase</div><div class="title">目标图片集合</div><div class="desc">保存封面、轮播、详情图，并预留单图 AI 处理状态。</div></a>
      </div>
      {{else}}<div class="error">当前账号没有源数据模块权限，已隐藏源数据操作入口。</div>{{end}}
      {{if .RecentActions}}
      <div class="section-title" style="margin-top:16px;">
        <h3>最近操作</h3>
        <span class="badge">统一日志</span>
      </div>
      <div class="table-wrap"><table>
        <thead><tr><th>模块</th><th>动作</th><th>目标</th><th>状态</th><th>操作人</th><th>说明</th><th>时间</th></tr></thead>
        <tbody>
        {{range .RecentActions}}
          <tr>
            <td><span class="badge">{{.Domain}}</span></td>
            <td><strong>{{.Label}}</strong></td>
            <td><div class="small muted">{{.Target}}</div></td>
            <td><span class="badge {{if eq .Status "success"}}ok{{else}}danger{{end}}">{{.Status}}</span></td>
            <td class="muted">{{.Actor}}</td>
            <td class="muted">{{.Message}}{{if .Note}}<br><span class="small muted">备注: {{.Note}}</span>{{end}}</td>
            <td class="small muted">{{.Created}}</td>
          </tr>
        {{end}}
        </tbody>
      </table></div>
      {{end}}
      {{end}}
    </section>

    <section class="panel">
      <h2>Target API Backlog</h2>
      <div class="table-wrap"><table>
        <thead><tr><th>模块</th><th>状态</th><th>当前情况</th><th>说明</th><th>操作</th></tr></thead>
        <tbody>
        {{range .Miniapp.Backlog}}
          <tr>
            <td><strong>{{.Area}}</strong></td>
            <td><span class="badge {{statusBadgeClass .Status}}">{{.Status}}</span></td>
            <td>{{.Summary}}</td>
            <td class="muted">{{.Detail}}</td>
            <td>{{if .ActionPath}}<a href="{{.ActionPath}}">{{.ActionLabel}}</a>{{end}}</td>
          </tr>
        {{else}}
          <tr><td colspan="5" class="muted">暂无 miniapp backlog 数据。</td></tr>
        {{end}}
        </tbody>
      </table></div>
    </section>

    <section class="panel">
      <h2>最近采购单</h2>
      {{if not .CanAccessProcurement}}<div class="error">当前账号没有采购模块权限，已隐藏采购模块操作入口。</div>{{end}}
      <div class="table-wrap"><table>
        <thead><tr><th>外部单号</th><th>状态</th><th>商品数</th><th>成本</th><th>风险</th><th>最近说明</th></tr></thead>
        <tbody>
        {{range .Procurement.RecentOrders}}
          <tr>
            <td><div><strong>{{.ExternalRef}}</strong></div><div class="small muted">{{.ID}}</div></td>
            <td><span class="badge {{statusClass .Status .RiskyItemCount}}">{{.Status}}</span></td>
            <td>{{.ItemCount}} / {{printf "%.2f" .TotalQty}}</td>
            <td>{{printf "%.2f" .TotalCostAmount}}</td>
            <td>{{if gt .RiskyItemCount 0}}<span class="badge danger">{{.RiskyItemCount}} risky</span>{{else}}<span class="badge ok">normal</span>{{end}}</td>
            <td class="muted">{{.LastActionNote}}</td>
          </tr>
        {{else}}
          <tr><td colspan="6" class="muted">暂无采购单。可先调用 <code>POST /api/pim/procurement/orders</code> 创建草稿单。</td></tr>
        {{end}}
        </tbody>
      </table></div>
    </section>

    <section class="panel">
      <h2>Quick Actions</h2>
      <div class="actions">
        {{range .QuickActions}}
        <a class="link-card" href="{{.Href}}"><div class="eyebrow">{{.Eyebrow}}</div><div class="title">{{.Title}}</div><div class="desc">{{.Desc}}</div></a>
        {{end}}
      </div>
    </section>

    <p class="lead small">Generated at {{.GeneratedAt}}</p>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("mrtang-admin").Funcs(template.FuncMap{
		"statusClass": func(status string, risky int) string {
			switch status {
			case pim.ProcurementStatusReceived:
				return "ok"
			case pim.ProcurementStatusCanceled:
				return "danger"
			default:
				if risky > 0 {
					return "danger"
				}
				return ""
			}
		},
		"statusBadgeClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case "done":
				return "ok"
			case "partial":
				return "warning"
			case "pending":
				return "danger"
			default:
				return ""
			}
		},
		"sourceModeClass": func(mode string) string {
			if strings.EqualFold(strings.TrimSpace(mode), "http") {
				return "ok"
			}
			return "warning"
		},
		"sourceModeLabel": func(mode string) string {
			if strings.EqualFold(strings.TrimSpace(mode), "http") {
				return "HTTP Source"
			}
			return "Snapshot Source"
		},
		"priorityLabel": func(priority string) string {
			switch strings.TrimSpace(priority) {
			case "homepage_dual_unit":
				return "Homepage Dual Unit"
			case "category_dual_unit":
				return "Category Dual Unit"
			case "visible_single_unit":
				return "Visible Single Unit"
			case "done_rr_detail":
				return "Done RR Detail"
			default:
				return priority
			}
		},
		"blank": func(value string, fallback string) string {
			return blankFallback(value, fallback)
		},
		"actionLabel": sourceActionTypeLabel,
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData); err != nil {
		return fmt.Sprintf("<pre>render mrtang admin failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "dashboard", "后台总览", "总览、待办、异常和模块入口。这里不承担完整批量操作。")
}
