package admin

import (
	"fmt"
	"html/template"
	"strings"
)

type adminNavItem struct {
	Label  string
	Href   string
	Active bool
}

func decorateAdminPageHTML(raw string, section string, title string, subtitle string) string {
	raw = injectAdminShellStyle(raw)
	raw = strings.Replace(raw, "<body>", "<body>"+renderAdminShellOpen(section, title, subtitle), 1)
	raw = strings.Replace(raw, "<div class=\"wrap\">", "<div class=\"admin-page wrap\">", 1)
	raw = strings.Replace(raw, "</body>", renderAdminShellClose()+"</body>", 1)
	return raw
}

func injectAdminShellStyle(raw string) string {
	const shellCSS = `
    .admin-shell { display:grid; grid-template-columns: 260px minmax(0,1fr); min-height:100vh; }
    .admin-sidebar {
      position: sticky;
      top: 0;
      align-self: start;
      height: 100vh;
      padding: 20px 16px;
      border-right: 1px solid rgba(123, 168, 203, 0.12);
      background: linear-gradient(180deg, rgba(5,15,24,.96) 0%, rgba(6,18,30,.92) 100%);
      backdrop-filter: blur(18px);
    }
    .admin-brand { padding: 10px 12px 18px; }
    .admin-brand .kicker { font-size: 11px; letter-spacing: .18em; text-transform: uppercase; color: var(--accent); }
    .admin-brand .name { margin-top: 8px; font-size: 24px; font-weight: 800; letter-spacing: -.04em; color: var(--ink); }
    .admin-brand .sub { margin-top: 6px; color: var(--muted); font-size: 12px; line-height: 1.5; }
    .admin-nav { display: grid; gap: 8px; margin-top: 10px; }
    .admin-nav a {
      display:block;
      padding: 12px 14px;
      border-radius: 14px;
      text-decoration:none;
      color: var(--muted);
      border: 1px solid transparent;
      background: transparent;
      transition: .18s ease;
    }
    .admin-nav a:hover { color: var(--ink); border-color: var(--line); background: rgba(255,255,255,.04); }
    .admin-nav a.active {
      color: var(--ink);
      border-color: rgba(94,230,255,.22);
      background: linear-gradient(180deg, rgba(94,230,255,.12) 0%, rgba(140,123,255,.08) 100%);
      box-shadow: inset 0 1px 0 rgba(255,255,255,.05);
    }
    .admin-nav .nav-title { font-size: 14px; font-weight: 700; }
    .admin-nav .nav-desc { margin-top: 4px; font-size: 12px; line-height: 1.4; }
    .admin-main { min-width: 0; }
    .admin-topbar {
      position: sticky;
      top: 0;
      z-index: 10;
      display:flex;
      justify-content:space-between;
      align-items:flex-end;
      gap: 16px;
      padding: 18px 24px 14px;
      border-bottom: 1px solid rgba(123, 168, 203, 0.12);
      background: linear-gradient(180deg, rgba(5,15,24,.92) 0%, rgba(5,15,24,.72) 100%);
      backdrop-filter: blur(16px);
    }
    .admin-topbar h1 { margin: 0; font-size: 28px; letter-spacing: -.04em; }
    .admin-topbar p { margin: 6px 0 0; color: var(--muted); font-size: 13px; }
    .admin-breadcrumbs { display:flex; flex-wrap:wrap; gap:8px; align-items:center; margin-bottom:8px; color: var(--muted); font-size:12px; }
    .admin-breadcrumbs a { color: var(--accent); text-decoration:none; }
    .admin-breadcrumbs .sep { opacity:.5; }
    .admin-topbar .top-actions { display:flex; flex-wrap:wrap; gap:8px; }
    .admin-topbar .top-actions a {
      padding: 9px 12px;
      border-radius: 12px;
      border: 1px solid var(--line);
      text-decoration: none;
      color: var(--accent);
      background: rgba(255,255,255,.04);
      font-size: 12px;
      font-weight: 700;
    }
    .admin-toast-stack {
      position: fixed;
      top: 18px;
      right: 22px;
      z-index: 40;
      display: grid;
      gap: 10px;
      width: min(380px, calc(100vw - 28px));
      pointer-events: none;
    }
    .admin-toast-stack .flash {
      margin: 0;
      box-shadow: 0 18px 40px rgba(0,0,0,.32);
      backdrop-filter: blur(14px);
      pointer-events: auto;
    }
    form[data-confirm] button,
    button[data-confirm] {
      position: relative;
    }
    .admin-toolbar {
      display:flex;
      flex-wrap:wrap;
      gap:10px;
      align-items:end;
      margin-bottom:14px;
      padding:14px;
      border:1px solid rgba(123, 168, 203, 0.12);
      border-radius:18px;
      background:rgba(255,255,255,.03);
    }
    .admin-page.wrap { max-width: none; margin: 0; padding: 24px; }
    @media (max-width: 980px) {
      .admin-shell { grid-template-columns: 1fr; }
      .admin-sidebar {
        position: relative;
        height: auto;
        border-right: 0;
        border-bottom: 1px solid rgba(123, 168, 203, 0.12);
      }
      .admin-topbar { position: relative; padding: 16px 20px 12px; }
      .admin-page.wrap { padding: 20px; }
    }
    @media (max-width: 640px) {
      .admin-toast-stack { left: 14px; right: 14px; width: auto; }
    }`

	const shellJS = `
  <script>
    (function () {
      function ready(fn) {
        if (document.readyState === "loading") {
          document.addEventListener("DOMContentLoaded", fn, { once: true });
          return;
        }
        fn();
      }
      ready(function () {
        var stack = document.querySelector(".admin-toast-stack");
        if (stack) {
          var flashes = Array.prototype.slice.call(document.querySelectorAll(".admin-page .flash"));
          flashes.forEach(function (flash, index) {
            stack.appendChild(flash);
            window.setTimeout(function () {
              flash.style.opacity = "0";
              flash.style.transform = "translateY(-6px)";
              flash.style.transition = "opacity .24s ease, transform .24s ease";
            }, 3200 + index * 300);
            window.setTimeout(function () {
              if (flash.parentNode) {
                flash.parentNode.removeChild(flash);
              }
            }, 3700 + index * 300);
          });
        }

        document.addEventListener("submit", function (event) {
          var form = event.target;
          if (!(form instanceof HTMLFormElement)) {
            return;
          }
          var message = form.getAttribute("data-confirm");
          if (!message) {
            return;
          }
          if (!window.confirm(message)) {
            event.preventDefault();
          }
        });

        document.addEventListener("click", function (event) {
          var target = event.target;
          if (!(target instanceof HTMLElement)) {
            return;
          }
          var button = target.closest("button[data-confirm]");
          if (!button) {
            return;
          }
          var message = button.getAttribute("data-confirm");
          if (message && !window.confirm(message)) {
            event.preventDefault();
          }
        });
      });
    })();
  </script>`

	raw = strings.Replace(raw, "</style>", shellCSS+"\n  </style>", 1)
	return strings.Replace(raw, "</body>", shellJS+"\n</body>", 1)
}

