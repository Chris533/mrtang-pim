package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderProcurementDetailHTML(order pim.ProcurementOrder, backHref string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>采购单详情</title>
  <style>
    :root { --panel:rgba(8,20,32,.92); --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:1180px; margin:0 auto; padding:24px; }
    .card { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:18px; padding:16px; box-shadow:var(--shadow); margin-bottom:14px; }
    .layout { display:grid; grid-template-columns:minmax(0,1fr) 320px; gap:14px; }
    .risk-item { border:1px solid var(--line); border-radius:14px; padding:12px; margin-top:10px; background:rgba(255,255,255,.03); }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; font-size:12px; font-weight:700; background:rgba(255,255,255,.06); border:1px solid rgba(255,255,255,.08); }
    .badge.warning { color:#ffd166; background:rgba(255,209,102,.12); border-color:rgba(255,209,102,.18); }
    .badge.danger { color:#ff6b8a; background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    pre { margin:0; white-space:pre-wrap; word-break:break-word; font-family:Consolas,monospace; font-size:12px; line-height:1.6; }
    .small { color:var(--muted); font-size:12px; }
    a { color:#5ee6ff; text-decoration:none; }
    @media (max-width: 980px) { .layout { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1 style="margin:0;">采购单详情</h1>
      <div class="small" style="margin-top:8px;">{{.Order.ExternalRef}} / {{.Order.ID}}</div>
      <div class="small">状态: {{.Order.Status}} / 风险项: {{.Order.RiskyItemCount}}</div>
      <div class="small">商品: {{.Order.ItemCount}} / 数量: {{printf "%.2f" .Order.TotalQty}} / 成本: {{printf "%.2f" .Order.TotalCostAmount}}</div>
      <div class="small" style="margin-top:8px;"><a href="{{.BackHref}}">返回采购页</a></div>
    </div>
    <div class="layout">
      <div class="card">
        <h3 style="margin-top:0;">风险商品</h3>
        {{range riskItems .Order.Summary}}
        <div class="risk-item">
          <div><strong>{{.Title}}</strong> <span class="badge {{riskClass .RiskLevel}}">{{riskLabel .RiskLevel}}</span></div>
          <div class="small">{{.OriginalSKU}} / {{.SupplierCode}}</div>
          <div class="small">数量 {{printf "%.2f" .Quantity}} {{.SalesUnit}} / 成本 {{printf "%.2f" .CostPrice}} / C价 {{printf "%.2f" .ConsumerPrice}}</div>
        </div>
        {{else}}
        <div class="small">当前采购单没有风险商品。</div>
        {{end}}
      </div>
      <div class="card">
        <h3 style="margin-top:0;">摘要 JSON</h3>
        <pre>{{.SummaryJSON}}</pre>
      </div>
    </div>
  </div>
</body>
</html>`

	var builder strings.Builder
	tpl := template.Must(template.New("procurement-detail").Funcs(template.FuncMap{
		"riskItems": procurementRiskItems,
		"riskLabel": func(level string) string {
			switch strings.ToLower(strings.TrimSpace(level)) {
			case "loss":
				return "亏损风险"
			case "warning":
				return "毛利预警"
			default:
				return level
			}
		},
		"riskClass": func(level string) string {
			if strings.EqualFold(level, "loss") {
				return "danger"
			}
			if strings.EqualFold(level, "warning") {
				return "warning"
			}
			return ""
		},
	}).Parse(page))
	if err := tpl.Execute(&builder, map[string]any{
		"Order":       order,
		"BackHref":    strings.TrimSpace(backHref),
		"SummaryJSON": prettyProcurementSummary(order.Summary),
	}); err != nil {
		return fmt.Sprintf("<pre>render procurement detail failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "procurement", "采购单详情", "查看采购单摘要、金额、风险和来源数据。")
}

func prettyProcurementSummary(summary pim.ProcurementSummary) string {
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Sprintf("marshal procurement summary failed: %s", err)
	}
	return string(raw)
}

func procurementRiskItems(summary pim.ProcurementSummary) []pim.ProcurementSummaryItem {
	items := make([]pim.ProcurementSummaryItem, 0)
	for _, supplier := range summary.Suppliers {
		for _, item := range supplier.Items {
			if strings.EqualFold(item.RiskLevel, "warning") || strings.EqualFold(item.RiskLevel, "loss") {
				items = append(items, item)
			}
		}
	}
	return items
}
