package admin

import (
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/pim"
)

func RenderTargetSyncRunDetailHTML(run pim.TargetSyncRun, backHref string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>同步运行详情</title>
  <style>
    :root { --panel:rgba(8,20,32,.92); --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --accent:#5ee6ff; --ok:#6ef2b4; --warning:#ffd166; --danger:#ff6b8a; --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:1280px; margin:0 auto; padding:24px; }
    .hero,.content { display:grid; gap:14px; }
    .hero { grid-template-columns:1fr .9fr; }
    .content { grid-template-columns:1fr; margin-top:14px; }
    .card { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:20px; padding:16px; box-shadow:var(--shadow); }
    .badge { display:inline-block; padding:4px 8px; border-radius:999px; font-size:11px; font-weight:700; border:1px solid rgba(255,255,255,.08); }
    .badge.ok { color:var(--ok); background:rgba(110,242,180,.12); border-color:rgba(110,242,180,.18); }
    .badge.warning { color:var(--warning); background:rgba(255,209,102,.12); border-color:rgba(255,209,102,.18); }
    .badge.danger { color:var(--danger); background:rgba(255,107,138,.12); border-color:rgba(255,107,138,.18); }
    .small { color:var(--muted); font-size:12px; }
    .stats { display:grid; grid-template-columns:repeat(auto-fit,minmax(150px,1fr)); gap:12px; margin-top:14px; }
    table { width:100%; border-collapse:collapse; }
    th, td { text-align:left; padding:10px 8px; border-bottom:1px solid var(--line); vertical-align:top; }
    th { color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:.05em; }
    .empty { color:var(--muted); }
    a { color:var(--accent); text-decoration:none; }
    @media (max-width: 980px) { .hero { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="hero">
      <div class="card">
        <div class="small">同步运行详情</div>
        <h2 style="margin:8px 0 0;">{{entityLabel .EntityType}} / {{.JobName}}</h2>
        <div style="margin-top:10px;">
          <span class="badge {{statusClass .Status}}">{{statusLabel .Status}}</span>
        </div>
        <div class="small" style="margin-top:10px;">范围：{{.ScopeLabel}} / 开始：{{.StartedAt}} / 完成：{{.FinishedAt}}</div>
        <div class="small">触发人：{{actorLabel .TriggeredByName .TriggeredByEmail}}</div>
        {{if .ErrorMessage}}<div class="small" style="margin-top:10px;">错误：{{.ErrorMessage}}</div>{{end}}
        <div class="small" style="margin-top:10px;"><a href="{{.BackHref}}">返回目标同步</a></div>
      </div>
      <div class="card">
        <div class="small">统计</div>
        <div class="stats">
          <div><div class="small">新增</div><div style="font-size:24px; font-weight:800;">{{.CreatedCount}}</div></div>
          <div><div class="small">更新</div><div style="font-size:24px; font-weight:800;">{{.UpdatedCount}}</div></div>
          <div><div class="small">未变</div><div style="font-size:24px; font-weight:800;">{{.UnchangedCount}}</div></div>
          <div><div class="small">范围</div><div style="font-size:24px; font-weight:800;">{{.ScopedNodeCount}}</div></div>
        </div>
      </div>
    </section>
    <section class="content">
      <div class="card">
        <h3 style="margin-top:0;">变更明细</h3>
        <table>
          <thead><tr><th>类型</th><th>目标</th><th>标识</th><th>路径/角色</th><th>说明</th></tr></thead>
          <tbody>
          {{range .Details}}
            <tr>
              <td><span class="badge {{changeClass .ChangeType}}">{{changeLabel .ChangeType}}</span></td>
              <td><strong>{{entityLabel .TargetType}}</strong><div class="small">{{.Label}}</div></td>
              <td class="small">{{.TargetKey}}</td>
              <td class="small">{{.Path}}</td>
              <td class="small">{{.Note}}</td>
            </tr>
          {{else}}
            <tr><td colspan="5" class="empty">这次运行没有记录到具体变更明细。</td></tr>
          {{end}}
          </tbody>
        </table>
      </div>
    </section>
  </div>
</body>
</html>`

	tpl := template.Must(template.New("target-sync-run-detail").Funcs(template.FuncMap{
		"entityLabel": targetSyncEntityLabelHTML,
		"statusClass": targetSyncStatusClassHTML,
		"statusLabel": targetSyncStatusLabelHTML,
		"changeClass": func(changeType string) string {
			switch strings.ToLower(strings.TrimSpace(changeType)) {
			case "created":
				return "ok"
			case "updated":
				return "warning"
			default:
				return ""
			}
		},
		"changeLabel": func(changeType string) string {
			switch strings.ToLower(strings.TrimSpace(changeType)) {
			case "created":
				return "新增"
			case "updated":
				return "更新"
			default:
				return changeType
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
	if err := tpl.Execute(&builder, map[string]any{
		"ID":               run.ID,
		"EntityType":       run.EntityType,
		"JobName":          run.JobName,
		"Status":           run.Status,
		"ScopeLabel":       run.ScopeLabel,
		"StartedAt":        run.StartedAt,
		"FinishedAt":       run.FinishedAt,
		"TriggeredByName":  run.TriggeredByName,
		"TriggeredByEmail": run.TriggeredByEmail,
		"CreatedCount":     run.CreatedCount,
		"UpdatedCount":     run.UpdatedCount,
		"UnchangedCount":   run.UnchangedCount,
		"ScopedNodeCount":  run.ScopedNodeCount,
		"ErrorMessage":     run.ErrorMessage,
		"Details":          run.Details,
		"BackHref":         strings.TrimSpace(backHref),
	}); err != nil {
		return fmt.Sprintf("<pre>render target sync run detail failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "target-sync", "同步运行详情", "查看单次目标站同步到底改了什么。")
}

func targetSyncEntityLabelHTML(entityType string) string {
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case pim.TargetSyncEntityProducts:
		return "商品规格"
	case pim.TargetSyncEntityAssets:
		return "图片资产"
	default:
		return "分类树"
	}
}

func targetSyncStatusClassHTML(status string) string {
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
}

func targetSyncStatusLabelHTML(status string) string {
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
}
