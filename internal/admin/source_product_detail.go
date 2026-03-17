package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

type sourceProductDetailPageData struct {
	Detail     pim.SourceProductDetail
	BackHref   string
	ActionBase string
	ReturnTo   string
}

func RenderSourceProductDetailHTML(detail pim.SourceProductDetail) string {
	return renderSourceProductDetailHTML(sourceProductDetailPageData{
		Detail:     detail,
		BackHref:   "/_/source-review-workbench",
		ActionBase: "/_/source-review-workbench/product",
		ReturnTo:   "/_/source-review-workbench",
	}, "源数据商品详情", "完整商品详情、规格、多单位和桥接状态。")
}

func RenderSourceProductDetailPageHTML(detail pim.SourceProductDetail, backHref string, actionBase string, returnTo string) string {
	return renderSourceProductDetailHTML(sourceProductDetailPageData{
		Detail:     detail,
		BackHref:   strings.TrimSpace(backHref),
		ActionBase: strings.TrimRight(strings.TrimSpace(actionBase), "/"),
		ReturnTo:   strings.TrimSpace(returnTo),
	}, "源数据商品详情", "从商品管理页进入的单商品详情。")
}

func renderSourceProductDetailHTML(pageData sourceProductDetailPageData, subtitle string, topSubtitle string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据商品详情</title>
  <style>
    :root { --bg:#06131f; --panel:rgba(8,20,32,.92); --card:rgba(10,24,38,.86); --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --accent:#5ee6ff; --accent-strong:#22b8cf; --danger:#ff6b8a; --ok:#6ef2b4; --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); color:var(--ink); }
    .wrap { max-width:1180px; margin:0 auto; padding:28px 24px 44px; }
    .card { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:22px; padding:18px; box-shadow:var(--shadow); margin-bottom:14px; backdrop-filter:blur(14px); }
    .hero { display:flex; gap:18px; align-items:flex-start; }
    .thumb { width:148px; height:148px; border-radius:20px; object-fit:cover; border:1px solid var(--line); background:#091521; }
    pre { margin:0; white-space:pre-wrap; word-break:break-word; font-family:Consolas,monospace; font-size:12px; line-height:1.6; color:#d9ebfb; }
    a { color:var(--accent); text-decoration:none; }
    .small { color:var(--muted); font-size:12px; }
    h1,h2,p { margin:0; }
    h1 { font-size:34px; letter-spacing:-.04em; }
    h2 { margin-bottom:10px; letter-spacing:-.02em; }
    .actions { display:flex; flex-wrap:wrap; gap:8px; margin-top:14px; }
    form { margin:0; }
    form.inline-note { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
    .note-input { min-width:220px; border:1px solid var(--line); border-radius:12px; padding:9px 11px; background:rgba(6,17,28,.86); color:var(--ink); font:inherit; }
    button { border:1px solid rgba(94,230,255,.24); border-radius:12px; padding:9px 12px; background:linear-gradient(135deg, var(--accent-strong) 0%, var(--accent) 100%); color:#04131d; cursor:pointer; font:inherit; font-weight:700; }
    button.secondary { background:rgba(255,255,255,.04); color:var(--accent); border-color:var(--line); }
    button.danger { background:rgba(255,107,138,.12); color:var(--danger); border-color:rgba(255,107,138,.22); }
    button.warn { background:rgba(255,209,102,.12); color:#ffd166; border-color:rgba(255,209,102,.22); }
    .status { display:inline-block; padding:5px 10px; border-radius:999px; font-size:11px; font-weight:700; background:rgba(255,255,255,.06); border:1px solid rgba(255,255,255,.08); }
    .history { display:grid; gap:10px; }
    .history-item { border:1px solid var(--line); border-radius:16px; padding:12px; background:rgba(255,255,255,.03); }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <div class="hero">
        {{if .Detail.PreviewURL}}<img class="thumb" src="{{.Detail.PreviewURL}}" alt="{{.Detail.Name}}">{{end}}
        <div>
          <h1>{{.Detail.Name}}</h1>
          <p class="small">{{.Detail.ProductID}}</p>
          <p class="small">status: {{reviewLabel .Detail.ReviewStatus}} | sourceType: {{.Detail.SourceType}}</p>
          <p class="small">{{.Detail.CategoryPath}}</p>
          {{if .Detail.ReviewedAt}}<p class="small">审核信息: {{actorLabel .Detail.ReviewedByName .Detail.ReviewedByMail}} / {{.Detail.ReviewedAt}}</p>{{end}}
          {{if .Detail.ReviewNote}}<p class="small">审核备注: {{.Detail.ReviewNote}}</p>{{end}}
          <p class="small">bridge: {{if .Detail.Bridge.Linked}}<span class="status">{{syncLabel .Detail.Bridge.SyncStatus .Detail.Bridge.Linked}}</span> {{.Detail.Bridge.SupplierRecordID}}{{else}}未桥接{{end}}</p>
          {{if .Detail.Bridge.VendureProductID}}<p class="small">vendure: {{.Detail.Bridge.VendureProductID}} / {{.Detail.Bridge.VendureVariantID}}</p>{{end}}
          {{if .Detail.Bridge.LastSyncError}}<p class="small">last error: {{.Detail.Bridge.LastSyncError}}</p>{{end}}
          <p class="small"><a href="{{.BackHref}}">返回上一页</a></p>
          <div class="actions">
            <form method="post" action="{{.ActionBase}}/status" class="inline-note" data-confirm="确认将这个商品标记为通过吗？">
              <input type="hidden" name="id" value="{{.Detail.ID}}">
              <input type="hidden" name="status" value="approved">
              <input type="hidden" name="returnTo" value="{{.ReturnTo}}">
              <input class="note-input" type="text" name="note" placeholder="审核备注">
              <button class="secondary" type="submit">通过</button>
            </form>
            <form method="post" action="{{.ActionBase}}/status" class="inline-note" data-confirm="确认拒绝这个商品吗？">
              <input type="hidden" name="id" value="{{.Detail.ID}}">
              <input type="hidden" name="status" value="rejected">
              <input type="hidden" name="returnTo" value="{{.ReturnTo}}">
              <input class="note-input" type="text" name="note" placeholder="驳回原因">
              <button class="danger" type="submit">拒绝</button>
            </form>
            <form method="post" action="{{.ActionBase}}/promote" class="inline-note" data-confirm="确认桥接这个商品吗？">
              <input type="hidden" name="id" value="{{.Detail.ID}}">
              <input type="hidden" name="returnTo" value="{{.ReturnTo}}">
              <input class="note-input" type="text" name="note" placeholder="桥接备注">
              <button type="submit">桥接</button>
            </form>
            <form method="post" action="{{.ActionBase}}/promote-sync" class="inline-note" data-confirm="确认桥接并同步这个商品吗？">
              <input type="hidden" name="id" value="{{.Detail.ID}}">
              <input type="hidden" name="returnTo" value="{{.ReturnTo}}">
              <input class="note-input" type="text" name="note" placeholder="同步备注">
              <button type="submit">桥接并同步</button>
            </form>
            {{if and .Detail.Bridge.Linked (eq .Detail.Bridge.SyncStatus "error")}}
            <form method="post" action="{{.ActionBase}}/retry-sync" class="inline-note" data-confirm="确认重试这个商品的同步吗？">
              <input type="hidden" name="id" value="{{.Detail.ID}}">
              <input type="hidden" name="returnTo" value="{{.ReturnTo}}">
              <input class="note-input" type="text" name="note" placeholder="重试备注">
              <button class="warn" type="submit">重试同步</button>
            </form>
            {{end}}
          </div>
        </div>
      </div>
    </div>
    <div class="card">
      <h2>最近操作</h2>
      <div class="history">
        {{range .Detail.RecentActions}}
        <div class="history-item">
          <div><strong>{{actionLabel .ActionType}}</strong> <span class="small">{{.Created}}</span></div>
          <div class="small">状态: {{.Status}} | 操作人: {{actorLabel .ActorName .ActorEmail}}</div>
          <div class="small">{{.Message}}</div>
          {{if .Note}}<div class="small">备注: {{.Note}}</div>{{end}}
        </div>
        {{else}}
        <div class="small">暂无最近操作。</div>
        {{end}}
      </div>
    </div>
    <div class="card"><h2>摘要</h2><pre>{{.Detail.SummaryJSON}}</pre></div>
    <div class="card"><h2>定价</h2><pre>{{.Detail.PricingJSON}}</pre></div>
    <div class="card"><h2>单位选项</h2><pre>{{.Detail.UnitOptions}}</pre></div>
    <div class="card"><h2>下单单位</h2><pre>{{.Detail.OrderUnits}}</pre></div>
    <div class="card"><h2>详情</h2><pre>{{.Detail.DetailJSON}}</pre></div>
    <div class="card"><h2>包装</h2><pre>{{.Detail.PackageJSON}}</pre></div>
    <div class="card"><h2>上下文</h2><pre>{{.Detail.ContextJSON}}</pre></div>
    <div class="card"><h2>来源分区</h2><pre>{{.Detail.SourceSections}}</pre></div>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("source-product-detail").Funcs(template.FuncMap{
		"reviewLabel": sourceReviewStatusLabel,
		"syncLabel":   sourceSyncStatusLabel,
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
		return fmt.Sprintf("<pre>render source product detail failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "source", subtitle, topSubtitle)
}