func renderAdminShellOpen(section string, title string, subtitle string) string {
	items := []adminNavItem{
		{Label: "总览", Href: "/_/mrtang-admin", Active: section == "dashboard"},
		{Label: "目标同步", Href: "/_/mrtang-admin/target-sync", Active: section == "target-sync"},
		{Label: "源数据", Href: "/_/mrtang-admin/source", Active: section == "source"},
		{Label: "采购", Href: "/_/mrtang-admin/procurement", Active: section == "procurement"},
		{Label: "审计", Href: "/_/mrtang-admin/audit", Active: section == "audit"},
		{Label: "集合", Href: "/_/", Active: section == "pocketbase"},
	}

	var nav strings.Builder
	for _, item := range items {
		cls := ""
		if item.Active {
			cls = " active"
		}
		desc := ""
		switch item.Label {
		case "总览":
			desc = "总览、待办、异常与模块入口"
		case "源数据":
			desc = "商品审核、图片处理、同步重试"
		case "目标同步":
			desc = "分类树、子分类和后续商品同步"
		case "采购":
			desc = "采购单、导出和状态流转"
		case "审计":
			desc = "统一查看源数据与采购动作"
		case "集合":
			desc = "原生 Admin 与集合入口"
		}
		nav.WriteString(fmt.Sprintf(`<a class="%s" href="%s"><div class="nav-title">%s</div><div class="nav-desc">%s</div></a>`, strings.TrimSpace("nav-link"+cls), item.Href, item.Label, desc))
	}

	breadcrumbs := renderAdminBreadcrumbs(section, title)

	return `<div class="admin-shell"><aside class="admin-sidebar"><div class="admin-brand"><div class="kicker">Mrtang Control</div><div class="name">后台</div><div class="sub">统一后台骨架。总览负责入口与异常，源数据与采购负责实际工作台。</div></div><nav class="admin-nav">` +
		nav.String() +
		`</nav></aside><div class="admin-main"><header class="admin-topbar"><div>` + breadcrumbs + `<h1>` + template.HTMLEscapeString(title) + `</h1><p>` + template.HTMLEscapeString(subtitle) + `</p></div><div class="top-actions"><a href="/_/mrtang-admin">总览</a><a href="/_/mrtang-admin/target-sync">目标同步</a><a href="/_/mrtang-admin/source">源数据首页</a><a href="/_/mrtang-admin/source/products">商品</a><a href="/_/mrtang-admin/source/assets">图片</a><a href="/_/mrtang-admin/source/logs">日志</a><a href="/_/mrtang-admin/procurement">采购</a><a href="/_/mrtang-admin/audit">审计</a></div></header><div class="admin-toast-stack" aria-live="polite"></div>`
}

