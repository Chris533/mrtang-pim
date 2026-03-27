package admin

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html/template"
	"strings"

	"mrtang-pim/internal/adminapp"
)

func RenderAdminAppShellHTML(title string, subtitle string, currentPath string, canAccessSource bool, canAccessProcurement bool) string {
	cssHref := "/_/mrtang-admin/app.css?v=" + adminAssetVersion("static/app.css")
	appVersion := adminAssetVersion("static/app.js")
	jsHref := "/_/mrtang-admin/app.js?v=" + appVersion
	const page = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="{{.CSSHref}}">
</head>
<body>
  <div id="admin-app"></div>
  <script>
    window.__MRTANG_ADMIN_BOOT__ = {
      title: {{printf "%q" .Title}},
      subtitle: {{printf "%q" .Subtitle}},
      currentPath: {{printf "%q" .CurrentPath}},
      canAccessSource: {{.CanAccessSource}},
      canAccessProcurement: {{.CanAccessProcurement}},
      version: {{printf "%q" .Version}}
    };
  </script>
  <script type="module" src="{{.JSHref}}"></script>
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
		CSSHref              string
		JSHref               string
		CanAccessSource      bool
		CanAccessProcurement bool
		Version              string
	}

	tpl := template.Must(template.New("admin-app-shell").Parse(page))
	var builder strings.Builder
	if err := tpl.Execute(&builder, shellData{
		Title:                strings.TrimSpace(title),
		Subtitle:             strings.TrimSpace(subtitle),
		CurrentPath:          strings.TrimSpace(currentPath),
		CSSHref:              cssHref,
		JSHref:               jsHref,
		CanAccessSource:      canAccessSource,
		CanAccessProcurement: canAccessProcurement,
		Version:              appVersion,
	}); err != nil {
		return fmt.Sprintf("<pre>render admin app shell failed: %s</pre>", template.HTMLEscapeString(err.Error()))
	}
	return builder.String()
}

func adminAssetVersion(name string) string {
	body, err := adminapp.Static.ReadFile(name)
	if err != nil || len(body) == 0 {
		return "dev"
	}
	sum := sha1.Sum(body)
	return hex.EncodeToString(sum[:6])
}
