package admin

import (
	"fmt"
	"html/template"
	"strings"
)

func RenderForbiddenPageHTML(title string, message string, backHref string) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>无权限访问</title>
  <style>
    :root { --ink:#edf7ff; --muted:#8aa3bb; --line:rgba(123,168,203,.16); --danger:#ff6b8a; --panel:rgba(8,20,32,.92); --shadow:0 24px 60px rgba(0,0,0,.34); }
    * { box-sizing:border-box; }
    body { margin:0; font-family:"Segoe UI","PingFang SC",sans-serif; color:var(--ink); background:radial-gradient(circle at top right, rgba(94,230,255,.16) 0, transparent 24%), linear-gradient(180deg,#07111b 0%, #06131f 55%, #04101b 100%); }
    .wrap { max-width:780px; margin:0 auto; padding:28px 24px 44px; }
    .card { background:linear-gradient(180deg, rgba(18,38,58,.82) 0%, var(--panel) 100%); border:1px solid var(--line); border-radius:22px; padding:24px; box-shadow:var(--shadow); }
    h1,p { margin:0; }
    h1 { font-size:32px; letter-spacing:-.04em; }
    p { margin-top:10px; color:var(--muted); line-height:1.7; }
    a { display:inline-block; margin-top:16px; color:var(--danger); text-decoration:none; font-weight:700; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1>{{.Title}}</h1>
      <p>{{.Message}}</p>
      <a href="{{.BackHref}}">返回后台总览</a>
    </div>
  </div>
</body>
</html>`

	var builder strings.Builder
	tpl := template.Must(template.New("forbidden").Parse(page))
	if err := tpl.Execute(&builder, map[string]string{
		"Title":    strings.TrimSpace(title),
		"Message":  strings.TrimSpace(message),
		"BackHref": defaultBackHref(backHref),
	}); err != nil {
		return fmt.Sprintf("<pre>render forbidden page failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return decorateAdminPageHTML(builder.String(), "dashboard", title, message)
}

func defaultBackHref(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/_/mrtang-admin"
	}
	return value
}
