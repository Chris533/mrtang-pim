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
      --paper: #f4efe7;
      --ink: #1f1f1a;
      --muted: #6f665e;
      --line: #d8ccbd;
      --card: #fffaf3;
      --accent: #8d3e16;
      --accent-soft: #f2e1d3;
      --danger: #b42318;
      --ok: #216e39;
      --warning: #9a6700;
    }
    * { box-sizing: border-box; }
    body { margin: 0; color: var(--ink); background:
      radial-gradient(circle at top right, #efe2d3 0, transparent 28%),
      linear-gradient(180deg, #f8f4ee 0%, var(--paper) 100%);
      font-family: "Segoe UI", "PingFang SC", sans-serif; }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 28px 24px 40px; }
    .hero { display: grid; grid-template-columns: 1.4fr .8fr; gap: 16px; margin-bottom: 18px; }
    .panel { background: var(--card); border: 1px solid var(--line); border-radius: 18px; padding: 18px; box-shadow: 0 18px 36px rgba(31,31,26,0.05); }
    h1, h2, h3, p { margin: 0; }
    h1 { font-size: 30px; }
    h2 { font-size: 18px; margin-bottom: 12px; }
    p.lead { margin-top: 8px; color: var(--muted); line-height: 1.55; }
    .actions, .grid, .mini-grid { display: grid; gap: 12px; }
    .actions { grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); margin-top: 16px; }
    .grid { grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); margin-top: 18px; }
    .mini-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
    .metric-grid { display:grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; }
    .link-card, .stat { display: block; background: white; border: 1px solid var(--line); border-radius: 14px; padding: 14px 16px; text-decoration: none; color: inherit; }
    .link-card:hover { border-color: var(--accent); transform: translateY(-1px); transition: .16s ease; }
    .eyebrow { font-size: 12px; letter-spacing: .06em; text-transform: uppercase; color: var(--muted); }
    .title { margin-top: 6px; font-weight: 700; }
    .desc { margin-top: 6px; color: var(--muted); font-size: 13px; line-height: 1.45; }
    .metric { font-size: 28px; font-weight: 800; margin-top: 8px; }
    .accent { color: var(--accent); }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 8px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    .badge { display: inline-block; border-radius: 999px; padding: 4px 8px; font-size: 12px; font-weight: 700; background: var(--accent-soft); color: var(--accent); }
    .badge.ok { background: #deefe1; color: var(--ok); }
    .badge.danger { background: #fde7e7; color: var(--danger); }
    .badge.warning { background: #fff2cc; color: var(--warning); }
    .muted { color: var(--muted); }
    .small { font-size: 12px; }
    .error { margin-top: 12px; padding: 10px 12px; border-radius: 12px; background: #fde7e7; color: var(--danger); font-size: 13px; }
    .inline-list { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
    .pill { display:inline-block; border:1px solid var(--line); background:#fff; border-radius:999px; padding:6px 10px; font-size:12px; }
    .section-title { display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:12px; }
    code { font-family: Consolas, monospace; background: #f7efe5; padding: 1px 5px; border-radius: 6px; }
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
          <a class="link-card" href="/_/procurement-workbench"><div class="eyebrow">Procurement</div><div class="title">采购工作台</div><div class="desc">Review、导出 CSV、手工推进到 ordered / received。</div></a>
          <a class="link-card" href="/_/#/collections/procurement_orders"><div class="eyebrow">PocketBase</div><div class="title">采购单集合</div><div class="desc">直接查看 <code>procurement_orders</code> 记录。</div></a>
          <a class="link-card" href="/_/#/collections/supplier_products"><div class="eyebrow">PocketBase</div><div class="title">供应商商品</div><div class="desc">审核标题、分类、价格和图片处理结果。</div></a>
          <a class="link-card" href="/_/#/collections/category_mappings"><div class="eyebrow">PocketBase</div><div class="title">分类映射</div><div class="desc">维护上游原始类目到业务类目的映射。</div></a>
        </div>
        {{if .ProcurementError}}<div class="error">采购数据加载失败：{{.ProcurementError}}</div>{{end}}
      </section>
      <aside class="panel">
        <h2>采购概览</h2>
        <div class="mini-grid">
          <div class="stat"><div class="eyebrow">Open</div><div class="metric">{{.Procurement.OpenOrderCount}}</div></div>
          <div class="stat"><div class="eyebrow">Risky</div><div class="metric accent">{{.Procurement.OpenRiskyOrders}}</div></div>
          <div class="stat"><div class="eyebrow">Recent</div><div class="metric">{{len .Procurement.RecentOrders}}</div></div>
        </div>
        <div class="grid">
          <a class="link-card" href="/api/pim/healthz"><div class="eyebrow">Health</div><div class="title">PIM 健康检查</div><div class="desc">快速确认服务已正常启动。</div></a>
          <a class="link-card" href="/api/miniapp/contracts/homepage"><div class="eyebrow">Source</div><div class="title">Miniapp 契约总览</div><div class="desc">查看源站 API 到本地接口映射。</div></a>
        </div>
      </aside>
    </div>

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
      <h2>Target API Backlog</h2>
      <table>
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
      </table>
    </section>

    <section class="panel">
      <h2>最近采购单</h2>
      <table>
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
      </table>
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
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData); err != nil {
		return fmt.Sprintf("<pre>render mrtang admin failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return builder.String()
}
