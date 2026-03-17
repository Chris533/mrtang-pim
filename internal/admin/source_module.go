package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderSourceModuleHTML(summary pim.SourceReviewWorkbenchSummary, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据模块</title>
  <style>
    :root {
      --bg: #06131f;
      --panel: rgba(8, 20, 32, 0.92);
      --card: rgba(10, 24, 38, 0.86);
      --ink: #edf7ff;
      --muted: #8aa3bb;
      --line: rgba(123, 168, 203, 0.16);
      --accent: #5ee6ff;
      --accent-soft: rgba(94, 230, 255, 0.14);
      --danger: #ff6b8a;
      --ok: #6ef2b4;
      --warning: #ffd166;
      --shadow: 0 24px 60px rgba(0,0,0,.34);
    }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); color:var(--ink); }
    .wrap { max-width:1320px; margin:0 auto; padding:24px; }
    .hero, .metrics, .modules, .content { display:grid; gap:14px; }
    .hero { grid-template-columns:1.3fr .9fr; }
    .metrics { grid-template-columns:repeat(auto-fit,minmax(170px,1fr)); }
    .modules { grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); }
    .content { grid-template-columns:1.15fr .85fr; margin-top:14px; }
    .card {
      background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%);
      border:1px solid var(--line);
      border-radius:22px;
      padding:18px;
      box-shadow:var(--shadow);
      backdrop-filter:blur(14px);
    }
    .stat, .link-card {
      display:block;
      background:linear-gradient(180deg, rgba(16,35,53,.88) 0%, rgba(9,23,36,.94) 100%);
      border:1px solid var(--line);
      border-radius:18px;
      padding:14px 16px;
      text-decoration:none;
      color:inherit;
      position:relative;
      overflow:hidden;
    }
    .stat::before, .link-card::before {
      content:"";
      position:absolute;
      inset:0 auto auto 0;
      width:100%;
      height:2px;
      background:linear-gradient(90deg, transparent 0%, rgba(94,230,255,.86) 24%, rgba(140,123,255,.7) 100%);
    }
    .link-card:hover { transform:translateY(-2px); border-color:rgba(94,230,255,.34); transition:.18s ease; }
    h2, h3, p { margin:0; }
    h2 { font-size:20px; letter-spacing:-.02em; }
    .lead { color:var(--muted); line-height:1.6; margin-top:8px; }
    .eyebrow { font-size:11px; letter-spacing:.14em; text-transform:uppercase; color:var(--accent); }
    .metric { font-size:30px; font-weight:800; letter-spacing:-.04em; margin-top:10px; }
    .title { margin-top:8px; font-weight:700; }
    .desc { margin-top:7px; color:var(--muted); font-size:13px; line-height:1.5; }
    .badge {
      display:inline-block;
      padding:5px 10px;
      border-radius:999px;
      border:1px solid rgba(255,255,255,.08);
      background:rgba(255,255,255,.06);
      font-size:11px;
      font-weight:700;
    }
    .badge.ok { color:var(--ok); background:rgba(110,242,180,.12); border-color:rgba(110,242,180,.18); }
    .badge.warning { color:var(--warning); background:rgba(255,209,102,.12); border-color:rgba(255,209,102,.18); }
    .badge.danger { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    .flash { margin-bottom:14px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { color:var(--ok); background:rgba(110,242,180,.12); }
    .flash.error { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.2); }
    .list { display:grid; gap:10px; }
    .list-item { border:1px solid var(--line); border-radius:16px; padding:12px 14px; background:rgba(8,20,32,.7); }
    .small { color:var(--muted); font-size:12px; }
    .mono { font-family:Consolas,monospace; font-size:12px; }
    @media (max-width: 980px) {
      .hero, .content { grid-template-columns:1fr; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}

    <section class="hero">
      <div class="card">
        <div class="eyebrow">源数据模块</div>
        <h2>目标分类、商品、图片的审核与同步入口</h2>
        <p class="lead">这个模块只承担 source 相关的管理职责。首页看板负责总览，这里负责 source 维度的待办、异常、最近动作和模块分流。</p>
      </div>
      <div class="card">
        <div class="eyebrow">待办队列</div>
        <div style="margin-top:12px; display:grid; gap:10px;">
          <a class="link-card" href="/_/mrtang-admin/source/products?productStatus=imported">
            <div class="title">待审批商品</div>
            <div class="desc">{{.Summary.ReadyToReviewCount}} 条待审核商品待处理</div>
          </a>
          <a class="link-card" href="/_/mrtang-admin/source/assets?assetStatus=failed">
            <div class="title">失败图片</div>
            <div class="desc">{{.Summary.AssetFailed}} 张图片需要重处理或人工确认</div>
          </a>
          <a class="link-card" href="/_/mrtang-admin/source/products?syncState=error">
            <div class="title">同步失败</div>
            <div class="desc">{{.Summary.SyncErrorCount}} 条已桥接商品需要重试同步</div>
          </a>
        </div>
      </div>
    </section>

    <section class="metrics" style="margin-top:14px;">
      <div class="stat"><div class="eyebrow">商品</div><div class="metric">{{.Summary.ProductCount}}</div><div class="small">{{.Summary.ImportedCount}} 待审核 / {{.Summary.ApprovedCount}} 待桥接 / {{.Summary.PromotedCount}} 已桥接</div></div>
      <div class="stat"><div class="eyebrow">图片</div><div class="metric">{{.Summary.AssetCount}}</div><div class="small">{{.Summary.AssetPending}} 待处理 / {{.Summary.AssetProcessed}} 已处理 / {{.Summary.AssetFailed}} 失败</div></div>
      <div class="stat"><div class="eyebrow">桥接</div><div class="metric">{{.Summary.LinkedCount}}</div><div class="small">{{.Summary.UnlinkedCount}} 未桥接 / {{.Summary.SyncedCount}} 已同步</div></div>
      <div class="stat"><div class="eyebrow">同步异常</div><div class="metric">{{.Summary.SyncErrorCount}}</div><div class="small">{{.Summary.FailedLinkedCount}} 条桥接记录同步失败</div></div>
      <div class="stat"><div class="eyebrow">待推进</div><div class="metric">{{.Summary.ReadyToPromoteCount}}</div><div class="small">{{.Summary.ReadyToReviewCount}} 待审核 / {{.Summary.ReadyToSyncCount}} 待同步</div></div>
    </section>

    <section class="modules" style="margin-top:14px;">
      <a class="link-card" href="/_/mrtang-admin/source/products">
        <div class="eyebrow">商品</div>
        <div class="title">商品管理页</div>
        <div class="desc">筛选、分页、批量通过、桥接、桥接并同步、重试同步。</div>
      </a>
      <a class="link-card" href="/_/mrtang-admin/source/assets?assetStatus=pending">
        <div class="eyebrow">图片</div>
        <div class="title">图片处理页</div>
        <div class="desc">独立的图片管理页，负责失败重试、批量处理和单图详情。</div>
      </a>
      <a class="link-card" href="/_/mrtang-admin/source/logs">
        <div class="eyebrow">日志</div>
        <div class="title">操作日志</div>
        <div class="desc">按动作、状态、目标筛选 source action logs。</div>
      </a>
    </section>

    <section class="content">
      <div class="card">
        <div style="display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:12px;">
          <h3>最近操作</h3>
          <a class="small" href="/_/source-review-workbench">打开兼容工作台</a>
        </div>
        <div class="list">
          {{range .Summary.RecentActions}}
            <div class="list-item">
              <div style="display:flex;justify-content:space-between;gap:10px;align-items:flex-start;">
                <div>
                  <strong>{{actionLabel .ActionType}}</strong>
                  <div class="small">{{.TargetType}} / {{.TargetLabel}}</div>
                  <div class="small mono">{{.TargetID}}</div>
                  {{if .Message}}<div class="small">{{.Message}}</div>{{end}}
                </div>
                <span class="badge {{logClass .Status}}">{{.Status}}</span>
              </div>
              <div class="small" style="margin-top:8px;">{{.Created}}</div>
            </div>
          {{else}}
            <div class="list-item"><span class="small">暂无操作日志。</span></div>
          {{end}}
        </div>
      </div>

      <div class="card">
        <div style="display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:12px;">
          <h3>异常与队列</h3>
          <span class="badge warning">运营优先级</span>
        </div>
        <div class="list">
          <div class="list-item">
            <div><strong>待审批商品</strong> <span class="badge warning">{{.Summary.ReadyToReviewCount}}</span></div>
            <div class="small">优先处理待审核商品，避免源数据到供应链的桥接积压。</div>
          </div>
          <div class="list-item">
            <div><strong>失败图片</strong> <span class="badge danger">{{.Summary.AssetFailed}}</span></div>
            <div class="small">失败资产优先重试，再决定是否人工替换或跳过。</div>
          </div>
          <div class="list-item">
            <div><strong>同步失败</strong> <span class="badge danger">{{.Summary.SyncErrorCount}}</span></div>
            <div class="small">已桥接商品同步失败时，优先去商品页重试同步。</div>
          </div>
        </div>
      </div>
    </section>
  </div>
</body>
</html>`

	type pageData struct {
		Summary      pim.SourceReviewWorkbenchSummary
		FlashMessage string
		FlashError   string
	}

	tpl := template.Must(template.New("source-module").Funcs(template.FuncMap{
		"logClass": func(status string) string {
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
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData{
		Summary:      summary,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}); err != nil {
		return fmt.Sprintf("<pre>render source module failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "source", "源数据模块", "源数据维度的概览、待办、异常和模块分流。")
}
