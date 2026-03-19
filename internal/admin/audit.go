package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"
)

type AuditFilter struct {
	Domain   string
	Status   string
	Query    string
	Page     int
	PageSize int
}

type AuditPageData struct {
	Items        []mrtangAdminRecentAction
	Filter       AuditFilter
	Total        int
	Page         int
	Pages        int
	SuccessCount int
	FailedCount  int
}

func RenderAuditPageHTML(data AuditPageData, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>统一审计</title>
  <style>
    :root { --panel:rgba(8,20,32,.92); --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --accent:#5ee6ff; --danger:#ff6b8a; --ok:#6ef2b4; --warning:#ffd166; --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:1260px; margin:0 auto; padding:24px; }
    .card { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:22px; padding:18px; box-shadow:var(--shadow); }
    .flash { margin-bottom:14px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { color:var(--ok); background:rgba(110,242,180,.12); }
    .flash.error { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.2); }
    .stats { display:grid; grid-template-columns:repeat(auto-fit,minmax(160px,1fr)); gap:12px; margin-bottom:14px; }
    .stat { border:1px solid var(--line); border-radius:18px; padding:14px 16px; background:rgba(255,255,255,.03); }
    .metric { font-size:28px; font-weight:800; letter-spacing:-.04em; margin-top:8px; }
    .filters { display:flex; flex-wrap:wrap; gap:10px; align-items:end; margin-bottom:14px; }
    .filters label { display:flex; flex-direction:column; gap:6px; font-size:12px; color:var(--muted); }
    input, select, button {
      border:1px solid var(--line); border-radius:12px; padding:9px 11px;
      background:rgba(6,17,28,.86); color:var(--ink); font:inherit;
    }
    button {
      cursor:pointer; background:linear-gradient(135deg, #22b8cf 0%, #5ee6ff 100%);
      color:#04131d; border-color:rgba(94,230,255,.28); font-weight:700;
    }
    .link { color:var(--accent); text-decoration:none; font-weight:600; }
    .table-wrap { overflow:auto; border-radius:16px; }
    table { width:100%; border-collapse:separate; border-spacing:0; min-width:900px; }
    th, td { text-align:left; padding:12px 8px; border-bottom:1px solid var(--line); vertical-align:top; }
    th { color:var(--muted); font-size:11px; text-transform:uppercase; letter-spacing:.12em; position:sticky; top:0; background:rgba(7,17,27,.96); backdrop-filter:blur(12px); z-index:1; }
    .badge { display:inline-block; border-radius:999px; padding:5px 10px; font-size:11px; font-weight:700; background:rgba(255,255,255,.06); border:1px solid rgba(255,255,255,.08); }
    .badge.ok { color:var(--ok); background:rgba(110,242,180,.12); border-color:rgba(110,242,180,.18); }
    .badge.danger { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    .muted { color:var(--muted); font-size:12px; }
  </style>
</head>
<body>
  <div class="wrap">
    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}
    <div class="card">
      <div class="stats">
        <div class="stat"><div class="muted">日志总数</div><div class="metric">{{.Data.Total}}</div></div>
        <div class="stat"><div class="muted">成功</div><div class="metric">{{.Data.SuccessCount}}</div></div>
        <div class="stat"><div class="muted">失败</div><div class="metric">{{.Data.FailedCount}}</div></div>
        <div class="stat"><div class="muted">分页</div><div class="metric">{{.Data.Page}} / {{.Data.Pages}}</div></div>
      </div>
      <form method="get" action="/_/mrtang-admin/audit" class="filters">
        <label>模块
          <select name="domain">
            <option value="">全部</option>
            <option value="源数据" {{if eq .Data.Filter.Domain "源数据"}}selected{{end}}>源数据</option>
            <option value="图片任务" {{if eq .Data.Filter.Domain "图片任务"}}selected{{end}}>图片任务</option>
            <option value="商品任务" {{if eq .Data.Filter.Domain "商品任务"}}selected{{end}}>商品任务</option>
            <option value="采购" {{if eq .Data.Filter.Domain "采购"}}selected{{end}}>采购</option>
          </select>
        </label>
        <label>状态
          <select name="status">
            <option value="">全部</option>
            <option value="success" {{if eq .Data.Filter.Status "success"}}selected{{end}}>成功</option>
            <option value="failed" {{if eq .Data.Filter.Status "failed"}}selected{{end}}>失败</option>
          </select>
        </label>
        <label>检索
          <input type="text" name="q" value="{{.Data.Filter.Query}}" placeholder="目标 / 操作人 / 备注">
        </label>
        <label>每页
          <select name="pageSize">
            <option value="20" {{if or (eq .Data.Filter.PageSize 0) (eq .Data.Filter.PageSize 20)}}selected{{end}}>20</option>
            <option value="50" {{if eq .Data.Filter.PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .Data.Filter.PageSize 100}}selected{{end}}>100</option>
          </select>
        </label>
        <button type="submit">应用筛选</button>
        <a class="link" href="/_/mrtang-admin/audit">重置</a>
        <a class="link" href="/_/mrtang-admin/audit?status=failed">只看失败</a>
      </form>
      <div class="table-wrap"><table>
        <thead><tr><th>模块</th><th>动作</th><th>目标</th><th>状态</th><th>操作人</th><th>说明</th><th>时间</th></tr></thead>
        <tbody>
        {{range .Data.Items}}
          <tr>
            <td><span class="badge">{{.Domain}}</span></td>
            <td><strong>{{.Label}}</strong></td>
            <td>{{.Target}}</td>
            <td><span class="badge {{if eq .Status "success"}}ok{{else}}danger{{end}}">{{.Status}}</span></td>
            <td class="muted">{{.Actor}}</td>
            <td class="muted">{{.Message}}{{if .Note}}<br>备注: {{.Note}}{{end}}</td>
            <td class="muted">{{.Created}}</td>
          </tr>
        {{else}}
          <tr><td colspan="7" class="muted">暂无统一审计记录。</td></tr>
        {{end}}
        </tbody>
      </table></div>
      <div style="display:flex; gap:10px; align-items:center; margin-top:12px;">
        {{if gt .Data.Page 1}}<a class="link" href="{{pageURL .Data.Filter (dec .Data.Page)}}">上一页</a>{{end}}
        <span class="muted">第 {{.Data.Page}} / {{.Data.Pages}} 页</span>
        {{if lt .Data.Page .Data.Pages}}<a class="link" href="{{pageURL .Data.Filter (inc .Data.Page)}}">下一页</a>{{end}}
      </div>
    </div>
  </div>
</body>
</html>`

	var builder strings.Builder
	tpl := template.Must(template.New("audit").Funcs(template.FuncMap{
		"inc": func(v int) int { return v + 1 },
		"dec": func(v int) int {
			if v <= 1 {
				return 1
			}
			return v - 1
		},
		"pageURL": func(filter AuditFilter, page int) string {
			return auditPageURL(filter, page)
		},
	}).Parse(page))
	if err := tpl.Execute(&builder, map[string]any{
		"Data":         data,
		"FlashMessage": strings.TrimSpace(flashMessage),
		"FlashError":   strings.TrimSpace(flashError),
	}); err != nil {
		return fmt.Sprintf("<pre>render audit page failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "dashboard", "统一审计", "汇总源数据与采购动作，统一回查最近操作。")
}

func auditPageURL(filter AuditFilter, page int) string {
	values := url.Values{}
	if v := strings.TrimSpace(filter.Domain); v != "" {
		values.Set("domain", v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		values.Set("status", v)
	}
	if v := strings.TrimSpace(filter.Query); v != "" {
		values.Set("q", v)
	}
	if filter.PageSize > 0 {
		values.Set("pageSize", strconv.Itoa(filter.PageSize))
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/_/mrtang-admin/audit?" + encoded
	}
	return "/_/mrtang-admin/audit"
}
