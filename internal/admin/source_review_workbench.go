package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderSourceReviewWorkbenchHTML(summary pim.SourceReviewWorkbenchSummary, filter pim.SourceReviewFilter, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据兼容工作台</title>
  <style>
    :root {
      --bg: #06131f;
      --panel: rgba(10, 24, 38, 0.86);
      --panel-strong: rgba(8, 20, 32, 0.94);
      --ink: #edf7ff;
      --muted: #89a4bc;
      --line: rgba(123, 168, 203, 0.16);
      --accent: #5ee6ff;
      --accent-strong: #22b8cf;
      --accent-soft: rgba(94, 230, 255, 0.14);
      --danger: #ff6b8a;
      --ok: #6ef2b4;
      --warning: #ffd166;
      --shadow: 0 24px 60px rgba(0, 0, 0, 0.34);
    }
    * { box-sizing:border-box; }
    body {
      margin:0;
      font-family:"Segoe UI","PingFang SC",sans-serif;
      background:
        radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 22%),
        radial-gradient(circle at left top, rgba(140,123,255,.12) 0, transparent 28%),
        linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%);
      color:var(--ink);
    }
    .wrap { max-width:1240px; margin:0 auto; padding:28px 24px 44px; }
    .hero { display:flex; justify-content:space-between; align-items:end; gap:16px; margin-bottom:20px; }
    .hero h1 { margin:0; font-size:34px; letter-spacing:-.04em; }
    .hero p { margin:8px 0 0; color:var(--muted); max-width:70ch; line-height:1.6; }
    .hero a { color:var(--accent); text-decoration:none; font-weight:600; }
    .stats,.grid { display:grid; gap:12px; }
    .stats { grid-template-columns:repeat(auto-fit,minmax(150px,1fr)); margin-bottom:20px; }
    .grid { grid-template-columns:1.4fr .9fr; }
    .card {
      background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel-strong) 100%);
      border:1px solid var(--line);
      border-radius:22px;
      padding:16px 18px;
      box-shadow:var(--shadow);
      backdrop-filter:blur(14px);
    }
    .metric { font-size:28px; font-weight:800; margin-top:8px; letter-spacing:-.04em; }
    .label { color:var(--muted); font-size:12px; letter-spacing:.12em; text-transform:uppercase; }
    .table-wrap { overflow:auto; border-radius:16px; }
    table { width:100%; border-collapse:separate; border-spacing:0; min-width:720px; }
    th,td { text-align:left; padding:12px 8px; border-bottom:1px solid var(--line); vertical-align:top; }
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
    .status.warning { background:rgba(255,209,102,.12); color:var(--warning); border-color:rgba(255,209,102,.18); }
    .status.danger { background:rgba(255,107,138,.12); color:var(--danger); border-color:rgba(255,107,138,.18); }
    .status.ok { background:rgba(110,242,180,.12); color:var(--ok); border-color:rgba(110,242,180,.18); }
    .actions { display:flex; flex-wrap:wrap; gap:8px; }
    form { margin:0; }
    button, select, input {
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
    button.danger { background:rgba(255,107,138,.12); color:var(--danger); border-color:rgba(255,107,138,.22); }
    .small { font-size:12px; color:var(--muted); }
    .flash { margin-bottom:16px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { background:rgba(110,242,180,.12); color:var(--ok); }
    .flash.error { background:rgba(255,107,138,.12); color:var(--danger); border-color:rgba(255,107,138,.18); }
    .toolbar { display:flex; flex-wrap:wrap; gap:10px; margin-bottom:14px; }
    .toolbar form { display:inline-flex; }
    .mono { font-family:Consolas,monospace; font-size:12px; }
    .thumb { width:56px; height:56px; border-radius:14px; object-fit:cover; border:1px solid var(--line); background:#091521; margin-right:10px; flex:0 0 auto; }
    .media { display:flex; align-items:flex-start; gap:10px; }
    .filters { display:flex; flex-wrap:wrap; gap:10px; margin-bottom:14px; align-items:end; }
    .filters label { display:flex; flex-direction:column; gap:6px; font-size:12px; color:var(--muted); }
    .section-head { display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:10px; }
    .bulkbar { display:flex; flex-wrap:wrap; gap:8px; margin:0 0 12px; }
    .bulkbar .small { display:flex; align-items:center; }
    .pager { display:flex; flex-wrap:wrap; gap:8px; align-items:center; margin-top:12px; }
    .pager a { color:var(--accent); text-decoration:none; font-weight:600; }
    .pick { width:34px; accent-color: var(--accent-strong); }
    .split { display:grid; grid-template-columns:1fr .9fr; gap:12px; margin-bottom:12px; }
    .issues { margin:0; padding-left:18px; }
    .issues li { margin:8px 0; }
    .log-list { display:grid; gap:8px; }
    .log-item { border:1px solid var(--line); border-radius:14px; padding:10px 12px; background:rgba(8,20,32,.72); }
    @media (max-width: 1000px) { .grid, .split { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="hero">
      <div>
        <h1>源数据兼容工作台</h1>
        <p>兼容保留的统一工作台。新后台默认请从源数据首页进入商品、图片和日志独立页面。</p>
      </div>
      <div>
        <a href="/_/mrtang-admin">Mrtang Admin</a>
        <span class="small"> | </span>
        <a href="/_/mrtang-admin/source">源数据首页</a>
        <span class="small"> | </span>
        <a href="/_/mrtang-admin/source/products">商品</a>
        <span class="small"> | </span>
        <a href="/_/mrtang-admin/source/assets">图片</a>
        <span class="small"> | </span>
        <a href="/_/">返回 Admin</a>
        <span class="small"> | </span>
        <a href="/_/#/collections/source_products">打开 source_products</a>
      </div>
    </div>

    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}

    <div class="stats">
      <div class="card"><div class="label">分类</div><div class="metric">{{.Summary.CategoryCount}}</div></div>
      <div class="card"><div class="label">商品</div><div class="metric">{{.Summary.ProductCount}}</div><div class="small">{{.Summary.ImportedCount}} imported / {{.Summary.ApprovedCount}} approved / {{.Summary.PromotedCount}} promoted</div></div>
      <div class="card"><div class="label">图片资产</div><div class="metric">{{.Summary.AssetCount}}</div><div class="small">{{.Summary.AssetPending}} pending / {{.Summary.AssetProcessed}} processed / {{.Summary.AssetFailed}} failed</div></div>
      <div class="card"><div class="label">桥接与同步</div><div class="metric">{{.Summary.LinkedCount}}</div><div class="small">{{.Summary.SyncedCount}} synced / {{.Summary.SyncErrorCount}} error / {{.Summary.UnlinkedCount}} unlinked</div></div>
      <div class="card"><div class="label">待处理</div><div class="metric">{{.Summary.ReadyToReviewCount}}</div><div class="small">{{.Summary.ReadyToPromoteCount}} 待 promote / {{.Summary.ReadyToSyncCount}} 待 sync</div></div>
      <div class="card"><div class="label">异常</div><div class="metric">{{.Summary.AssetFailed}}</div><div class="small">{{.Summary.FailedLinkedCount}} linked sync error</div></div>
    </div>

    <div class="split">
      <div class="card">
        <div class="section-head">
          <h2>图片失败聚合</h2>
          <div class="small">{{.Summary.AssetFailed}} failed assets</div>
        </div>
        <ul class="issues">
          {{range .Summary.AssetFailureReasons}}
            <li><strong>{{.Count}}</strong> <span class="small">{{.Message}}</span></li>
          {{else}}
            <li class="small">当前没有失败原因。</li>
          {{end}}
        </ul>
      </div>
      <div class="card">
        <div class="section-head">
          <h2>图片快捷动作</h2>
          <div class="small">优先重试失败资产</div>
        </div>
        <div class="toolbar">
          <form method="post" action="/_/source-review-workbench/assets/reprocess-failed"><button type="submit">批量重处理失败图片</button></form>
          <a class="small" href="/_/source-review-workbench?assetStatus=failed">只看失败图片</a>
          <a class="small" href="/_/source-review-workbench?assetStatus=processed">只看处理成功</a>
        </div>
      </div>
    </div>

    <div class="split">
      <div class="card">
        <div class="section-head">
          <h2>最近操作</h2>
          <div class="small">{{len .Summary.RecentActions}} recent actions</div>
        </div>
        <div class="log-list">
          {{range .Summary.RecentActions}}
            <div class="log-item">
              <div><strong>{{actionLabel .ActionType}}</strong> <span class="status {{logClass .Status}}">{{.Status}}</span></div>
              <div class="small">{{.TargetType}} / {{.TargetLabel}}</div>
              <div class="small mono">{{.TargetID}}</div>
              {{if .Message}}<div class="small">{{.Message}}</div>{{end}}
              <div class="small">{{.Created}}</div>
            </div>
          {{else}}
            <div class="small">暂无 source action logs。</div>
          {{end}}
        </div>
      </div>
      <div class="card">
        <div class="section-head">
          <h2>同步重试</h2>
          <div class="small">优先处理 linked sync error</div>
        </div>
        <div class="toolbar">
          <a class="small" href="/_/source-review-workbench?syncState=error">只看同步失败</a>
          <a class="small" href="/_/source-review-workbench?productStatus=approved">只看待 promote</a>
          <a class="small" href="/_/source-review-workbench?assetStatus=failed">只看失败图片</a>
        </div>
      </div>
    </div>

    <div class="card">
      <form method="get" action="/_/source-review-workbench" class="filters">
        <label>商品状态
          <select name="productStatus">
            <option value="">all</option>
            <option value="imported" {{if eq .Filter.ProductStatus "imported"}}selected{{end}}>imported</option>
            <option value="approved" {{if eq .Filter.ProductStatus "approved"}}selected{{end}}>approved</option>
            <option value="promoted" {{if eq .Filter.ProductStatus "promoted"}}selected{{end}}>promoted</option>
            <option value="rejected" {{if eq .Filter.ProductStatus "rejected"}}selected{{end}}>rejected</option>
          </select>
        </label>
        <label>图片状态
          <select name="assetStatus">
            <option value="">all</option>
            <option value="pending" {{if eq .Filter.AssetStatus "pending"}}selected{{end}}>pending</option>
            <option value="processed" {{if eq .Filter.AssetStatus "processed"}}selected{{end}}>processed</option>
            <option value="failed" {{if eq .Filter.AssetStatus "failed"}}selected{{end}}>failed</option>
          </select>
        </label>
        <label>同步状态
          <select name="syncState">
            <option value="">all</option>
            <option value="unlinked" {{if eq .Filter.SyncState "unlinked"}}selected{{end}}>unlinked</option>
            <option value="linked" {{if eq .Filter.SyncState "linked"}}selected{{end}}>linked</option>
            <option value="error" {{if eq .Filter.SyncState "error"}}selected{{end}}>error</option>
            <option value="synced" {{if eq .Filter.SyncState "synced"}}selected{{end}}>synced</option>
          </select>
        </label>
        <label>检索
          <input type="text" name="q" value="{{.Filter.Query}}" placeholder="name / productId / assetKey">
        </label>
        <label>每页
          <select name="pageSize">
            <option value="12" {{if eq .Filter.PageSize 12}}selected{{end}}>12</option>
            <option value="24" {{if or (eq .Filter.PageSize 0) (eq .Filter.PageSize 24)}}selected{{end}}>24</option>
            <option value="48" {{if eq .Filter.PageSize 48}}selected{{end}}>48</option>
          </select>
        </label>
        <button type="submit" class="secondary">应用筛选</button>
        <a class="small" href="/_/source-review-workbench">重置</a>
        <a class="small" href="/_/source-review-workbench?productStatus=imported">只看待审批</a>
        <a class="small" href="/_/source-review-workbench?assetStatus=pending">只看待处理图片</a>
        <a class="small" href="/_/source-review-workbench?productStatus=approved">只看已审批未推送</a>
        <a class="small" href="/_/source-review-workbench?syncState=unlinked">只看未桥接</a>
        <a class="small" href="/_/source-review-workbench?syncState=error">只看同步失败</a>
        <a class="small" href="/_/source-review-workbench?syncState=synced">只看已同步</a>
      </form>
      <div class="toolbar">
        <form method="post" action="/_/source-review-workbench/assets/process"><button type="submit">批量处理待处理图片</button></form>
        <form method="post" action="/_/source-review-workbench/products/promote"><button type="submit">批量推送已审批商品</button></form>
        <form method="post" action="/_/source-review-workbench/supplier-products/sync"><button type="submit">同步已批准商品到 Backend</button></form>
      </div>
    </div>

    <div class="grid">
      <section class="card">
        <div class="section-head">
          <h2>商品审批</h2>
          <div class="small">第 {{.Summary.ProductPage}} / {{.Summary.ProductPages}} 页，共 {{len .Summary.Products}} 条当前页记录</div>
        </div>
        <form method="post" action="/_/source-review-workbench/products/batch-status">
          <div class="bulkbar">
            <input type="hidden" name="status" value="approved">
            <button type="submit" class="secondary">批量 Approve</button>
            <button type="submit" class="danger" formaction="/_/source-review-workbench/products/batch-status" onclick="this.form.status.value='rejected'">批量 Reject</button>
            <button type="submit" formaction="/_/source-review-workbench/products/batch-promote">批量 Promote</button>
            <button type="submit" formaction="/_/source-review-workbench/products/batch-promote-sync">批量 Promote &amp; Sync</button>
            <button type="submit" class="warn" formaction="/_/source-review-workbench/products/batch-retry-sync">批量 Retry Sync</button>
          </div>
          <div class="table-wrap"><table>
          <thead><tr><th>商品</th><th>状态</th><th>规格</th><th>同步</th><th>图片</th><th>操作</th></tr></thead>
          <tbody>
          {{range .Summary.Products}}
            <tr>
              <td>
                <div class="media">
                  <input class="pick" type="checkbox" name="productIds" value="{{.ID}}">
                  {{if .PreviewURL}}<img class="thumb" src="{{.PreviewURL}}" alt="{{.Name}}">{{end}}
                  <div>
                    <div><strong>{{.Name}}</strong></div>
                    <div class="small mono">{{.ProductID}}</div>
                    <div class="small">{{.CategoryPath}}</div>
                    <div class="small"><a href="/_/source-review-workbench/product?id={{.ID}}">查看详情</a></div>
                  </div>
                </div>
              </td>
              <td><span class="status {{reviewClass .ReviewStatus}}">{{.ReviewStatus}}</span><div class="small">{{.SourceType}}</div></td>
              <td>
                <div>{{printf "%.2f" .DefaultPrice}}</div>
                <div class="small">{{.UnitCount}} units {{if .HasMultiUnit}}/ multi-unit{{end}}</div>
              </td>
              <td>
                {{if .Bridge.Linked}}
                <div><span class="status {{syncClass .Bridge.SyncStatus}}">{{.Bridge.SyncStatus}}</span></div>
                <div class="small mono">{{.Bridge.SupplierRecordID}}</div>
                {{if .Bridge.VendureProductID}}<div class="small">Vendure {{.Bridge.VendureProductID}}</div>{{end}}
                {{if .Bridge.LastSyncError}}<div class="small">{{.Bridge.LastSyncError}}</div>{{end}}
                {{else}}
                <div class="small">unlinked</div>
                {{end}}
              </td>
              <td>
                <div>{{.AssetCount}} assets</div>
                <div class="small">{{.ProcessedCount}} processed / {{.FailedCount}} failed</div>
              </td>
              <td>
                <div class="actions">
                  <form method="post" action="/_/source-review-workbench/product/status">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <input type="hidden" name="status" value="approved">
                    <button class="secondary" type="submit">Approve</button>
                  </form>
                  <form method="post" action="/_/source-review-workbench/product/status">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <input type="hidden" name="status" value="rejected">
                    <button class="danger" type="submit">Reject</button>
                  </form>
                  <form method="post" action="/_/source-review-workbench/product/promote">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <button type="submit">Promote</button>
                  </form>
                  <form method="post" action="/_/source-review-workbench/product/promote-sync">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <button type="submit">Promote & Sync</button>
                  </form>
                  {{if and .Bridge.Linked (eq .Bridge.SyncStatus "error")}}
                  <form method="post" action="/_/source-review-workbench/product/retry-sync">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <button class="warn" type="submit">Retry Sync</button>
                  </form>
                  {{end}}
                </div>
              </td>
            </tr>
          {{else}}
            <tr><td colspan="6" class="small">暂无 source_products 记录。</td></tr>
          {{end}}
          </tbody>
          </table></div>
        </form>
        <div class="pager">
          {{if gt .Summary.ProductPage 1}}<a href="{{productPageURL .Filter (dec .Summary.ProductPage)}}">上一页</a>{{end}}
          <span class="small">产品分页 {{.Summary.ProductPage}} / {{.Summary.ProductPages}}</span>
          {{if lt .Summary.ProductPage .Summary.ProductPages}}<a href="{{productPageURL .Filter (inc .Summary.ProductPage)}}">下一页</a>{{end}}
        </div>
      </section>

      <section class="card">
        <div class="section-head">
          <h2>图片 AI 处理</h2>
          <div class="small">第 {{.Summary.AssetPage}} / {{.Summary.AssetPages}} 页，共 {{len .Summary.Assets}} 条当前页记录</div>
        </div>
        <form method="post" action="/_/source-review-workbench/assets/batch-process">
          <div class="bulkbar">
            <button type="submit">批量 Process Selected</button>
          </div>
          <div class="table-wrap"><table>
          <thead><tr><th>图片</th><th>状态</th><th>操作</th></tr></thead>
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
                    <div class="small"><a href="/_/source-review-workbench/asset?id={{.ID}}">查看图片详情</a></div>
                  </div>
                </div>
              </td>
              <td>
                <span class="status {{assetClass .ImageProcessingStatus}}">{{.ImageProcessingStatus}}</span>
                <div class="small">{{.ImageProcessingError}}</div>
              </td>
              <td>
                <div class="actions">
                  <form method="post" action="/_/source-review-workbench/assets/process">
                    <input type="hidden" name="id" value="{{.ID}}">
                    <button type="submit">Process</button>
                  </form>
                  {{if .ProcessedImageURL}}<a class="small" href="{{.ProcessedImageURL}}" target="_blank" rel="noreferrer">处理图</a>{{end}}
                  <a class="small" href="{{.SourceURL}}" target="_blank" rel="noreferrer">原图</a>
                </div>
              </td>
            </tr>
          {{else}}
            <tr><td colspan="3" class="small">暂无 source_assets 记录。</td></tr>
          {{end}}
          </tbody>
          </table></div>
        </form>
        <div class="pager">
          {{if gt .Summary.AssetPage 1}}<a href="{{assetPageURL .Filter (dec .Summary.AssetPage)}}">上一页</a>{{end}}
          <span class="small">资产分页 {{.Summary.AssetPage}} / {{.Summary.AssetPages}}</span>
          {{if lt .Summary.AssetPage .Summary.AssetPages}}<a href="{{assetPageURL .Filter (inc .Summary.AssetPage)}}">下一页</a>{{end}}
        </div>
      </section>
    </div>
  </div>
</body>
</html>`

	type pageData struct {
		Summary      pim.SourceReviewWorkbenchSummary
		Filter       pim.SourceReviewFilter
		FlashMessage string
		FlashError   string
	}

	tpl := template.Must(template.New("source-review-workbench").Funcs(template.FuncMap{
		"reviewClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case "approved", "promoted":
				return "ok"
			case "rejected":
				return "danger"
			default:
				return "warning"
			}
		},
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
		"syncClass": func(status string) string {
			switch strings.ToLower(strings.TrimSpace(status)) {
			case pim.StatusSynced:
				return "ok"
			case pim.StatusError:
				return "danger"
			case pim.StatusApproved, pim.StatusReady:
				return "warning"
			default:
				return ""
			}
		},
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
		"inc":         func(v int) int { return v + 1 },
		"dec": func(v int) int {
			if v <= 1 {
				return 1
			}
			return v - 1
		},
		"productPageURL": func(filter pim.SourceReviewFilter, page int) string {
			return sourceWorkbenchURL(filter, page, filter.AssetPage)
		},
		"assetPageURL": func(filter pim.SourceReviewFilter, page int) string {
			return sourceWorkbenchURL(filter, filter.ProductPage, page)
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData{
		Summary:      summary,
		Filter:       filter,
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}); err != nil {
		return fmt.Sprintf("<pre>render source review workbench failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "source", "Source Module", "商品审核、图片处理、同步重试和操作日志。")
}

func sourceWorkbenchURL(filter pim.SourceReviewFilter, productPage int, assetPage int) string {
	values := url.Values{}
	if v := strings.TrimSpace(filter.ProductStatus); v != "" {
		values.Set("productStatus", v)
	}
	if v := strings.TrimSpace(filter.AssetStatus); v != "" {
		values.Set("assetStatus", v)
	}
	if v := strings.TrimSpace(filter.SyncState); v != "" {
		values.Set("syncState", v)
	}
	if v := strings.TrimSpace(filter.Query); v != "" {
		values.Set("q", v)
	}
	if filter.PageSize > 0 {
		values.Set("pageSize", strconv.Itoa(filter.PageSize))
	}
	if productPage > 1 {
		values.Set("productPage", strconv.Itoa(productPage))
	}
	if assetPage > 1 {
		values.Set("assetPage", strconv.Itoa(assetPage))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/_/source-review-workbench"
	}
	return "/_/source-review-workbench?" + encoded
}
