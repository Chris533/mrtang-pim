package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"mrtang-pim/internal/pim"
)

type SourceLogFilter struct {
	ActionType string
	Status     string
	TargetType string
	Actor      string
	Query      string
	Page       int
	PageSize   int
}

type SourceLogsPageData struct {
	Items        []pim.SourceActionLog
	Filter       SourceLogFilter
	Total        int
	Page         int
	Pages        int
	PageSize     int
	SuccessCount int
	FailedCount  int
}

func RenderSourceLogsHTML(data SourceLogsPageData, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据操作日志</title>
  <style>
    :root {
      --panel: rgba(8, 20, 32, 0.92);
      --ink: #edf7ff;
      --muted: #8aa3bb;
      --line: rgba(123,168,203,.16);
      --accent: #5ee6ff;
      --danger: #ff6b8a;
      --ok: #6ef2b4;
      --warning: #ffd166;
      --shadow: 0 24px 60px rgba(0,0,0,.34);
    }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:1380px; margin:0 auto; padding:24px; }
    .stats { display:grid; grid-template-columns:repeat(auto-fit,minmax(170px,1fr)); gap:14px; margin-bottom:14px; }
    .card, .stat {
      background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%);
      border:1px solid var(--line);
      border-radius:22px;
      padding:18px;
      box-shadow:var(--shadow);
      backdrop-filter:blur(14px);
    }
    .stat { border-radius:18px; padding:14px 16px; position:relative; overflow:hidden; }
    .stat::before {
      content:"";
      position:absolute;
      inset:0 auto auto 0;
      width:100%;
      height:2px;
      background:linear-gradient(90deg, transparent 0%, rgba(94,230,255,.86) 24%, rgba(140,123,255,.7) 100%);
    }
    .eyebrow { font-size:11px; letter-spacing:.14em; text-transform:uppercase; color:var(--accent); }
    .metric { font-size:30px; font-weight:800; letter-spacing:-.04em; margin-top:10px; }
    .small { color:var(--muted); font-size:12px; }
    .mono { font-family:Consolas,monospace; font-size:12px; }
    .filters, .pager { display:flex; flex-wrap:wrap; gap:10px; align-items:end; }
    .filters { margin-bottom:14px; }
    .filters label { display:flex; flex-direction:column; gap:6px; font-size:12px; color:var(--muted); }
    input, select, button {
      border:1px solid var(--line);
      border-radius:12px;
      padding:9px 11px;
      background:rgba(6,17,28,.86);
      color:var(--ink);
      font:inherit;
    }
    button {
      cursor:pointer;
      background:linear-gradient(135deg, #22b8cf 0%, #5ee6ff 100%);
      color:#04131d;
      border-color:rgba(94,230,255,.28);
      font-weight:700;
    }
    button.secondary { background:rgba(255,255,255,.04); color:var(--accent); }
    .flash { margin-bottom:14px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { color:var(--ok); background:rgba(110,242,180,.12); }
    .flash.error { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.2); }
    .table-wrap { overflow:auto; border-radius:16px; }
    table { width:100%; border-collapse:separate; border-spacing:0; min-width:900px; }
    th, td { text-align:left; padding:12px 8px; border-bottom:1px solid var(--line); vertical-align:top; }
    th { color:var(--muted); font-size:11px; text-transform:uppercase; letter-spacing:.12em; position:sticky; top:0; background:rgba(7,17,27,.96); backdrop-filter:blur(12px); z-index:1; }
    .status {
      display:inline-block;
      padding:5px 10px;
      border-radius:999px;
      font-size:11px;
      font-weight:700;
      background:rgba(255,255,255,.06);
      border:1px solid rgba(255,255,255,.08);
    }
    .status.ok { color:var(--ok); background:rgba(110,242,180,.12); border-color:rgba(110,242,180,.18); }
    .status.warning { color:var(--warning); background:rgba(255,209,102,.12); border-color:rgba(255,209,102,.18); }
    .status.danger { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    .link { color:var(--accent); text-decoration:none; font-weight:600; }
  </style>
</head>
<body>
  <div class="wrap">
    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}

    <section class="stats">
      <div class="stat"><div class="eyebrow">日志总数</div><div class="metric">{{.Data.Total}}</div><div class="small">当前筛选命中的日志数</div></div>
      <div class="stat"><div class="eyebrow">成功动作</div><div class="metric">{{.Data.SuccessCount}}</div><div class="small">执行成功</div></div>
      <div class="stat"><div class="eyebrow">失败动作</div><div class="metric">{{.Data.FailedCount}}</div><div class="small">需要回查</div></div>
      <div class="stat"><div class="eyebrow">分页</div><div class="metric">{{.Data.Page}}</div><div class="small">第 {{.Data.Page}} / {{.Data.Pages}} 页</div></div>
    </section>

    <section class="card">
      <form method="get" action="/_/mrtang-admin/source/logs" class="filters">
        <label>动作
          <select name="actionType">
            <option value="">全部</option>
            <option value="update_review_status" {{if eq .Data.Filter.ActionType "update_review_status"}}selected{{end}}>审核状态变更</option>
            <option value="promote_product" {{if eq .Data.Filter.ActionType "promote_product"}}selected{{end}}>桥接商品</option>
            <option value="promote_and_sync" {{if eq .Data.Filter.ActionType "promote_and_sync"}}selected{{end}}>桥接并同步</option>
            <option value="retry_sync" {{if eq .Data.Filter.ActionType "retry_sync"}}selected{{end}}>重试同步</option>
            <option value="process_asset" {{if eq .Data.Filter.ActionType "process_asset"}}selected{{end}}>处理图片</option>
          </select>
        </label>
        <label>状态
          <select name="status">
            <option value="">全部</option>
            <option value="success" {{if eq .Data.Filter.Status "success"}}selected{{end}}>成功</option>
            <option value="failed" {{if eq .Data.Filter.Status "failed"}}selected{{end}}>失败</option>
          </select>
        </label>
        <label>目标
          <select name="targetType">
            <option value="">全部</option>
            <option value="product" {{if eq .Data.Filter.TargetType "product"}}selected{{end}}>商品</option>
            <option value="asset" {{if eq .Data.Filter.TargetType "asset"}}selected{{end}}>图片</option>
          </select>
        </label>
        <label>操作人
          <input type="text" name="actor" value="{{.Data.Filter.Actor}}" placeholder="邮箱或姓名">
        </label>
        <label>检索
          <input type="text" name="q" value="{{.Data.Filter.Query}}" placeholder="target label / id / message">
        </label>
        <label>每页
          <select name="pageSize">
            <option value="20" {{if or (eq .Data.Filter.PageSize 0) (eq .Data.Filter.PageSize 20)}}selected{{end}}>20</option>
            <option value="50" {{if eq .Data.Filter.PageSize 50}}selected{{end}}>50</option>
            <option value="100" {{if eq .Data.Filter.PageSize 100}}selected{{end}}>100</option>
          </select>
        </label>
        <button type="submit" class="secondary">应用筛选</button>
        <a class="link small" href="/_/mrtang-admin/source/logs">重置</a>
        <a class="link small" href="/_/mrtang-admin/source/logs?status=failed">只看失败</a>
        <a class="link small" href="/_/mrtang-admin/source/logs?actionType=retry_sync">只看重试同步</a>
        <a class="link small" href="/_/mrtang-admin/source/logs?actionType=process_asset">只看图片处理</a>
      </form>

      <div class="table-wrap"><table>
        <thead><tr><th>动作</th><th>目标</th><th>状态</th><th>操作人</th><th>说明</th><th>时间</th></tr></thead>
        <tbody>
        {{range .Data.Items}}
          <tr>
            <td><strong>{{actionLabel .ActionType}}</strong></td>
            <td><div>{{.TargetType}} / {{.TargetLabel}}</div><div class="small mono">{{.TargetID}}</div></td>
            <td><span class="status {{statusClass .Status}}">{{.Status}}</span></td>
            <td><div class="small">{{actorLabel .ActorName .ActorEmail}}</div></td>
            <td class="small">{{.Message}}{{if .Note}}<br><span class="mono">备注: {{.Note}}</span>{{end}}</td>
            <td class="small">{{.Created}}</td>
          </tr>
        {{else}}
          <tr><td colspan="6" class="small">暂无源数据操作日志。</td></tr>
        {{end}}
        </tbody>
      </table></div>

      <div class="pager" style="margin-top:12px;">
        {{if gt .Data.Page 1}}<a class="link" href="{{pageURL .Data.Filter (dec .Data.Page)}}">上一页</a>{{end}}
        <span class="small">日志分页 {{.Data.Page}} / {{.Data.Pages}}</span>
        {{if lt .Data.Page .Data.Pages}}<a class="link" href="{{pageURL .Data.Filter (inc .Data.Page)}}">下一页</a>{{end}}
      </div>
    </section>
  </div>
</body>
</html>`

	type pageVM struct {
		Data         SourceLogsPageData
		FlashMessage string
		FlashError   string
	}

	tpl := template.Must(template.New("source-logs").Funcs(template.FuncMap{
		"statusClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case "success":
				return "ok"
			case "failed":
				return "danger"
			default:
				return "warning"
			}
		},
		"actionLabel": sourceActionTypeLabel,
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
		"pageURL": func(filter SourceLogFilter, page int) string {
			return sourceLogsURL(filter, page)
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageVM{
		Data:         data,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}); err != nil {
		return fmt.Sprintf("<pre>render source logs failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "source", "Source Logs", "source 操作日志、失败追踪和按动作筛选。")
}

func sourceLogsURL(filter SourceLogFilter, page int) string {
	values := url.Values{}
	if v := strings.TrimSpace(filter.ActionType); v != "" {
		values.Set("actionType", v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		values.Set("status", v)
	}
	if v := strings.TrimSpace(filter.TargetType); v != "" {
		values.Set("targetType", v)
	}
	if v := strings.TrimSpace(filter.Actor); v != "" {
		values.Set("actor", v)
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
	encoded := values.Encode()
	if encoded == "" {
		return "/_/mrtang-admin/source/logs"
	}
	return "/_/mrtang-admin/source/logs?" + encoded
}
