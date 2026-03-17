package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderSourceAssetsHTML(summary pim.SourceReviewWorkbenchSummary, filter pim.SourceReviewFilter, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据图片</title>
  <style>
    :root {
      --panel: rgba(8, 20, 32, 0.92);
      --ink: #edf7ff;
      --muted: #8aa3bb;
      --line: rgba(123,168,203,.16);
      --accent: #5ee6ff;
      --accent-strong: #22b8cf;
      --danger: #ff6b8a;
      --ok: #6ef2b4;
      --warning: #ffd166;
      --shadow: 0 24px 60px rgba(0,0,0,.34);
    }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:1380px; margin:0 auto; padding:24px; }
    .stats, .layout, .top { display:grid; gap:14px; }
    .stats { grid-template-columns:repeat(auto-fit,minmax(170px,1fr)); }
    .top { grid-template-columns:1.05fr .95fr; margin-bottom:14px; }
    .layout { grid-template-columns:minmax(0,1fr) 320px; }
    .card {
      background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%);
      border:1px solid var(--line);
      border-radius:22px;
      padding:18px;
      box-shadow:var(--shadow);
      backdrop-filter:blur(14px);
    }
    .stat {
      background:linear-gradient(180deg, rgba(16,35,53,.88) 0%, rgba(9,23,36,.94) 100%);
      border:1px solid var(--line);
      border-radius:18px;
      padding:14px 16px;
      position:relative;
      overflow:hidden;
    }
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
    .filters, .toolbar, .bulkbar, .actions { display:flex; flex-wrap:wrap; gap:10px; }
    .filters { align-items:end; margin-bottom:14px; }
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
      background:linear-gradient(135deg, var(--accent-strong) 0%, var(--accent) 100%);
      color:#04131d;
      border-color:rgba(94,230,255,.28);
      font-weight:700;
    }
    button.secondary { background:rgba(255,255,255,.04); color:var(--accent); }
    button.warn { background:rgba(255,209,102,.12); color:var(--warning); border-color:rgba(255,209,102,.22); }
    .flash { margin-bottom:14px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { color:var(--ok); background:rgba(110,242,180,.12); }
    .flash.error { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.2); }
    .table-wrap { overflow:auto; border-radius:16px; }
    table { width:100%; border-collapse:separate; border-spacing:0; min-width:860px; }
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
    .media { display:flex; gap:10px; align-items:flex-start; }
    .thumb { width:56px; height:56px; border-radius:14px; object-fit:cover; border:1px solid var(--line); background:#091521; flex:0 0 auto; }
    .pick { width:34px; accent-color: var(--accent-strong); }
    .pager { display:flex; flex-wrap:wrap; gap:10px; align-items:center; margin-top:12px; }
    .pager a, .link { color:var(--accent); text-decoration:none; font-weight:600; }
    .queue-list { display:grid; gap:10px; }
    .queue-item { border:1px solid var(--line); border-radius:16px; padding:12px 14px; background:rgba(8,20,32,.72); }
    .issues { margin:0; padding-left:18px; }
    .issues li { margin:8px 0; }
    @media (max-width: 1120px) { .layout, .top { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}

    <section class="top">
      <div class="card">
        <div class="eyebrow">图片</div>
        <h2 style="margin:8px 0 0;">图片处理与失败重试</h2>
        <p class="small" style="margin-top:8px;">这里单独承担 source_assets 的筛选、分页、批量处理、失败重试和单图详情。图片工作流不再挤在 products 页里。</p>
      </div>
      <div class="card">
        <div class="eyebrow">失败原因</div>
        <ul class="issues" style="margin-top:10px;">
          {{range .Summary.AssetFailureReasons}}
            <li><strong>{{.Count}}</strong> <span class="small">{{.Message}}</span></li>
          {{else}}
            <li class="small">当前没有失败原因。</li>
          {{end}}
        </ul>
      </div>
    </section>

    <section class="stats">
      <div class="stat"><div class="eyebrow">图片</div><div class="metric">{{.Summary.AssetCount}}</div><div class="small">全部图片资产</div></div>
      <div class="stat"><div class="eyebrow">待处理</div><div class="metric">{{.Summary.AssetPending}}</div><div class="small">待 AI 处理</div></div>
      <div class="stat"><div class="eyebrow">已处理</div><div class="metric">{{.Summary.AssetProcessed}}</div><div class="small">已处理成功</div></div>
      <div class="stat"><div class="eyebrow">失败</div><div class="metric">{{.Summary.AssetFailed}}</div><div class="small">需要重试或人工介入</div></div>
    </section>

    <section class="layout" style="margin-top:14px;">
      <div class="card">
        <form method="get" action="/_/mrtang-admin/source/assets" class="filters admin-toolbar">
          <label>图片状态
            <select name="assetStatus">
              <option value="">全部</option>
              <option value="pending" {{if eq .Filter.AssetStatus "pending"}}selected{{end}}>待处理</option>
              <option value="processed" {{if eq .Filter.AssetStatus "processed"}}selected{{end}}>已处理</option>
              <option value="failed" {{if eq .Filter.AssetStatus "failed"}}selected{{end}}>失败</option>
            </select>
          </label>
          <label>检索
            <input type="text" name="q" value="{{.Filter.Query}}" placeholder="图片名 / assetKey / productId">
          </label>
          <label>每页
            <select name="pageSize">
              <option value="12" {{if eq .Filter.PageSize 12}}selected{{end}}>12</option>
              <option value="24" {{if or (eq .Filter.PageSize 0) (eq .Filter.PageSize 24)}}selected{{end}}>24</option>
              <option value="48" {{if eq .Filter.PageSize 48}}selected{{end}}>48</option>
            </select>
          </label>
          <button type="submit" class="secondary">应用筛选</button>
          <a class="link small" href="/_/mrtang-admin/source/assets">重置</a>
          <a class="link small" href="/_/mrtang-admin/source/assets?assetStatus=pending">待处理</a>
          <a class="link small" href="/_/mrtang-admin/source/assets?assetStatus=failed">失败图片</a>
          <a class="link small" href="/_/mrtang-admin/source/assets?assetStatus=processed">处理成功</a>
        </form>

        <div class="toolbar admin-toolbar" style="margin-bottom:12px;">
          <form method="post" action="/_/mrtang-admin/source/assets/process" data-confirm="确认批量处理当前待处理图片吗？">
            <input type="hidden" name="returnTo" value="{{.CurrentURL}}">
            <button type="submit">批量处理待处理图片</button>
          </form>
          <form method="post" action="/_/mrtang-admin/source/assets/reprocess-failed" data-confirm="确认批量重处理失败图片吗？">
            <input type="hidden" name="returnTo" value="{{.CurrentURL}}">
            <button type="submit" class="warn">批量重处理失败图片</button>
          </form>
        </div>

        <form method="post" action="/_/mrtang-admin/source/assets/batch-process">
          <input type="hidden" name="returnTo" value="{{.CurrentURL}}">
          <div class="bulkbar admin-toolbar" style="margin-bottom:12px;">
            <button type="submit" data-confirm="确认处理当前选中的图片吗？">批量处理所选图片</button>
          </div>
          <div class="table-wrap"><table>
            <thead><tr><th>图片</th><th>状态</th><th>来源</th><th>操作</th></tr></thead>
            <tbody>
            {{range .Summary.Assets}}
              <tr>
                <td>
                  <div class="media">
                    <input class="pick" type="checkbox" name="assetIds" value="{{.ID}}">
                    {{if .PreviewURL}}<img class="thumb" src="{{.PreviewURL}}" alt="{{.Name}}">{{end}}
                    <div>
                      <div><strong>{{.Name}}</strong></div>
                      <div class="small mono">{{.AssetKey}}</div>
                      <div class="small">{{.AssetRole}} / {{.ProductID}}</div>
                      <div class="small"><a class="link" href="/_/mrtang-admin/source/assets/detail?id={{.ID}}&returnTo={{urlquery $.CurrentURL}}">查看图片详情</a></div>
                    </div>
                  </div>
                </td>
                <td>
                  <span class="status {{assetClass .ImageProcessingStatus}}">{{assetLabel .ImageProcessingStatus}}</span>
                  {{if .ImageProcessingError}}<div class="small">{{.ImageProcessingError}}</div>{{end}}
                </td>
                <td>
                  {{if .ProcessedImageURL}}<div class="small">处理图已生成</div>{{end}}
                  <div class="small"><a class="link" href="{{.SourceURL}}" target="_blank" rel="noreferrer">原图</a>{{if .ProcessedImageURL}} / <a class="link" href="{{.ProcessedImageURL}}" target="_blank" rel="noreferrer">处理图</a>{{end}}</div>
                </td>
                <td>
                  <div class="actions">
                    <form method="post" action="/_/mrtang-admin/source/assets/process" data-confirm="确认处理这张图片吗？">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <input type="hidden" name="returnTo" value="{{$.CurrentURL}}">
                      <button type="submit">处理</button>
                    </form>
                  </div>
                </td>
              </tr>
            {{else}}
              <tr><td colspan="4" class="small">暂无 source_assets 记录。</td></tr>
            {{end}}
            </tbody>
          </table></div>
        </form>

        <div class="pager">
          {{if gt .Summary.AssetPage 1}}<a href="{{pageURL .Filter (dec .Summary.AssetPage)}}">上一页</a>{{end}}
          <span class="small">资产分页 {{.Summary.AssetPage}} / {{.Summary.AssetPages}}</span>
          {{if lt .Summary.AssetPage .Summary.AssetPages}}<a href="{{pageURL .Filter (inc .Summary.AssetPage)}}">下一页</a>{{end}}
        </div>
      </div>

      <aside class="card">
        <div style="display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:12px;">
          <h3 style="margin:0;">资产队列</h3>
          <a class="link small" href="/_/mrtang-admin/source">返回源数据首页</a>
        </div>
        <div class="queue-list">
          <div class="queue-item">
            <div><strong>待处理</strong> <span class="status warning">{{.Summary.AssetPending}}</span></div>
            <div class="small">待处理图片可直接批量处理。</div>
          </div>
          <div class="queue-item">
            <div><strong>失败重试</strong> <span class="status danger">{{.Summary.AssetFailed}}</span></div>
            <div class="small">失败图片优先看失败原因，再批量重处理。</div>
          </div>
          <div class="queue-item">
            <div><strong>已处理</strong> <span class="status ok">{{.Summary.AssetProcessed}}</span></div>
            <div class="small">处理成功后可在详情页对比原图和处理图。</div>
          </div>
        </div>
      </aside>
    </section>
  </div>
</body>
</html>`

	type pageData struct {
		Summary      pim.SourceReviewWorkbenchSummary
		Filter       pim.SourceReviewFilter
		CurrentURL   string
		FlashMessage string
		FlashError   string
	}

	tpl := template.Must(template.New("source-assets").Funcs(template.FuncMap{
		"assetClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case pim.ImageStatusProcessed:
				return "ok"
			case pim.ImageStatusFailed:
				return "danger"
			default:
				return "warning"
			}
		},
		"assetLabel": sourceAssetStatusLabel,
		"inc":        func(v int) int { return v + 1 },
		"dec": func(v int) int {
			if v <= 1 {
				return 1
			}
			return v - 1
		},
		"pageURL": func(filter pim.SourceReviewFilter, page int) string {
			return sourceAssetsURL(filter, page)
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData{
		Summary:      summary,
		Filter:       filter,
		CurrentURL:   sourceAssetsURL(filter, filter.AssetPage),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}); err != nil {
		return fmt.Sprintf("<pre>render source assets failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "source", "源数据图片", "独立的图片管理页，负责处理、重试和详情对比。")
}

func sourceAssetsURL(filter pim.SourceReviewFilter, page int) string {
	values := url.Values{}
	if v := strings.TrimSpace(filter.AssetStatus); v != "" {
		values.Set("assetStatus", v)
	}
	if v := strings.TrimSpace(filter.Query); v != "" {
		values.Set("q", v)
	}
	if filter.PageSize > 0 {
		values.Set("pageSize", strconv.Itoa(filter.PageSize))
	}
	if page > 1 {
		values.Set("assetPage", strconv.Itoa(page))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/_/mrtang-admin/source/assets"
	}
	return "/_/mrtang-admin/source/assets?" + encoded
}
