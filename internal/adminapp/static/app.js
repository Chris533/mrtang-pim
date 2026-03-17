import {
  h,
  html,
  render,
  useEffect,
  useMemo,
  useState
} from "./vendor/htm-preact-standalone.mjs";
const boot = window.__MRTANG_ADMIN_BOOT__ || {};

function parseQuery() {
  const params = new URLSearchParams(window.location.search);
  return { message: params.get("message") || "", error: params.get("error") || "" };
}

function exportedKeyAlias(key) {
  if (!key) return key;
  if (key === "id") return "ID";
  const initial = key.charAt(0).toUpperCase() + key.slice(1);
  return initial
    .replace(/Id/g, "ID")
    .replace(/Ids/g, "IDs")
    .replace(/Url/g, "URL")
    .replace(/Urls/g, "URLs")
    .replace(/Json/g, "JSON")
    .replace(/Sku/g, "SKU");
}

function withExportedKeys(value) {
  if (Array.isArray(value)) {
    return value.map((item) => withExportedKeys(item));
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  const next = {};
  Object.entries(value).forEach(([key, current]) => {
    const normalized = withExportedKeys(current);
    next[key] = normalized;
    const alias = exportedKeyAlias(key);
    if (alias && alias !== key && next[alias] === undefined) {
      next[alias] = normalized;
    }
  });
  return next;
}

async function fetchJSON(url) {
  const response = await fetch(url, { headers: { Accept: "application/json" }, credentials: "same-origin" });
  if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
  return withExportedKeys(await response.json());
}

async function postForm(url, values) {
  const body = new URLSearchParams();
  Object.entries(values || {}).forEach(([key, value]) => {
    if (value === undefined || value === null) return;
    body.set(key, String(value));
  });
  const response = await fetch(url, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/x-www-form-urlencoded; charset=UTF-8",
    },
    credentials: "same-origin",
    body,
  });
  const text = await response.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    data = { message: text };
  }
  if (!response.ok) {
    throw new Error((data && data.message) || text || `HTTP ${response.status}`);
  }
  return data || {};
}

function useResource(url, deps = []) {
  const [state, setState] = useState({ loading: true, error: "", data: null });
  useEffect(() => {
    let active = true;
    setState({ loading: true, error: "", data: null });
    fetchJSON(url)
      .then((data) => active && setState({ loading: false, error: "", data }))
      .catch((error) => active && setState({ loading: false, error: error.message || "加载失败", data: null }));
    return () => { active = false; };
  }, [url, ...deps]);
  return state;
}

function tone(value) {
  const normalized = (value || "").toLowerCase();
  if (["raw", "success", "raw_live", "ok", "synced", "completed", "processed", "downloaded"].includes(normalized)) return "success";
  if (["failed", "error", "explicit_write"].includes(normalized)) return "danger";
  return "warning";
}

function sourceModeLabel(mode) {
  const normalized = (mode || "").toLowerCase();
  if (normalized === "raw") return "RAW Source";
  if (normalized === "snapshot") return "Snapshot Source";
  return mode || "未识别来源";
}

function checkoutStatusLabel(status) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "raw_live") return "raw live";
  if (normalized === "raw_readonly") return "raw 只读";
  if (normalized === "explicit_write") return "显式真实写入";
  return "fallback";
}

function syncStatusLabel(status) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "pending") return "待执行";
  if (normalized === "running") return "执行中";
  if (normalized === "success") return "成功";
  if (normalized === "partial") return "部分成功";
  if (normalized === "failed") return "失败";
  return status || "-";
}

function progressStageLabel(stage) {
  const normalized = (stage || "").toLowerCase();
  if (normalized === "queued") return "排队中";
  if (normalized === "loading_dataset") return "加载数据集";
  if (normalized === "categories") return "写入分类";
  if (normalized === "products") return "写入商品规格";
  if (normalized === "assets") return "写入图片资源";
  if (normalized === "completed") return "已完成";
  return stage || "-";
}

function originalImageStatusLabel(status) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "pending") return "待下载";
  if (normalized === "downloading") return "下载中";
  if (normalized === "downloaded") return "已下载";
  if (normalized === "failed") return "下载失败";
  return status || "-";
}

function sourceAssetJobTypeLabel(jobType, mode) {
  const normalized = (jobType || "").toLowerCase();
  if (normalized === "download_original") return "原图下载";
  if (normalized === "process_asset") {
    return (mode || "").toLowerCase() === "failed" ? "失败图片重处理" : "图片处理";
  }
  return jobType || "-";
}

function sourceAssetJobRecentError(item) {
  if (item && item.Error) return item.Error;
  const logs = (item && item.Logs) || [];
  for (let index = logs.length - 1; index >= 0; index -= 1) {
    const message = (logs[index] && logs[index].Message) || "";
    if (message.includes("失败")) return message;
  }
  return "";
}

function sourceAssetJobTargetHref(item) {
  const jobType = ((item && item.JobType) || "").toLowerCase();
  const mode = ((item && item.Mode) || "").toLowerCase();
  if (jobType === "download_original") {
    return buildURL("/_/mrtang-admin/source/assets", { originalStatus: "failed" });
  }
  if (jobType === "process_asset" && mode === "failed") {
    return buildURL("/_/mrtang-admin/source/assets", { assetStatus: "failed" });
  }
  if (jobType === "process_asset") {
    return buildURL("/_/mrtang-admin/source/assets", { assetStatus: "pending" });
  }
  return "/_/mrtang-admin/source/assets";
}

function sourceAssetJobTargetLabel(item) {
  const jobType = ((item && item.JobType) || "").toLowerCase();
  const mode = ((item && item.Mode) || "").toLowerCase();
  if (jobType === "download_original") return "查看原图失败图片";
  if (jobType === "process_asset" && mode === "failed") return "查看处理失败图片";
  if (jobType === "process_asset") return "查看待处理图片";
  return "查看相关图片";
}

function NavLink({ href, label, active }) {
  return html`<a class=${`nav-link${active ? " active" : ""}`} href=${href}>${label}</a>`;
}

function FlashStack() {
  const query = useMemo(() => parseQuery(), []);
  if (!query.message && !query.error) return null;
  return html`<div class="flash-stack">
    ${query.message ? html`<div class="flash ok">${query.message}</div>` : null}
    ${query.error ? html`<div class="flash error">${query.error}</div>` : null}
  </div>`;
}

function StatusBadge({ label, currentTone }) {
  return html`<span class=${`status-badge ${currentTone}`}>${label}</span>`;
}

function MetricCard({ eyebrow, value, detail }) {
  return html`<div class="metric-card"><div class="metric-kicker">${eyebrow}</div><div class="metric-value">${value}</div>${detail ? html`<div class="small" style="margin-top:8px;">${detail}</div>` : null}</div>`;
}

function AppLayout({ title, subtitle, currentPath, children }) {
  const navItems = [
    { href: "/_/mrtang-admin", label: "总览", visible: true },
    { href: "/_/mrtang-admin/target-sync", label: "抓取入库", visible: !!boot.canAccessSource },
    { href: "/_/mrtang-admin/source", label: "源数据", visible: !!boot.canAccessSource },
    { href: "/_/mrtang-admin/procurement", label: "采购", visible: !!boot.canAccessProcurement },
    { href: "/_/mrtang-admin/audit", label: "审计", visible: true },
  ].filter((item) => item.visible);
  const topLinks = [
    ...navItems,
    ...(boot.canAccessSource ? [
      { href: "/_/mrtang-admin/source/categories", label: "分类" },
      { href: "/_/mrtang-admin/source/products", label: "商品" },
      { href: "/_/mrtang-admin/source/assets", label: "图片" },
      { href: "/_/mrtang-admin/source/asset-jobs", label: "任务" },
      { href: "/_/mrtang-admin/source/logs", label: "日志" },
    ] : []),
  ];

  return html`
    <div class="admin-shell">
      <aside class="admin-sidebar">
        <div class="brand">
          <div class="brand-kicker">Mrtang Admin</div>
          <div class="brand-title">统一后台</div>
          <div class="brand-desc">页面先开壳，再异步加载各模块数据。raw 慢时只影响局部卡片，不再拖死整页。</div>
        </div>
        <div class="nav-group">
          <div class="nav-label">导航</div>
          ${navItems.map((item) => html`<${NavLink} key=${item.href} href=${item.href} label=${item.label} active=${currentPath === item.href} />`)}
        </div>
      </aside>
      <main class="admin-main">
        <header class="admin-topbar">
          <div>
            <div class="breadcrumbs"><a href="/_/mrtang-admin">后台</a><span>/</span><span>${title}</span></div>
            <h1 class="page-title">${title}</h1>
            <p class="page-subtitle">${subtitle}</p>
          </div>
          <div class="top-actions">
            ${topLinks.map((item) => html`<a href=${item.href}>${item.label}</a>`)}
          </div>
        </header>
        <${FlashStack} />
        ${children}
      </main>
    </div>
  `;
}

function LoadingSection({ label }) {
  return html`<div class="loading-state">${label}加载中...</div>`;
}

function ErrorSection({ error }) {
  return html`<div class="error-state">加载失败：${error}</div>`;
}

function ActionNotice({ state }) {
  if (!state) return null;
  if (state.error) return html`<div class="flash error" style="margin-top:14px;">${state.error}</div>`;
  if (state.message) return html`<div class="flash ok" style="margin-top:14px;">
    <div>${state.message}</div>
    ${state.href ? html`<div style="margin-top:10px;"><a class="btn secondary" href=${state.href}>${state.hrefLabel || "查看结果"}</a></div>` : null}
  </div>`;
  return null;
}

