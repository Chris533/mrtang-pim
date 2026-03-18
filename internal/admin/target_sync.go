package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/config"
	"mrtang-pim/internal/pim"
)

func RenderTargetSyncHTML(cfg config.Config, summary pim.TargetSyncSummary, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源站抓取入库</title>
  <style>
    :root { --panel:rgba(8,20,32,.92); --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --accent:#5ee6ff; --ok:#6ef2b4; --warning:#ffd166; --danger:#ff6b8a; --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:1320px; margin:0 auto; padding:24px; }
    .hero,.metrics,.content,.actions { display:grid; gap:14px; }
    .hero { grid-template-columns:1.2fr .8fr; }
    .metrics { grid-template-columns:repeat(auto-fit,minmax(170px,1fr)); }
    .content { grid-template-columns:1.1fr .9fr; margin-top:14px; }
    .actions { grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); margin-top:14px; }
    .card, .stat, .action { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:20px; padding:16px; box-shadow:var(--shadow); }
    .stat .metric { font-size:28px; font-weight:800; margin-top:10px; }
    .eyebrow { font-size:11px; letter-spacing:.14em; text-transform:uppercase; color:var(--accent); }
    .small, .muted { font-size:12px; color:var(--muted); }
    .flash { margin-bottom:14px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { color:var(--ok); background:rgba(110,242,180,.12); }
    .flash.error { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    .action { display:block; text-decoration:none; color:inherit; }
    button, select { border:1px solid var(--line); border-radius:10px; padding:8px 10px; background:rgba(6,17,28,.86); color:var(--ink); font:inherit; }
    button { cursor:pointer; background:linear-gradient(135deg,#22b8cf 0%, #5ee6ff 100%); color:#04131d; font-weight:700; }
    .secondary { background:rgba(255,255,255,.04); color:var(--accent); }
    .list { display:grid; gap:10px; }
    .list-item { border:1px solid var(--line); border-radius:14px; padding:12px; background:rgba(255,255,255,.03); }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; font-size:11px; font-weight:700; border:1px solid rgba(255,255,255,.08); }
    .badge.ok { color:var(--ok); background:rgba(110,242,180,.12); border-color:rgba(110,242,180,.18); }
    .badge.warning { color:var(--warning); background:rgba(255,209,102,.12); border-color:rgba(255,209,102,.18); }
    .badge.danger { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    table { width:100%; border-collapse:collapse; }
    th, td { text-align:left; padding:10px 8px; border-bottom:1px solid var(--line); vertical-align:top; }
    th { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:.05em; }
    @media (max-width: 980px) { .hero,.content { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}
    <section class="hero">
      <div class="card">
        <div class="eyebrow">抓取入库</div>
        <h2 style="margin:8px 0 0;">分类树、商品规格与图片抓取入库入口</h2>
        <p class="small" style="margin-top:10px;">当前已经并入三条链：分类树、商品与多单位规格、图片资产。统一登记任务、统一执行、统一查看最近运行。</p>
        <div class="actions">
          <form method="post" action="/_/mrtang-admin/target-sync/jobs/ensure">
            <input type="hidden" name="entityType" value="category_tree">
            <input type="hidden" name="scopeKey" value="">
            <button type="submit" style="width:100%; text-align:left;">保存当前源站分类抓取任务</button>
          </form>
          <form method="post" action="/_/mrtang-admin/target-sync/jobs/run" data-confirm="确认按当前源站结果抓取分类入库吗？">
            <input type="hidden" name="entityType" value="category_tree">
            <input type="hidden" name="scopeKey" value="">
            <button type="submit" style="width:100%; text-align:left;">按当前源站结果抓分类</button>
          </form>
          <form method="post" action="/_/mrtang-admin/target-sync/jobs/run" data-confirm="确认按当前源站结果抓取商品规格入库吗？">
            <input type="hidden" name="entityType" value="products">
            <input type="hidden" name="scopeKey" value="">
            <button type="submit" style="width:100%; text-align:left;">按当前源站结果抓商品规格</button>
          </form>
          <form method="post" action="/_/mrtang-admin/target-sync/jobs/run" data-confirm="确认按当前源站结果抓取图片入库吗？">
            <input type="hidden" name="entityType" value="assets">
            <input type="hidden" name="scopeKey" value="">
            <button type="submit" style="width:100%; text-align:left;">按当前源站结果抓图片</button>
          </form>
        </div>
      </div>
      <div class="card">
        <div class="eyebrow">当前来源</div>
        <div style="margin-top:10px;"><span class="badge {{modeClass .Summary.SourceMode}}">{{modeLabel .Summary.SourceMode}}</span></div>
        <div class="list" style="margin-top:12px;">
          <div class="list-item">
            <strong>接口接入提示</strong>
            <div class="small" style="margin-top:6px;">
              {{if .RequiresAuth}}当前 API 需要携带 Authorization: Bearer ...。{{else}}当前 API 默认公开，无需本地 Bearer 头。{{end}}
              {{if .SourceURL}}<br>当前上游地址：<code>{{.SourceURL}}</code>{{end}}
            </div>
          </div>
          <div class="list-item"><strong>顶级分类</strong><div class="small">{{.Summary.TopLevelCount}} 个</div></div>
          <div class="list-item"><strong>目标分类节点</strong><div class="small">{{.Summary.ExpectedNodeCount}} 个</div></div>
          <div class="list-item"><strong>本地已落库分类</strong><div class="small">{{.Summary.CategoryCount}} 个</div></div>
          <div class="list-item"><strong>目标商品 / 图片</strong><div class="small">{{.Summary.ExpectedProductCount}} 个商品 / {{.Summary.ExpectedAssetCount}} 张图片</div></div>
        </div>
      </div>
    </section>

    <section class="metrics" style="margin-top:14px;">
      <div class="stat"><div class="eyebrow">抓取任务</div><div class="metric">{{.Summary.JobCount}}</div></div>
      <div class="stat"><div class="eyebrow">运行记录</div><div class="metric">{{.Summary.RunCount}}</div></div>
      <div class="stat"><div class="eyebrow">新增差异</div><div class="metric">{{.Summary.DiffNewCount}}</div></div>
      <div class="stat"><div class="eyebrow">变更差异</div><div class="metric">{{.Summary.DiffChangedCount}}</div></div>
      <div class="stat"><div class="eyebrow">缺失差异</div><div class="metric">{{.Summary.DiffMissingCount}}</div></div>
      <div class="stat"><div class="eyebrow">商品差异</div><div class="metric">{{.Summary.ProductDiffNewCount}}</div><div class="small">新增 {{.Summary.ProductDiffNewCount}} / 变更 {{.Summary.ProductDiffChangedCount}}</div></div>
      <div class="stat"><div class="eyebrow">图片差异</div><div class="metric">{{.Summary.AssetDiffNewCount}}</div><div class="small">新增 {{.Summary.AssetDiffNewCount}} / 变更 {{.Summary.AssetDiffChangedCount}}</div></div>
    </section>

    <section class="content">
      <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
          <h3 style="margin:0;">抓取入库后待审核</h3>
          <span class="small">目标站刷新后直接回到 source 流</span>
        </div>
        <div class="actions" style="margin-top:0;">
          <a class="action" href="/_/mrtang-admin/source/products?productStatus=imported">
            <div class="eyebrow">待审核商品</div>
            <div style="font-size:24px; font-weight:800; margin-top:8px;">{{.Summary.SourceImportedCount}}</div>
            <div class="small" style="margin-top:8px;">新增或变更后的商品与规格会自动回到 imported。</div>
          </a>
          <a class="action" href="/_/mrtang-admin/source/products?productStatus=approved">
            <div class="eyebrow">待加入发布队列商品</div>
            <div style="font-size:24px; font-weight:800; margin-top:8px;">{{.Summary.SourceApprovedCount}}</div>
            <div class="small" style="margin-top:8px;">审核通过后加入发布队列，再进入 supplier_products 和 backend 发布链。</div>
          </a>
          <a class="action" href="/_/mrtang-admin/source/assets?assetStatus=pending">
            <div class="eyebrow">待处理图片</div>
            <div style="font-size:24px; font-weight:800; margin-top:8px;">{{.Summary.SourceAssetPendingCount}}</div>
            <div class="small" style="margin-top:8px;">变更图片会自动重置为 pending，重新进入图片处理流。</div>
          </a>
          <a class="action" href="/_/mrtang-admin/source/assets?assetStatus=failed">
            <div class="eyebrow">失败图片</div>
            <div style="font-size:24px; font-weight:800; margin-top:8px;">{{.Summary.SourceAssetFailedCount}}</div>
            <div class="small" style="margin-top:8px;">失败图片可在 source 模块继续重试或人工处理。</div>
          </a>
        </div>
      </div>

      <div class="card">
        <h3 style="margin-top:0;">审核说明</h3>
        <div class="list">
          <div class="list-item">
            <strong>商品与规格</strong>
            <div class="small" style="margin-top:6px;">当商品标题、分类、默认单位、单位数量、默认价格或资源数量发生变化时，会自动回到 <code>imported</code>，要求重新审核。</div>
          </div>
          <div class="list-item">
            <strong>图片资产</strong>
            <div class="small" style="margin-top:6px;">当图片地址、角色或排序变化时，会自动清空处理结果并回到 <code>pending</code>，重新进入图片处理链。</div>
          </div>
        </div>
      </div>
    </section>

    <section class="content">
      <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
          <h3 style="margin:0;">Checkout 来源矩阵</h3>
          <span class="small">这里显示的是当前实际 contractId，不是静态推断</span>
        </div>
        <table>
          <thead><tr><th>链路</th><th>当前状态</th><th>contractId</th><th>说明</th></tr></thead>
          <tbody>
          {{range .Summary.CheckoutSources}}
            <tr>
              <td><strong>{{.Label}}</strong></td>
              <td><span class="badge {{sourceStatusClass .Status}}">{{sourceStatusLabel .Status}}</span></td>
              <td class="small"><code>{{blank .ContractID "-"}}</code></td>
              <td class="small">{{.Note}}</td>
            </tr>
          {{else}}
            <tr><td colspan="4" class="small">当前还没有 checkout 来源数据。</td></tr>
          {{end}}
          </tbody>
        </table>
      </div>

      <div class="card">
        <h3 style="margin-top:0;">真实写操作提示</h3>
        <div class="list">
          <div class="list-item">
            <strong>添加收货地址</strong>
            <div class="small" style="margin-top:6px;"><code>POST /api/miniapp/cart-order/order/address/add</code> 在 raw 模式下会真实调用源站。必须显式传请求体，不再默认复用 fallback 样本。</div>
          </div>
          <div class="list-item">
            <strong>提交订单</strong>
            <div class="small" style="margin-top:6px;"><code>POST /api/miniapp/cart-order/order/submit</code> 在 raw 模式下会真实下单。必须显式传请求体，建议只在确认购物车、地址和运费后手动触发。</div>
          </div>
          <div class="list-item">
            <strong>安全边界</strong>
            <div class="small" style="margin-top:6px;">后台总览、抓取入库和 source 审核流不会自动执行写操作；raw 自动抓取阶段仍保持只读优先。</div>
          </div>
        </div>
      </div>
    </section>

    <section class="content">
      <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
          <h3 style="margin:0;">按顶级分类抓取入库</h3>
          <span class="small">分类、商品规格、图片都支持按顶级分类执行</span>
        </div>
        <div class="list">
          {{range .Summary.ScopeOptions}}
            {{if .Key}}
            <div class="list-item">
              <div style="display:flex; justify-content:space-between; gap:12px; align-items:flex-start;">
                <div>
                  <strong>{{.Label}}</strong>
                  <div class="small">{{.NodeCount}} 个分类节点</div>
                </div>
                <div style="display:flex; gap:8px; flex-wrap:wrap;">
                  <form method="post" action="/_/mrtang-admin/target-sync/jobs/ensure">
                    <input type="hidden" name="entityType" value="category_tree">
                    <input type="hidden" name="scopeKey" value="{{.Key}}">
                    <button class="secondary" type="submit">登记分类任务</button>
                  </form>
                  <form method="post" action="/_/mrtang-admin/target-sync/jobs/run" data-confirm="确认执行该顶级分类抓取入库吗？">
                    <input type="hidden" name="entityType" value="category_tree">
                    <input type="hidden" name="scopeKey" value="{{.Key}}">
                    <button type="submit">抓取分类</button>
                  </form>
                  <form method="post" action="/_/mrtang-admin/target-sync/jobs/run" data-confirm="确认执行该顶级分类下的商品规格抓取入库吗？">
                    <input type="hidden" name="entityType" value="products">
                    <input type="hidden" name="scopeKey" value="{{.Key}}">
                    <button class="secondary" type="submit">抓取商品规格</button>
                  </form>
                  <form method="post" action="/_/mrtang-admin/target-sync/jobs/run" data-confirm="确认执行该顶级分类下的图片抓取入库吗？">
                    <input type="hidden" name="entityType" value="assets">
                    <input type="hidden" name="scopeKey" value="{{.Key}}">
                    <button class="secondary" type="submit">抓取图片</button>
                  </form>
                </div>
                <div class="small" style="margin-top:8px;">商品 {{.ProductCount}} 个 / 图片 {{.AssetCount}} 张</div>
              </div>
            </div>
            {{end}}
          {{end}}
        </div>
      </div>

      <div class="card">
        <h3 style="margin-top:0;">分类差异</h3>
        <table>
          <thead><tr><th>类型</th><th>分类</th><th>路径</th></tr></thead>
          <tbody>
          {{range .Summary.CategoryDiffs}}
            <tr>
              <td><span class="badge {{diffClass .DiffType}}">{{diffLabel .DiffType}}</span></td>
              <td><strong>{{.Label}}</strong><div class="small">{{.SourceKey}}</div></td>
              <td class="small">{{.CategoryPath}}</td>
            </tr>
          {{else}}
            <tr><td colspan="3" class="small">当前分类树没有检测到差异。</td></tr>
          {{end}}
          </tbody>
        </table>
      </div>

      <div class="card">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
          <h3 style="margin:0;">最近真实写操作</h3>
          <span class="small">仅记录 raw 模式下的显式真实写入</span>
        </div>
        <table>
          <thead><tr><th>时间</th><th>操作</th><th>结果</th><th>contractId</th></tr></thead>
          <tbody>
          {{range .Summary.RecentMiniappWrites}}
            <tr>
              <td class="small">{{blank .CreatedAt "-"}}</td>
              <td><strong>{{.OperationLabel}}</strong><div class="small">{{.OperationID}}</div></td>
              <td><span class="badge {{writeStatusClass .Status}}">{{writeStatusLabel .Status}}</span><div class="small">{{blank .Message "-"}}</div></td>
              <td class="small"><code>{{blank .ContractID "-"}}</code></td>
            </tr>
          {{else}}
            <tr><td colspan="4" class="small">目前还没有记录到 raw 模式下的真实写操作。</td></tr>
          {{end}}
          </tbody>
        </table>
      </div>
    </section>

    <section class="content">
      <div class="card">
        <h3 style="margin-top:0;">抓取任务</h3>
        <table>
          <thead><tr><th>任务</th><th>范围</th><th>状态</th><th>最近运行</th></tr></thead>
          <tbody>
          {{range .Summary.Jobs}}
            <tr>
              <td><strong>{{.Name}}</strong><div class="small">{{entityLabel .EntityType}} / {{.JobKey}}</div></td>
              <td>{{scopeLabel .ScopeType .ScopeLabel}}</td>
              <td><span class="badge {{statusClass .Status}}">{{statusLabel .Status}}</span></td>
              <td class="small">{{blank .LastRunAt "-"}}{{if .LastError}}<br>{{.LastError}}{{end}}</td>
            </tr>
          {{else}}
            <tr><td colspan="4" class="small">还没有抓取任务，先保存一个分类抓取任务。</td></tr>
          {{end}}
          </tbody>
        </table>
      </div>

      <div class="card">
        <h3 style="margin-top:0;">最近运行</h3>
        <table>
          <thead><tr><th>运行</th><th>结果</th><th>统计</th></tr></thead>
          <tbody>
          {{range .Summary.Runs}}
            <tr>
              <td><strong><a href="/_/mrtang-admin/target-sync/run?id={{.ID}}">{{.JobName}}</a></strong><div class="small">{{entityLabel .EntityType}} / {{blank .FinishedAt .StartedAt}}</div></td>
              <td><span class="badge {{statusClass .Status}}">{{statusLabel .Status}}</span><div class="small">{{actorLabel .TriggeredByName .TriggeredByEmail}}</div></td>
              <td class="small">新增 {{.CreatedCount}} / 更新 {{.UpdatedCount}} / 未变 {{.UnchangedCount}} / 范围 {{.ScopedNodeCount}}{{if .ErrorMessage}}<br>{{.ErrorMessage}}{{end}}</td>
            </tr>
          {{else}}
            <tr><td colspan="3" class="small">还没有运行记录。</td></tr>
          {{end}}
          </tbody>
        </table>
      </div>
    </section>
  </div>
</body>
</html>`

	type pageData struct {
		Summary      pim.TargetSyncSummary
		FlashMessage string
		FlashError   string
		RequiresAuth bool
		SourceURL    string
	}

	tpl := template.Must(template.New("target-sync").Funcs(template.FuncMap{
		"blank": func(value string, fallback string) string {
			if strings.TrimSpace(value) == "" {
				return fallback
			}
			return value
		},
		"modeClass": func(mode string) string {
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "raw":
				return "ok"
			case "snapshot":
				return "warning"
			}
			return ""
		},
		"modeLabel": func(mode string) string {
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case "raw":
				return "RAW 真实源站模式"
			case "snapshot":
				return "快照模式"
			}
			return "未识别模式"
		},
		"sourceStatusClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case "raw_live":
				return "ok"
			case "raw_readonly", "fallback":
				return "warning"
			case "explicit_write":
				return "danger"
			default:
				return ""
			}
		},
		"sourceStatusLabel": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case "raw_live":
				return "raw live"
			case "raw_readonly":
				return "raw 只读"
			case "explicit_write":
				return "显式真实写入"
			default:
				return "fallback"
			}
		},
		"writeStatusClass": func(status string) string {
			if strings.EqualFold(strings.TrimSpace(status), "success") {
				return "ok"
			}
			if strings.EqualFold(strings.TrimSpace(status), "failed") {
				return "danger"
			}
			return ""
		},
		"writeStatusLabel": func(status string) string {
			if strings.EqualFold(strings.TrimSpace(status), "success") {
				return "成功"
			}
			if strings.EqualFold(strings.TrimSpace(status), "failed") {
				return "失败"
			}
			return "未知"
		},
		"scopeLabel": func(scopeType string, scopeLabel string) string {
			if strings.EqualFold(scopeType, pim.TargetSyncScopeAll) {
				return "全量"
			}
			return scopeLabel
		},
		"entityLabel": func(entityType string) string {
			switch strings.ToLower(strings.TrimSpace(entityType)) {
			case pim.TargetSyncEntityProducts:
				return "商品规格"
			case pim.TargetSyncEntityAssets:
				return "图片资产"
			default:
				return "分类树"
			}
		},
		"statusClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case pim.TargetSyncStatusSuccess:
				return "ok"
			case pim.TargetSyncStatusPartial, pim.TargetSyncStatusRunning:
				return "warning"
			case pim.TargetSyncStatusFailed:
				return "danger"
			default:
				return ""
			}
		},
		"statusLabel": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case pim.TargetSyncStatusPending:
				return "待执行"
			case pim.TargetSyncStatusRunning:
				return "执行中"
			case pim.TargetSyncStatusSuccess:
				return "成功"
			case pim.TargetSyncStatusPartial:
				return "部分成功"
			case pim.TargetSyncStatusFailed:
				return "失败"
			default:
				return status
			}
		},
		"diffClass": func(diff string) string {
			switch strings.ToLower(strings.TrimSpace(diff)) {
			case "new":
				return "ok"
			case "changed":
				return "warning"
			case "missing":
				return "danger"
			default:
				return ""
			}
		},
		"diffLabel": func(diff string) string {
			switch strings.ToLower(strings.TrimSpace(diff)) {
			case "new":
				return "待新增"
			case "changed":
				return "待更新"
			case "missing":
				return "本地多余"
			default:
				return diff
			}
		},
		"actorLabel": func(name string, email string) string {
			if strings.TrimSpace(name) != "" {
				return name
			}
			if strings.TrimSpace(email) != "" {
				return email
			}
			return "系统"
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData{
		Summary:      summary,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
		RequiresAuth: strings.TrimSpace(cfg.MiniApp.AuthorizedAccountID) != "",
		SourceURL:    strings.TrimSpace(cfg.MiniApp.SourceURL),
	}); err != nil {
		return fmt.Sprintf("<pre>render ingest sync failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "target-sync", "抓取入库", "先完成分类树和子分类抓取入库，再逐步并入商品、规格和图片。")
}
