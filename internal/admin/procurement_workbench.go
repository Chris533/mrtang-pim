package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderProcurementWorkbenchHTML(summary pim.ProcurementWorkbenchSummary) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>采购工作台</title>
  <style>
    :root {
      --panel: rgba(8, 20, 32, 0.92);
      --card: rgba(10, 24, 38, 0.78);
      --ink: #edf7ff;
      --muted: #8aa3bb;
      --line: rgba(123,168,203,.16);
      --accent: #5ee6ff;
      --accent-strong: #22b8cf;
      --warning: #ffd166;
      --danger: #ff6b8a;
      --ok: #6ef2b4;
      --shadow: 0 24px 60px rgba(0,0,0,.34);
    }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: "Segoe UI", "PingFang SC", sans-serif; background: radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); color: var(--ink); }
    .wrap { max-width: 1180px; margin: 0 auto; padding: 24px; }
    .hero { display:grid; grid-template-columns:1.3fr .9fr; gap:16px; margin-bottom:20px; }
    .hero h1 { margin: 0; font-size: 30px; letter-spacing:-.04em; }
    .hero p { margin: 8px 0 0; color: var(--muted); line-height:1.6; }
    .hero a, .link { color: var(--accent); text-decoration: none; font-weight: 600; }
    .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 20px; }
    .card { background: linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border: 1px solid var(--line); border-radius: 18px; padding: 14px 16px; box-shadow: var(--shadow); backdrop-filter: blur(14px); }
    .metric { font-size: 26px; font-weight: 700; margin-top: 6px; }
    .label { color: var(--muted); font-size: 13px; }
    .layout { display:grid; grid-template-columns:minmax(0,1fr) 320px; gap:14px; }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; padding: 10px 8px; border-bottom: 1px solid var(--line); vertical-align: top; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
    .status { display: inline-block; padding: 4px 8px; border-radius: 999px; font-size: 12px; font-weight: 700; background: rgba(255,255,255,.06); border:1px solid rgba(255,255,255,.08); }
    .status.warning { background: rgba(255,209,102,.12); color: var(--warning); border-color: rgba(255,209,102,.18); }
    .status.danger { background: rgba(255,107,138,.12); color: var(--danger); border-color: rgba(255,107,138,.18); }
    .status.ok { background: rgba(110,242,180,.12); color: var(--ok); border-color: rgba(110,242,180,.18); }
    .actions { display: flex; flex-wrap: wrap; gap: 8px; }
    form { margin: 0; }
    button, select, input {
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 8px 10px;
      background: rgba(6,17,28,.86);
      color: var(--ink);
      font: inherit;
    }
    button { cursor: pointer; background: linear-gradient(135deg, var(--accent-strong) 0%, var(--accent) 100%); color: #04131d; border-color: rgba(94,230,255,.28); font-weight:700; }
    button.secondary { background: rgba(255,255,255,.04); color: var(--accent); }
    .toolbar { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 14px; }
    .toolbar label { display:flex; flex-direction:column; gap:6px; color:var(--muted); font-size:12px; }
    .note { width: 180px; }
    .muted { color: var(--muted); font-size: 12px; }
    .risk { font-weight: 700; color: var(--danger); }
    .small { font-size: 12px; color: var(--muted); }
    .log-list { display:grid; gap:10px; margin-top:14px; }
    .log-item { border:1px solid var(--line); border-radius:14px; padding:12px; background:rgba(255,255,255,.03); }
    .queue { display:grid; gap:10px; }
    .queue-item { border:1px solid var(--line); border-radius:14px; padding:12px; background:rgba(255,255,255,.03); }
    .risk-tags { display:flex; flex-wrap:wrap; gap:6px; margin-top:6px; }
    @media (max-width: 1120px) { .hero, .layout { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div class="card">
        <div class="label">采购模块</div>
        <h1>采购工作台</h1>
        <p>采购页现在完全纳入统一后台骨架，负责草稿单复核、导出、状态流转、风险识别和最近采购动作追踪。</p>
        <p class="small" style="margin-top:12px;"><a class="link" href="/_/mrtang-admin">返回总览</a> / <a class="link" href="/_/mrtang-admin/audit">打开统一审计</a> / <a class="link" href="/_/#/collections/procurement_orders">查看采购集合</a></p>
      </div>
      <div class="card">
        <div class="label">当前待办</div>
        <div class="queue" style="margin-top:12px;">
          <div class="queue-item"><strong>待复核</strong><div class="small">{{.DraftOrders}} 张草稿单需要先确认风险与数量。</div></div>
          <div class="queue-item"><strong>待导出 / 待下单</strong><div class="small">{{.ReviewedOrders}} 已复核，{{.ExportedOrders}} 已导出待采购。</div></div>
          <div class="queue-item"><strong>风险采购单</strong><div class="small">{{.OpenRiskyOrders}} 张未完成采购单含风险项，优先跟进。</div></div>
        </div>
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

    <div class="layout">
    <div class="card">
      <div class="toolbar admin-toolbar">
        <div class="muted">当前页采购单：{{len .RecentOrders}} 条</div>
        <label>状态
          <select name="status" onchange="location.href=this.value">
            <option value="{{pageURL . "" .FilterRisk}}">全部</option>
            <option value="{{pageURL . "draft" .FilterRisk}}" {{if eq .FilterStatus "draft"}}selected{{end}}>草稿</option>
            <option value="{{pageURL . "reviewed" .FilterRisk}}" {{if eq .FilterStatus "reviewed"}}selected{{end}}>已复核</option>
            <option value="{{pageURL . "exported" .FilterRisk}}" {{if eq .FilterStatus "exported"}}selected{{end}}>已导出</option>
            <option value="{{pageURL . "ordered" .FilterRisk}}" {{if eq .FilterStatus "ordered"}}selected{{end}}>已下单</option>
            <option value="{{pageURL . "received" .FilterRisk}}" {{if eq .FilterStatus "received"}}selected{{end}}>已收货</option>
            <option value="{{pageURL . "canceled" .FilterRisk}}" {{if eq .FilterStatus "canceled"}}selected{{end}}>已取消</option>
          </select>
        </label>
        <label>风险
          <select name="risk" onchange="location.href=this.value">
            <option value="{{pageURL . .FilterStatus ""}}">全部</option>
            <option value="{{pageURL . .FilterStatus "has_risk"}}" {{if eq .FilterRisk "has_risk"}}selected{{end}}>有风险</option>
            <option value="{{pageURL . .FilterStatus "loss"}}" {{if eq .FilterRisk "loss"}}selected{{end}}>亏损风险</option>
            <option value="{{pageURL . .FilterStatus "warning"}}" {{if eq .FilterRisk "warning"}}selected{{end}}>毛利预警</option>
            <option value="{{pageURL . .FilterStatus "normal"}}" {{if eq .FilterRisk "normal"}}selected{{end}}>仅正常</option>
          </select>
        </label>
        <form method="get" action="/_/mrtang-admin/procurement" style="display:flex; gap:8px; align-items:end;">
          <input type="hidden" name="status" value="{{.FilterStatus}}">
          <input type="hidden" name="risk" value="{{.FilterRisk}}">
          <label>检索
            <input type="text" name="q" value="{{.Query}}" placeholder="外部单号 / 备注">
          </label>
          <label>每页
            <select name="pageSize">
              <option value="10" {{if eq .PageSize 10}}selected{{end}}>10</option>
              <option value="20" {{if or (eq .PageSize 0) (eq .PageSize 20)}}selected{{end}}>20</option>
              <option value="50" {{if eq .PageSize 50}}selected{{end}}>50</option>
            </select>
          </label>
          <button class="secondary" type="submit">筛选</button>
        </form>
        <a class="link" href="/_/mrtang-admin/audit?domain=采购">只看采购审计</a>
      </div>
      <div style="overflow:auto; border-radius:14px;">
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
              <div class="small"><a class="link" href="/_/mrtang-admin/procurement/detail?id={{.ID}}&returnTo={{urlquery $.CurrentURL}}">查看详情</a></div>
            </td>
            <td><span class="status {{statusClass .Status .RiskyItemCount}}">{{statusLabel .Status}}</span></td>
            <td>
              <div>{{.ItemCount}} 项 / {{printf "%.2f" .TotalQty}}</div>
              <div class="small">{{.SupplierCount}} 个供应商</div>
            </td>
            <td>
              <div>成本 {{printf "%.2f" .TotalCostAmount}}</div>
            </td>
            <td>
              {{if gt .RiskyItemCount 0}}
              <span class="risk">{{.RiskyItemCount}} 个风险项</span>
              <div class="risk-tags">
                {{range riskBadges .Summary}}{{.}}{{end}}
              </div>
              {{else}}
              <span class="small">正常</span>
              {{end}}
            </td>
            <td>
              <div>{{.LastActionNote}}</div>
              <div class="small">{{.Updated}}</div>
            </td>
            <td>
              <div class="actions">
                <form method="post" action="/_/mrtang-admin/procurement/order/review" data-confirm="确认复核这张采购单吗？">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <input class="note" type="text" name="note" placeholder="复核备注">
                  <button class="secondary" type="submit">复核</button>
                </form>
                <form method="post" action="/_/mrtang-admin/procurement/order/export" data-confirm="确认导出这张采购单的 CSV 吗？">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <input class="note" type="text" name="note" placeholder="导出备注">
                  <button class="secondary" type="submit">导出 CSV</button>
                </form>
                <form method="post" action="/_/mrtang-admin/procurement/order/status" data-confirm="确认更新这张采购单的状态吗？">
                  <input type="hidden" name="id" value="{{.ID}}">
                  <select name="status">
                    <option value="reviewed">已复核</option>
                    <option value="exported">已导出</option>
                    <option value="ordered">已下单</option>
                    <option value="received">已收货</option>
                    <option value="canceled">已取消</option>
                  </select>
                  <input class="note" type="text" name="note" placeholder="状态备注">
                  <button type="submit">更新状态</button>
                </form>
              </div>
            </td>
          </tr>
        {{else}}
          <tr><td colspan="7" class="muted">暂无采购单。先调用采购单创建接口生成草稿单。</td></tr>
        {{end}}
        </tbody>
      </table>
      </div>
      <div style="display:flex; gap:10px; align-items:center; margin-top:12px;">
        {{if gt .Page 1}}<a class="link" href="{{pagerURL . (dec .Page)}}">上一页</a>{{end}}
        <span class="small">第 {{.Page}} / {{.Pages}} 页</span>
        {{if lt .Page .Pages}}<a class="link" href="{{pagerURL . (inc .Page)}}">下一页</a>{{end}}
      </div>
    </div>

    <aside class="card">
      <div class="label">采购状态说明</div>
      <div class="queue" style="margin-top:12px;">
        <div class="queue-item"><strong>草稿</strong><div class="small">刚创建的采购单，还未人工复核。</div></div>
        <div class="queue-item"><strong>已复核 / 已导出</strong><div class="small">已确认内容，等待导出或已交给采购执行。</div></div>
        <div class="queue-item"><strong>已下单 / 已收货</strong><div class="small">采购执行中或已经完成履约。</div></div>
      </div>
      {{if .RecentActions}}
      <div class="label" style="margin-top:16px;">最近采购操作</div>
      <div class="log-list">
        {{range .RecentActions}}
        <div class="log-item">
          <div><strong>{{actionLabel .ActionType}}</strong> <span class="small">{{.Created}}</span></div>
          <div class="small">{{.ExternalRef}} / {{.OrderID}}</div>
          <div class="small">状态: {{.Status}} | 操作人: {{actorLabel .ActorName .ActorEmail}}</div>
          <div class="small">{{.Message}}</div>
          {{if .Note}}<div class="small">备注: {{.Note}}</div>{{end}}
        </div>
        {{end}}
      </div>
      {{end}}
    </aside>
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
		"statusLabel": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case pim.ProcurementStatusDraft:
				return "草稿"
			case pim.ProcurementStatusReviewed:
				return "已复核"
			case pim.ProcurementStatusExported:
				return "已导出"
			case pim.ProcurementStatusOrdered:
				return "已下单"
			case pim.ProcurementStatusReceived:
				return "已收货"
			case pim.ProcurementStatusCanceled:
				return "已取消"
			default:
				return status
			}
		},
		"actionLabel": func(action string) string {
			switch strings.ToLower(strings.TrimSpace(action)) {
			case "create_order":
				return "创建采购单"
			case "export_order":
				return "导出采购单"
			case "update_status":
				return "更新采购状态"
			default:
				return action
			}
		},
		"actorLabel": func(name string, email string) string {
			name = strings.TrimSpace(name)
			email = strings.TrimSpace(email)
			if name != "" && email != "" && !strings.EqualFold(name, email) {
				return name + " / " + email
			}
			if name != "" {
				return name
			}
			if email != "" {
				return email
			}
			return "系统"
		},
		"inc": func(v int) int { return v + 1 },
		"dec": func(v int) int {
			if v <= 1 {
				return 1
			}
			return v - 1
		},
		"pageURL": func(summary pim.ProcurementWorkbenchSummary, status string, risk string) string {
			return procurementPageURL(summary, status, risk, summary.Page)
		},
		"riskBadges": func(summary pim.ProcurementSummary) []template.HTML {
			items := make([]template.HTML, 0, 2)
			if procurementSummaryHasRiskLevel(summary, "loss") {
				items = append(items, template.HTML(`<span class="status danger">亏损风险</span>`))
			}
			if procurementSummaryHasRiskLevel(summary, "warning") {
				items = append(items, template.HTML(`<span class="status warning">毛利预警</span>`))
			}
			return items
		},
		"pagerURL": func(summary pim.ProcurementWorkbenchSummary, page int) string {
			return procurementPageURL(summary, summary.FilterStatus, summary.FilterRisk, page)
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, summary); err != nil {
		return fmt.Sprintf("<pre>render procurement workbench failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "procurement", "采购工作台", "统一的采购管理页面，负责复核、导出、下单和收货推进。")
}

func procurementPageURL(summary pim.ProcurementWorkbenchSummary, status string, risk string, page int) string {
	values := url.Values{}
	if v := strings.TrimSpace(status); v != "" {
		values.Set("status", v)
	}
	if v := strings.TrimSpace(risk); v != "" {
		values.Set("risk", v)
	}
	if v := strings.TrimSpace(summary.Query); v != "" {
		values.Set("q", v)
	}
	if summary.PageSize > 0 {
		values.Set("pageSize", strconv.Itoa(summary.PageSize))
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/_/mrtang-admin/procurement?" + encoded
	}
	return "/_/mrtang-admin/procurement"
}

func procurementSummaryHasRiskLevel(summary pim.ProcurementSummary, level string) bool {
	for _, supplier := range summary.Suppliers {
		for _, item := range supplier.Items {
			if strings.EqualFold(item.RiskLevel, level) {
				return true
			}
		}
	}
	return false
}
