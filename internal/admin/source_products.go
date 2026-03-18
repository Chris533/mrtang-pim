package admin

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderSourceProductsHTML(summary pim.SourceReviewWorkbenchSummary, filter pim.SourceReviewFilter, flashMessage string, flashError string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>源数据商品</title>
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
    .stats, .layout { display:grid; gap:14px; }
    .stats { grid-template-columns:repeat(auto-fit,minmax(170px,1fr)); }
    .layout { grid-template-columns:minmax(0,1fr) 320px; margin-top:14px; }
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
    button.danger { background:rgba(255,107,138,.12); color:var(--danger); border-color:rgba(255,107,138,.22); }
    .flash { margin-bottom:14px; padding:12px 14px; border-radius:14px; border:1px solid var(--line); }
    .flash.ok { color:var(--ok); background:rgba(110,242,180,.12); }
    .flash.error { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.2); }
    .table-wrap { overflow:auto; border-radius:16px; }
    table { width:100%; border-collapse:separate; border-spacing:0; min-width:920px; }
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
    @media (max-width: 1120px) { .layout { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    {{if .FlashMessage}}<div class="flash ok">{{.FlashMessage}}</div>{{end}}
    {{if .FlashError}}<div class="flash error">{{.FlashError}}</div>{{end}}

    <section class="stats">
      <div class="stat"><div class="eyebrow">商品</div><div class="metric">{{.Summary.ProductCount}}</div><div class="small">{{.Summary.ImportedCount}} 待审核 / {{.Summary.ApprovedCount}} 待加入发布队列 / {{.Summary.PromotedCount}} 已加入发布队列</div></div>
      <div class="stat"><div class="eyebrow">当前筛选</div><div class="metric">{{len .Summary.Products}}</div><div class="small">当前页 {{.Summary.ProductPage}} / {{.Summary.ProductPages}}</div></div>
      <div class="stat"><div class="eyebrow">发布队列</div><div class="metric">{{.Summary.LinkedCount}}</div><div class="small">{{.Summary.UnlinkedCount}} 未进入发布队列</div></div>
      <div class="stat"><div class="eyebrow">同步</div><div class="metric">{{.Summary.SyncedCount}}</div><div class="small">{{.Summary.SyncErrorCount}} 失败 / {{.Summary.ReadyToSyncCount}} 待同步</div></div>
    </section>

    <section class="layout">
      <div class="card">
        <form method="get" action="/_/mrtang-admin/source/products" class="filters admin-toolbar">
          <label>商品状态
            <select name="productStatus">
              <option value="">全部</option>
              <option value="imported" {{if eq .Filter.ProductStatus "imported"}}selected{{end}}>待审核</option>
              <option value="approved" {{if eq .Filter.ProductStatus "approved"}}selected{{end}}>待加入发布队列</option>
              <option value="promoted" {{if eq .Filter.ProductStatus "promoted"}}selected{{end}}>已加入发布队列</option>
              <option value="rejected" {{if eq .Filter.ProductStatus "rejected"}}selected{{end}}>已拒绝</option>
            </select>
          </label>
          <label>同步状态
            <select name="syncState">
              <option value="">全部</option>
              <option value="unlinked" {{if eq .Filter.SyncState "unlinked"}}selected{{end}}>未进入发布队列</option>
              <option value="linked" {{if eq .Filter.SyncState "linked"}}selected{{end}}>已加入发布队列</option>
              <option value="error" {{if eq .Filter.SyncState "error"}}selected{{end}}>同步失败</option>
              <option value="synced" {{if eq .Filter.SyncState "synced"}}selected{{end}}>已同步</option>
            </select>
          </label>
          <label>检索
            <input type="text" name="q" value="{{.Filter.Query}}" placeholder="商品名 / productId / 分类">
          </label>
          <label>每页
            <select name="pageSize">
              <option value="12" {{if eq .Filter.PageSize 12}}selected{{end}}>12</option>
              <option value="24" {{if or (eq .Filter.PageSize 0) (eq .Filter.PageSize 24)}}selected{{end}}>24</option>
              <option value="48" {{if eq .Filter.PageSize 48}}selected{{end}}>48</option>
            </select>
          </label>
          <button type="submit" class="secondary">应用筛选</button>
          <a class="link small" href="/_/mrtang-admin/source/products">重置</a>
          <a class="link small" href="/_/mrtang-admin/source/products?productStatus=imported">待审批</a>
          <a class="link small" href="/_/mrtang-admin/source/products?productStatus=approved">待加入发布队列</a>
          <a class="link small" href="/_/mrtang-admin/source/products?syncState=error">同步失败</a>
          <a class="link small" href="/_/mrtang-admin/source/products?syncState=unlinked">未进入发布队列</a>
          <a class="link small" href="/_/mrtang-admin/source/products?syncState=synced">已同步</a>
        </form>

        <form method="post" action="/_/mrtang-admin/source/products/batch-status">
          <input type="hidden" name="returnTo" value="{{.CurrentURL}}">
          <div class="bulkbar admin-toolbar" style="margin-bottom:12px;">
            <input type="hidden" name="status" value="approved">
            <button type="submit" class="secondary" data-confirm="确认批量通过当前选中的商品吗？">批量通过</button>
            <button type="submit" class="danger" formaction="/_/mrtang-admin/source/products/batch-status" onclick="this.form.status.value='rejected'" data-confirm="确认批量拒绝当前选中的商品吗？">批量拒绝</button>
            <button type="submit" formaction="/_/mrtang-admin/source/products/batch-promote" data-confirm="确认将当前选中的商品批量加入发布队列吗？">批量加入发布队列</button>
            <button type="submit" formaction="/_/mrtang-admin/source/products/batch-promote-sync" data-confirm="确认将当前选中的商品批量加入发布队列并发布到 Backend 吗？">批量加入发布队列并发布</button>
            <button type="submit" class="warn" formaction="/_/mrtang-admin/source/products/batch-retry-sync" data-confirm="确认批量重试当前选中商品发布到 Backend 吗？">批量重试发布</button>
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
                      <div class="small"><a class="link" href="/_/mrtang-admin/source/products/detail?id={{.ID}}&returnTo={{urlquery $.CurrentURL}}">查看详情</a></div>
                    </div>
                  </div>
                </td>
                <td><span class="status {{reviewClass .ReviewStatus}}">{{reviewLabel .ReviewStatus}}</span><div class="small">{{.SourceType}}</div></td>
                <td><div>{{printf "%.2f" .DefaultPrice}}</div><div class="small">{{.UnitCount}} units {{if .HasMultiUnit}}/ multi-unit{{end}}</div></td>
                <td>
                  {{if .Bridge.Linked}}
                    <div><span class="status {{syncClass .Bridge.SyncStatus}}">{{syncLabel .Bridge.SyncStatus .Bridge.Linked}}</span></div>
                    <div class="small mono">{{.Bridge.SupplierRecordID}}</div>
                    {{if .Bridge.VendureProductID}}<div class="small">后端 {{.Bridge.VendureProductID}}</div>{{end}}
                    {{if .Bridge.LastSyncError}}<div class="small">{{.Bridge.LastSyncError}}</div>{{end}}
                  {{else}}
                    <div class="small">unlinked</div>
                  {{end}}
                </td>
                <td><div>{{.AssetCount}} assets</div><div class="small">{{.ProcessedCount}} processed / {{.FailedCount}} failed</div></td>
                <td>
                  <div class="actions">
                    <form method="post" action="/_/mrtang-admin/source/products/status" data-confirm="确认将这个商品标记为通过吗？">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <input type="hidden" name="status" value="approved">
                      <input type="hidden" name="returnTo" value="{{$.CurrentURL}}">
                      <button class="secondary" type="submit">通过</button>
                    </form>
                    <form method="post" action="/_/mrtang-admin/source/products/status" data-confirm="确认拒绝这个商品吗？">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <input type="hidden" name="status" value="rejected">
                      <input type="hidden" name="returnTo" value="{{$.CurrentURL}}">
                      <button class="danger" type="submit">拒绝</button>
                    </form>
                    <form method="post" action="/_/mrtang-admin/source/products/promote" data-confirm="确认将这个商品加入发布队列吗？">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <input type="hidden" name="returnTo" value="{{$.CurrentURL}}">
                      <button type="submit">加入发布队列</button>
                    </form>
                    <form method="post" action="/_/mrtang-admin/source/products/promote-sync" data-confirm="确认将这个商品加入发布队列并立即发布到 Backend 吗？">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <input type="hidden" name="returnTo" value="{{$.CurrentURL}}">
                      <button type="submit">加入发布队列并发布</button>
                    </form>
                    {{if and .Bridge.Linked (eq .Bridge.SyncStatus "error")}}
                    <form method="post" action="/_/mrtang-admin/source/products/retry-sync" data-confirm="确认重试这个商品发布到 Backend 吗？">
                      <input type="hidden" name="id" value="{{.ID}}">
                      <input type="hidden" name="returnTo" value="{{$.CurrentURL}}">
                      <button class="warn" type="submit">重试发布</button>
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
          {{if gt .Summary.ProductPage 1}}<a href="{{pageURL .Filter (dec .Summary.ProductPage)}}">上一页</a>{{end}}
          <span class="small">产品分页 {{.Summary.ProductPage}} / {{.Summary.ProductPages}}</span>
          {{if lt .Summary.ProductPage .Summary.ProductPages}}<a href="{{pageURL .Filter (inc .Summary.ProductPage)}}">下一页</a>{{end}}
        </div>
      </div>

      <aside class="card">
        <div style="display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:12px;">
          <h3 style="margin:0;">快捷队列</h3>
          <a class="link small" href="/_/mrtang-admin/source">返回源数据首页</a>
        </div>
        <div class="queue-list">
          <div class="queue-item">
            <div><strong>待审批</strong> <span class="status warning">{{.Summary.ReadyToReviewCount}}</span></div>
            <div class="small">优先处理 imported 商品。</div>
          </div>
          <div class="queue-item">
            <div><strong>待加入发布队列</strong> <span class="status warning">{{.Summary.ReadyToPromoteCount}}</span></div>
            <div class="small">已通过商品需要进入发布队列，后续再同步到 Backend。</div>
          </div>
          <div class="queue-item">
            <div><strong>同步失败</strong> <span class="status danger">{{.Summary.SyncErrorCount}}</span></div>
            <div class="small">已加入发布队列商品失败后在这里重试发布。</div>
          </div>
          <div class="queue-item">
            <div><strong>失败图片</strong> <span class="status danger">{{.Summary.AssetFailed}}</span></div>
            <div class="small"><a class="link" href="/_/mrtang-admin/source/assets?assetStatus=failed">去图片页处理</a></div>
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

	tpl := template.Must(template.New("source-products").Funcs(template.FuncMap{
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
		"reviewLabel": sourceReviewStatusLabel,
		"syncLabel":   sourceSyncStatusLabel,
		"inc":         func(v int) int { return v + 1 },
		"dec": func(v int) int {
			if v <= 1 {
				return 1
			}
			return v - 1
		},
		"pageURL": func(filter pim.SourceReviewFilter, page int) string {
			return sourceProductsURL(filter, page)
		},
	}).Parse(page))

	var builder strings.Builder
	if err := tpl.Execute(&builder, pageData{
		Summary:      summary,
		Filter:       filter,
		CurrentURL:   sourceProductsURL(filter, filter.ProductPage),
		FlashMessage: strings.TrimSpace(flashMessage),
		FlashError:   strings.TrimSpace(flashError),
	}); err != nil {
		return fmt.Sprintf("<pre>render source products failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}

	return decorateAdminPageHTML(builder.String(), "source", "源数据商品", "独立的商品管理页，负责筛选、分页和批量审核、加入发布队列、发布。")
}

func sourceProductsURL(filter pim.SourceReviewFilter, page int) string {
	values := url.Values{}
	if v := strings.TrimSpace(filter.ProductStatus); v != "" {
		values.Set("productStatus", v)
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
	if page > 1 {
		values.Set("productPage", strconv.Itoa(page))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/_/mrtang-admin/source/products"
	}
	return "/_/mrtang-admin/source/products?" + encoded
}
