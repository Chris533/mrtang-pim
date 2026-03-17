package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

type sourceAssetDetailPageData struct {
	Detail     pim.SourceAssetDetail
	BackHref   string
	ActionBase string
	ReturnTo   string
}

func RenderSourceAssetDetailHTML(detail pim.SourceAssetDetail) string {
	return renderSourceAssetDetailHTML(sourceAssetDetailPageData{
		Detail:     detail,
		BackHref:   "/_/source-review-workbench",
		ActionBase: "/_/source-review-workbench/assets",
		ReturnTo:   "/_/source-review-workbench",
	}, "源数据图片详情", "原图、处理图和来源载荷。")
}

func RenderSourceAssetDetailPageHTML(detail pim.SourceAssetDetail, backHref string, actionBase string, returnTo string) string {
	return renderSourceAssetDetailHTML(sourceAssetDetailPageData{
		Detail:     detail,
		BackHref:   strings.TrimSpace(backHref),
		ActionBase: strings.TrimRight(strings.TrimSpace(actionBase), "/"),
		ReturnTo:   strings.TrimSpace(returnTo),
	}, "源数据图片详情", "从图片管理页进入的单图详情。")
}

func renderSourceAssetDetailHTML(pageData sourceAssetDetailPageData, title string, subtitle string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据图片详情</title>
  <style>
    :root { --bg:#06131f; --panel:rgba(8,20,32,.92); --card:rgba(10,24,38,.86); --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --accent:#5ee6ff; --accent-strong:#22b8cf; --danger:#ff6b8a; --ok:#6ef2b4; --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); color:var(--ink); }
    .wrap { max-width:1180px; margin:0 auto; padding:28px 24px 44px; }
    .card { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:22px; padding:18px; box-shadow:var(--shadow); margin-bottom:14px; backdrop-filter:blur(14px); }
    .hero { display:flex; justify-content:space-between; gap:16px; align-items:start; }
    .pair { display:grid; grid-template-columns:1fr 1fr; gap:14px; }
    .panel img { width:100%; max-height:520px; object-fit:contain; border:1px solid var(--line); border-radius:16px; background:#091521; }
    .meta { color:var(--muted); font-size:12px; }
    .status { display:inline-block; padding:5px 10px; border-radius:999px; font-size:11px; font-weight:700; background:rgba(255,255,255,.06); border:1px solid rgba(255,255,255,.08); }
    .status.ok { background:rgba(110,242,180,.12); color:var(--ok); border-color:rgba(110,242,180,.18); }
    .status.danger { background:rgba(255,107,138,.12); color:var(--danger); border-color:rgba(255,107,138,.18); }
    .actions { display:flex; flex-wrap:wrap; gap:8px; margin-top:12px; }
    form { margin:0; }
    form.inline-note { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
    .note-input { min-width:220px; border:1px solid var(--line); border-radius:12px; padding:9px 11px; background:rgba(6,17,28,.86); color:var(--ink); font:inherit; }
    button { border:1px solid rgba(94,230,255,.24); border-radius:12px; padding:9px 12px; background:linear-gradient(135deg, var(--accent-strong) 0%, var(--accent) 100%); color:#04131d; cursor:pointer; font:inherit; font-weight:700; }
    a { color:var(--accent); text-decoration:none; }
    pre { margin:0; white-space:pre-wrap; word-break:break-word; font-family:Consolas,monospace; font-size:12px; line-height:1.6; color:#d9ebfb; }
    h1 { font-size:34px; letter-spacing:-.04em; margin:0 0 4px; }
    h2 { margin:0 0 10px; }
    .history { display:grid; gap:10px; }
    .history-item { border:1px solid var(--line); border-radius:16px; padding:12px; background:rgba(255,255,255,.03); }
    @media (max-width: 900px) { .pair { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <div class="hero">
        <div>
          <h1>{{.Detail.Name}}</h1>
          <div class="meta">{{.Detail.AssetKey}}</div>
          <div class="meta">{{.Detail.AssetRole}} / {{.Detail.ProductID}}</div>
          <div class="meta">status: <span class="status {{assetClass .Detail.ImageProcessingStatus}}">{{assetLabel .Detail.ImageProcessingStatus}}</span></div>
          {{if .Detail.ImageProcessingError}}<div class="meta">last error: {{.Detail.ImageProcessingError}}</div>{{end}}
          {{if .Detail.ProcessedImageSource}}<div class="meta">processed by: {{.Detail.ProcessedImageSource}}</div>{{end}}
          <div class="meta"><a href="{{.BackHref}}">返回上一页</a></div>
        </div>
        <div class="actions">
          <form method="post" action="{{.ActionBase}}/process" class="inline-note" data-confirm="确认处理或重处理这张图片吗？">
            <input type="hidden" name="id" value="{{.Detail.ID}}">
            <input type="hidden" name="returnTo" value="{{.ReturnTo}}">
            <input class="note-input" type="text" name="note" placeholder="处理备注">
            <button type="submit">处理 / 重处理</button>
          </form>
          {{if .Detail.SourceURL}}<a href="{{.Detail.SourceURL}}" target="_blank" rel="noreferrer">打开原图</a>{{end}}
          {{if .Detail.ProcessedImageURL}}<a href="{{.Detail.ProcessedImageURL}}" target="_blank" rel="noreferrer">打开处理图</a>{{end}}
        </div>
      </div>
    </div>

    <div class="pair">
      <div class="card panel">
        <h2>原图</h2>
        {{if .Detail.SourceURL}}<img src="{{.Detail.SourceURL}}" alt="{{.Detail.Name}} 原图">{{else}}<div class="meta">暂无原图</div>{{end}}
      </div>
      <div class="card panel">
        <h2>处理图</h2>
        {{if .Detail.ProcessedImageURL}}<img src="{{.Detail.ProcessedImageURL}}" alt="{{.Detail.Name}} 处理图">{{else}}<div class="meta">暂无处理图</div>{{end}}
      </div>
    </div>

    <div class="card">
      <h2>最近操作</h2>
      <div class="history">
        {{range .Detail.RecentActions}}
        <div class="history-item">
          <div><strong>{{actionLabel .ActionType}}</strong> <span class="meta">{{.Created}}</span></div>
          <div class="meta">状态: {{.Status}} | 操作人: {{actorLabel .ActorName .ActorEmail}}</div>
          <div class="meta">{{.Message}}</div>
          {{if .Note}}<div class="meta">备注: {{.Note}}</div>{{end}}
        </div>
        {{else}}
        <div class="meta">暂无最近操作。</div>
        {{end}}
      </div>
    </div>

    <div class="card">
      <h2>来源载荷</h2>
      <pre>{{.Detail.SourcePayloadJSON}}</pre>
    </div>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("source-asset-detail").Funcs(template.FuncMap{
		"assetClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case pim.ImageStatusProcessed:
				return "ok"
			case pim.ImageStatusFailed:
				return "danger"
			default:
				return ""
			}
		},
		"assetLabel":  sourceAssetStatusLabel,
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
	}).Parse(page))
	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData); err != nil {
		return fmt.Sprintf("<pre>render source asset detail failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "source", title, subtitle)
}