function DashboardPage() {
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const resource = useResource("/api/pim/admin/dashboard", [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="总览数据" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const data = resource.data || {};
  const miniapp = data.Miniapp || {};
  const source = data.SourceCapture || {};
  const procurement = data.Procurement || {};

  async function importSource(scope) {
    setActionState({ busy: "import-source", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/import", { scope });
      setActionState({ busy: "", message: result.message || "源数据导入完成。", error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "导入源数据失败" });
    }
  }

  return html`
    <div class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">Miniapp Source Coverage</div>
        <h2 class="card-title">当前数据源覆盖</h2>
        <div class="inline-pills">
          <${StatusBadge} label=${sourceModeLabel(miniapp.SourceMode)} currentTone=${tone(miniapp.SourceMode)} />
          <span class="pill">configMode: <code>${miniapp.ConfigSourceMode || "-"}</code></span>
          <span class="pill">datasetSource: <code>${miniapp.DatasetSource || "-"}</code></span>
          <span class="pill">sourceURL: <code>${miniapp.SourceURL || "-"}</code></span>
        </div>
        ${data.MiniappError ? html`<div class="flash error" style="margin-top:14px;">${data.MiniappError}</div>` : null}
        <div class="metric-grid section">
          <${MetricCard} eyebrow="Contracts" value=${miniapp.ContractCount || 0} detail=${`Dataset source: ${miniapp.DatasetSource || "-"}`} />
          <${MetricCard} eyebrow="Homepage" value=${miniapp.HomepageSectionCount || 0} detail=${`${miniapp.HomepageProductCount || 0} 个首页商品`} />
          <${MetricCard} eyebrow="Category Tree" value=${miniapp.CategoryTopLevelCount || 0} detail=${`${miniapp.CategoryNodeCount || 0} 个分类节点`} />
          <${MetricCard} eyebrow="Category Sections" value=${miniapp.CategorySectionCount || 0} detail=${`${miniapp.CategorySectionWithProducts || 0} 个带商品`} />
          <${MetricCard} eyebrow="Products" value=${miniapp.ProductTotal || 0} detail=${`${miniapp.ProductRRDetailCount || 0} rr_detail / ${miniapp.ProductSkeletonCount || 0} skeleton`} />
          <${MetricCard} eyebrow="Checkout" value=${miniapp.OrderOperationCount || 0} detail=${`${miniapp.CartOperationCount || 0} cart / ${miniapp.FreightScenarioCount || 0} freight`} />
        </div>
        <div class="inline-pills">
          <span class="pill">multiUnitVisible: <code>${miniapp.MultiUnitTotal || 0}</code></span>
          <span class="pill">categoryProducts: <code>${miniapp.CategoryProductCount || 0}</code></span>
        </div>
      </div></section>

      <section class="card"><div class="card-body">
        <div class="card-kicker">Source Capture</div>
        <h2 class="card-title">PocketBase 落库状态</h2>
        ${data.SourceError ? html`<div class="flash error" style="margin-top:14px;">${data.SourceError}</div>` : null}
        <div class="metric-grid section">
          <${MetricCard} eyebrow="Categories" value=${source.CategoryCount || 0} />
          <${MetricCard} eyebrow="Products" value=${source.ProductCount || 0} detail=${`${source.ImportedCount || 0} imported / ${source.ApprovedCount || 0} approved / ${source.PromotedCount || 0} promoted`} />
          <${MetricCard} eyebrow="Assets" value=${source.AssetCount || 0} detail=${`${source.ProcessedAssetCount || 0} processed / ${source.FailedAssetCount || 0} failed`} />
          <${MetricCard} eyebrow="Bridge" value=${source.LinkedCount || 0} detail=${`${source.SyncedCount || 0} synced / ${source.SyncErrorCount || 0} error`} />
        </div>
      </div></section>
    </div>

    <section class="section card"><div class="card-body">
      <div class="card-kicker">入口与动作</div>
      <h2 class="card-title">关键操作</h2>
      <${ActionNotice} state=${actionState} />
      <div class="ops-grid section">
        ${(data.QuickActions || []).map((item) => html`<a class="action-card" href=${item.Href}><div class="card-kicker">${item.Eyebrow}</div><div class="card-title">${item.Title}</div><div class="card-desc">${item.Desc}</div></a>`)}
        ${data.CanAccessSource ? html`<div class="action-card"><div class="card-kicker">Import</div><div class="card-title">抓取并保存分类、商品、图片</div><div class="card-desc">从当前 miniapp dataset 导入到 source collections。</div><div class="action-row"><button class="btn" type="button" disabled=${actionState.busy === "import-source"} onClick=${() => importSource("all")}>${actionState.busy === "import-source" ? "导入中..." : "立即导入"}</button></div></div>` : null}
      </div>
    </div></section>

    <section class="section split-grid">
      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">近期动作</div>
        <h2 class="card-title">统一最近操作</h2>
        <div class="table-wrap section"><table><thead><tr><th>域</th><th>目标</th><th>结果</th><th>操作人</th><th>时间</th></tr></thead><tbody>
          ${(data.RecentActions || []).length ? (data.RecentActions || []).map((item) => html`<tr><td><strong>${item.Domain || "-"}</strong><div class="small">${item.Label || "-"}</div></td><td>${item.Target || "-"}</td><td><${StatusBadge} label=${item.Status || "-"} currentTone=${tone(item.Status)} /><div class="small">${item.Message || "-"}</div></td><td>${item.Actor || "-"}</td><td class="small">${item.Created || "-"}</td></tr>`) : html`<tr><td colspan="5" class="small">还没有最近动作。</td></tr>`}
        </tbody></table></div>
      </div></div></section>

      <section class="card"><div class="card-body">
        <div class="card-kicker">采购概览</div>
        <h2 class="card-title">当前采购状态</h2>
        ${data.ProcurementError ? html`<div class="flash error" style="margin-top:14px;">${data.ProcurementError}</div>` : null}
        <div class="metric-grid section">
          <${MetricCard} eyebrow="未完成" value=${procurement.OpenOrderCount || 0} />
          <${MetricCard} eyebrow="风险单" value=${procurement.OpenRiskyOrders || 0} />
          <${MetricCard} eyebrow="最近单据" value=${(procurement.RecentOrders || []).length} />
        </div>
      </div></section>
    </section>
  `;
}

function confirmSubmit(message, event) {
  if (!window.confirm(message)) {
    event.preventDefault();
  }
}

function TargetSyncPage() {
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "", href: "", hrefLabel: "" });
  const [activeRunId, setActiveRunId] = useState("");
  const [activeRun, setActiveRun] = useState(null);
  const [activeRunError, setActiveRunError] = useState("");
  const resource = useResource("/api/pim/admin/target-sync", [reloadKey]);

  useEffect(() => {
    if (!resource.data || activeRunId) return;
    const running = ((resource.data.summary || {}).Runs || []).find((item) => (item.Status || "").toLowerCase() === "running");
    if (running && running.ID) {
      setActiveRunId(running.ID);
      setActiveRun(running);
    }
  }, [resource.data, activeRunId]);

  useEffect(() => {
    if (!activeRunId) return undefined;
    let cancelled = false;
    const poll = async () => {
      try {
        const payload = await fetchJSON(buildURL("/api/pim/admin/target-sync/run", { id: activeRunId }));
        if (cancelled) return;
        const run = payload.run || {};
        setActiveRun(run);
        setActiveRunError("");
        if ((run.Status || "").toLowerCase() !== "running") {
          const link = runResultLink(run.EntityType);
          setActionState({
            busy: "",
            message: runResultMessage(run.EntityType, { run }),
            error: "",
            href: link.href,
            hrefLabel: link.hrefLabel,
          });
          setActiveRunId("");
          setReloadKey((value) => value + 1);
        }
      } catch (error) {
        if (cancelled) return;
        setActiveRunError(error.message || "轮询抓取进度失败");
      }
    };
    poll();
    const timer = window.setInterval(poll, 1500);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [activeRunId]);

  if (resource.loading) return html`<${LoadingSection} label="抓取入库" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};

  async function ensureJob(entityType, scopeKey = "", scopeLabel = "") {
    setActionState({ busy: `ensure:${entityType}:${scopeKey}`, message: "", error: "", href: "", hrefLabel: "" });
    try {
      const result = await postForm("/api/pim/admin/target-sync/jobs/ensure", { entityType, scopeKey, scopeLabel });
      setActionState({ busy: "", message: result.message || "已保存抓取入库任务。", error: "", href: "", hrefLabel: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "保存抓取入库任务失败", href: "", hrefLabel: "" });
    }
  }

  function runResultLink(entityType) {
    if (entityType === "category_tree") return { href: "/_/mrtang-admin/source/categories", hrefLabel: "查看已入库分类" };
    if (entityType === "products") return { href: "/_/mrtang-admin/source/products", hrefLabel: "查看已入库商品" };
    if (entityType === "assets") return { href: "/_/mrtang-admin/source/assets", hrefLabel: "查看已入库图片" };
    return { href: "", hrefLabel: "" };
  }

  function runResultMessage(entityType, result) {
    const run = (result && result.run) || {};
    const created = run.createdCount || 0;
    const updated = run.updatedCount || 0;
    const unchanged = run.unchangedCount || 0;
    if (entityType === "category_tree") {
      return `分类抓取入库完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    if (entityType === "products") {
      return `商品规格抓取入库完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    if (entityType === "assets") {
      return `图片抓取入库完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    return result.message || "抓取入库执行完成。";
  }

  async function runJob(entityType, scopeKey = "", scopeLabel = "", confirmMessage = "") {
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: `run:${entityType}:${scopeKey}`, message: "", error: "", href: "", hrefLabel: "" });
    try {
      const result = await postForm("/api/pim/admin/target-sync/jobs/run", { entityType, scopeKey, scopeLabel });
      const run = (result && result.run) || null;
      setActiveRun(run);
      setActiveRunId((run && run.ID) || "");
      setActiveRunError("");
      setActionState({
        busy: "",
        message: result.message || "抓取入库任务已启动。",
        error: "",
        href: run && run.ID ? buildURL("/_/mrtang-admin/target-sync/run", { id: run.ID }) : "",
        hrefLabel: run && run.ID ? "查看运行详情" : "",
      });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "执行抓取入库失败", href: "", hrefLabel: "" });
      setReloadKey((value) => value + 1);
    }
  }

  const scopeOptions = summary.ScopeOptions || [];
  const progressTotal = (activeRun && activeRun.ProgressTotal) || 0;
  const progressDone = (activeRun && activeRun.ProgressDone) || 0;
  const progressPercent = progressTotal > 0 ? Math.min(100, Math.round((progressDone / progressTotal) * 100)) : 0;
  const recentRuns = summary.Runs || [];

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">抓取入库</div>
        <h2 class="card-title">分类树、商品规格与图片抓取入库</h2>
        ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
        ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
        <${ActionNotice} state=${actionState} />
        <div class="action-row">
          <button class="btn secondary" type="button" disabled=${actionState.busy === "ensure:category_tree:"} onClick=${() => ensureJob("category_tree", "", "全量")}>${actionState.busy === "ensure:category_tree:" ? "保存中..." : "保存全量分类抓取任务"}</button>
          <button class="btn" type="button" disabled=${actionState.busy === "run:category_tree:"} onClick=${() => runJob("category_tree", "", "全量", "确认立即执行全量分类抓取入库吗？")}>${actionState.busy === "run:category_tree:" ? "启动中..." : "立即执行分类抓取入库"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "run:products:"} onClick=${() => runJob("products", "", "全量", "确认立即执行全量商品规格抓取入库吗？")}>${actionState.busy === "run:products:" ? "启动中..." : "立即执行商品规格抓取入库"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "run:assets:"} onClick=${() => runJob("assets", "", "全量", "确认立即执行全量图片抓取入库吗？")}>${actionState.busy === "run:assets:" ? "启动中..." : "立即执行图片抓取入库"}</button>
        </div>
        <div class="inline-pills section">
          <${StatusBadge} label=${sourceModeLabel(summary.SourceMode)} currentTone=${tone(summary.SourceMode)} />
          <span class="pill">sourceURL: <code>${payload.sourceURL || "-"}</code></span>
          <span class="pill">${payload.requiresAuth ? "当前 API 需要 Bearer 鉴权" : "当前 API 默认公开"}</span>
        </div>
        <div class="metric-grid section">
          <${MetricCard} eyebrow="抓取任务" value=${summary.JobCount || 0} />
          <${MetricCard} eyebrow="运行记录" value=${summary.RunCount || 0} />
          <${MetricCard} eyebrow="顶级分类" value=${summary.TopLevelCount || 0} />
          <${MetricCard} eyebrow="分类节点" value=${summary.ExpectedNodeCount || 0} />
          <${MetricCard} eyebrow="目标商品" value=${summary.ExpectedProductCount || 0} detail=${`${summary.ExpectedMultiUnitCount || 0} 个多单位`} />
          <${MetricCard} eyebrow="目标图片" value=${summary.ExpectedAssetCount || 0} />
        </div>
      </div></section>

      <section class="card"><div class="card-body">
        <div class="card-kicker">当前任务</div>
        <h2 class="card-title">运行进度与阶段日志</h2>
        ${activeRun ? html`
          <div class="inline-pills">
            <${StatusBadge} label=${syncStatusLabel(activeRun.Status)} currentTone=${tone(activeRun.Status)} />
            <span class="pill">任务：<code>${activeRun.JobName || "-"}</code></span>
            <span class="pill">范围：<code>${activeRun.ScopeLabel || "-"}</code></span>
          </div>
          <div class="section">
            <div class="small">阶段：${progressStageLabel(activeRun.CurrentStage)}${activeRun.CurrentItem ? ` / ${activeRun.CurrentItem}` : ""}</div>
            <div class="small" style="margin-top:6px;">进度：${progressDone} / ${progressTotal || "-"}${progressTotal > 0 ? ` (${progressPercent}%)` : ""}</div>
            <div style="margin-top:10px; height:10px; border-radius:999px; background:rgba(255,255,255,.08); overflow:hidden;">
              <div style=${`height:100%; width:${progressPercent}%; background:linear-gradient(90deg,#5ee6ff 0%, #6ef2b4 100%); transition:width .25s ease;`}></div>
            </div>
          </div>
          ${activeRunError ? html`<div class="flash error" style="margin-top:14px;">${activeRunError}</div>` : null}
          <div class="table-wrap section"><table><thead><tr><th>时间</th><th>阶段</th><th>级别</th><th>日志</th></tr></thead><tbody>
            ${(activeRun.Logs || []).length ? (activeRun.Logs || []).slice().reverse().map((item) => html`<tr><td class="small">${item.Time || "-"}</td><td class="small">${progressStageLabel(item.Stage)}</td><td><${StatusBadge} label=${item.Level || "-"} currentTone=${tone(item.Level)} /></td><td class="small">${item.Message || "-"}</td></tr>`) : html`<tr><td colspan="4" class="small">当前还没有阶段日志。</td></tr>`}
          </tbody></table></div>
        ` : html`<div class="small">当前没有执行中的抓取入库任务。启动后这里会实时显示阶段、进度和日志。</div>`}
      </div></section>
    </section>

    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">抓取结果入口</div>
        <h2 class="card-title">先查看已入库结果，再进入 source 审核流</h2>
        <div class="ops-grid section">
          <a class="action-card" href="/_/mrtang-admin/source/categories"><div class="card-kicker">已入库分类</div><div class="metric-value">${summary.CategoryCount || 0}</div><div class="card-desc">查看抓取保存下来的分类树、层级和分类商品数。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products"><div class="card-kicker">已入库商品</div><div class="metric-value">${summary.SourceProductCount || 0}</div><div class="card-desc">查看抓取保存下来的商品、规格和审核状态。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets"><div class="card-kicker">已入库图片</div><div class="metric-value">${summary.SourceAssetCount || 0}</div><div class="card-desc">查看抓取保存下来的封面、轮播和详情图。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products?productStatus=imported"><div class="card-kicker">待审核商品</div><div class="metric-value">${summary.SourceImportedCount || 0}</div><div class="card-desc">商品和规格变化后自动回到 imported。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products?productStatus=approved"><div class="card-kicker">待桥接商品</div><div class="metric-value">${summary.SourceApprovedCount || 0}</div><div class="card-desc">审核通过后继续桥接到 supplier_products。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets?assetStatus=pending"><div class="card-kicker">待处理图片</div><div class="metric-value">${summary.SourceAssetPendingCount || 0}</div><div class="card-desc">图片变化后自动重置为 pending。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets?assetStatus=failed"><div class="card-kicker">失败图片</div><div class="metric-value">${summary.SourceAssetFailedCount || 0}</div><div class="card-desc">在图片模块继续重试或人工处理。</div></a>
        </div>
      </div></section>

      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">按范围抓取入库</div>
        <h2 class="card-title">顶级分类批次</h2>
        <div class="table-wrap section"><table><thead><tr><th>分类</th><th>节点</th><th>商品</th><th>图片</th><th>动作</th></tr></thead><tbody>
          ${scopeOptions.length ? scopeOptions.filter((item) => item.Key).map((item) => html`<tr>
            <td><strong>${item.Label || "-"}</strong><div class="small">${item.Key || "-"}</div></td>
            <td>${item.NodeCount || 0}</td>
            <td>${item.ProductCount || 0}</td>
            <td>${item.AssetCount || 0}</td>
            <td>
              <div class="action-row">
                <button class="btn secondary" type="button" disabled=${actionState.busy === `ensure:category_tree:${item.Key}`} onClick=${() => ensureJob("category_tree", item.Key, item.Label || item.Key)}>保存分类任务</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:category_tree:${item.Key}`} onClick=${() => runJob("category_tree", item.Key, item.Label || item.Key, `确认执行 ${item.Label || item.Key} 的分类抓取入库吗？`)}>抓分类</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:products:${item.Key}`} onClick=${() => runJob("products", item.Key, item.Label || item.Key, `确认执行 ${item.Label || item.Key} 的商品规格抓取入库吗？`)}>抓商品</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:assets:${item.Key}`} onClick=${() => runJob("assets", item.Key, item.Label || item.Key, `确认执行 ${item.Label || item.Key} 的图片抓取入库吗？`)}>抓图片</button>
              </div>
            </td>
          </tr>`) : html`<tr><td colspan="5" class="small">当前没有可用的顶级分类范围。</td></tr>`}
        </tbody></table></div>
      </div></div></section>
    </section>

    <section class="section split-grid">
      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">Checkout 来源矩阵</div>
        <h2 class="card-title">当前实际 contractId</h2>
        <div class="table-wrap section"><table><thead><tr><th>链路</th><th>状态</th><th>contractId</th><th>说明</th></tr></thead><tbody>
          ${(summary.CheckoutSources || []).length ? (summary.CheckoutSources || []).map((item) => html`<tr><td><strong>${item.Label || "-"}</strong></td><td><${StatusBadge} label=${checkoutStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td><td class="small"><code>${item.ContractID || "-"}</code></td><td class="small">${item.Note || "-"}</td></tr>`) : html`<tr><td colspan="4" class="small">当前还没有 checkout 来源数据。</td></tr>`}
        </tbody></table></div>
      </div></div></section>

      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">最近真实写操作</div>
        <h2 class="card-title">仅记录 raw 模式下的显式真实写入</h2>
        <div class="table-wrap section"><table><thead><tr><th>时间</th><th>操作</th><th>结果</th><th>contractId</th></tr></thead><tbody>
          ${(summary.RecentMiniappWrites || []).length ? (summary.RecentMiniappWrites || []).map((item) => html`<tr><td class="small">${item.CreatedAt || "-"}</td><td><strong>${item.OperationLabel || "-"}</strong><div class="small">${item.OperationID || "-"}</div></td><td><${StatusBadge} label=${item.Status || "-"} currentTone=${tone(item.Status)} /><div class="small">${item.Message || "-"}</div></td><td class="small"><code>${item.ContractID || "-"}</code></td></tr>`) : html`<tr><td colspan="4" class="small">目前还没有记录到 raw 模式下的真实写操作。</td></tr>`}
        </tbody></table></div>
      </div></div></section>
    </section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">最近运行</div>
      <h2 class="card-title">抓取入库历史</h2>
      <div class="table-wrap section"><table><thead><tr><th>任务</th><th>状态</th><th>范围</th><th>进度</th><th>结果</th><th>时间</th></tr></thead><tbody>
        ${recentRuns.length ? recentRuns.map((run) => {
          const total = run.ProgressTotal || run.ScopedNodeCount || 0;
          const done = run.ProgressDone || 0;
          return html`<tr>
            <td><a href=${buildURL("/_/mrtang-admin/target-sync/run", { id: run.ID })}>${run.JobName || "-"}</a><div class="small">${run.EntityType || "-"}</div></td>
            <td><${StatusBadge} label=${syncStatusLabel(run.Status)} currentTone=${tone(run.Status)} /></td>
            <td>${run.ScopeLabel || "-"}</td>
            <td class="small">${done} / ${total || "-"}</td>
            <td class="small">新增 ${run.CreatedCount || 0} / 更新 ${run.UpdatedCount || 0} / 未变 ${run.UnchangedCount || 0}</td>
            <td class="small">${run.LastProgressAt || run.FinishedAt || run.StartedAt || "-"}</td>
          </tr>`;
        }) : html`<tr><td colspan="6" class="small">当前还没有抓取入库运行记录。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function routePath(pathname) {
  if ((pathname || "").startsWith("/_/mrtang-admin/source/categories")) return "source-categories";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/products/detail")) return "source-product-detail";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/products")) return "source-products";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/asset-jobs/detail")) return "source-asset-job-detail";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/asset-jobs")) return "source-asset-jobs";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/assets/detail")) return "source-asset-detail";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/assets")) return "source-assets";
  if ((pathname || "").startsWith("/_/mrtang-admin/source")) return "source";
  if ((pathname || "").startsWith("/_/mrtang-admin/procurement/detail")) return "procurement-detail";
  if ((pathname || "").startsWith("/_/mrtang-admin/procurement")) return "procurement";
  if ((pathname || "").startsWith("/_/mrtang-admin/target-sync")) return "target-sync";
  if (pathname === "/_/mrtang-admin/source") return "source";
  if (pathname === "/_/mrtang-admin/source/categories") return "source-categories";
  if (pathname === "/_/mrtang-admin/source/products") return "source-products";
  if (pathname === "/_/mrtang-admin/source/products/detail") return "source-product-detail";
  if (pathname === "/_/mrtang-admin/source/asset-jobs") return "source-asset-jobs";
  if (pathname === "/_/mrtang-admin/source/asset-jobs/detail") return "source-asset-job-detail";
  if (pathname === "/_/mrtang-admin/source/assets") return "source-assets";
  if (pathname === "/_/mrtang-admin/source/assets/detail") return "source-asset-detail";
  if (pathname === "/_/mrtang-admin/procurement") return "procurement";
  if (pathname === "/_/mrtang-admin/procurement/detail") return "procurement-detail";
  if (pathname === "/_/mrtang-admin/target-sync") return "target-sync";
  return "dashboard";
}

function buildURL(base, params) {
  const url = new URL(base, window.location.origin);
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "" || value === 0) return;
    url.searchParams.set(key, value);
  });
  return url.pathname + url.search;
}

function SourceModulePage() {
  const resource = useResource("/api/pim/admin/source");
  if (resource.loading) return html`<${LoadingSection} label="源数据模块" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">源数据概览</div>
        <h2 class="card-title">审核与处理总览</h2>
        ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
        ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
        <div class="metric-grid section">
          <${MetricCard} eyebrow="商品总数" value=${summary.ProductCount || 0} detail=${`${summary.ImportedCount || 0} 待审核 / ${summary.ApprovedCount || 0} 待桥接`} />
          <${MetricCard} eyebrow="图片总数" value=${summary.AssetCount || 0} detail=${`${summary.AssetPending || 0} 待处理 / ${summary.AssetFailed || 0} 失败`} />
          <${MetricCard} eyebrow="已桥接" value=${summary.LinkedCount || 0} detail=${`${summary.SyncedCount || 0} 已同步 / ${summary.SyncErrorCount || 0} 同步失败`} />
          <${MetricCard} eyebrow="分类" value=${summary.CategoryCount || 0} />
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">快捷入口</div>
        <h2 class="card-title">下一步处理</h2>
        <div class="ops-grid section">
          <a class="action-card" href="/_/mrtang-admin/source/categories"><div class="card-kicker">分类</div><div class="card-title">分类管理</div><div class="card-desc">查看抓取入库后的分类树、层级和分类商品数量。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products?productStatus=imported"><div class="card-kicker">商品</div><div class="card-title">待审核商品</div><div class="card-desc">直接筛出 imported 商品。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products?syncState=error"><div class="card-kicker">商品</div><div class="card-title">同步失败商品</div><div class="card-desc">查看 linked sync error 并继续重试。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets?assetStatus=pending"><div class="card-kicker">图片</div><div class="card-title">待处理图片</div><div class="card-desc">进入图片页执行批量处理。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/asset-jobs"><div class="card-kicker">任务</div><div class="card-title">图片任务历史</div><div class="card-desc">查看原图下载和图片处理的历史任务、失败与重试。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/logs"><div class="card-kicker">日志</div><div class="card-title">源数据日志</div><div class="card-desc">查看最近审核、桥接和图片处理动作。</div></a>
        </div>
      </div></section>
    </section>
  `;
}

function SourceCategoriesPage() {
  const qs = new URLSearchParams(window.location.search);
  const q = qs.get("q") || "";
  const page = qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/categories", { q, page, pageSize });
  const resource = useResource(apiURL);
  if (resource.loading) return html`<${LoadingSection} label="源数据分类" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const items = summary.Items || [];
  const rootNodes = items.filter((item) => !item.ParentKey);
  const childMap = {};
  items.forEach((item) => {
    const key = item.ParentKey || "__root__";
    if (!childMap[key]) childMap[key] = [];
    childMap[key].push(item);
  });

  function renderCategoryTree(parentKey, depth = 0) {
    const nodes = childMap[parentKey || "__root__"] || [];
    if (!nodes.length) return null;
    return html`<div class="section" style=${depth ? `margin-left:${depth * 18}px;` : ""}>
      ${nodes.map((item) => html`
        <div class="action-card" style="margin-bottom:10px;">
          <div class="card-kicker">深度 ${item.Depth || 0}${item.HasChildren ? " / 有子分类" : " / 叶子"}</div>
          <div class="card-title">${item.Label || "-"}</div>
          <div class="card-desc">${item.CategoryPath || "-"}</div>
          <div class="action-row" style="margin-top:10px;">
            <span class="pill">商品 <code>${item.ProductCount || 0}</code></span>
            <a class="btn secondary" href=${`/_/mrtang-admin/source/products?categoryKey=${encodeURIComponent(item.SourceKey || "")}`}>查看该分类商品</a>
          </div>
          ${renderCategoryTree(item.SourceKey, depth + 1)}
        </div>
      `)}
    </div>`;
  }

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">分类列表</h2>
      ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
      ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
      <form class="action-row" method="get" action="/_/mrtang-admin/source/categories">
        <input type="text" name="q" placeholder="搜索分类名 / sourceKey / 路径" defaultValue=${filter.Query || ""} />
        <select name="pageSize" defaultValue=${String(filter.PageSize || 24)}>
          <option value="12">12</option>
          <option value="24">24</option>
          <option value="48">48</option>
        </select>
        <button class="btn secondary" type="submit">应用筛选</button>
        <a class="btn secondary" href="/_/mrtang-admin/source/categories">重置</a>
      </form>
      <div class="inline-pills">
        <span class="pill">分类总数 <code>${summary.CategoryCount || 0}</code></span>
        <span class="pill">顶级分类 <code>${summary.TopLevelCount || 0}</code></span>
        <span class="pill">叶子分类 <code>${summary.LeafCount || 0}</code></span>
        <span class="pill">带图分类 <code>${summary.WithImageCount || 0}</code></span>
      </div>
    </div></section>

    <section class="section card"><div class="card-body">
      <div class="card-kicker">分类树</div>
      <h2 class="card-title">按层级查看</h2>
      ${rootNodes.length ? renderCategoryTree("", 0) : html`<div class="small">当前筛选下没有分类树节点。</div>`}
    </div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">列表</div>
      <h2 class="card-title">已落库分类</h2>
      <div class="table-wrap section"><table><thead><tr><th>分类</th><th>路径</th><th>层级</th><th>商品数</th><th>图片</th></tr></thead><tbody>
        ${items.length ? items.map((item) => html`
          <tr>
            <td><strong>${item.Label || "-"}</strong><div class="small">${item.SourceKey || "-"}</div></td>
            <td class="small">${item.CategoryPath || "-"}</td>
            <td class="small">深度 ${item.Depth || 0}${item.HasChildren ? " / 有子分类" : " / 叶子"}</td>
            <td><span class="pill"><code>${item.ProductCount || 0}</code></span><div class="small"><a href=${`/_/mrtang-admin/source/products?categoryKey=${encodeURIComponent(item.SourceKey || "")}`}>查看商品</a></div></td>
            <td>${item.ImageURL ? html`<a href=${item.ImageURL} target="_blank" rel="noreferrer">查看</a>` : html`<span class="small">无</span>`}</td>
          </tr>
        `) : html`<tr><td colspan="5" class="small">当前筛选下没有分类。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function SourceProductsPage() {
  const qs = new URLSearchParams(window.location.search);
  const categoryKey = qs.get("categoryKey") || "";
  const productStatus = qs.get("productStatus") || "";
  const syncState = qs.get("syncState") || "";
  const q = qs.get("q") || "";
  const page = qs.get("productPage") || qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/products", { categoryKey, productStatus, syncState, q, productPage: page, pageSize });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const resource = useResource(apiURL, [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="源数据商品" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const products = summary.Products || [];

  async function productAction(url, values, confirmMessage, busyKey, successMessage) {
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, values);
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "操作失败" });
    }
  }

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">商品列表</h2>
      ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
      ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
      <${ActionNotice} state=${actionState} />
      <form class="action-row" method="get" action="/_/mrtang-admin/source/products">
        <select name="productStatus" defaultValue=${filter.ProductStatus || ""}>
          <option value="">全部审核状态</option>
          <option value="imported">待审核</option>
          <option value="approved">待桥接</option>
          <option value="promoted">已桥接</option>
          <option value="rejected">已拒绝</option>
        </select>
        <select name="syncState" defaultValue=${filter.SyncState || ""}>
          <option value="">全部同步状态</option>
          <option value="unlinked">未桥接</option>
          <option value="error">同步失败</option>
          <option value="synced">已同步</option>
        </select>
        <input type="text" name="categoryKey" placeholder="分类 key" defaultValue=${filter.CategoryKey || ""} />
        <input type="text" name="q" placeholder="搜索商品名 / productId" defaultValue=${filter.Query || ""} />
        <select name="pageSize" defaultValue=${String(filter.PageSize || 24)}>
          <option value="12">12</option>
          <option value="24">24</option>
          <option value="48">48</option>
        </select>
        <button class="btn secondary" type="submit">应用筛选</button>
        <a class="btn secondary" href="/_/mrtang-admin/source/products">重置</a>
      </form>
      <div class="inline-pills">
        ${filter.CategoryKey ? html`<span class="pill">分类 <code>${filter.CategoryKey}</code></span>` : null}
        <span class="pill">总数 <code>${summary.ProductCount || 0}</code></span>
        <span class="pill">待审核 <code>${summary.ImportedCount || 0}</code></span>
        <span class="pill">待桥接 <code>${summary.ApprovedCount || 0}</code></span>
        <span class="pill">同步失败 <code>${summary.SyncErrorCount || 0}</code></span>
      </div>
    </div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">列表</div>
      <h2 class="card-title">商品批次</h2>
      <div class="table-wrap section"><table><thead><tr><th>商品</th><th>分类 / 单位</th><th>审核</th><th>桥接</th><th>动作</th></tr></thead><tbody>
        ${products.length ? products.map((item) => html`
          <tr>
            <td><strong>${item.Name || "-"}</strong><div class="small">${item.ProductID || "-"}</div></td>
            <td class="small">${item.CategoryPath || "-"}<br />${item.UnitCount || 0} 个单位 / ${item.HasMultiUnit ? "多单位" : "单单位"}</td>
            <td><${StatusBadge} label=${item.ReviewStatus || "-"} currentTone=${tone(item.ReviewStatus)} /></td>
            <td><${StatusBadge} label=${(item.Bridge && item.Bridge.SyncStatus) || (item.Bridge && item.Bridge.Linked ? "linked" : "unlinked")} currentTone=${tone((item.Bridge && item.Bridge.SyncStatus) || (item.Bridge && item.Bridge.Linked ? "warning" : "error"))} /></td>
            <td>
              <div class="action-row">
                <a class="btn secondary" href=${`/_/mrtang-admin/source/products/detail?id=${encodeURIComponent(item.ID || "")}&returnTo=${encodeURIComponent(window.location.pathname + window.location.search)}`}>详情</a>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `approve:${item.ID || ""}`} onClick=${() => productAction("/api/pim/admin/source/products/status", { id: item.ID || "", status: "approved" }, "确认将这个商品标记为通过吗？", `approve:${item.ID || ""}`, "商品审核状态已更新。")}>${actionState.busy === `approve:${item.ID || ""}` ? "处理中..." : "通过"}</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `promote:${item.ID || ""}`} onClick=${() => productAction("/api/pim/admin/source/products/promote", { id: item.ID || "" }, "确认桥接这个商品吗？", `promote:${item.ID || ""}`, "商品已桥接到同步链。")}>${actionState.busy === `promote:${item.ID || ""}` ? "处理中..." : "桥接"}</button>
                ${(item.Bridge && (item.Bridge.SyncStatus || "").toLowerCase() === "error") ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === `retry:${item.ID || ""}`} onClick=${() => productAction("/api/pim/admin/source/products/retry-sync", { id: item.ID || "" }, "确认重试这个商品的同步吗？", `retry:${item.ID || ""}`, "已触发商品同步重试。")}>${actionState.busy === `retry:${item.ID || ""}` ? "处理中..." : "重试同步"}</button>` : null}
              </div>
            </td>
          </tr>
        `) : html`<tr><td colspan="5" class="small">当前筛选下没有商品。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function SourceAssetsPage() {
  const qs = new URLSearchParams(window.location.search);
  const assetStatus = qs.get("assetStatus") || "";
  const originalStatus = qs.get("originalStatus") || "";
  const q = qs.get("q") || "";
  const page = qs.get("assetPage") || qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/assets", { assetStatus, originalStatus, q, assetPage: page, pageSize });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [activeDownloadId, setActiveDownloadId] = useState("");
  const [activeDownload, setActiveDownload] = useState(null);
  const [activeDownloadError, setActiveDownloadError] = useState("");
  const [activeProcessId, setActiveProcessId] = useState("");
  const [activeProcess, setActiveProcess] = useState(null);
  const [activeProcessError, setActiveProcessError] = useState("");
  const resource = useResource(apiURL, [reloadKey]);
  const jobsResource = useResource("/api/pim/admin/source/asset-jobs?pageSize=5", [reloadKey]);

  useEffect(() => {
    if (!activeDownloadId) return undefined;
    let cancelled = false;
    const poll = async () => {
      try {
        const payload = await fetchJSON(buildURL("/api/pim/admin/source/assets/download-progress", { id: activeDownloadId }));
        if (cancelled) return;
        const progress = payload.progress || {};
        setActiveDownload(progress);
        setActiveDownloadError("");
        if ((progress.Status || "").toLowerCase() !== "running") {
          setActiveDownloadId("");
          setReloadKey((value) => value + 1);
          setActionState({
            busy: "",
            message: `原图批量下载完成：成功 ${progress.Processed || 0}，失败 ${progress.Failed || 0}。`,
            error: "",
          });
        }
      } catch (error) {
        if (cancelled) return;
        setActiveDownloadError(error.message || "轮询原图下载进度失败");
      }
    };
    poll();
    const timer = window.setInterval(poll, 1500);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [activeDownloadId]);

  useEffect(() => {
    if (!activeProcessId) return undefined;
    let cancelled = false;
    const poll = async () => {
      try {
        const payload = await fetchJSON(buildURL("/api/pim/admin/source/assets/process-progress", { id: activeProcessId }));
        if (cancelled) return;
        const progress = payload.progress || {};
        setActiveProcess(progress);
        setActiveProcessError("");
        if ((progress.Status || "").toLowerCase() !== "running") {
          setActiveProcessId("");
          setReloadKey((value) => value + 1);
          setActionState({
            busy: "",
            message: `图片批量处理完成：成功 ${progress.Processed || 0}，失败 ${progress.Failed || 0}。`,
            error: "",
          });
        }
      } catch (error) {
        if (cancelled) return;
        setActiveProcessError(error.message || "轮询图片处理进度失败");
      }
    };
    poll();
    const timer = window.setInterval(poll, 1500);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [activeProcessId]);
  if (resource.loading) return html`<${LoadingSection} label="源数据图片" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const assets = summary.Assets || [];

  async function assetAction(url, values, confirmMessage, busyKey, successMessage) {
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, values);
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "操作失败" });
    }
  }

  async function startDownloadPending() {
    if (!window.confirm("确认批量下载待下载原图吗？")) return;
    setActionState({ busy: "download-pending", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/download-pending", {});
      const progress = result.progress || {};
      setActiveDownload(progress);
      setActiveDownloadId(progress.ID || "");
      setActiveDownloadError("");
      setActionState({ busy: "", message: result.message || "原图批量下载任务已启动。", error: "" });
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "批量下载原图失败" });
    }
  }

  async function startProcessBatch(url, confirmMessage, busyKey, defaultMessage) {
    if (!window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, {});
      const progress = result.progress || {};
      setActiveProcess(progress);
      setActiveProcessId(progress.ID || "");
      setActiveProcessError("");
      setActionState({ busy: "", message: result.message || defaultMessage, error: "" });
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || defaultMessage || "批量处理图片失败" });
    }
  }

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">图片列表</h2>
      ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
      ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
      <${ActionNotice} state=${actionState} />
      <form class="action-row" method="get" action="/_/mrtang-admin/source/assets">
        <select name="assetStatus" defaultValue=${filter.AssetStatus || ""}>
          <option value="">全部图片状态</option>
          <option value="pending">待处理</option>
          <option value="processed">已处理</option>
          <option value="failed">处理失败</option>
        </select>
        <select name="originalStatus" defaultValue=${filter.OriginalStatus || ""}>
          <option value="">全部原图状态</option>
          <option value="pending">待下载</option>
          <option value="downloaded">已下载</option>
          <option value="failed">下载失败</option>
        </select>
        <input type="text" name="q" placeholder="搜索商品名 / assetKey" defaultValue=${filter.Query || ""} />
        <select name="pageSize" defaultValue=${String(filter.PageSize || 24)}>
          <option value="12">12</option>
          <option value="24">24</option>
          <option value="48">48</option>
        </select>
        <button class="btn secondary" type="submit">应用筛选</button>
        <a class="btn secondary" href="/_/mrtang-admin/source/assets">重置</a>
      </form>
      <div class="inline-pills">
        <span class="pill">总数 <code>${summary.AssetCount || 0}</code></span>
        <span class="pill">原图待下载 <code>${summary.AssetOriginalPending || 0}</code></span>
        <span class="pill">原图已下载 <code>${summary.AssetOriginalDownloaded || 0}</code></span>
        <span class="pill">原图失败 <code>${summary.AssetOriginalFailed || 0}</code></span>
        <span class="pill">待处理 <code>${summary.AssetPending || 0}</code></span>
        <span class="pill">失败 <code>${summary.AssetFailed || 0}</code></span>
        <span class="pill">已处理 <code>${summary.AssetProcessed || 0}</code></span>
        <a class="pill" href=${buildURL("/_/mrtang-admin/source/assets", { originalStatus: "failed" })}>原图失败</a>
        <a class="pill" href=${buildURL("/_/mrtang-admin/source/assets", { assetStatus: "failed" })}>处理失败</a>
        <a class="pill" href="/_/mrtang-admin/source/asset-jobs">查看任务历史</a>
      </div>
      ${(summary.AssetFailureReasons || []).length ? html`<div class="inline-pills section">
        ${(summary.AssetFailureReasons || []).map((reason) => html`<span class="pill">${reason.Message || "未知失败"} <code>${reason.Count || 0}</code></span>`)}
      </div>` : null}
    </div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">列表</div>
      <h2 class="card-title">图片批次</h2>
      ${activeDownload ? html`<div class="flash ok" style="margin-bottom:12px;">
        <div>原图批量下载：${(activeDownload.Status || "").toLowerCase() === "running" ? "执行中" : "已完成"}</div>
        <div class="small" style="margin-top:8px;">已处理 ${activeDownload.Processed || 0} / ${activeDownload.Total || 0}，失败 ${activeDownload.Failed || 0}${activeDownload.CurrentItem ? `，当前项：${activeDownload.CurrentItem}` : ""}</div>
        ${(activeDownload.Logs || []).length ? html`<div class="small" style="margin-top:8px;">${(activeDownload.Logs || []).slice(-5).map((item) => `${item.Time || "-"} ${item.Message || "-"}`).join(" / ")}</div>` : null}
        ${activeDownload.ID ? html`<div class="small" style="margin-top:8px;"><a href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: activeDownload.ID, returnTo: window.location.pathname + window.location.search })}>查看任务详情</a></div>` : null}
      </div>` : null}
      ${activeDownloadError ? html`<div class="flash error" style="margin-bottom:12px;">${activeDownloadError}</div>` : null}
      ${activeProcess ? html`<div class="flash ok" style="margin-bottom:12px;">
        <div>图片批量处理：${(activeProcess.Mode || "").toLowerCase() === "failed" ? "失败图片重处理" : "待处理图片"} / ${(activeProcess.Status || "").toLowerCase() === "running" ? "执行中" : "已完成"}</div>
        <div class="small" style="margin-top:8px;">已处理 ${activeProcess.Processed || 0} / ${activeProcess.Total || 0}，失败 ${activeProcess.Failed || 0}${activeProcess.CurrentItem ? `，当前项：${activeProcess.CurrentItem}` : ""}</div>
        ${(activeProcess.Logs || []).length ? html`<div class="small" style="margin-top:8px;">${(activeProcess.Logs || []).slice(-5).map((item) => `${item.Time || "-"} ${item.Message || "-"}`).join(" / ")}</div>` : null}
        ${activeProcess.ID ? html`<div class="small" style="margin-top:8px;"><a href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: activeProcess.ID, returnTo: window.location.pathname + window.location.search })}>查看任务详情</a></div>` : null}
      </div>` : null}
      ${activeProcessError ? html`<div class="flash error" style="margin-bottom:12px;">${activeProcessError}</div>` : null}
      <div class="action-row" style="margin-bottom:12px;">
        <button class="btn secondary" type="button" disabled=${actionState.busy === "download-pending" || !!activeDownloadId} onClick=${startDownloadPending}>${actionState.busy === "download-pending" || !!activeDownloadId ? "下载中..." : "批量下载待下载原图"}</button>
        <button class="btn secondary" type="button" disabled=${actionState.busy === "process-pending" || !!activeProcessId} onClick=${() => startProcessBatch("/api/pim/admin/source/assets/process-pending", "确认批量处理待处理图片吗？", "process-pending", "图片批量处理任务已启动。")}>${actionState.busy === "process-pending" || (!!activeProcessId && ((activeProcess && (activeProcess.Mode || "").toLowerCase()) !== "failed")) ? "处理中..." : "批量处理待处理图片"}</button>
        <button class="btn secondary" type="button" disabled=${actionState.busy === "reprocess-failed" || !!activeProcessId} onClick=${() => startProcessBatch("/api/pim/admin/source/assets/reprocess-failed", "确认批量重处理失败图片吗？", "reprocess-failed", "失败图片重处理任务已启动。")}>${actionState.busy === "reprocess-failed" || (!!activeProcessId && ((activeProcess && (activeProcess.Mode || "").toLowerCase()) === "failed")) ? "处理中..." : "批量重处理失败图片"}</button>
      </div>
      <div class="table-wrap section"><table><thead><tr><th>图片</th><th>商品</th><th>原图</th><th>处理</th><th>错误</th><th>动作</th></tr></thead><tbody>
        ${assets.length ? assets.map((item) => html`
          <tr>
            <td><strong>${item.AssetRole || "-"}</strong><div class="small">${item.AssetKey || "-"}</div></td>
            <td>${item.Name || "-"}<div class="small">${item.ProductID || "-"}</div></td>
            <td>
              <${StatusBadge} label=${originalImageStatusLabel(item.OriginalImageStatus)} currentTone=${tone(item.OriginalImageStatus)} />
              <div class="small" style="margin-top:8px;">
                ${item.OriginalImageURL ? html`<a href=${item.OriginalImageURL} target="_blank" rel="noreferrer">查看原图文件</a>` : (item.SourceURL ? html`<a href=${item.SourceURL} target="_blank" rel="noreferrer">查看源地址</a>` : "-")}
              </div>
            </td>
            <td>
              <${StatusBadge} label=${item.ImageProcessingStatus || "-"} currentTone=${tone(item.ImageProcessingStatus)} />
              <div class="small" style="margin-top:8px;">
                ${item.ProcessedImageURL ? html`<a href=${item.ProcessedImageURL} target="_blank" rel="noreferrer">查看处理图</a>` : "-"}
              </div>
            </td>
            <td class="small">${item.OriginalImageError || item.ImageProcessingError || "-"}</td>
            <td>
              <div class="action-row">
                <a class="btn secondary" href=${`/_/mrtang-admin/source/assets/detail?id=${encodeURIComponent(item.ID || "")}&returnTo=${encodeURIComponent(window.location.pathname + window.location.search)}`}>详情</a>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `download:${item.ID || ""}`} onClick=${() => assetAction("/api/pim/admin/source/assets/download", { id: item.ID || "" }, "确认下载这张图片的原图吗？", `download:${item.ID || ""}`, "已下载原图。")}>${actionState.busy === `download:${item.ID || ""}` ? "下载中..." : "下载原图"}</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `asset:${item.ID || ""}`} onClick=${() => assetAction("/api/pim/admin/source/assets/process", { id: item.ID || "" }, "确认处理这张图片吗？", `asset:${item.ID || ""}`, "图片已进入处理流程。")}>${actionState.busy === `asset:${item.ID || ""}` ? "处理中..." : "处理"}</button>
              </div>
            </td>
          </tr>
        `) : html`<tr><td colspan="6" class="small">当前筛选下没有图片。</td></tr>`}
      </tbody></table></div>
    </div></div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">最近任务</div>
      <h2 class="card-title">原图下载与图片处理</h2>
      ${jobsResource.loading ? html`<div class="small">最近任务加载中...</div>` : null}
      ${jobsResource.error ? html`<div class="flash error">${jobsResource.error}</div>` : null}
      ${!jobsResource.loading && !jobsResource.error ? html`<div class="table-wrap section"><table><thead><tr><th>任务</th><th>状态</th><th>进度</th><th>错误摘要</th><th>动作</th></tr></thead><tbody>
        ${(((jobsResource.data || {}).summary || {}).Items || []).length ? (((jobsResource.data || {}).summary || {}).Items || []).map((item) => html`
          <tr>
            <td><strong>${sourceAssetJobTypeLabel(item.JobType, item.Mode)}</strong><div class="small">${item.StartedAt || item.Created || "-"}</div></td>
            <td><${StatusBadge} label=${syncStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td>
            <td class="small">${item.Processed || 0} / ${item.Total || 0}<br />失败 ${item.Failed || 0}</td>
            <td class="small">${sourceAssetJobRecentError(item) || "-"}</td>
            <td><div class="action-row"><a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: item.ID || "", returnTo: window.location.pathname + window.location.search })}>详情</a><a class="btn secondary" href=${sourceAssetJobTargetHref(item)}>${sourceAssetJobTargetLabel(item)}</a></div></td>
          </tr>
        `) : html`<tr><td colspan="5" class="small">还没有图片任务记录。</td></tr>`}
      </tbody></table></div>` : null}
    </div></div></section>
  `;
}

function SourceAssetJobsPage() {
  const qs = new URLSearchParams(window.location.search);
  const jobType = qs.get("jobType") || "";
  const status = qs.get("status") || "";
  const q = qs.get("q") || "";
  const page = qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/asset-jobs", { jobType, status, q, page, pageSize });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const resource = useResource(apiURL, [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="图片任务" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const items = summary.Items || [];

  async function retryJob(item) {
    if (!window.confirm(`确认重新执行“${sourceAssetJobTypeLabel(item.JobType, item.Mode)}”吗？`)) return;
    setActionState({ busy: item.ID || "", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/asset-jobs/retry", { id: item.ID || "" });
      const nextJob = result.job || {};
      setActionState({
        busy: "",
        message: result.message || "图片任务已重新启动。",
        error: "",
      });
      if (nextJob.ID) {
        window.location.href = buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: nextJob.ID, returnTo: window.location.pathname + window.location.search });
        return;
      }
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "重新执行图片任务失败" });
    }
  }

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">图片任务历史</h2>
      ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
      ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
      <${ActionNotice} state=${actionState} />
      <form class="action-row" method="get" action="/_/mrtang-admin/source/asset-jobs">
        <select name="jobType" defaultValue=${filter.JobType || ""}>
          <option value="">全部任务类型</option>
          <option value="download_original">原图下载</option>
          <option value="process_asset">图片处理</option>
        </select>
        <select name="status" defaultValue=${filter.Status || ""}>
          <option value="">全部状态</option>
          <option value="running">执行中</option>
          <option value="completed">已完成</option>
          <option value="failed">失败</option>
        </select>
        <input type="text" name="q" placeholder="搜索当前项 / 错误" defaultValue=${filter.Query || ""} />
        <select name="pageSize" defaultValue=${String(filter.PageSize || 20)}>
          <option value="10">10</option>
          <option value="20">20</option>
          <option value="50">50</option>
        </select>
        <button class="btn secondary" type="submit">应用筛选</button>
        <a class="btn secondary" href="/_/mrtang-admin/source/asset-jobs">重置</a>
      </form>
      <div class="inline-pills">
        <span class="pill">总任务 <code>${summary.TotalJobs || 0}</code></span>
        <span class="pill">执行中 <code>${summary.RunningJobs || 0}</code></span>
        <span class="pill">已完成 <code>${summary.CompletedJobs || 0}</code></span>
        <span class="pill">失败 <code>${summary.FailedJobs || 0}</code></span>
      </div>
    </div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">列表</div>
      <h2 class="card-title">历史任务</h2>
      <div class="table-wrap section"><table><thead><tr><th>任务</th><th>状态</th><th>进度</th><th>当前项 / 错误</th><th>时间</th><th>动作</th></tr></thead><tbody>
        ${items.length ? items.map((item) => html`
          <tr>
            <td>
              <strong>${sourceAssetJobTypeLabel(item.JobType, item.Mode)}</strong>
              <div class="small">${item.Mode ? `模式：${item.Mode}` : item.JobType || "-"}</div>
              <div class="small">${item.ID || "-"}</div>
            </td>
            <td><${StatusBadge} label=${syncStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td>
            <td class="small">${item.Processed || 0} / ${item.Total || 0}<br />失败 ${item.Failed || 0}</td>
            <td class="small">${item.CurrentItem || "-"}${sourceAssetJobRecentError(item) ? html`<div style="margin-top:8px;">${sourceAssetJobRecentError(item)}</div>` : null}</td>
            <td class="small">${item.StartedAt || item.Created || "-"}<br />${item.FinishedAt || "-"}</td>
            <td>
              <div class="action-row">
                <a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: item.ID || "", returnTo: window.location.pathname + window.location.search })}>详情</a>
                <a class="btn secondary" href=${sourceAssetJobTargetHref(item)}>${sourceAssetJobTargetLabel(item)}</a>
                ${item.CanRetry ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === (item.ID || "")} onClick=${() => retryJob(item)}>${actionState.busy === (item.ID || "") ? "处理中..." : "重新执行"}</button>` : html`<span class="pill">执行中</span>`}
              </div>
            </td>
          </tr>
        `) : html`<tr><td colspan="6" class="small">当前筛选下没有图片任务。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function SourceAssetJobDetailPage() {
  const qs = new URLSearchParams(window.location.search);
  const id = qs.get("id") || "";
  const returnTo = qs.get("returnTo") || "/_/mrtang-admin/source/asset-jobs";
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [reloadKey, setReloadKey] = useState(0);
  const resource = useResource(buildURL("/api/pim/admin/source/asset-jobs/detail", { id, returnTo }), [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="图片任务详情" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const detail = payload.detail || {};
  const backHref = payload.returnTo || returnTo;

  async function retryCurrent() {
    if (!window.confirm(`确认重新执行“${sourceAssetJobTypeLabel(detail.JobType, detail.Mode)}”吗？`)) return;
    setActionState({ busy: "retry", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/asset-jobs/retry", { id: detail.ID || "" });
      const nextJob = result.job || {};
      if (nextJob.ID) {
        window.location.href = buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: nextJob.ID, returnTo: backHref });
        return;
      }
      setActionState({ busy: "", message: result.message || "图片任务已重新启动。", error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "重新执行图片任务失败" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">任务详情</div>
        <h2 class="card-title">${sourceAssetJobTypeLabel(detail.JobType, detail.Mode)}</h2>
        <${ActionNotice} state=${actionState} />
        <div class="inline-pills">
          <span class="pill">任务 ID <code>${detail.ID || "-"}</code></span>
          <${StatusBadge} label=${syncStatusLabel(detail.Status)} currentTone=${tone(detail.Status)} />
          <span class="pill">模式 <code>${detail.Mode || "-"}</code></span>
        </div>
        <div class="action-row" style="margin-top:12px;">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          <a class="btn secondary" href=${sourceAssetJobTargetHref(detail)}>${sourceAssetJobTargetLabel(detail)}</a>
          ${detail.CanRetry ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === "retry"} onClick=${retryCurrent}>${actionState.busy === "retry" ? "处理中..." : "重新执行"}</button>` : html`<span class="pill">任务仍在执行中</span>`}
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">进度</div>
        <h2 class="card-title">执行状态</h2>
        <div class="metric-grid section">
          <${MetricCard} eyebrow="总数" value=${detail.Total || 0} />
          <${MetricCard} eyebrow="已处理" value=${detail.Processed || 0} />
          <${MetricCard} eyebrow="失败" value=${detail.Failed || 0} />
          <${MetricCard} eyebrow="当前项" value=${detail.CurrentItem || "-"} />
        </div>
        <div class="small">开始：${detail.StartedAt || "-"} / 结束：${detail.FinishedAt || "-"}</div>
        ${detail.Error ? html`<div class="flash error" style="margin-top:14px;">${detail.Error}</div>` : null}
      </div></section>
    </section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">日志</div>
      <h2 class="card-title">最近任务日志</h2>
      <div class="table-wrap section"><table><thead><tr><th>时间</th><th>内容</th></tr></thead><tbody>
        ${(detail.Logs || []).length ? (detail.Logs || []).map((item) => html`<tr><td class="small">${item.Time || "-"}</td><td>${item.Message || "-"}</td></tr>`) : html`<tr><td colspan="2" class="small">当前任务还没有日志。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function ProcurementPage() {
  const qs = new URLSearchParams(window.location.search);
  const status = qs.get("status") || "";
  const risk = qs.get("risk") || "";
  const q = qs.get("q") || "";
  const page = qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/procurement", { status, risk, q, page, pageSize });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [notes, setNotes] = useState({});
  const resource = useResource(apiURL, [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="采购模块" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const orders = summary.RecentOrders || [];
  const recentActions = summary.RecentActions || [];

  function setOrderNote(id, value) {
    setNotes((current) => ({ ...current, [id]: value }));
  }

  async function procurementAction(url, values, confirmMessage, busyKey, successMessage) {
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, values);
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "操作失败" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">采购概览</div>
        <h2 class="card-title">采购工作台</h2>
        ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
        ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
        <${ActionNotice} state=${actionState} />
        <form class="action-row" method="get" action="/_/mrtang-admin/procurement">
          <select name="status" defaultValue=${summary.FilterStatus || ""}>
            <option value="">全部状态</option>
            <option value="draft">草稿</option>
            <option value="reviewed">已复核</option>
            <option value="exported">已导出</option>
            <option value="ordered">已下单</option>
            <option value="received">已收货</option>
            <option value="canceled">已取消</option>
          </select>
          <select name="risk" defaultValue=${summary.FilterRisk || ""}>
            <option value="">全部风险</option>
            <option value="has_risk">有风险</option>
            <option value="loss">亏损风险</option>
            <option value="warning">毛利预警</option>
            <option value="normal">仅正常</option>
          </select>
          <input type="text" name="q" placeholder="外部单号 / 备注" defaultValue=${summary.Query || ""} />
          <select name="pageSize" defaultValue=${String(summary.PageSize || 20)}>
            <option value="10">10</option>
            <option value="20">20</option>
            <option value="50">50</option>
          </select>
          <button class="btn secondary" type="submit">应用筛选</button>
          <a class="btn secondary" href="/_/mrtang-admin/procurement">重置</a>
        </form>
        <div class="metric-grid section">
          <${MetricCard} eyebrow="总采购单" value=${summary.TotalOrders || 0} />
          <${MetricCard} eyebrow="草稿" value=${summary.DraftOrders || 0} />
          <${MetricCard} eyebrow="已复核" value=${summary.ReviewedOrders || 0} />
          <${MetricCard} eyebrow="已导出" value=${summary.ExportedOrders || 0} />
          <${MetricCard} eyebrow="已下单" value=${summary.OrderedOrders || 0} />
          <${MetricCard} eyebrow="未完成风险单" value=${summary.OpenRiskyOrders || 0} />
        </div>
      </div></section>

      <section class="card"><div class="card-body">
        <div class="card-kicker">待办</div>
        <h2 class="card-title">当前队列</h2>
        <div class="ops-grid section">
          <div class="action-card"><div class="card-kicker">待复核</div><div class="metric-value">${summary.DraftOrders || 0}</div><div class="card-desc">草稿单需要先确认风险与数量。</div></div>
          <div class="action-card"><div class="card-kicker">待导出</div><div class="metric-value">${summary.ReviewedOrders || 0}</div><div class="card-desc">已复核，等待导出或继续推进采购。</div></div>
          <div class="action-card"><div class="card-kicker">待收货</div><div class="metric-value">${summary.OrderedOrders || 0}</div><div class="card-desc">已下单，等待到货和收货确认。</div></div>
          <a class="action-card" href="/_/mrtang-admin/audit?domain=采购"><div class="card-kicker">审计</div><div class="card-title">查看采购审计</div><div class="card-desc">打开统一审计，只看采购动作。</div></a>
        </div>
      </div></section>
    </section>

    <section class="section split-grid">
      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">列表</div>
        <h2 class="card-title">采购单</h2>
        <div class="table-wrap section"><table><thead><tr><th>外部单号</th><th>状态</th><th>商品</th><th>金额</th><th>风险</th><th>说明</th><th>动作</th></tr></thead><tbody>
          ${orders.length ? orders.map((item) => html`
            <tr>
              <td><strong>${item.ExternalRef || "-"}</strong><div class="small">${item.ID || "-"}</div><div class="small"><a href=${`/_/mrtang-admin/procurement/detail?id=${encodeURIComponent(item.ID || "")}&returnTo=${encodeURIComponent(window.location.pathname + window.location.search)}`}>查看详情</a></div></td>
              <td><${StatusBadge} label=${item.Status || "-"} currentTone=${item.RiskyItemCount > 0 ? "warning" : tone(item.Status)} /></td>
              <td>${item.ItemCount || 0} 项 / ${(item.TotalQty || 0).toFixed ? item.TotalQty.toFixed(2) : item.TotalQty}<div class="small">${item.SupplierCount || 0} 个供应商</div></td>
              <td>成本 ${typeof item.TotalCostAmount === "number" ? item.TotalCostAmount.toFixed(2) : item.TotalCostAmount || "0.00"}</td>
              <td>${item.RiskyItemCount > 0 ? html`<span class="small">${item.RiskyItemCount} 个风险项</span>` : html`<span class="small">正常</span>`}</td>
              <td><div>${item.LastActionNote || "-"}</div><div class="small">${item.Updated || "-"}</div></td>
              <td>
                <div class="action-row">
                  <input type="text" value=${notes[item.ID || ""] || ""} onInput=${(event) => setOrderNote(item.ID || "", event.currentTarget.value)} placeholder="操作备注" />
                  <button class="btn secondary" type="button" disabled=${actionState.busy === `review:${item.ID || ""}`} onClick=${() => procurementAction("/api/pim/admin/procurement/order/review", { id: item.ID || "", note: notes[item.ID || ""] || "" }, "确认复核这张采购单吗？", `review:${item.ID || ""}`, "采购单已复核。")}>${actionState.busy === `review:${item.ID || ""}` ? "处理中..." : "复核"}</button>
                  <button class="btn secondary" type="button" disabled=${actionState.busy === `export:${item.ID || ""}`} onClick=${() => procurementAction("/api/pim/admin/procurement/order/export", { id: item.ID || "", note: notes[item.ID || ""] || "" }, "确认导出这张采购单的 CSV 吗？", `export:${item.ID || ""}`, "采购单已导出。")}>${actionState.busy === `export:${item.ID || ""}` ? "处理中..." : "导出"}</button>
                </div>
              </td>
            </tr>
          `) : html`<tr><td colspan="7" class="small">暂无采购单。</td></tr>`}
        </tbody></table></div>
      </div></div></section>

      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">最近采购动作</div>
        <h2 class="card-title">动作日志</h2>
        <div class="table-wrap section"><table><thead><tr><th>动作</th><th>结果</th><th>操作人</th><th>时间</th></tr></thead><tbody>
          ${recentActions.length ? recentActions.map((item) => html`
            <tr>
              <td><strong>${item.ActionType || "-"}</strong><div class="small">${item.ExternalRef || item.OrderID || "-"}</div></td>
              <td><${StatusBadge} label=${item.Status || "-"} currentTone=${tone(item.Status)} /><div class="small">${item.Message || "-"}</div></td>
              <td>${item.ActorName || item.ActorEmail || "-"}</td>
              <td class="small">${item.Created || "-"}</td>
            </tr>
          `) : html`<tr><td colspan="4" class="small">还没有最近采购动作。</td></tr>`}
        </tbody></table></div>
      </div></div></section>
    </section>
  `;
}

