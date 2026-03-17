package admin

import (
	"fmt"
	"html/template"
	"strings"
)

func RenderAdminAppShellHTML(title string, subtitle string, currentPath string, canAccessSource bool, canAccessProcurement bool) string {
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="/_/mrtang-admin/app.css">
</head>
<body>
  <div id="admin-app"></div>
  <script>
    window.__MRTANG_ADMIN_BOOT__ = {
      title: {{printf "%q" .Title}},
      subtitle: {{printf "%q" .Subtitle}},
      currentPath: {{printf "%q" .CurrentPath}},
      canAccessSource: {{.CanAccessSource}},
      canAccessProcurement: {{.CanAccessProcurement}}
    };
  </script>
  <script type="module" src="/_/mrtang-admin/app.js"></script>
  <noscript>
    <main style="max-width:960px;margin:48px auto;padding:0 20px;font-family:Segoe UI,PingFang SC,sans-serif;color:#eef6ff;background:#08121c;">
      <h1>后台需要启用 JavaScript</h1>
      <p>当前总览和抓取入库已经迁到无构建 Preact 壳子。若要暂时回退，可在地址后加 <code>?legacy=1</code>。</p>
    </main>
  </noscript>
</body>
</html>`

	type shellData struct {
		Title                string
		Subtitle             string
		CurrentPath          string
		CanAccessSource      bool
		CanAccessProcurement bool
	}

	tpl := template.Must(template.New("admin-app-shell").Parse(page))
	var builder strings.Builder
	if err := tpl.Execute(&builder, shellData{
		Title:                strings.TrimSpace(title),
		Subtitle:             strings.TrimSpace(subtitle),
		CurrentPath:          strings.TrimSpace(currentPath),
		CanAccessSource:      canAccessSource,
		CanAccessProcurement: canAccessProcurement,
	}); err != nil {
		return fmt.Sprintf("<pre>render admin app shell failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return builder.String()
}
