package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderProcurementWorkbenchHTML(summary pim.ProcurementWorkbenchSummary) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Procurement Workbench</title>
  <style>
    :root {
      --bg: #f5f1e8;
      --card: #fffaf0;
      --ink: #1d1d1b;
      --muted: #6e6259;
      --line: #d9cdbf;
      --accent: #8b3d16;
      --warning: #c27c0e;
      --danger: #b42318;
      --ok: #216e39;
    }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: "Segoe UI", "PingFang SC", sans-serif; background: linear-gradient(180deg, #efe6d7 0%, var(--bg) 100%); color: var(--ink); }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 24px; }
    .hero { display: flex; justify-content: space-between; align-items: end; gap: 16px; margin-bottom: 20px; }
    .hero h1 { margin: 0; font-size: 28px; }
    .hero p { margin: 6px 0 0; color: var(--muted); }
    .hero a { color: var(--accent); text-decoration: none; font-weight: 600; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 20px; }
    .card { background: var(--card); border: 1px solid var(--line); border-radius: 16px; padding: 14px 16px; box-shadow: 0 12px 24px rgba(29,29,27,0.04); }
    .metric { font-size: 26px; font-weight: 700; margin-top: 6px; }
    .label { color: var(--muted); font-size: 13px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; padding: 10px 8px; border-bottom: 1px solid var(--line); vertical-align: top; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    .status { display: inline-block; padding: 4px 8px; border-radius: 999px; font-size: 12px; font-weight: 700; background: #eee3d4; }
    .status.warning { background: #fff0d0; color: var(--warning); }
    .status.danger { background: #fde7e7; color: var(--danger); }
    .status.ok { background: #deefe1; color: var(--ok); }
    .actions { display: flex; flex-wrap: wrap; gap: 8px; }
    form { margin: 0; }
    button, select, input {
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 8px 10px;
      background: white;
      color: var(--ink);
      font: inherit;
    }
    button { cursor: pointer; background: var(--accent); color: white; border-color: var(--accent); }
    button.secondary { background: white; color: var(--accent); }
    .toolbar { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 14px; }
    .note { width: 180px; }
    .muted { color: var(--muted); font-size: 12px; }
    .risk { font-weight: 700; color: var(--danger); }
    .small { font-size: 12px; color: var(--muted); }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1>采购工作台</h1>
        <p>在 PocketBase Admin 内完成 review、导出、手工下单和收货推进。</p>
      </div>
      <div>
        <a href="/_/mrtang-admin">Mrtang Admin</a>
        <span class="small"> | </span>
        <a href="/_/">返回 Admin</a>
        <span class="small"> | </span>
        <a href="/_/#/collections/procurement_orders">打开 procurement_orders 集合</a>
      </div>
    </div>

    <div class="stats">
      <div class="card"><div class="label">总采购单</div><div class="metric">{{.TotalOrders}}</div></div>
      <div class="card"><div class="label">草稿</div><div class="metric">{{.DraftOrders}}</div></div>
      <div class="card"><div class="label">已复核</div><div class="metric">{{.ReviewedOrders}}</div></div>
      <div class="card"><div class="label">已导出</div><div class="metric">{{.ExportedOrders}}</div></div>
      <div class="card"><div class="label">已下单</div><div class="metric">{{.OrderedOrders}}</div></div>
      <div class="card"><div class="label">已收货</div><div class="metric">{{.ReceivedOrders}}</div></div>
      <div class="card"><div class="label">未完成风险单</div><div class="metric">{{.OpenRiskyOrders}}</div></div>
    </div>

    <div class="card">
      <div class="toolbar">
        <div class="muted">最近采购单：{{len .RecentOrders}} 条</div>
      </div>
      <table>
        <thead>
          <tr>
            <th>外部单号</th>
            <th>状态</th>
            <th>商品</th>
            <th>金额</th>
            <th>风险</th>
            <th>说明</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
        {{range .RecentOrders}}
          <tr>
            <td>
              <div><strong>{{.ExternalRef}}</strong></div>
              <div class="small">{{.ID}}</div>
            </td>
            <td><span class="status {{statusClass .Status .RiskyItemCount}}">{{.Status}}</span></td>
            <td>
              <div>{{.ItemCount}} items / {{printf "%.2f" .TotalQty}}</div>
              <div class="small">{{.SupplierCount}} suppliers</div>
            </td>
            <td>
              <div>成本 {{printf "%.2f" .TotalCostAmount}}</div>
            </td>
            <td>
              {{if gt .RiskyItemCount 0}}
              <span class="risk">{{.RiskyItemCount}} risky</span>
              {{else}}
              <span class="small">normal</span>
              {{end}}
            </td>
            <td>
              <div>{{.LastActionNote}}</div>
              <div class="small">{{.Updated}}</div>
            </td>
            <td>
              <div class="actions">
                <form method="post" action="/_/procurement-workbench/order/review">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <input class="note" type="text" name="note" placeholder="review note">
                  <button class="secondary" type="submit">Review</button>
                </form>
                <form method="post" action="/_/procurement-workbench/order/export">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <button class="secondary" type="submit">Export CSV</button>
                </form>
                <form method="post" action="/_/procurement-workbench/order/status">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <select name="status">
                    <option value="reviewed">reviewed</option>
                    <option value="exported">exported</option>
                    <option value="ordered">ordered</option>
                    <option value="received">received</option>
                    <option value="canceled">canceled</option>
                  </select>
                  <input class="note" type="text" name="note" placeholder="status note">
                  <button type="submit">Update</button>
                </form>
              </div>
            </td>
          </tr>
        {{else}}
          <tr><td colspan="7" class="muted">暂无采购单。先调用 procurement order create 接口生成草稿单。</td></tr>
        {{end}}
        </tbody>
      </table>
    </div>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("procurement-workbench").Funcs(template.FuncMap{
		"statusClass": func(status string, risky int) string {
			switch status {
			case pim.ProcurementStatusReceived:
				return "ok"
			case pim.ProcurementStatusCanceled:
				return "danger"
			default:
				if risky > 0 {
					return "warning"
				}
				return ""
			}
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, summary); err != nil {
		return fmt.Sprintf("<pre>render procurement workbench failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return builder.String()
}