function SourceProductDetailPage() {
  const qs = new URLSearchParams(window.location.search);
  const id = qs.get("id") || "";
  const returnTo = qs.get("returnTo") || "/_/mrtang-admin/source/products";
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [approveNote, setApproveNote] = useState("");
  const [rejectNote, setRejectNote] = useState("");
  const resource = useResource(buildURL("/api/pim/admin/source/products/detail", { id, returnTo }), [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="商品详情" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const detail = payload.detail || {};
  const backHref = payload.returnTo || returnTo;

  async function detailAction(url, values, confirmMessage, busyKey, successMessage) {
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, values);
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "操作失败" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">商品详情</div>
        <h2 class="card-title">${detail.Name || "-"}</h2>
        ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
        ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
        <${ActionNotice} state=${actionState} />
        <div class="inline-pills">
          <span class="pill">productId: <code>${detail.ProductID || "-"}</code></span>
          <${StatusBadge} label=${detail.ReviewStatus || "-"} currentTone=${tone(detail.ReviewStatus)} />
          <span class="pill">sourceType: <code>${detail.SourceType || "-"}</code></span>
        </div>
        <div class="small" style="margin-top:12px;">${detail.CategoryPath || "-"}</div>
        <div class="action-row">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          <input type="text" value=${approveNote} onInput=${(event) => setApproveNote(event.currentTarget.value)} placeholder="审核备注" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "approve"} onClick=${() => detailAction("/api/pim/admin/source/products/status", { id: detail.ID || "", status: "approved", note: approveNote }, "确认将这个商品标记为通过吗？", "approve", "商品审核状态已更新。")}>${actionState.busy === "approve" ? "处理中..." : "通过"}</button>
          <input type="text" value=${rejectNote} onInput=${(event) => setRejectNote(event.currentTarget.value)} placeholder="驳回原因" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "reject"} onClick=${() => detailAction("/api/pim/admin/source/products/status", { id: detail.ID || "", status: "rejected", note: rejectNote }, "确认拒绝这个商品吗？", "reject", "商品审核状态已更新。")}>${actionState.busy === "reject" ? "处理中..." : "拒绝"}</button>
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">桥接状态</div>
        <h2 class="card-title">同步链状态</h2>
        <div class="inline-pills">
          <${StatusBadge} label=${(detail.Bridge && detail.Bridge.SyncStatus) || (detail.Bridge && detail.Bridge.Linked ? "linked" : "unlinked")} currentTone=${tone((detail.Bridge && detail.Bridge.SyncStatus) || (detail.Bridge && detail.Bridge.Linked ? "warning" : "error"))} />
          <span class="pill">supplierRecord: <code>${(detail.Bridge && detail.Bridge.SupplierRecordID) || "-"}</code></span>
          <span class="pill">vendure: <code>${(detail.Bridge && detail.Bridge.VendureProductID) || "-"} / ${(detail.Bridge && detail.Bridge.VendureVariantID) || "-"}</code></span>
        </div>
        ${(detail.Bridge && detail.Bridge.LastSyncError) ? html`<div class="flash error" style="margin-top:14px;">${detail.Bridge.LastSyncError}</div>` : null}
        <div class="action-row" style="margin-top:12px;">
          <button class="btn secondary" type="button" disabled=${actionState.busy === "promote"} onClick=${() => detailAction("/api/pim/admin/source/products/promote", { id: detail.ID || "" }, "确认桥接这个商品吗？", "promote", "商品已桥接到同步链。")}>${actionState.busy === "promote" ? "处理中..." : "桥接"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "promote-sync"} onClick=${() => detailAction("/api/pim/admin/source/products/promote-sync", { id: detail.ID || "" }, "确认桥接并同步这个商品吗？", "promote-sync", "商品已桥接并同步到后端。")}>${actionState.busy === "promote-sync" ? "处理中..." : "桥接并同步"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "retry-sync"} onClick=${() => detailAction("/api/pim/admin/source/products/retry-sync", { id: detail.ID || "" }, "确认重试这个商品的同步吗？", "retry-sync", "已触发商品同步重试。")}>${actionState.busy === "retry-sync" ? "处理中..." : "重试同步"}</button>
        </div>
      </div></section>
    </section>

    <section class="section card"><div class="card-body">
      <div class="card-kicker">数据块</div>
      <h2 class="card-title">摘要与详情</h2>
      <div class="split-grid section">
        <div class="table-card"><div class="table-card"><div class="card-body"><div class="card-kicker">摘要</div><pre>${detail.SummaryJSON || "-"}</pre></div></div></div>
        <div class="table-card"><div class="table-card"><div class="card-body"><div class="card-kicker">定价</div><pre>${detail.PricingJSON || "-"}</pre></div></div></div>
      </div>
      <div class="split-grid section">
        <div class="table-card"><div class="table-card"><div class="card-body"><div class="card-kicker">单位选项</div><pre>${detail.UnitOptions || "-"}</pre></div></div></div>
        <div class="table-card"><div class="table-card"><div class="card-body"><div class="card-kicker">下单单位</div><pre>${detail.OrderUnits || "-"}</pre></div></div></div>
      </div>
      <div class="table-card section"><div class="table-card"><div class="card-body"><div class="card-kicker">详情</div><pre>${detail.DetailJSON || "-"}</pre></div></div></div>
    </div></section>
  `;
}

function SourceAssetDetailPage() {
  const qs = new URLSearchParams(window.location.search);
  const id = qs.get("id") || "";
  const returnTo = qs.get("returnTo") || "/_/mrtang-admin/source/assets";
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [note, setNote] = useState("");
  const resource = useResource(buildURL("/api/pim/admin/source/assets/detail", { id, returnTo }), [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="图片详情" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const detail = payload.detail || {};
  const backHref = payload.returnTo || returnTo;

  async function processAsset() {
    if (!window.confirm("确认处理这张图片吗？")) return;
    setActionState({ busy: "process", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/process", { id: detail.ID || "", note });
      setActionState({ busy: "", message: result.message || "图片已进入处理流程。", error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "处理图片失败" });
    }
  }

  async function downloadOriginal() {
    if (!window.confirm("确认下载这张图片的原图吗？")) return;
    setActionState({ busy: "download", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/download", { id: detail.ID || "", note });
      setActionState({ busy: "", message: result.message || "原图已下载。", error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "下载原图失败" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">图片详情</div>
        <h2 class="card-title">${detail.Name || "-"}</h2>
        <${ActionNotice} state=${actionState} />
        <div class="inline-pills">
          <span class="pill">assetKey: <code>${detail.AssetKey || "-"}</code></span>
          <${StatusBadge} label=${originalImageStatusLabel(detail.OriginalImageStatus)} currentTone=${tone(detail.OriginalImageStatus)} />
          <${StatusBadge} label=${detail.ImageProcessingStatus || "-"} currentTone=${tone(detail.ImageProcessingStatus)} />
          <span class="pill">role: <code>${detail.AssetRole || "-"}</code></span>
        </div>
        ${(detail.OriginalImageError) ? html`<div class="flash error" style="margin-top:14px;">${detail.OriginalImageError}</div>` : null}
        ${(detail.ImageProcessingError) ? html`<div class="flash error" style="margin-top:14px;">${detail.ImageProcessingError}</div>` : null}
        <div class="action-row">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          <input type="text" value=${note} onInput=${(event) => setNote(event.currentTarget.value)} placeholder="处理备注" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "download"} onClick=${downloadOriginal}>${actionState.busy === "download" ? "下载中..." : "下载原图"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "process"} onClick=${processAsset}>${actionState.busy === "process" ? "处理中..." : "处理 / 重处理"}</button>
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">图片预览</div>
        <h2 class="card-title">源地址 / 原图文件 / 处理图</h2>
        <div class="split-grid section">
          <div>
            <div class="small" style="margin-bottom:8px;">源地址</div>
            ${detail.SourceURL ? html`<img alt="源地址图片" src=${detail.SourceURL} style="width:100%;max-height:420px;object-fit:contain;border:1px solid var(--line);border-radius:16px;background:#091521;" /><div class="small" style="margin-top:8px;"><a href=${detail.SourceURL} target="_blank" rel="noreferrer">打开源地址</a></div>` : html`<div class="small">暂无源地址</div>`}
          </div>
          <div>
            <div class="small" style="margin-bottom:8px;">原图文件</div>
            ${detail.OriginalImageURL ? html`<img alt="原图文件" src=${detail.OriginalImageURL} style="width:100%;max-height:420px;object-fit:contain;border:1px solid var(--line);border-radius:16px;background:#091521;" /><div class="small" style="margin-top:8px;"><a href=${detail.OriginalImageURL} target="_blank" rel="noreferrer">打开原图文件</a></div>` : html`<div class="small">尚未下载原图</div>`}
          </div>
          <div>
            <div class="small" style="margin-bottom:8px;">处理图</div>
            ${detail.ProcessedImageURL ? html`<img alt="处理图" src=${detail.ProcessedImageURL} style="width:100%;max-height:420px;object-fit:contain;border:1px solid var(--line);border-radius:16px;background:#091521;" /><div class="small" style="margin-top:8px;"><a href=${detail.ProcessedImageURL} target="_blank" rel="noreferrer">打开处理图</a></div>` : html`<div class="small">暂无处理图</div>`}
          </div>
        </div>
      </div></section>
    </section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">来源载荷</div>
      <pre>${detail.SourcePayloadJSON || "-"}</pre>
    </div></div></section>
  `;
}