func renderAdminShellClose() string {
	return `</div></div>`
}

func renderAdminBreadcrumbs(section string, title string) string {
	items := []adminNavItem{{Label: "后台", Href: "/_/mrtang-admin"}}
	switch section {
	case "target-sync":
		items = append(items, adminNavItem{Label: "目标同步", Href: "/_/mrtang-admin/target-sync"})
	case "source":
		items = append(items, adminNavItem{Label: "源数据", Href: "/_/mrtang-admin/source"})
		if title == "源数据商品" || title == "源数据商品详情" {
			items = append(items, adminNavItem{Label: "商品", Href: "/_/mrtang-admin/source/products"})
		}
		if title == "源数据图片" || title == "源数据图片详情" {
			items = append(items, adminNavItem{Label: "图片", Href: "/_/mrtang-admin/source/assets"})
		}
		if title == "源数据操作日志" {
			items = append(items, adminNavItem{Label: "日志", Href: "/_/mrtang-admin/source/logs"})
		}
	case "procurement":
		items = append(items, adminNavItem{Label: "采购", Href: "/_/mrtang-admin/procurement"})
	case "audit":
		items = append(items, adminNavItem{Label: "审计", Href: "/_/mrtang-admin/audit"})
	}
	if title != "" && title != "统一后台入口" && title != "源数据首页" && title != "采购工作台" && title != "统一审计" &&
		title != "源数据商品" && title != "源数据图片" && title != "源数据操作日志" {
		items = append(items, adminNavItem{Label: title})
	}

	var builder strings.Builder
	builder.WriteString(`<nav class="admin-breadcrumbs" aria-label="面包屑">`)
	for i, item := range items {
		if i > 0 {
			builder.WriteString(`<span class="sep">/</span>`)
		}
		if item.Href != "" {
			builder.WriteString(`<a href="` + template.HTMLEscapeString(item.Href) + `">` + template.HTMLEscapeString(item.Label) + `</a>`)
			continue
		}
		builder.WriteString(`<span>` + template.HTMLEscapeString(item.Label) + `</span>`)
	}
	builder.WriteString(`</nav>`)
	return builder.String()
}