function ProcurementDetailPage() {
  const qs = new URLSearchParams(window.location.search);
  const id = qs.get("id") || "";
  const returnTo = qs.get("returnTo") || "/_/mrtang-admin/procurement";
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [reviewNote, setReviewNote] = useState("");
  const [exportNote, setExportNote] = useState("");
  const resource = useResource(buildURL("/api/pim/admin/procurement/detail", { id, returnTo }), [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="采购详情" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const order = payload.order || {};
  const backHref = payload.returnTo || returnTo;
  const suppliers = (order.Summary && order.Summary.Suppliers) || [];
  const riskyItems = suppliers.flatMap((supplier) => (supplier.Items || []).filter((item) => ["loss", "warning"].includes((item.RiskLevel || "").toLowerCase())));

  async function procurementDetailAction(url, values, confirmMessage, busyKey, successMessage) {
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, values);
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "操作失败" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">采购详情</div>
        <h2 class="card-title">${order.ExternalRef || "-"}</h2>
        ${(payload.flashError) ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
        <${ActionNotice} state=${actionState} />
        <div class="inline-pills">
          <span class="pill">id: <code>${order.ID || "-"}</code></span>
          <${StatusBadge} label=${order.Status || "-"} currentTone=${tone(order.Status)} />
          <span class="pill">风险项 <code>${order.RiskyItemCount || 0}</code></span>
        </div>
        <div class="small" style="margin-top:12px;">商品 ${order.ItemCount || 0} 项 / 数量 ${typeof order.TotalQty === "number" ? order.TotalQty.toFixed(2) : order.TotalQty || "0.00"} / 成本 ${typeof order.TotalCostAmount === "number" ? order.TotalCostAmount.toFixed(2) : order.TotalCostAmount || "0.00"}</div>
        <div class="action-row">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          <input type="text" value=${reviewNote} onInput=${(event) => setReviewNote(event.currentTarget.value)} placeholder="复核备注" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "review"} onClick=${() => procurementDetailAction("/api/pim/admin/procurement/order/review", { id: order.ID || "", note: reviewNote }, "确认复核这张采购单吗？", "review", "采购单已复核。")}>${actionState.busy === "review" ? "处理中..." : "复核"}</button>
          <input type="text" value=${exportNote} onInput=${(event) => setExportNote(event.currentTarget.value)} placeholder="导出备注" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "export"} onClick=${() => procurementDetailAction("/api/pim/admin/procurement/order/export", { id: order.ID || "", note: exportNote }, "确认导出这张采购单吗？", "export", "采购单已导出。")}>${actionState.busy === "export" ? "处理中..." : "导出"}</button>
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">风险商品</div>
        <h2 class="card-title">优先处理</h2>
        ${riskyItems.length ? html`<div class="ops-grid section">
          ${riskyItems.map((item) => html`<div class="action-card"><div class="card-kicker">${item.RiskLevel || "-"}</div><div class="card-title">${item.Title || "-"}</div><div class="card-desc">${item.OriginalSKU || "-"} / ${item.SupplierCode || "-"}</div><div class="small">数量 ${typeof item.Quantity === "number" ? item.Quantity.toFixed(2) : item.Quantity || "0"} ${item.SalesUnit || ""} / 成本 ${typeof item.CostPrice === "number" ? item.CostPrice.toFixed(2) : item.CostPrice || "0.00"} / C价 ${typeof item.ConsumerPrice === "number" ? item.ConsumerPrice.toFixed(2) : item.ConsumerPrice || "0.00"}</div></div>`)}
        </div>` : html`<div class="small">当前采购单没有风险商品。</div>`}
      </div></section>
    </section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">摘要</div>
      <pre>${JSON.stringify(order.Summary || {}, null, 2)}</pre>
    </div></div></section>
  `;
}

function App() {
  const currentPath = window.location.pathname;
  const currentRoute = routePath(currentPath);
  return html`
    <${AppLayout}
      title=${currentRoute === "target-sync" ? "抓取入库" : currentRoute === "source" ? "源数据" : currentRoute === "source-categories" ? "源数据分类" : currentRoute === "source-products" ? "源数据商品" : currentRoute === "source-product-detail" ? "商品详情" : currentRoute === "source-assets" ? "源数据图片" : currentRoute === "source-asset-detail" ? "图片详情" : currentRoute === "source-asset-jobs" ? "图片任务" : currentRoute === "source-asset-job-detail" ? "图片任务详情" : currentRoute === "procurement" ? "采购" : currentRoute === "procurement-detail" ? "采购详情" : "总览"}
      subtitle=${currentRoute === "target-sync"
        ? "先开页面，再异步拉抓取摘要、来源矩阵和最近写操作；raw 慢时也只影响局部。"
        : currentRoute === "source"
          ? "先看 source 模块概览，再分流到商品、图片和日志；数据异步加载，不阻塞整页。"
          : currentRoute === "source-categories"
            ? "分类树抓取入库结果和已落库分类都在这里查看；页面先开壳，再异步加载。"
          : currentRoute === "source-products"
            ? "商品审核、桥接、同步重试改成前端异步列表；现有动作端点继续复用。"
            : currentRoute === "source-product-detail"
              ? "详情页也切到前端异步渲染，动作端点继续复用现有 POST 路由。"
            : currentRoute === "source-assets"
              ? "图片状态、失败聚合和批量处理改成前端异步列表；现有动作端点继续复用。"
              : currentRoute === "source-asset-detail"
                ? "详情页也切到前端异步渲染，动作端点继续复用现有 POST 路由。"
                : currentRoute === "source-asset-jobs"
                  ? "原图下载和图片处理任务都在这里追踪；失败后也可以直接重新执行。"
                  : currentRoute === "source-asset-job-detail"
                    ? "任务详情会显示进度、错误和最近日志，刷新页面后也能继续追踪。"
                : currentRoute === "procurement"
                  ? "采购列表、风险筛选和最近动作改成前端异步加载；详情页也已接入同一前端壳子。"
                  : currentRoute === "procurement-detail"
                    ? "详情页也切到前端异步渲染，风险商品和原始摘要不再阻塞整页。"
              : "后台首页先秒开壳子，再异步拉 coverage、source capture 和最近动作。"}
      currentPath=${currentPath}
    >
      ${currentRoute === "target-sync"
        ? html`<${TargetSyncPage} />`
        : currentRoute === "source"
          ? html`<${SourceModulePage} />`
        : currentRoute === "source-categories"
          ? html`<${SourceCategoriesPage} />`
        : currentRoute === "source-products"
          ? html`<${SourceProductsPage} />`
        : currentRoute === "source-product-detail"
          ? html`<${SourceProductDetailPage} />`
        : currentRoute === "source-assets"
          ? html`<${SourceAssetsPage} />`
        : currentRoute === "source-asset-detail"
          ? html`<${SourceAssetDetailPage} />`
        : currentRoute === "source-asset-jobs"
          ? html`<${SourceAssetJobsPage} />`
        : currentRoute === "source-asset-job-detail"
          ? html`<${SourceAssetJobDetailPage} />`
        : currentRoute === "procurement"
          ? html`<${ProcurementPage} />`
        : currentRoute === "procurement-detail"
          ? html`<${ProcurementDetailPage} />`
        : html`<${DashboardPage} />`}
    </${AppLayout}>
  `;
}

render(html`<${App} />`, document.getElementById("admin-app"));
