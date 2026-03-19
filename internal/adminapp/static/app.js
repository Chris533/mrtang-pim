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
    const error = new Error((data && data.message) || text || `HTTP ${response.status}`);
    error.payload = data;
    error.status = response.status;
    throw error;
  }
  return withExportedKeys(data || {});
}

function useResource(url, deps = []) {
  const [state, setState] = useState({ loading: true, error: "", data: null });
  useEffect(() => {
    let active = true;
    if (!url) {
      setState({ loading: false, error: "", data: null });
      return () => { active = false; };
    }
    setState({ loading: true, error: "", data: null });
    fetchJSON(url)
      .then((data) => active && setState({ loading: false, error: "", data }))
      .catch((error) => active && setState({ loading: false, error: error.message || "加载失败", data: null }));
    return () => { active = false; };
  }, [url, ...deps]);
  return state;
}

function classifyLoadError(error) {
  const message = String(error || "").trim();
  const lowered = message.toLowerCase();
  if (!message) return { title: "加载失败", detail: "请求没有返回可识别的错误信息。" };
  if (lowered.includes("deadline exceeded") || lowered.includes("timeout")) {
    return { title: "请求超时", detail: "源站返回过慢，当前区块已局部降级。你可以稍后重试。", raw: message };
  }
  if (lowered.includes("authorization") || lowered.includes("unauthorized") || lowered.includes("forbidden")) {
    return { title: "鉴权失败", detail: "当前会话的 Bearer 或登录上下文不可用，请先检查 raw 续活状态。", raw: message };
  }
  if (message.includes("返回空登录数据") || lowered.includes("empty")) {
    return { title: "上下文缺失", detail: "源站没有返回有效登录或业务上下文，请检查 openId、contactsId、customerId 是否匹配当前会话。", raw: message };
  }
  return { title: "加载失败", detail: message, raw: message };
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

function rawWarmupLabel(status) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "success") return "续活成功";
  if (normalized === "running") return "续活中";
  if (normalized === "partial") return "部分成功";
  if (normalized === "failed") return "续活失败";
  if (normalized === "skipped") return "已跳过";
  if (normalized === "idle") return "未执行";
  return status || "-";
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

function targetSyncRunFailedBranches(run) {
  const details = (run && run.Details) || [];
  const entityType = String((run && (run.EntityType || run.entityType)) || "").toLowerCase();
  const seen = new Set();
  return details.filter((item) => {
    const changeType = String(item.ChangeType || item.changeType || "").toLowerCase();
    const targetType = String(item.TargetType || item.targetType || "").toLowerCase();
    const targetKey = String(item.TargetKey || item.targetKey || "").trim();
    if (changeType !== "failed" || targetType !== entityType || !targetKey || seen.has(targetKey)) return false;
    seen.add(targetKey);
    return true;
  });
}

function retryFailedBranchesLabel(run) {
  const entityType = String((run && (run.EntityType || run.entityType)) || "").toLowerCase();
  const count = targetSyncRunFailedBranches(run).length;
  if (entityType === "category_sources") return `重跑失败分类来源分支（${count}）`;
  if (entityType === "products") return `重跑失败商品分支（${count}）`;
  return `重跑失败分支（${count}）`;
}

function retryFailedBranchesConfirmMessage(run) {
  const entityType = String((run && (run.EntityType || run.entityType)) || "").toLowerCase();
  const count = targetSyncRunFailedBranches(run).length;
  if (entityType === "category_sources") return `确认重跑 ${count} 个失败分类来源分支吗？`;
  if (entityType === "products") return `确认重跑 ${count} 个失败商品分支吗？`;
  return `确认重跑 ${count} 个失败分支吗？`;
}

function retryFailedBranchesStartedMessage(run, count) {
  const entityType = String((run && (run.EntityType || run.entityType)) || "").toLowerCase();
  if (entityType === "category_sources") return `已启动 ${count} 个失败分类来源分支重跑任务。`;
  if (entityType === "products") return `已启动 ${count} 个失败商品分支重跑任务。`;
  return `已启动 ${count} 个失败分支重跑任务。`;
}

function progressStageLabel(stage) {
  const normalized = (stage || "").toLowerCase();
  if (normalized === "queued") return "排队中";
  if (normalized === "loading_dataset") return "加载数据集";
  if (normalized === "categories") return "写入分类";
  if (normalized === "category_sources") return "写入分类商品来源";
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
  if (normalized === "missing") return "无源图地址";
  return status || "-";
}

function sourceAssetJobTypeLabel(jobType, mode) {
  const normalized = (jobType || "").toLowerCase();
  if (normalized === "download_original") return "原图下载";
  if (normalized === "process_asset") {
    return (mode || "").toLowerCase().includes("failed") ? "失败图片重处理" : "图片处理";
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

function normalizeIDList(values) {
  const items = Array.isArray(values)
    ? values
    : String(values || "")
        .split(",")
        .map((item) => item.trim());
  const seen = new Set();
  return items.filter((item) => {
    if (!item || seen.has(item)) return false;
    seen.add(item);
    return true;
  });
}

function sourceAssetJobTargetHref(item) {
  const assetIDs = normalizeIDList((item && item.AssetIDs) || []);
  if (assetIDs.length) {
    return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(",") });
  }
  const jobType = ((item && item.JobType) || "").toLowerCase();
  const mode = ((item && item.Mode) || "").toLowerCase();
  if (jobType === "download_original") {
    return buildURL("/_/mrtang-admin/source/assets", { originalStatus: "failed" });
  }
  if (jobType === "process_asset" && mode.includes("failed")) {
    return buildURL("/_/mrtang-admin/source/assets", { assetStatus: "failed" });
  }
  if (jobType === "process_asset") {
    return buildURL("/_/mrtang-admin/source/assets", { assetStatus: "pending" });
  }
  return "/_/mrtang-admin/source/assets";
}

function sourceAssetJobTargetLabel(item) {
  const assetIDs = normalizeIDList((item && item.AssetIDs) || []);
  if (assetIDs.length) return "查看本任务图片";
  const jobType = ((item && item.JobType) || "").toLowerCase();
  const mode = ((item && item.Mode) || "").toLowerCase();
  if (jobType === "download_original") return "查看原图失败图片";
  if (jobType === "process_asset" && mode.includes("failed")) return "查看处理失败图片";
  if (jobType === "process_asset") return "查看待处理图片";
  return "查看相关图片";
}

function sourceAssetJobRetryLabel(item) {
  return ((item && item.FailedItems) || []).length ? "仅重跑失败项" : "重新执行";
}

function sourceAssetJobModeLabel(mode) {
  const normalized = (mode || "").toLowerCase();
  if (normalized === "selected") return "选中项";
  if (normalized === "selected_failed") return "选中失败项";
  if (normalized === "failed") return "失败项";
  if (normalized === "failed_only") return "失败项";
  if (normalized === "pending") return "待处理";
  return "全量";
}

function backendCategoryStatusLabel(status) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "published") return "已创建到 Backend";
  if (normalized === "mapped") return "已保存待创建";
  if (normalized === "error") return "创建失败";
  return "待创建";
}

function sourceAssetJobSelectionCount(item) {
  return normalizeIDList((item && item.AssetIDs) || []).length;
}

function sourceAssetJobSuccessRate(item) {
  const total = Number((item && item.Total) || 0);
  if (!total) return "0%";
  const processed = Number((item && item.Processed) || 0);
  return `${Math.round((processed / total) * 100)}%`;
}

function sourceAssetJobSuccessLabel(item) {
  return `成功 ${item && item.Processed ? item.Processed : 0}`;
}

function sourceAssetJobSuccessHref(item) {
  const assetIDs = normalizeIDList((item && item.AssetIDs) || []);
  const jobType = ((item && item.JobType) || "").toLowerCase();
  if (!assetIDs.length) return "";
  if (jobType === "download_original") {
    return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(","), originalStatus: "downloaded" });
  }
  if (jobType === "process_asset") {
    return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(","), assetStatus: "processed" });
  }
  return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(",") });
}

function sourceAssetJobFailureHref(item) {
  const assetIDs = normalizeIDList((item && item.AssetIDs) || []);
  const jobType = ((item && item.JobType) || "").toLowerCase();
  if (!assetIDs.length) return sourceAssetJobTargetHref(item);
  if (jobType === "download_original") {
    return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(","), originalStatus: "failed" });
  }
  if (jobType === "process_asset") {
    return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(","), assetStatus: "failed" });
  }
  return buildURL("/_/mrtang-admin/source/assets", { assetIds: assetIDs.join(",") });
}

function sourceProductJobTypeLabel(jobType) {
  const normalized = (jobType || "").toLowerCase();
  if (normalized === "retry_sync") return "商品重试发布";
  if (normalized === "promote_sync") return "加入发布队列并发布";
  if (normalized === "promote") return "加入发布队列";
  return jobType || "-";
}

function sourceProductJobModeLabel(mode) {
  const normalized = (mode || "").toLowerCase();
  if (normalized === "selected") return "选中项";
  if (normalized === "filtered") return "当前筛选结果";
  return "全量";
}

function sourceProductJobRecentError(item) {
  if (item && item.Error) return item.Error;
  const logs = (item && item.Logs) || [];
  for (let index = logs.length - 1; index >= 0; index -= 1) {
    const message = (logs[index] && logs[index].Message) || "";
    if (message.includes("失败")) return message;
  }
  return "";
}

function sourceProductJobRetryLabel(item) {
  return ((item && item.FailedItems) || []).length ? "仅重跑失败项" : "重新执行";
}

function sourceProductJobFailedHref(item) {
  const failedRecordIDs = normalizeIDList(((item && item.FailedItems) || []).map((failed) => failed.RecordID));
  const ids = failedRecordIDs.length ? failedRecordIDs : normalizeIDList((item && item.ProductIDs) || []);
  if (!ids.length) return buildURL("/_/mrtang-admin/source/products", { syncState: "error" });
  return buildURL("/_/mrtang-admin/source/products", { syncState: "error", productIds: ids.join(",") });
}

function sourceProductJobRemaining(item) {
  const total = Number((item && item.Total) || 0);
  const processed = Number((item && item.Processed) || 0);
  const failed = Number((item && item.Failed) || 0);
  return Math.max(total - processed - failed, 0);
}

function sourceProductJobSummaryText(item) {
  return `成功 ${item.Processed || 0} / 总数 ${item.Total || 0} / 失败 ${item.Failed || 0} / 剩余 ${sourceProductJobRemaining(item)}`;
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
    { href: "/_/mrtang-admin/backend-release", label: "发布准备", visible: !!boot.canAccessSource },
    { href: "/_/mrtang-admin/procurement", label: "采购", visible: !!boot.canAccessProcurement },
    { href: "/_/mrtang-admin/audit", label: "审计", visible: true },
  ].filter((item) => item.visible);
  const topLinks = [
    ...navItems,
    ...(boot.canAccessSource ? [
      { href: "/_/mrtang-admin/source/categories", label: "分类" },
      { href: "/_/mrtang-admin/source/products", label: "商品" },
      { href: "/_/mrtang-admin/source/product-jobs", label: "发布任务" },
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

function buildPaginationItems(currentPage, totalPages) {
  const page = Math.max(1, Number(currentPage) || 1);
  const pages = Math.max(1, Number(totalPages) || 1);
  const visible = new Set([1, pages, page - 1, page, page + 1, page - 2, page + 2]);
  const sorted = Array.from(visible)
    .filter((item) => item >= 1 && item <= pages)
    .sort((left, right) => left - right);
  const items = [];
  for (let index = 0; index < sorted.length; index += 1) {
    const value = sorted[index];
    if (index > 0 && value-sorted[index - 1] > 1) {
      items.push("ellipsis");
    }
    items.push(value);
  }
  return items;
}

function Pagination({ basePath, pageParam, currentPage, totalPages, params }) {
  const page = Math.max(1, Number(currentPage) || 1);
  const pages = Math.max(1, Number(totalPages) || 1);
  if (pages <= 1) return null;
  const items = buildPaginationItems(page, pages);
  const baseParams = { ...(params || {}) };
  delete baseParams[pageParam];
  return html`
    <div class="pagination">
      <a class=${`page-link ${page <= 1 ? "disabled" : ""}`} href=${page <= 1 ? "#" : buildURL(basePath, { ...baseParams, [pageParam]: page - 1 })}>上一页</a>
      ${items.map((item) => item === "ellipsis"
        ? html`<span class="page-ellipsis">…</span>`
        : html`<a class=${`page-link ${item === page ? "active" : ""}`} href=${buildURL(basePath, { ...baseParams, [pageParam]: item })}>${item}</a>`)}
      <a class=${`page-link ${page >= pages ? "disabled" : ""}`} href=${page >= pages ? "#" : buildURL(basePath, { ...baseParams, [pageParam]: page + 1 })}>下一页</a>
    </div>
  `;
}

function DashboardPage() {
  const [reloadKey, setReloadKey] = useState(0);
  const [miniappReloadKey, setMiniappReloadKey] = useState(0);
  const [miniappLiveEnabled, setMiniappLiveEnabled] = useState(false);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const resource = useResource("/api/pim/admin/dashboard", [reloadKey]);
  const miniappResource = useResource(miniappLiveEnabled ? "/api/pim/admin/dashboard/miniapp-live" : "", [reloadKey, miniappReloadKey, miniappLiveEnabled]);
  if (resource.loading) return html`<${LoadingSection} label="总览数据" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const data = resource.data || {};
  const miniappPayload = miniappResource.data || {};
  const miniapp = miniappPayload.Miniapp || data.Miniapp || {};
  const miniappError = miniappPayload.MiniappError || "";
  const miniappErrorInfo = classifyLoadError(miniappError);
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
          <${StatusBadge} label=${sourceModeLabel(miniapp.SourceMode || "-")} currentTone=${tone(miniapp.SourceMode)} />
          <span class="pill">configMode: <code>${miniapp.ConfigSourceMode || "-"}</code></span>
          <span class="pill">datasetSource: <code>${miniapp.DatasetSource || "-"}</code></span>
          <span class="pill">sourceURL: <code>${miniapp.SourceURL || "-"}</code></span>
          ${miniapp.RawAuthStatus && miniapp.RawAuthStatus.Enabled ? html`<span class="pill">续活状态: <strong>${rawWarmupLabel(miniapp.RawAuthStatus.Status)}</strong></span>` : null}
        </div>
        <div class="action-row" style="margin-top:12px;">
          <button class="btn secondary" type="button" onClick=${() => { setMiniappLiveEnabled(true); setMiniappReloadKey((value) => value + 1); }}>
            ${miniappResource.loading ? "刷新中..." : "刷新实时摘要"}
          </button>
        </div>
        ${miniappResource.loading ? html`<div class="small" style="margin-top:14px;">正在加载实时源站摘要...</div>` : null}
        ${miniappError ? html`<div class="flash error" style="margin-top:14px;">
          <div><strong>${miniappErrorInfo.title}</strong></div>
          <div class="small" style="margin-top:8px;">${miniappErrorInfo.detail}</div>
          <div class="small" style="margin-top:8px;"><code>${miniappErrorInfo.raw || miniappError}</code></div>
        </div>` : null}
        ${miniapp.UsedStoredData ? html`<div class="flash ok" style="margin-top:14px;">当前默认展示已落库分类/商品/图片结果，不会自动刷新源站。只有点“刷新实时摘要”时才会请求实时源站。</div>` : null}
        ${miniapp.RawAuthStatus && miniapp.RawAuthStatus.Enabled ? html`<div class=${`flash ${((miniapp.RawAuthStatus.Status || "").toLowerCase() === "failed" ? "error" : "ok")}`} style="margin-top:14px;">
          <div>${miniapp.RawAuthStatus.Message || "raw 登录续活状态未知。"}</div>
          <div class="small" style="margin-top:8px;">上次尝试：${miniapp.RawAuthStatus.LastAttemptAt || "-"} / 最近成功：${miniapp.RawAuthStatus.LastSuccessAt || "-"} / OpenID：${miniapp.RawAuthStatus.OpenID || "未配置"}</div>
        </div>` : null}
        <div class="metric-grid section">
          <${MetricCard} eyebrow="Contracts" value=${miniappResource.loading ? "..." : (miniapp.ContractCount || 0)} detail=${`Dataset source: ${miniapp.DatasetSource || "-"}`} />
          <${MetricCard} eyebrow="Homepage" value=${miniappResource.loading ? "..." : (miniapp.HomepageSectionCount || 0)} detail=${`${miniapp.HomepageProductCount || 0} 个首页商品`} />
          <${MetricCard} eyebrow="Category Tree" value=${miniappResource.loading ? "..." : (miniapp.CategoryTopLevelCount || 0)} detail=${`${miniapp.CategoryNodeCount || 0} 个分类节点`} />
          <${MetricCard} eyebrow="Category Sections" value=${miniappResource.loading ? "..." : (miniapp.CategorySectionCount || 0)} detail=${`${miniapp.CategorySectionWithProducts || 0} 个带商品`} />
          <${MetricCard} eyebrow="Products" value=${miniappResource.loading ? "..." : (miniapp.ProductTotal || 0)} detail=${`${miniapp.ProductRRDetailCount || 0} rr_detail / ${miniapp.ProductSkeletonCount || 0} skeleton`} />
          <${MetricCard} eyebrow="Checkout" value=${miniappResource.loading ? "..." : (miniapp.OrderOperationCount || 0)} detail=${`${miniapp.CartOperationCount || 0} cart / ${miniapp.FreightScenarioCount || 0} freight`} />
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
          <${MetricCard} eyebrow="商品" value=${source.ProductCount || 0} detail=${`${source.ImportedCount || 0} 待审核 / ${source.ApprovedCount || 0} 待加入发布队列 / ${source.PromotedCount || 0} 已加入发布队列`} />
          <${MetricCard} eyebrow="Assets" value=${source.AssetCount || 0} detail=${`${source.ProcessedAssetCount || 0} processed / ${source.FailedAssetCount || 0} failed`} />
          <${MetricCard} eyebrow="Bridge" value=${source.LinkedCount || 0} detail=${`${source.SyncedCount || 0} synced / ${source.SyncErrorCount || 0} error`} />
        </div>
      </div></section>
    </div>

    <section class="section card"><div class="card-body">
      <div class="card-kicker">Backend Readiness</div>
      <h2 class="card-title">Backend 发布准备度</h2>
      <div class="inline-pills">
        <${StatusBadge} label=${data.BackendReadiness && data.BackendReadiness.Ready ? "可联调" : "待补字段"} currentTone=${data.BackendReadiness && data.BackendReadiness.Ready ? "success" : "warning"} />
        <span class="pill">Variant 字段: <code>${(data.BackendReadiness && data.BackendReadiness.VariantFieldConfigured) || 0}/${(data.BackendReadiness && data.BackendReadiness.VariantFieldTotal) || 0}</code></span>
        <span class="pill">Product 字段: <code>${(data.BackendReadiness && data.BackendReadiness.ProductFieldConfigured) || 0}/${(data.BackendReadiness && data.BackendReadiness.ProductFieldTotal) || 0}</code></span>
        <span class="pill">分类映射: <code>${(data.BackendReadiness && data.BackendReadiness.MappedCategoryCount) || 0}</code></span>
      </div>
      <div class="metric-grid section">
        <${MetricCard} eyebrow="分类映射" value=${(data.BackendReadiness && data.BackendReadiness.MappedCategoryCount) || 0} detail=${`${(data.BackendReadiness && data.BackendReadiness.PublishedCategoryCount) || 0} published / ${(data.BackendReadiness && data.BackendReadiness.PendingCategoryCount) || 0} pending`} />
        <${MetricCard} eyebrow="待发布商品" value=${(data.BackendReadiness && data.BackendReadiness.PromotedProductCount) || 0} detail=${`${(data.BackendReadiness && data.BackendReadiness.SyncedProductCount) || 0} 已同步`} />
      </div>
      ${data.BackendReadiness && data.BackendReadiness.MissingFields && data.BackendReadiness.MissingFields.length
        ? html`<div class="flash warning" style="margin-top:14px;">
            <div><strong>Vendure custom fields 仍未配置完整</strong></div>
            <div class="small" style="margin-top:8px;">缺失：${data.BackendReadiness.MissingFields.join("、")}</div>
            <div class="small" style="margin-top:8px;">先按文档配置字段，再做正式联调同步更稳。</div>
          </div>`
        : html`<div class="flash ok" style="margin-top:14px;">Vendure 发布所需的最小 custom fields 已配置，可进入小批量联调。</div>`}
      <div class="small" style="margin-top:12px;">文档参考：<code>docs/backend-miniapp-plan.md</code> / <code>docs/backend-release-contract.md</code></div>
    </div></section>

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

function BackendReleasePage() {
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [previewID, setPreviewID] = useState("");
  const resource = useResource("/api/pim/admin/backend-release", [reloadKey]);
  const previewResource = useResource(previewID ? buildURL("/api/pim/admin/backend-release/product-preview", { id: previewID }) : "", [previewID]);

  if (resource.loading) return html`<${LoadingSection} label="发布准备" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;

  const payload = resource.data || {};
  const summary = payload.summary || {};
  const categories = summary.Categories || [];
  const branches = summary.Branches || [];
  const products = summary.Products || [];
  const suggestedCategories = summary.SuggestedCategories || [];
  const recommendedProducts = summary.RecommendedProducts || [];
  const suggestionMap = useMemo(() => {
    const next = {};
    suggestedCategories.forEach((item) => {
      const key = item.SourceKey || item.sourceKey || "";
      if (!key || next[key]) return;
      next[key] = item;
    });
    return next;
  }, [suggestedCategories]);
  const visibleCategories = useMemo(() => {
    return [...categories].sort((left, right) => {
      const leftHasSuggestion = !!suggestionMap[left.SourceKey || left.sourceKey || ""];
      const rightHasSuggestion = !!suggestionMap[right.SourceKey || right.sourceKey || ""];
      if (leftHasSuggestion !== rightHasSuggestion) {
        return leftHasSuggestion ? -1 : 1;
      }
      return String(left.SourcePath || left.sourcePath || "").localeCompare(String(right.SourcePath || right.sourcePath || ""), "zh-CN");
    });
  }, [categories, suggestionMap]);
  const suggestedKeys = useMemo(() => suggestedCategories.map((item) => item.SourceKey || item.sourceKey || "").filter(Boolean), [suggestedCategories]);
  const failedCategoryKeys = useMemo(() => visibleCategories
    .filter((item) => String(item.PublishStatus || item.publishStatus || "").toLowerCase() === "error")
    .map((item) => item.SourceKey || item.sourceKey || "")
    .filter(Boolean), [visibleCategories]);
  const pendingRootKeys = useMemo(() => branches
    .filter((item) => Number(item.PendingCount || item.pendingCount || 0) > 0)
    .map((item) => item.RootKey || item.rootKey || "")
    .filter(Boolean), [branches]);

  useEffect(() => {
    if (previewID) return;
    const first = recommendedProducts[0] || products[0];
    if (first && first.ID) {
      setPreviewID(first.ID);
    }
  }, [previewID, recommendedProducts, products]);

  async function saveMapping(item, form) {
    const key = item.SourceKey || item.sourceKey || "";
    if (!key) return;
    const backendCollection = String(form.backendCollection || "").trim();
    const backendPath = String(form.backendPath || "").trim();
    if (!backendCollection && !backendPath) {
      setActionState({
        busy: "",
        message: "",
        error: "请至少填写 backend collection 或 backend path；空值保存后状态仍会保持 pending。",
      });
      return;
    }
    setActionState({ busy: `mapping:${key}`, message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/backend-release/category-mappings", {
        sourceKey: key,
        backendCollection,
        backendPath,
        note: form.note || "",
      });
      setActionState({
        busy: "",
        message: result.message || `分类映射已保存：${backendCollection || "-"} / ${backendPath || "-"}`,
        error: "",
      });
      window.scrollTo({ top: 0, behavior: "smooth" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "保存分类发布映射失败" });
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  }

  async function publishCategory(item, form, useSuggestion) {
    const key = item.SourceKey || item.sourceKey || "";
    if (!key) return;
    const suggestion = suggestionMap[key] || {};
    const backendCollection = String(useSuggestion ? (suggestion.SuggestedCollection || suggestion.suggestedCollection || "") : (form.backendCollection || "")).trim();
    const backendPath = String(useSuggestion ? (suggestion.SuggestedBackendPath || suggestion.suggestedBackendPath || "") : (form.backendPath || "")).trim();
    if (!backendCollection && !backendPath) {
      setActionState({
        busy: "",
        message: "",
        error: "请先填写 backend collection 或 backend path，或者直接使用按建议创建。",
      });
      return;
    }
    const actionKey = `publish:${key}`;
    setActionState({ busy: actionKey, message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/backend-release/category-publish", {
        sourceKey: key,
        backendCollection,
        backendPath,
        note: form.note || "",
      });
      setActionState({
        busy: "",
        message: result.message || `分类已创建到 Backend：${backendCollection || "-"} / ${backendPath || "-"}`,
        error: "",
      });
      window.scrollTo({ top: 0, behavior: "smooth" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "创建 backend 分类失败" });
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  }

  async function publishBatch(sourceKeys, label) {
    const keys = Array.from(new Set((sourceKeys || []).filter(Boolean)));
    if (!keys.length) {
      setActionState({ busy: "", message: "", error: `当前没有可执行“${label}”的分类。` });
      return;
    }
    if (!window.confirm(`确认批量执行“${label}”吗？共 ${keys.length} 个分类。`)) return;
    setActionState({ busy: `batch:${label}`, message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/backend-release/category-publish-batch", {
        sourceKeys: keys.join(","),
      });
      const batch = result.result || {};
      const errorSummary = (batch.Errors || []).slice(0, 3).join("；");
      setActionState({
        busy: "",
        message: result.message || `${label}已完成。`,
        error: errorSummary,
      });
      window.scrollTo({ top: 0, behavior: "smooth" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || `${label}失败` });
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  }

  async function cleanupAssets() {
    if (!window.confirm("确认清理 backend 中未被任何商品、规格或分类引用的 PIM 图片吗？此操作会删除历史冗余孤儿图。")) return;
    setActionState({ busy: "cleanup-assets", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/backend-release/cleanup-assets", {});
      setActionState({
        busy: "",
        message: result.message || "backend 冗余图片已清理。",
        error: "",
      });
      window.scrollTo({ top: 0, behavior: "smooth" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "清理 backend 冗余图片失败" });
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">Backend 发布</div>
        <h2 class="card-title">分类创建与商品发布准备</h2>
        <${ActionNotice} state=${actionState} />
        <div class="metric-grid section">
          <${MetricCard} eyebrow="分类创建" value=${summary.PublishedCount || 0} detail=${`顶级分支已创建 ${(summary.PublishedRootCount || 0)} / 待创建 ${(summary.PendingCategoryCount || 0)} / 失败 ${(summary.ErrorCategoryCount || 0)}`} />
          <${MetricCard} eyebrow="商品联调" value=${summary.ProductCount || 0} detail=${`待同步 ${(summary.ReadyProductCount || 0)} / 已同步 ${(summary.SyncedProductCount || 0)}`} />
          <${MetricCard} eyebrow="商品同步错误" value=${summary.ErrorProductCount || 0} />
        </div>
        <div class="flash warning" style="margin-top:14px;">
          <div><strong>推荐顺序</strong></div>
          <div class="small" style="margin-top:8px;">先配置 Vendure 字段，再直接创建 backend 分类，最后挑 1 到 3 个商品做 payload 预览和试同步。</div>
        </div>
        <div class="action-row" style="margin-top:14px;">
          <button class="btn secondary" type="button" disabled=${actionState.busy === "batch:按建议批量创建" || !suggestedKeys.length} onClick=${() => publishBatch(suggestedKeys, "按建议批量创建")}>
            ${actionState.busy === "batch:按建议批量创建" ? "创建中..." : `按建议批量创建（${suggestedKeys.length}）`}
          </button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "batch:重试失败分类" || !failedCategoryKeys.length} onClick=${() => publishBatch(failedCategoryKeys, "重试失败分类")}>
            ${actionState.busy === "batch:重试失败分类" ? "重试中..." : `重试失败分类（${failedCategoryKeys.length}）`}
          </button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "batch:创建全部待创建顶级分类" || !pendingRootKeys.length} onClick=${() => publishBatch(pendingRootKeys, "创建全部待创建顶级分类")}>
            ${actionState.busy === "batch:创建全部待创建顶级分类" ? "创建中..." : `创建全部待创建顶级分类（${pendingRootKeys.length}）`}
          </button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "cleanup-assets"} onClick=${cleanupAssets}>
            ${actionState.busy === "cleanup-assets" ? "清理中..." : "清理 backend 冗余图片"}
          </button>
        </div>
      </div></section>

      <section class="card"><div class="card-body">
        <div class="card-kicker">文档</div>
        <h2 class="card-title">联调参考</h2>
        <div class="small">字段映射与发布模型：<code>docs/backend-release-contract.md</code></div>
        <div class="small" style="margin-top:8px;">Vendure 字段配置：<code>docs/vendure-field-setup.md</code></div>
        <div class="small" style="margin-top:8px;">整体发布前规划：<code>docs/backend-miniapp-plan.md</code></div>
        <div class="small" style="margin-top:8px;">Backend 分类模型：<code>../mrtang-backend/docs/category-release-model.md</code></div>
      </div></section>
    </section>

    <section class="section split-grid">
      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">顶级分支创建概览</div>
        <h2 class="card-title">为什么 backend 里可能只看到一个顶级分类</h2>
        <div class="small">当前批量创建会递归补齐父子层级，所以一次可能创建很多分类，但它们可能都属于同一个顶级分支。</div>
        <div class="table-wrap section"><table><thead><tr><th>顶级分支</th><th>总数</th><th>已创建</th><th>待创建</th><th>失败</th></tr></thead><tbody>
          ${branches.length ? branches.map((item) => html`<tr>
            <td><strong>${item.Label || item.label || "-"}</strong><div class="small"><code>${item.RootKey || item.rootKey || "-"}</code></div></td>
            <td>${item.TotalCount || item.totalCount || 0}</td>
            <td>${item.PublishedCount || item.publishedCount || 0}</td>
            <td>${item.PendingCount || item.pendingCount || 0}</td>
            <td>${item.ErrorCount || item.errorCount || 0}</td>
          </tr>`) : html`<tr><td colspan="5" class="small">当前没有顶级分支摘要。</td></tr>`}
        </tbody></table></div>
      </div></div></section>

      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">最小分类创建样例</div>
        <h2 class="card-title">建议先创建这些 Backend 分类</h2>
        <div class="small">更推荐直接“按建议创建”或“创建到 Backend”，而不是只保存本地映射。</div>
        <div class="small" style="margin-top:8px;">下方分类列表会优先展示有建议路径的分类行，便于你直接创建。</div>
        <div class="table-wrap section"><table><thead><tr><th>源分类</th><th>建议 collection</th><th>建议 path</th><th>说明</th></tr></thead><tbody>
          ${suggestedCategories.length ? suggestedCategories.map((item) => html`<tr>
            <td><strong>${item.Label || "-"}</strong><div class="small">${item.SourcePath || "-"}</div></td>
            <td><code>${item.SuggestedCollection || "-"}</code></td>
            <td><code>${item.SuggestedBackendPath || "-"}</code></td>
            <td class="small">${item.Reason || "-"}</td>
          </tr>`) : html`<tr><td colspan="4" class="small">当前没有新的分类映射建议。</td></tr>`}
        </tbody></table></div>
      </div></div></section>

      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">联调候选商品</div>
        <h2 class="card-title">建议先试这 1 到 3 个商品</h2>
        <div class="table-wrap section"><table><thead><tr><th>商品</th><th>特征</th><th>状态</th><th>预览</th></tr></thead><tbody>
          ${recommendedProducts.length ? recommendedProducts.map((item) => html`<tr>
            <td><strong>${item.Title || "-"}</strong><div class="small">${item.SupplierCode || "-"} / ${item.SKU || "-"}</div></td>
            <td><div class="small">${item.Reason || "基础联调样例"}</div><div class="small">Audience: ${item.TargetAudience || "ALL"} / Rate: ${item.ConversionRate || 1}</div></td>
            <td><${StatusBadge} label=${item.SyncStatus || "-"} currentTone=${tone(item.SyncStatus)} /></td>
            <td><button class="btn secondary" type="button" disabled=${!item.ReadyForPreview} onClick=${() => setPreviewID(item.ID || "")}>预览 payload</button></td>
          </tr>`) : html`<tr><td colspan="4" class="small">当前没有推荐联调商品。</td></tr>`}
        </tbody></table></div>
      </div></div></section>
    </section>

    <section class="section card"><div class="card-body">
      <div class="card-kicker">Backend 分类创建</div>
      <h2 class="card-title">source category -> backend collection</h2>
      <div class="table-wrap section"><table><thead><tr><th>源分类</th><th>backend collection</th><th>backend path</th><th>状态</th><th>操作</th></tr></thead><tbody>
        ${visibleCategories.length ? visibleCategories.map((item) => html`<${BackendCategoryMappingRow} key=${item.SourceKey || item.sourceKey} item=${item} suggestion=${suggestionMap[item.SourceKey || item.sourceKey || ""]} onSave=${saveMapping} onPublish=${publishCategory} busy=${actionState.busy} />`) : html`<tr><td colspan="5" class="small">还没有分类映射数据。</td></tr>`}
      </tbody></table></div>
    </div></section>

    <section class="section split-grid">
      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">待联调商品</div>
        <h2 class="card-title">最近 supplier_products</h2>
        <div class="table-wrap section"><table><thead><tr><th>商品</th><th>分类</th><th>字段</th><th>状态</th><th>预览</th></tr></thead><tbody>
          ${products.length ? products.map((item) => html`<tr>
            <td><strong>${item.Title || "-"}</strong><div class="small">${item.SupplierCode || "-"} / ${item.SKU || "-"}</div></td>
            <td>${item.NormalizedCategory || "-"}</td>
            <td><div class="small">Audience: ${item.TargetAudience || "ALL"}</div><div class="small">Rate: ${item.ConversionRate || 1}</div><div class="small">${item.HasProcessedImage ? "已有处理图" : "仅原图"}</div></td>
            <td><${StatusBadge} label=${item.SyncStatus || "-"} currentTone=${tone(item.SyncStatus)} /></td>
            <td><button class="btn secondary" type="button" disabled=${!item.ReadyForPreview} onClick=${() => setPreviewID(item.ID || "")}>预览 payload</button></td>
          </tr>`) : html`<tr><td colspan="5" class="small">当前还没有可联调商品。</td></tr>`}
        </tbody></table></div>
      </div></div></section>

      <section class="card"><div class="card-body">
        <div class="card-kicker">Payload Preview</div>
        <h2 class="card-title">Vendure 同步预览</h2>
        ${!previewID ? html`<div class="small">从左侧选一个商品，查看将要发送给 Vendure 的 payload。</div>` : null}
        ${previewID && previewResource.loading ? html`<${LoadingSection} label="payload 预览" />` : null}
        ${previewID && previewResource.error ? html`<div class="flash error">${previewResource.error}</div>` : null}
        ${previewID && previewResource.data && previewResource.data.preview && previewResource.data.preview.Payload ? html`
          <pre class="json-block">${JSON.stringify(previewResource.data.preview.Payload, null, 2)}</pre>
        ` : null}
      </div></section>
    </section>
  `;
}

function BackendCategoryMappingRow({ item, suggestion, onSave, onPublish, busy }) {
  const [backendCollection, setBackendCollection] = useState(item.BackendCollection || item.backendCollection || "");
  const [backendPath, setBackendPath] = useState(item.BackendPath || item.backendPath || "");
  const [note, setNote] = useState(item.Note || item.note || "");
  const key = item.SourceKey || item.sourceKey || "";
  const hasMapping = !!String(backendCollection || "").trim() || !!String(backendPath || "").trim();
  const hasSuggestion = !!suggestion;
  return html`<tr>
    <td><strong>${item.Label || item.label || "-"}</strong><div class="small">${item.SourcePath || item.sourcePath || "-"}</div></td>
    <td><input class="input" value=${backendCollection} onInput=${(event) => setBackendCollection(event.target.value)} placeholder="collections/meat/chicken" /></td>
    <td><input class="input" value=${backendPath} onInput=${(event) => setBackendPath(event.target.value)} placeholder="鸡产品/鸡副/鸡块" /></td>
    <td>
      <${StatusBadge} label=${backendCategoryStatusLabel(item.PublishStatus || item.publishStatus || "pending")} currentTone=${tone(item.PublishStatus || item.publishStatus)} />
      ${(item.LastError || item.lastError) ? html`<div class="small">${item.LastError || item.lastError}</div>` : null}
      ${item.BackendCollectionID || item.backendCollectionId ? html`<div class="small">backend id: <code>${item.BackendCollectionID || item.backendCollectionId}</code></div>` : null}
      ${item.PublishedAt || item.publishedAt ? html`<div class="small">最近创建：${item.PublishedAt || item.publishedAt}</div>` : null}
      ${!hasMapping && hasSuggestion ? html`<div class="small">当前未填写自定义路径，可直接按建议创建。</div>` : null}
      ${!hasMapping && !hasSuggestion ? html`<div class="small">当前未填写 backend 映射，请手工输入后创建。</div>` : null}
    </td>
    <td>
      <div class="action-row">
        ${suggestion ? html`<button class="btn secondary" type="button" disabled=${busy === `publish:${key}`} onClick=${() => onPublish(item, { backendCollection, backendPath, note }, true)}>${busy === `publish:${key}` ? "创建中..." : "按建议创建"}</button>` : null}
        <button class="btn secondary" type="button" disabled=${busy === `publish:${key}` || (!hasMapping && !hasSuggestion)} onClick=${() => onPublish(item, { backendCollection, backendPath, note }, false)}>${busy === `publish:${key}` ? "创建中..." : (String(item.PublishStatus || item.publishStatus || "").toLowerCase() === "published" ? "重新同步到 Backend" : "创建到 Backend")}</button>
        <button class="btn secondary" type="button" disabled=${busy === `mapping:${key}` || !hasMapping} onClick=${() => onSave(item, { backendCollection, backendPath, note })}>${busy === `mapping:${key}` ? "保存中..." : "仅保存本地路径"}</button>
      </div>
    </td>
  </tr>`;
}

function confirmSubmit(message, event) {
  if (!window.confirm(message)) {
    event.preventDefault();
  }
}

function TargetSyncPage() {
  const [reloadKey, setReloadKey] = useState(0);
  const [liveReloadKey, setLiveReloadKey] = useState(0);
  const [checkoutReloadKey, setCheckoutReloadKey] = useState(0);
  const [liveEnabled, setLiveEnabled] = useState(false);
  const [checkoutEnabled, setCheckoutEnabled] = useState(false);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "", href: "", hrefLabel: "" });
  const [activeRunId, setActiveRunId] = useState("");
  const [activeRun, setActiveRun] = useState(null);
  const [activeRunError, setActiveRunError] = useState("");
  const resource = useResource("/api/pim/admin/target-sync", [reloadKey]);
  const liveResource = useResource(liveEnabled ? "/api/pim/admin/target-sync/live" : "", [reloadKey, liveReloadKey, liveEnabled]);
  const checkoutResource = useResource(checkoutEnabled ? "/api/pim/admin/target-sync/checkout-live" : "", [reloadKey, checkoutReloadKey, checkoutEnabled]);

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
  const livePayload = liveResource.data || {};
  const liveSummary = livePayload.summary || {};
  const checkoutPayload = checkoutResource.data || {};
  const checkoutSummary = checkoutPayload.summary || {};

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
    if (entityType === "category_sources") return { href: "/_/mrtang-admin/source/categories", hrefLabel: "查看分类与商品归属" };
    if (entityType === "category_rebuild") return { href: "/_/mrtang-admin/source/products", hrefLabel: "查看商品分类归属" };
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
      return `抓分类树完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    if (entityType === "category_sources") {
      return `刷新分类商品来源完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    if (entityType === "category_rebuild") {
      return `重建分类商品归属完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    if (entityType === "products") {
      return `按已保存分类来源抓商品规格完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
    }
    if (entityType === "assets") {
      return `按当前源站结果抓图片完成：新增 ${created}，更新 ${updated}，未变化 ${unchanged}。`;
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
      const payloadRun = error && error.payload && error.payload.run;
      if (payloadRun && payloadRun.ID) {
        setActiveRun(payloadRun);
        setActiveRunId(payloadRun.ID);
        setActiveRunError("");
        setActionState({
          busy: "",
          message: "已有同类抓取任务在执行中，已切换到当前任务进度。",
          error: "",
          href: buildURL("/_/mrtang-admin/target-sync/run", { id: payloadRun.ID }),
          hrefLabel: "查看运行详情",
        });
        setReloadKey((value) => value + 1);
        return;
      }
      setActionState({ busy: "", message: "", error: error.message || "执行抓取入库失败", href: "", hrefLabel: "" });
      setReloadKey((value) => value + 1);
    }
  }

  async function retryFailedBranches(run) {
    const runId = run && run.ID;
    const failedBranches = targetSyncRunFailedBranches(run);
    if (!runId || !failedBranches.length) {
      setActionState({ busy: "", message: "", error: "当前运行记录没有可重跑的失败分支。", href: "", hrefLabel: "" });
      return;
    }
    if (!window.confirm(retryFailedBranchesConfirmMessage(run))) return;
    setActionState({ busy: `retry-failed:${runId}`, message: "", error: "", href: "", hrefLabel: "" });
    try {
      const result = await postForm("/api/pim/admin/target-sync/run/retry-failed-branches", { runId });
      const runs = result.runs || [];
      const firstRun = runs[0] || null;
      setActionState({
        busy: "",
        message: result.message || retryFailedBranchesStartedMessage(run, runs.length),
        error: "",
        href: firstRun && firstRun.ID ? buildURL("/_/mrtang-admin/target-sync/run", { id: firstRun.ID }) : "",
        hrefLabel: firstRun && firstRun.ID ? "查看首个重跑任务" : "",
      });
      if (firstRun && firstRun.ID) {
        setActiveRun(firstRun);
        setActiveRunId(firstRun.ID);
      }
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "重跑失败分支失败", href: "", hrefLabel: "" });
    }
  }

  const scopeOptions = liveSummary.ScopeOptions || [];
  const progressTotal = (activeRun && activeRun.ProgressTotal) || 0;
  const progressDone = (activeRun && activeRun.ProgressDone) || 0;
  const progressPercent = progressTotal > 0 ? Math.min(100, Math.round((progressDone / progressTotal) * 100)) : 0;
  const recentRuns = summary.Runs || [];
  const effectiveSourceMode = liveSummary.SourceMode || summary.SourceMode;
  const liveError = livePayload.flashError || "";
  const checkoutError = checkoutPayload.flashError || "";
  const liveErrorInfo = classifyLoadError(liveError);
  const checkoutErrorInfo = classifyLoadError(checkoutError);
  const displayedTopLevelCount = liveEnabled ? (liveSummary.TopLevelCount || 0) : (summary.TopLevelCount || 0);
  const displayedNodeCount = liveEnabled ? (liveSummary.ExpectedNodeCount || 0) : (summary.CategoryCount || 0);
  const displayedProductCount = liveEnabled ? (liveSummary.ExpectedProductCount || 0) : (summary.SourceProductCount || 0);
  const displayedAssetCount = liveEnabled ? (liveSummary.ExpectedAssetCount || 0) : (summary.SourceAssetCount || 0);
  const displayedScopeOptions = (liveEnabled ? liveSummary.ScopeOptions : summary.ScopeOptions) || [];

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">抓取入库</div>
        <h2 class="card-title">先抓分类树，再刷新分类商品来源，再按来源抓商品规格</h2>
        ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
        ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
        <${ActionNotice} state=${actionState} />
        <div class="action-row">
          <button class="btn secondary" type="button" disabled=${actionState.busy === "ensure:category_tree:"} onClick=${() => ensureJob("category_tree", "", "分类树")}>${actionState.busy === "ensure:category_tree:" ? "保存中..." : "保存分类树任务"}</button>
          <button class="btn" type="button" disabled=${actionState.busy === "run:category_tree:"} onClick=${() => runJob("category_tree", "", "分类树", "确认立即抓取分类树吗？")}>${actionState.busy === "run:category_tree:" ? "启动中..." : "抓分类树"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "run:category_sources:"} onClick=${() => runJob("category_sources", "", "分类商品来源", "确认立即刷新分类商品来源吗？这一步会请求各分类路径下的商品列表，但不会抓商品详情。")}>${actionState.busy === "run:category_sources:" ? "启动中..." : "刷新分类商品来源"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "run:category_rebuild:"} onClick=${() => runJob("category_rebuild", "", "分类商品归属", "确认基于全部已保存分类来源重建商品分类归属吗？这一步不会请求源站。")}>${actionState.busy === "run:category_rebuild:" ? "启动中..." : "全量重建分类商品归属"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "run:products:"} onClick=${() => runJob("products", "", "商品规格", "确认基于已保存分类商品来源抓取商品规格吗？若没有来源，将先即时刷新来源。")}>${actionState.busy === "run:products:" ? "启动中..." : "按已保存来源抓商品规格"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "run:assets:"} onClick=${() => runJob("assets", "", "商品图片", "确认立即抓取图片入库吗？图片仍基于当前商品结果生成。")}>${actionState.busy === "run:assets:" ? "启动中..." : "按已保存商品抓图片"}</button>
        </div>
        <div class="small" style="margin-top:10px;">默认流程现在是：先抓分类树，再刷新分类商品来源，再按已保存来源抓商品规格。商品规格抓取不再重新请求分类接口；只有没有已保存来源时，才会即时补刷新。</div>
        <div class="action-row" style="margin-top:12px;">
          <button class="btn secondary" type="button" onClick=${() => { setLiveEnabled(true); setLiveReloadKey((value) => value + 1); }}>
            ${liveResource.loading ? "刷新中..." : "刷新实时范围摘要"}
          </button>
          <button class="btn secondary" type="button" onClick=${() => { setCheckoutEnabled(true); setCheckoutReloadKey((value) => value + 1); }}>
            ${checkoutResource.loading ? "刷新中..." : "刷新实时 checkout 摘要"}
          </button>
        </div>
        <div class="inline-pills section">
          <${StatusBadge} label=${sourceModeLabel(effectiveSourceMode)} currentTone=${tone(effectiveSourceMode)} />
          <span class="pill">sourceURL: <code>${payload.sourceURL || "-"}</code></span>
          <span class="pill">${payload.requiresAuth ? "当前 API 需要 Bearer 鉴权" : "当前 API 默认公开"}</span>
          ${summary.RawAuthStatus && summary.RawAuthStatus.Enabled ? html`<span class="pill">续活状态: <strong>${rawWarmupLabel(summary.RawAuthStatus.Status)}</strong></span>` : null}
        </div>
        ${summary.RawAuthStatus && summary.RawAuthStatus.Enabled ? html`<div class=${`flash ${((summary.RawAuthStatus.Status || "").toLowerCase() === "failed" ? "error" : "ok")}`} style="margin-top:14px;">
          <div>${summary.RawAuthStatus.Message || "raw 登录续活状态未知。"}</div>
          <div class="small" style="margin-top:8px;">上次尝试：${summary.RawAuthStatus.LastAttemptAt || "-"} / 最近成功：${summary.RawAuthStatus.LastSuccessAt || "-"} / OpenID：${summary.RawAuthStatus.OpenID || "未配置"}</div>
        </div>` : null}
        <div class="metric-grid section">
          <${MetricCard} eyebrow="抓取任务" value=${summary.JobCount || 0} />
          <${MetricCard} eyebrow="运行记录" value=${summary.RunCount || 0} />
          <${MetricCard} eyebrow="分类商品来源" value=${summary.SourceCategorySectionCount || 0} detail="已保存的分类路径 -> 商品列表来源快照" />
          <${MetricCard} eyebrow="顶级分类" value=${liveResource.loading ? "..." : displayedTopLevelCount} />
          <${MetricCard} eyebrow="分类节点" value=${liveResource.loading ? "..." : displayedNodeCount} />
          <${MetricCard} eyebrow="目标商品" value=${liveResource.loading ? "..." : displayedProductCount} detail=${liveEnabled ? (liveResource.loading ? "正在加载实时源站摘要" : `${liveSummary.ExpectedMultiUnitCount || 0} 个多单位`) : "默认显示已落库商品数"} />
          <${MetricCard} eyebrow="目标图片" value=${liveResource.loading ? "..." : displayedAssetCount} />
        </div>
        ${!liveEnabled ? html`<div class="flash ok" style="margin-top:14px;">当前默认展示已保存结果，不会自动刷新源站分类与商品来源。只有点“刷新实时范围摘要”时才会请求实时源站。</div>` : null}
        ${liveError ? html`<div class="flash error" style="margin-top:14px;">
          <div><strong>${liveErrorInfo.title}</strong></div>
          <div class="small" style="margin-top:8px;">${liveErrorInfo.detail}</div>
          <div class="small" style="margin-top:8px;"><code>${liveErrorInfo.raw || liveError}</code></div>
        </div>` : null}
        ${liveSummary.UsedStoredData ? html`<div class="flash ok" style="margin-top:14px;">当前范围摘要已自动回退到已落库分类/商品/图片结果。若分类商品来源已存在，商品归属会优先复用已保存来源，不必每次重新实时读取全部分类商品列表。</div>` : null}
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
            ${targetSyncRunFailedBranches(activeRun).length ? html`<div class="action-row" style="margin-top:12px;">
              <button class="btn secondary" type="button" disabled=${actionState.busy === `retry-failed:${activeRun.ID}`} onClick=${() => retryFailedBranches(activeRun)}>
                ${actionState.busy === `retry-failed:${activeRun.ID}` ? "启动中..." : retryFailedBranchesLabel(activeRun)}
              </button>
            </div>` : null}
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
          <a class="action-card" href="/_/mrtang-admin/source/categories"><div class="card-kicker">已入库分类</div><div class="metric-value">${summary.CategoryCount || 0}</div><div class="card-desc">查看分类树、层级和分类商品归属统计。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products"><div class="card-kicker">已入库商品</div><div class="metric-value">${summary.SourceProductCount || 0}</div><div class="card-desc">查看抓取保存下来的商品、规格和审核状态。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets"><div class="card-kicker">已入库图片</div><div class="metric-value">${summary.SourceAssetCount || 0}</div><div class="card-desc">查看抓取保存下来的封面、轮播和详情图。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products?productStatus=imported"><div class="card-kicker">待审核商品</div><div class="metric-value">${summary.SourceImportedCount || 0}</div><div class="card-desc">商品和规格变化后自动回到 imported。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/products?productStatus=approved"><div class="card-kicker">待加入发布队列商品</div><div class="metric-value">${summary.SourceApprovedCount || 0}</div><div class="card-desc">审核通过后加入发布队列，再进入 supplier_products 和 backend 发布链。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets?assetStatus=pending"><div class="card-kicker">待处理图片</div><div class="metric-value">${summary.SourceAssetPendingCount || 0}</div><div class="card-desc">图片变化后自动重置为 pending。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets?assetStatus=failed"><div class="card-kicker">失败图片</div><div class="metric-value">${summary.SourceAssetFailedCount || 0}</div><div class="card-desc">在图片模块继续重试或人工处理。</div></a>
        </div>
      </div></section>

      <section class="table-card"><div class="table-card"><div class="card-body">
        <div class="card-kicker">按范围抓取入库</div>
        <h2 class="card-title">顶级分类批次</h2>
        ${liveEnabled && liveResource.loading ? html`<div class="small section">正在加载实时源站分类范围摘要...</div>` : null}
        ${liveError ? html`<div class="flash error" style="margin-top:14px;">
          <div><strong>${liveErrorInfo.title}</strong></div>
          <div class="small" style="margin-top:8px;">${liveErrorInfo.detail}</div>
          <div class="small" style="margin-top:8px;"><code>${liveErrorInfo.raw || liveError}</code></div>
        </div>` : null}
        ${!liveEnabled ? html`<div class="flash ok" style="margin-top:14px;">当前表格默认基于已保存分类树、分类商品来源和商品结果生成，不会自动刷新源站。</div>` : null}
        ${liveSummary.UsedStoredData ? html`<div class="flash ok" style="margin-top:14px;">当前表格已回退到已落库结果统计；可以继续查看分类归属和数量，等源站恢复后再做实时抓取。</div>` : null}
        <div class="table-wrap section"><table><thead><tr><th>分类</th><th>节点</th><th>商品</th><th>图片</th><th>动作</th></tr></thead><tbody>
          ${displayedScopeOptions.length ? displayedScopeOptions.filter((item) => item.Key).map((item) => html`<tr>
            <td><strong>${item.Label || "-"}</strong><div class="small">${item.Key || "-"}</div></td>
            <td>${item.NodeCount || 0}</td>
            <td>${item.ProductCount || 0}</td>
            <td>${item.AssetCount || 0}</td>
            <td>
              <div class="action-row">
                <button class="btn secondary" type="button" disabled=${actionState.busy === `ensure:category_tree:${item.Key}`} onClick=${() => ensureJob("category_tree", item.Key, item.Label || item.Key)}>保存分类树任务</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:category_tree:${item.Key}`} onClick=${() => runJob("category_tree", item.Key, item.Label || item.Key, `确认执行 ${item.Label || item.Key} 的分类树抓取吗？`)}>抓分类树</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:category_sources:${item.Key}`} onClick=${() => runJob("category_sources", item.Key, item.Label || item.Key, `确认刷新 ${item.Label || item.Key} 的分类商品来源吗？这一步不会抓商品详情。`)}>刷新分类商品来源</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:category_rebuild:${item.Key}`} onClick=${() => runJob("category_rebuild", item.Key, item.Label || item.Key, `确认仅基于 ${item.Label || item.Key} 的已保存来源重建商品分类归属吗？这一步不会请求源站。`)}>按该分类重建归属</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:products:${item.Key}`} onClick=${() => runJob("products", item.Key, item.Label || item.Key, `确认基于 ${item.Label || item.Key} 已保存分类商品来源抓取商品规格吗？若没有来源，将先即时刷新来源。`)}>按已保存来源抓商品</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `run:assets:${item.Key}`} onClick=${() => runJob("assets", item.Key, item.Label || item.Key, `确认执行 ${item.Label || item.Key} 的图片抓取入库吗？`)}>按已保存商品抓图片</button>
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
        ${!checkoutEnabled ? html`<div class="flash ok" style="margin-top:14px;">checkout 来源矩阵默认不自动刷新；只有点“刷新实时 checkout 摘要”时才会读取实时源站。</div>` : null}
        ${checkoutEnabled && checkoutResource.loading ? html`<div class="small section">正在加载 checkout 来源矩阵...</div>` : null}
        ${checkoutError ? html`<div class="flash error" style="margin-top:14px;">
          <div><strong>${checkoutErrorInfo.title}</strong></div>
          <div class="small" style="margin-top:8px;">${checkoutErrorInfo.detail}</div>
          <div class="small" style="margin-top:8px;"><code>${checkoutErrorInfo.raw || checkoutError}</code></div>
        </div>` : null}
        <div class="table-wrap section"><table><thead><tr><th>链路</th><th>状态</th><th>contractId</th><th>说明</th></tr></thead><tbody>
          ${(checkoutSummary.CheckoutSources || []).length ? (checkoutSummary.CheckoutSources || []).map((item) => html`<tr><td><strong>${item.Label || "-"}</strong></td><td><${StatusBadge} label=${checkoutStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td><td class="small"><code>${item.ContractID || "-"}</code></td><td class="small">${item.Note || "-"}</td></tr>`) : html`<tr><td colspan="4" class="small">当前还没有 checkout 来源数据。</td></tr>`}
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
            <td class="small">
              新增 ${run.CreatedCount || 0} / 更新 ${run.UpdatedCount || 0} / 未变 ${run.UnchangedCount || 0}
              ${targetSyncRunFailedBranches(run).length ? html`<div class="action-row" style="margin-top:8px;">
                <button class="btn secondary" type="button" disabled=${actionState.busy === `retry-failed:${run.ID}`} onClick=${() => retryFailedBranches(run)}>
                  ${actionState.busy === `retry-failed:${run.ID}` ? "启动中..." : retryFailedBranchesLabel(run)}
                </button>
              </div>` : null}
            </td>
            <td class="small">${run.LastProgressAt || run.FinishedAt || run.StartedAt || "-"}</td>
          </tr>`;
        }) : html`<tr><td colspan="6" class="small">当前还没有抓取入库运行记录。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function routePath(pathname) {
  if ((pathname || "").startsWith("/_/mrtang-admin/backend-release")) return "backend-release";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/categories")) return "source-categories";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/products/detail")) return "source-product-detail";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/product-jobs/detail")) return "source-product-job-detail";
  if ((pathname || "").startsWith("/_/mrtang-admin/source/product-jobs")) return "source-product-jobs";
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
  if (pathname === "/_/mrtang-admin/source/product-jobs") return "source-product-jobs";
  if (pathname === "/_/mrtang-admin/source/product-jobs/detail") return "source-product-job-detail";
  if (pathname === "/_/mrtang-admin/source/asset-jobs") return "source-asset-jobs";
  if (pathname === "/_/mrtang-admin/source/asset-jobs/detail") return "source-asset-job-detail";
  if (pathname === "/_/mrtang-admin/source/assets") return "source-assets";
  if (pathname === "/_/mrtang-admin/source/assets/detail") return "source-asset-detail";
  if (pathname === "/_/mrtang-admin/procurement") return "procurement";
  if (pathname === "/_/mrtang-admin/procurement/detail") return "procurement-detail";
  if (pathname === "/_/mrtang-admin/target-sync") return "target-sync";
  if (pathname === "/_/mrtang-admin/backend-release") return "backend-release";
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

function normalizeCategoryKeyList(values) {
  const items = Array.isArray(values)
    ? values
    : String(values || "")
        .split(",")
        .map((item) => item.trim());
  const seen = new Set();
  return items.filter((item) => {
    if (!item || seen.has(item)) return false;
    seen.add(item);
    return true;
  });
}

function categoryGroupLabel(item) {
  const path = String((item && (item.CategoryPath || item.categoryPath)) || "").trim();
  if (!path) return "未分组";
  return path.split("/").map((part) => part.trim()).filter(Boolean)[0] || "未分组";
}

function groupCategoryItems(items) {
  const groups = new Map();
  (items || []).forEach((item) => {
    const key = categoryGroupLabel(item);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(item);
  });
  return Array.from(groups.entries())
    .map(([label, categories]) => ({
      label,
      categories: categories.sort((left, right) => {
        const leftDepth = Number(left.Depth || 0);
        const rightDepth = Number(right.Depth || 0);
        if (leftDepth !== rightDepth) return leftDepth - rightDepth;
        return String(left.Label || "").localeCompare(String(right.Label || ""), "zh-Hans-CN");
      }),
    }))
    .sort((left, right) => left.label.localeCompare(right.label, "zh-Hans-CN"));
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
          <${MetricCard} eyebrow="商品总数" value=${summary.ProductCount || 0} detail=${`${summary.ImportedCount || 0} 待审核 / ${summary.ApprovedCount || 0} 待加入发布队列`} />
          <${MetricCard} eyebrow="图片总数" value=${summary.AssetCount || 0} detail=${`${summary.AssetPending || 0} 待处理 / ${summary.AssetFailed || 0} 失败`} />
          <${MetricCard} eyebrow="已加入发布队列" value=${summary.LinkedCount || 0} detail=${`${summary.SyncedCount || 0} 已同步 / ${summary.SyncErrorCount || 0} 同步失败`} />
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
          <a class="action-card" href="/_/mrtang-admin/source/product-jobs"><div class="card-kicker">发布任务</div><div class="card-title">商品发布任务</div><div class="card-desc">查看批量发布、重试发布的历史任务、失败项与重跑。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/assets?assetStatus=pending"><div class="card-kicker">图片</div><div class="card-title">待处理图片</div><div class="card-desc">进入图片页执行批量处理。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/asset-jobs"><div class="card-kicker">任务</div><div class="card-title">图片任务历史</div><div class="card-desc">查看原图下载和图片处理的历史任务、失败与重试。</div></a>
          <a class="action-card" href="/_/mrtang-admin/source/logs"><div class="card-kicker">日志</div><div class="card-title">源数据日志</div><div class="card-desc">查看最近审核、加入发布队列和图片处理动作。</div></a>
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
  const currentJob = summary.CurrentJob || null;
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
            <a class="btn secondary" href=${`/_/mrtang-admin/source/products?categoryKeys=${encodeURIComponent(item.SourceKey || "")}`}>查看该分类商品</a>
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
        <span class="pill">商品数按观察到分类链统计</span>
      </div>
    </div></section>

    <section class="section card"><div class="card-body">
      <div class="card-kicker">分类树</div>
      <h2 class="card-title">按层级查看</h2>
      ${rootNodes.length ? renderCategoryTree("", 0) : html`<div class="small">当前筛选下没有分类树节点。</div>`}
    </div></section>

    ${currentJob ? html`
      <section class="section card"><div class="card-body">
        <div class="card-kicker">当前任务</div>
        <h2 class="card-title">${sourceProductJobTypeLabel(currentJob.JobType)}</h2>
        <div class="inline-pills">
          <${StatusBadge} label=${syncStatusLabel(currentJob.Status)} currentTone=${tone(currentJob.Status)} />
          <span class="pill">范围 <code>${sourceProductJobModeLabel(currentJob.Mode)}</code></span>
          <span class="pill">进度 <code>${sourceProductJobSummaryText(currentJob)}</code></span>
        </div>
        <div class="small" style="margin-top:12px;">当前项：${currentJob.CurrentItem || "-"}</div>
        ${sourceProductJobRecentError(currentJob) ? html`<div class="flash error" style="margin-top:12px;">${sourceProductJobRecentError(currentJob)}</div>` : null}
        <div class="action-row" style="margin-top:12px;">
          <a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: currentJob.ID || "", returnTo: window.location.pathname + window.location.search })}>查看任务详情</a>
          ${currentJob.Failed ? html`<a class="btn secondary" href=${sourceProductJobFailedHref(currentJob)}>查看失败商品</a>` : null}
        </div>
      </div></section>
    ` : null}

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">列表</div>
      <h2 class="card-title">已落库分类</h2>
      <div class="table-wrap section"><table><thead><tr><th>分类</th><th>路径</th><th>层级</th><th>商品数</th><th>图片</th></tr></thead><tbody>
        ${items.length ? items.map((item) => html`
          <tr>
            <td><strong>${item.Label || "-"}</strong><div class="small">${item.SourceKey || "-"}</div></td>
            <td class="small">${item.CategoryPath || "-"}</td>
            <td class="small">深度 ${item.Depth || 0}${item.HasChildren ? " / 有子分类" : " / 叶子"}</td>
            <td><span class="pill"><code>${item.ProductCount || 0}</code></span><div class="small"><a href=${`/_/mrtang-admin/source/products?categoryKeys=${encodeURIComponent(item.SourceKey || "")}`}>查看商品</a></div></td>
            <td>${item.ImageURL ? html`<a href=${item.ImageURL} target="_blank" rel="noreferrer">查看</a>` : html`<span class="small">无</span>`}</td>
          </tr>
        `) : html`<tr><td colspan="5" class="small">当前筛选下没有分类。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function SourceProductsPage() {
  const qs = new URLSearchParams(window.location.search);
  const categoryKeys = normalizeCategoryKeyList(qs.get("categoryKeys") || qs.get("categoryKey") || "");
  const productStatus = qs.get("productStatus") || "";
  const syncState = qs.get("syncState") || "";
  const productIds = qs.get("productIds") || "";
  const q = qs.get("q") || "";
  const page = qs.get("productPage") || qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/products", { categoryKeys: categoryKeys.join(","), productStatus, syncState, productIds, q, productPage: page, pageSize });
  const categoryOptionsURL = buildURL("/api/pim/admin/source/categories", { pageSize: 300 });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [selectedIDs, setSelectedIDs] = useState([]);
  const [selectedCategoryKeys, setSelectedCategoryKeys] = useState(categoryKeys);
  const [categoryPickerOpen, setCategoryPickerOpen] = useState(categoryKeys.length > 0);
  const [categorySearch, setCategorySearch] = useState("");
  const resource = useResource(apiURL, [reloadKey]);
  const jobsResource = useResource("/api/pim/admin/source/product-jobs?pageSize=5", [reloadKey]);
  const categoryOptionsResource = useResource(categoryOptionsURL);
  const categoryOptionsSummary = (categoryOptionsResource.data || {}).summary || {};
  const categoryItems = categoryOptionsSummary.Items || [];
  const categoryByKey = useMemo(() => {
    const map = new Map();
    categoryItems.forEach((item) => {
      map.set(item.SourceKey || item.sourceKey || "", item);
    });
    return map;
  }, [categoryItems]);
  const filteredCategoryGroups = useMemo(() => {
    const keyword = categorySearch.trim().toLowerCase();
    const filtered = !keyword
      ? categoryItems
      : categoryItems.filter((item) => {
          const search = [
            item.Label || "",
            item.SourceKey || "",
            item.CategoryPath || "",
          ]
            .join(" ")
            .toLowerCase();
          return search.includes(keyword);
        });
    return groupCategoryItems(filtered);
  }, [categoryItems, categorySearch]);
  if (resource.loading) return html`<${LoadingSection} label="源数据商品" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const products = summary.Products || [];
  const currentJob = (((jobsResource.data || {}).summary || {}).CurrentJob) || null;
  const selectedCategoryItems = selectedCategoryKeys
    .map((key) => categoryByKey.get(key))
    .filter(Boolean);
  const currentProductPage = Math.max(1, Number(summary.ProductPage || filter.ProductPage || 1));
  const productPages = Math.max(1, Number(summary.ProductPages || 1));
  const visibleIDs = products.map((item) => item.ID || "").filter(Boolean);
  const selectedVisibleIDs = normalizeIDList(selectedIDs).filter((id) => visibleIDs.includes(id));
  const allVisibleSelected = products.length > 0 && selectedVisibleIDs.length === visibleIDs.length;
  const filteredActionPayload = {
    categoryKeys: filter.CategoryKeys || filter.CategoryKey || "",
    productStatus: filter.ProductStatus || "",
    syncState: filter.SyncState || "",
    productIds: filter.ProductIDs || "",
    q: filter.Query || "",
  };

  useEffect(() => {
    setSelectedIDs((current) => normalizeIDList(current).filter((id) => visibleIDs.includes(id)));
  }, [visibleIDs.join(",")]);

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

  async function batchProductAction(url, values, confirmMessage, busyKey, successMessage) {
    if (!selectedVisibleIDs.length) return;
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, { ...values, productIds: selectedVisibleIDs.join(",") });
      const job = result.job || {};
      if (job.ID) {
        window.location.href = buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: job.ID, returnTo: window.location.pathname + window.location.search });
        return;
      }
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
      setSelectedIDs([]);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "批量操作失败" });
    }
  }

  async function filteredProductAction(url, values, confirmMessage, busyKey, successMessage) {
    if (!(summary.ProductCount || 0)) return;
    if (confirmMessage && !window.confirm(confirmMessage)) return;
    setActionState({ busy: busyKey, message: "", error: "" });
    try {
      const result = await postForm(url, { ...filteredActionPayload, ...values });
      const job = result.job || {};
      if (job.ID) {
        window.location.href = buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: job.ID, returnTo: window.location.pathname + window.location.search });
        return;
      }
      setActionState({ busy: "", message: result.message || successMessage, error: "" });
      setReloadKey((value) => value + 1);
      setSelectedIDs([]);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || successMessage || "批量操作失败" });
    }
  }

  function toggleSelected(id) {
    setSelectedIDs((current) => {
      const normalized = normalizeIDList(current);
      if (normalized.includes(id)) {
        return normalized.filter((item) => item !== id);
      }
      return [...normalized, id];
    });
  }

  function toggleSelectAll() {
    if (allVisibleSelected) {
      setSelectedIDs((current) => normalizeIDList(current).filter((id) => !visibleIDs.includes(id)));
      return;
    }
    setSelectedIDs((current) => normalizeIDList([...current, ...visibleIDs]));
  }

  function toggleCategoryKey(key) {
    setSelectedCategoryKeys((current) => {
      if (current.includes(key)) return current.filter((item) => item !== key);
      return [...current, key];
    });
  }

  const selectedCategorySummary = selectedCategoryItems.length
    ? `${selectedCategoryItems.length} 个分类`
    : selectedCategoryKeys.length
      ? `${selectedCategoryKeys.length} 个分类`
      : "全部分类";

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">商品列表</h2>
      ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
      ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
      <${ActionNotice} state=${actionState} />
      <form class="section" method="get" action="/_/mrtang-admin/source/products">
        <div class="filter-grid">
          <label class="control-field">
            <span class="control-label">审核状态</span>
            <select class="control-select" name="productStatus" defaultValue=${filter.ProductStatus || ""}>
              <option value="">全部审核状态</option>
              <option value="imported">待审核</option>
              <option value="approved">待加入发布队列</option>
              <option value="promoted">已加入发布队列</option>
              <option value="rejected">已拒绝</option>
            </select>
          </label>
          <label class="control-field">
            <span class="control-label">发布状态</span>
            <select class="control-select" name="syncState" defaultValue=${filter.SyncState || ""}>
              <option value="">全部同步状态</option>
              <option value="unlinked">未进入发布队列</option>
              <option value="error">同步失败</option>
              <option value="synced">已同步</option>
            </select>
          </label>
          <label class="control-field control-field--wide">
            <span class="control-label">搜索</span>
            <input class="control-input" type="text" name="q" placeholder="搜索商品名 / productId" defaultValue=${filter.Query || ""} />
          </label>
          <label class="control-field">
            <span class="control-label">每页数量</span>
            <select class="control-select" name="pageSize" defaultValue=${String(filter.PageSize || 24)}>
              <option value="12">12</option>
              <option value="24">24</option>
              <option value="48">48</option>
            </select>
          </label>
        </div>

        <div class="category-picker section">
          <div class="control-label">分类筛选</div>
          ${selectedCategoryKeys.length ? html`<input type="hidden" name="categoryKeys" value=${selectedCategoryKeys.join(",")} />` : null}
          ${filter.ProductIDs ? html`<input type="hidden" name="productIds" value=${filter.ProductIDs || ""} />` : null}
          <button class="category-picker-trigger" type="button" onClick=${() => setCategoryPickerOpen((current) => !current)}>
            <div>
              <div class="category-picker-title">${selectedCategorySummary}</div>
              <div class="small">${selectedCategoryItems.length ? selectedCategoryItems.map((item) => item.Label || item.SourceKey).slice(0, 3).join("、") : "未限制分类，将显示全部商品。"}</div>
            </div>
            <span class="pill">${categoryPickerOpen ? "收起" : "展开选择"}</span>
          </button>
          ${selectedCategoryItems.length ? html`
            <div class="selected-chip-group">
              ${selectedCategoryItems.map((item) => html`
                <button class="selected-chip" type="button" onClick=${() => toggleCategoryKey(item.SourceKey || "")}>
                  <span>${item.Label || item.SourceKey || "-"}</span>
                  <span>×</span>
                </button>
              `)}
            </div>
          ` : null}
          ${categoryPickerOpen ? html`
            <div class="category-picker-panel">
              <div class="category-picker-toolbar">
                <input
                  class="control-input"
                  type="search"
                  value=${categorySearch}
                  placeholder="搜索分类名 / 路径 / sourceKey"
                  onInput=${(event) => setCategorySearch(event.currentTarget.value)}
                />
                <button class="btn secondary" type="button" onClick=${() => setSelectedCategoryKeys([])}>清空分类</button>
              </div>
              ${categoryOptionsResource.loading ? html`<div class="small">分类选项加载中...</div>` : null}
              ${categoryOptionsResource.error ? html`<div class="flash error">${categoryOptionsResource.error}</div>` : null}
              ${!categoryOptionsResource.loading && !categoryOptionsResource.error ? html`
                <div class="category-group-list">
                  ${filteredCategoryGroups.length ? filteredCategoryGroups.map((group) => html`
                    <section class="category-group">
                      <div class="category-group-title">${group.label}</div>
                      <div class="category-option-list">
                        ${group.categories.map((item) => {
                          const categoryKey = item.SourceKey || "";
                          const active = selectedCategoryKeys.includes(categoryKey);
                          return html`
                            <button
                              class=${`category-option${active ? " active" : ""}`}
                              type="button"
                              onClick=${() => toggleCategoryKey(categoryKey)}
                            >
                              <div class="category-option-title">${item.Label || "-"}</div>
                              <div class="category-option-meta">
                                <span>深度 ${item.Depth || 0}</span>
                                <span>${item.ProductCount || 0} 个商品</span>
                              </div>
                              <div class="small">${item.CategoryPath || categoryKey}</div>
                            </button>
                          `;
                        })}
                      </div>
                    </section>
                  `) : html`<div class="small">没有匹配的分类。</div>`}
                </div>
              ` : null}
            </div>
          ` : null}
        </div>

        <div class="action-row">
          <button class="btn secondary" type="submit">应用筛选</button>
          <a class="btn secondary" href="/_/mrtang-admin/source/products">重置</a>
        </div>
      </form>
      <div class="inline-pills">
        ${selectedCategoryItems.length ? selectedCategoryItems.map((item) => html`<span class="pill">分类 <code>${item.Label || item.SourceKey}</code></span>`) : null}
        ${filter.ProductIDs ? html`<span class="pill">任务商品范围 <code>${normalizeIDList((filter.ProductIDs || "").split(",")).length}</code></span>` : null}
        <span class="pill">总数 <code>${summary.ProductCount || 0}</code></span>
        <span class="pill">待审核 <code>${summary.ImportedCount || 0}</code></span>
        <span class="pill">待加入发布队列 <code>${summary.ApprovedCount || 0}</code></span>
        <span class="pill">同步失败 <code>${summary.SyncErrorCount || 0}</code></span>
        <span class="pill">当前筛选 <code>${summary.FilteredProductCount || summary.ProductCount || 0}</code></span>
        ${selectedVisibleIDs.length ? html`<span class="pill">当前选中 <code>${selectedVisibleIDs.length}</code></span>` : null}
      </div>
    </div></section>

    ${currentJob ? html`
      <section class="section card"><div class="card-body">
        <div class="card-kicker">当前任务</div>
        <h2 class="card-title">${sourceProductJobTypeLabel(currentJob.JobType)}</h2>
        <div class="inline-pills">
          <${StatusBadge} label=${syncStatusLabel(currentJob.Status)} currentTone=${tone(currentJob.Status)} />
          <span class="pill">范围 <code>${sourceProductJobModeLabel(currentJob.Mode)}</code></span>
          <span class="pill">进度 <code>${sourceProductJobSummaryText(currentJob)}</code></span>
        </div>
        <div class="small" style="margin-top:12px;">当前项：${currentJob.CurrentItem || "-"}</div>
        ${sourceProductJobRecentError(currentJob) ? html`<div class="flash error" style="margin-top:12px;">${sourceProductJobRecentError(currentJob)}</div>` : null}
        <div class="action-row" style="margin-top:12px;">
          <a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: currentJob.ID || "", returnTo: window.location.pathname + window.location.search })}>查看任务详情</a>
          ${currentJob.Failed ? html`<a class="btn secondary" href=${sourceProductJobFailedHref(currentJob)}>查看本任务失败商品</a>` : null}
        </div>
      </div></section>
    ` : null}

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">列表</div>
      <h2 class="card-title">商品批次</h2>
      <div class="action-row" style="margin-bottom:12px;">
        <button class="btn secondary" type="button" disabled=${!(summary.FilteredProductCount || 0) || actionState.busy === "filtered-approve"} onClick=${() => filteredProductAction("/api/pim/admin/source/products/batch-status-filtered", { status: "approved" }, `确认将当前筛选结果 ${summary.FilteredProductCount || 0} 个商品标记为通过吗？`, "filtered-approve", "按当前筛选结果批量通过已完成。")}>${actionState.busy === "filtered-approve" ? "处理中..." : "按当前筛选结果批量通过"}</button>
        <button class="btn secondary" type="button" disabled=${!(summary.FilteredProductCount || 0) || actionState.busy === "filtered-reject"} onClick=${() => filteredProductAction("/api/pim/admin/source/products/batch-status-filtered", { status: "rejected" }, `确认拒绝当前筛选结果 ${summary.FilteredProductCount || 0} 个商品吗？`, "filtered-reject", "按当前筛选结果批量拒绝已完成。")}>${actionState.busy === "filtered-reject" ? "处理中..." : "按当前筛选结果批量拒绝"}</button>
        <button class="btn secondary" type="button" disabled=${!(summary.FilteredProductCount || 0) || actionState.busy === "filtered-promote"} onClick=${() => filteredProductAction("/api/pim/admin/source/products/batch-promote-filtered", {}, `确认将当前筛选结果 ${summary.FilteredProductCount || 0} 个商品加入发布队列吗？`, "filtered-promote", "按当前筛选结果批量加入发布队列已完成。")}>${actionState.busy === "filtered-promote" ? "处理中..." : "按当前筛选结果加入发布队列"}</button>
        <button class="btn secondary" type="button" disabled=${!(summary.FilteredProductCount || 0) || actionState.busy === "filtered-promote-sync"} onClick=${() => filteredProductAction("/api/pim/admin/source/products/batch-promote-sync-filtered", {}, `确认将当前筛选结果 ${summary.FilteredProductCount || 0} 个商品加入发布队列并发布到 Backend 吗？`, "filtered-promote-sync", "按当前筛选结果批量发布已完成。")}>${actionState.busy === "filtered-promote-sync" ? "处理中..." : "按当前筛选结果加入发布队列并发布"}</button>
        <button class="btn secondary" type="button" disabled=${!(summary.FilteredProductCount || 0) || actionState.busy === "filtered-retry-sync"} onClick=${() => filteredProductAction("/api/pim/admin/source/products/batch-retry-sync-filtered", {}, `确认重试当前筛选结果 ${summary.FilteredProductCount || 0} 个商品发布到 Backend 吗？`, "filtered-retry-sync", "按当前筛选结果批量重试发布已完成。")}>${actionState.busy === "filtered-retry-sync" ? "处理中..." : "按当前筛选结果重试发布"}</button>
      </div>
      <div class="action-row" style="margin-bottom:12px;">
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "batch-approve"} onClick=${() => batchProductAction("/api/pim/admin/source/products/batch-status", { status: "approved" }, `确认将选中的 ${selectedVisibleIDs.length} 个商品标记为通过吗？`, "batch-approve", "批量更新商品审核状态已完成。")}>${actionState.busy === "batch-approve" ? "处理中..." : "选中项批量通过"}</button>
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "batch-reject"} onClick=${() => batchProductAction("/api/pim/admin/source/products/batch-status", { status: "rejected" }, `确认拒绝选中的 ${selectedVisibleIDs.length} 个商品吗？`, "batch-reject", "批量更新商品审核状态已完成。")}>${actionState.busy === "batch-reject" ? "处理中..." : "选中项批量拒绝"}</button>
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "batch-promote"} onClick=${() => batchProductAction("/api/pim/admin/source/products/batch-promote", {}, `确认将选中的 ${selectedVisibleIDs.length} 个商品加入发布队列吗？`, "batch-promote", "批量加入发布队列已完成。")}>${actionState.busy === "batch-promote" ? "处理中..." : "选中项批量加入发布队列"}</button>
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "batch-promote-sync"} onClick=${() => batchProductAction("/api/pim/admin/source/products/batch-promote-sync", {}, `确认将选中的 ${selectedVisibleIDs.length} 个商品加入发布队列并发布到 Backend 吗？`, "batch-promote-sync", "批量加入发布队列并发布已完成。")}>${actionState.busy === "batch-promote-sync" ? "处理中..." : "选中项批量加入发布队列并发布"}</button>
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "batch-retry-sync"} onClick=${() => batchProductAction("/api/pim/admin/source/products/batch-retry-sync", {}, `确认重试选中的 ${selectedVisibleIDs.length} 个商品发布到 Backend 吗？`, "batch-retry-sync", "批量重试发布已完成。")}>${actionState.busy === "batch-retry-sync" ? "处理中..." : "选中项批量重试发布"}</button>
        ${selectedVisibleIDs.length ? html`<button class="btn secondary" type="button" onClick=${() => setSelectedIDs([])}>清空选择</button>` : null}
      </div>
      <div class="table-wrap section"><table><thead><tr><th><input type="checkbox" checked=${allVisibleSelected} onChange=${toggleSelectAll} /></th><th>商品</th><th>发布分类 / 观察到分类</th><th>审核</th><th>发布队列</th><th>动作</th></tr></thead><tbody>
        ${products.length ? products.map((item) => html`
          <tr>
            <td><input type="checkbox" checked=${selectedVisibleIDs.includes(item.ID || "")} onChange=${() => toggleSelected(item.ID || "")} /></td>
            <td><strong>${item.Name || "-"}</strong><div class="small">${item.ProductID || "-"}</div></td>
            <td class="small">
              <div><strong>发布分类：</strong>${item.CategoryPath || "-"}</div>
              <div style="margin-top:4px;"><strong>观察到分类：</strong>${(item.ObservedCategoryPaths && item.ObservedCategoryPaths.length) ? item.ObservedCategoryPaths.join("；") : "-"}</div>
              <div style="margin-top:4px;">${item.UnitCount || 0} 个单位 / ${item.HasMultiUnit ? "多单位" : "单单位"}</div>
            </td>
            <td><${StatusBadge} label=${item.ReviewStatus || "-"} currentTone=${tone(item.ReviewStatus)} /></td>
            <td><${StatusBadge} label=${(item.Bridge && item.Bridge.SyncStatus) || (item.Bridge && item.Bridge.Linked ? "linked" : "unlinked")} currentTone=${tone((item.Bridge && item.Bridge.SyncStatus) || (item.Bridge && item.Bridge.Linked ? "warning" : "error"))} /></td>
            <td>
              <div class="action-row">
                <a class="btn secondary" href=${`/_/mrtang-admin/source/products/detail?id=${encodeURIComponent(item.ID || "")}&returnTo=${encodeURIComponent(window.location.pathname + window.location.search)}`}>详情</a>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `approve:${item.ID || ""}`} onClick=${() => productAction("/api/pim/admin/source/products/status", { id: item.ID || "", status: "approved" }, "确认将这个商品标记为通过吗？", `approve:${item.ID || ""}`, "商品审核状态已更新。")}>${actionState.busy === `approve:${item.ID || ""}` ? "处理中..." : "通过"}</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `promote:${item.ID || ""}`} onClick=${() => productAction("/api/pim/admin/source/products/promote", { id: item.ID || "" }, "确认将这个商品加入发布队列吗？", `promote:${item.ID || ""}`, "商品已加入发布队列。")}>${actionState.busy === `promote:${item.ID || ""}` ? "处理中..." : "加入发布队列"}</button>
                ${(item.Bridge && (item.Bridge.SyncStatus || "").toLowerCase() === "error") ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === `retry:${item.ID || ""}`} onClick=${() => productAction("/api/pim/admin/source/products/retry-sync", { id: item.ID || "" }, "确认重试这个商品发布到 Backend 吗？", `retry:${item.ID || ""}`, "已触发商品发布重试。")}>${actionState.busy === `retry:${item.ID || ""}` ? "处理中..." : "重试发布"}</button>` : null}
              </div>
            </td>
          </tr>
        `) : html`<tr><td colspan="6" class="small">当前筛选下没有商品。</td></tr>`}
      </tbody></table></div>
      <div class="action-row" style="align-items:center; justify-content:space-between;">
        <div class="small">第 ${currentProductPage} / ${productPages} 页，共 ${summary.FilteredProductCount || summary.ProductCount || 0} 条当前筛选结果。跨页全选请使用“按当前筛选结果...”按钮。</div>
        <${Pagination}
          basePath="/_/mrtang-admin/source/products"
          pageParam="productPage"
          currentPage=${currentProductPage}
          totalPages=${productPages}
          params=${{
            productStatus: filter.ProductStatus || "",
            syncState: filter.SyncState || "",
            categoryKeys: filter.CategoryKeys || filter.CategoryKey || "",
            productIds: filter.ProductIDs || "",
            q: filter.Query || "",
            pageSize: filter.PageSize || 24,
          }}
        />
      </div>
    </div></div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">最近任务</div>
      <h2 class="card-title">商品发布与重试</h2>
      ${jobsResource.loading ? html`<div class="small">最近任务加载中...</div>` : null}
      ${jobsResource.error ? html`<div class="flash error">${jobsResource.error}</div>` : null}
      ${!jobsResource.loading && !jobsResource.error ? html`<div class="table-wrap section"><table><thead><tr><th>任务</th><th>状态</th><th>进度</th><th>错误摘要</th><th>动作</th></tr></thead><tbody>
        ${(((jobsResource.data || {}).summary || {}).Items || []).length ? (((jobsResource.data || {}).summary || {}).Items || []).map((item) => html`
          <tr>
            <td><strong>${sourceProductJobTypeLabel(item.JobType)}</strong><div class="small">${sourceProductJobModeLabel(item.Mode)} / ${item.StartedAt || item.Created || "-"}</div></td>
            <td><${StatusBadge} label=${syncStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td>
            <td class="small">${sourceProductJobSummaryText(item)}</td>
            <td class="small">${sourceProductJobRecentError(item) || "-"}</td>
            <td><div class="action-row"><a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: item.ID || "", returnTo: window.location.pathname + window.location.search })}>详情</a>${item.Failed ? html`<a class="btn secondary" href=${sourceProductJobFailedHref(item)}>查看失败商品</a>` : null}</div></td>
          </tr>
        `) : html`<tr><td colspan="5" class="small">还没有商品发布任务记录。</td></tr>`}
      </tbody></table></div>` : null}
    </div></div></section>
  `;
}

function SourceAssetsPage() {
  const qs = new URLSearchParams(window.location.search);
  const assetStatus = qs.get("assetStatus") || "";
  const originalStatus = qs.get("originalStatus") || "";
  const assetIds = qs.get("assetIds") || "";
  const q = qs.get("q") || "";
  const page = qs.get("assetPage") || qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/assets", { assetStatus, originalStatus, assetIds, q, assetPage: page, pageSize });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [selectedIDs, setSelectedIDs] = useState(normalizeIDList(assetIds));
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
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const assets = summary.Assets || [];
  const currentAssetPage = Number(summary.AssetPage || filter.AssetPage || page || 1) || 1;
  const assetPages = Number(summary.AssetPages || 1) || 1;
  const filterAssetIDs = normalizeIDList(filter.AssetIDs || assetIds);
  const visibleIDs = assets.map((item) => item.ID || "").filter(Boolean);
  const selectedVisibleIDs = normalizeIDList(selectedIDs).filter((id) => visibleIDs.includes(id));
  const allVisibleSelected = assets.length > 0 && selectedVisibleIDs.length === visibleIDs.length;
  const activeProcessMode = ((activeProcess && activeProcess.Mode) || "").toLowerCase();
  const filteredActionPayload = {
    assetStatus: filter.AssetStatus || "",
    originalStatus: filter.OriginalStatus || "",
    assetIds: filter.AssetIDs || "",
    q: filter.Query || "",
  };

  useEffect(() => {
    setSelectedIDs((current) => {
      const normalizedCurrent = normalizeIDList(current);
      const retained = normalizedCurrent.filter((id) => visibleIDs.includes(id));
      if (retained.length) return retained;
      if (filterAssetIDs.length) return filterAssetIDs.filter((id) => visibleIDs.includes(id));
      return retained;
    });
  }, [visibleIDs.join(","), filter.AssetIDs || ""]);
  if (resource.loading) return html`<${LoadingSection} label="源数据图片" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;

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
      const progress = error && error.payload && error.payload.progress;
      if (progress && progress.ID) {
        setActiveDownload(progress);
        setActiveDownloadId(progress.ID);
        setActionState({ busy: "", message: "已有原图下载任务执行中，已切换到当前任务进度。", error: "" });
        return;
      }
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
      const progress = error && error.payload && error.payload.progress;
      if (progress && progress.ID) {
        setActiveProcess(progress);
        setActiveProcessId(progress.ID);
        setActionState({ busy: "", message: "已有图片处理任务执行中，已切换到当前任务进度。", error: "" });
        return;
      }
      setActionState({ busy: "", message: "", error: error.message || defaultMessage || "批量处理图片失败" });
    }
  }

  async function startSelectedDownload() {
    if (!selectedVisibleIDs.length) return;
    if (!window.confirm(`确认下载选中的 ${selectedVisibleIDs.length} 张图片原图吗？`)) return;
    setActionState({ busy: "download-selected", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/download-selected", { assetIds: selectedVisibleIDs.join(",") });
      const progress = result.progress || {};
      setActiveDownload(progress);
      setActiveDownloadId(progress.ID || "");
      setActiveDownloadError("");
      setActionState({ busy: "", message: result.message || "选中图片原图下载任务已启动。", error: "" });
    } catch (error) {
      const progress = error && error.payload && error.payload.progress;
      if (progress && progress.ID) {
        setActiveDownload(progress);
        setActiveDownloadId(progress.ID);
        setActionState({ busy: "", message: "已有原图下载任务执行中，已切换到当前任务进度。", error: "" });
        return;
      }
      setActionState({ busy: "", message: "", error: error.message || "启动选中图片原图下载失败" });
    }
  }

  async function startFilteredDownload() {
    if (!summary.FilteredAssetCount) return;
    if (!window.confirm(`确认下载当前筛选结果 ${summary.FilteredAssetCount} 张图片的原图吗？`)) return;
    setActionState({ busy: "download-filtered", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/download-filtered", filteredActionPayload);
      const progress = result.progress || {};
      setActiveDownload(progress);
      setActiveDownloadId(progress.ID || "");
      setActiveDownloadError("");
      setActionState({ busy: "", message: result.message || "当前筛选结果原图下载任务已启动。", error: "" });
    } catch (error) {
      const progress = error && error.payload && error.payload.progress;
      if (progress && progress.ID) {
        setActiveDownload(progress);
        setActiveDownloadId(progress.ID);
        setActionState({ busy: "", message: "已有原图下载任务执行中，已切换到当前任务进度。", error: "" });
        return;
      }
      setActionState({ busy: "", message: "", error: error.message || "启动当前筛选结果原图下载失败" });
    }
  }

  async function startSelectedProcess(failedOnly) {
    if (!selectedVisibleIDs.length) return;
    const title = failedOnly ? "重处理选中失败图片" : "处理选中图片";
    if (!window.confirm(`确认${title}（${selectedVisibleIDs.length} 张）吗？`)) return;
    setActionState({ busy: failedOnly ? "process-selected-failed" : "process-selected", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/process-selected", {
        assetIds: selectedVisibleIDs.join(","),
        failedOnly: failedOnly ? "true" : "false",
      });
      const progress = result.progress || {};
      setActiveProcess(progress);
      setActiveProcessId(progress.ID || "");
      setActiveProcessError("");
      setActionState({ busy: "", message: result.message || `${title}任务已启动。`, error: "" });
    } catch (error) {
      const progress = error && error.payload && error.payload.progress;
      if (progress && progress.ID) {
        setActiveProcess(progress);
        setActiveProcessId(progress.ID);
        setActionState({ busy: "", message: "已有图片处理任务执行中，已切换到当前任务进度。", error: "" });
        return;
      }
      setActionState({ busy: "", message: "", error: error.message || `${title}失败` });
    }
  }

  async function startFilteredProcess(failedOnly) {
    if (!summary.FilteredAssetCount) return;
    const title = failedOnly ? "重处理当前筛选结果失败图片" : "处理当前筛选结果图片";
    if (!window.confirm(`确认${title}（${summary.FilteredAssetCount} 张）吗？`)) return;
    setActionState({ busy: failedOnly ? "process-filtered-failed" : "process-filtered", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/assets/process-filtered", {
        ...filteredActionPayload,
        failedOnly: failedOnly ? "true" : "false",
      });
      const progress = result.progress || {};
      setActiveProcess(progress);
      setActiveProcessId(progress.ID || "");
      setActiveProcessError("");
      setActionState({ busy: "", message: result.message || `${title}任务已启动。`, error: "" });
    } catch (error) {
      const progress = error && error.payload && error.payload.progress;
      if (progress && progress.ID) {
        setActiveProcess(progress);
        setActiveProcessId(progress.ID);
        setActionState({ busy: "", message: "已有图片处理任务执行中，已切换到当前任务进度。", error: "" });
        return;
      }
      setActionState({ busy: "", message: "", error: error.message || `${title}失败` });
    }
  }

  function toggleSelected(id) {
    setSelectedIDs((current) => {
      const normalized = normalizeIDList(current);
      if (normalized.includes(id)) {
        return normalized.filter((item) => item !== id);
      }
      return [...normalized, id];
    });
  }

  function toggleSelectAll() {
    if (allVisibleSelected) {
      setSelectedIDs((current) => normalizeIDList(current).filter((id) => !visibleIDs.includes(id)));
      return;
    }
    setSelectedIDs((current) => normalizeIDList([...current, ...visibleIDs]));
  }

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">图片列表</h2>
      ${payload.flashError ? html`<div class="flash error" style="margin-top:14px;">${payload.flashError}</div>` : null}
      ${payload.flashMessage ? html`<div class="flash ok" style="margin-top:14px;">${payload.flashMessage}</div>` : null}
      <${ActionNotice} state=${actionState} />
      <form class="action-row" method="get" action="/_/mrtang-admin/source/assets">
        ${filter.AssetIDs ? html`<input type="hidden" name="assetIds" value=${filter.AssetIDs} />` : null}
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
        <span class="pill">当前筛选 <code>${summary.FilteredAssetCount || 0}</code></span>
        ${filterAssetIDs.length ? html`<span class="pill">任务图片 <code>${filterAssetIDs.length}</code></span>` : null}
        ${selectedVisibleIDs.length ? html`<span class="pill">当前选中 <code>${selectedVisibleIDs.length}</code></span>` : null}
        <a class="pill" href=${buildURL("/_/mrtang-admin/source/assets", { originalStatus: "failed" })}>原图失败</a>
        <a class="pill" href=${buildURL("/_/mrtang-admin/source/assets", { assetStatus: "failed" })}>处理失败</a>
        <a class="pill" href="/_/mrtang-admin/source/asset-jobs">查看任务历史</a>
        ${filterAssetIDs.length ? html`<a class="pill" href="/_/mrtang-admin/source/assets">退出任务图片视图</a>` : null}
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
        <div>图片批量处理：${activeProcessMode.includes("failed") ? "失败图片重处理" : "待处理图片"} / ${(activeProcess.Status || "").toLowerCase() === "running" ? "执行中" : "已完成"}</div>
        <div class="small" style="margin-top:8px;">已处理 ${activeProcess.Processed || 0} / ${activeProcess.Total || 0}，失败 ${activeProcess.Failed || 0}${activeProcess.CurrentItem ? `，当前项：${activeProcess.CurrentItem}` : ""}</div>
        ${(activeProcess.Logs || []).length ? html`<div class="small" style="margin-top:8px;">${(activeProcess.Logs || []).slice(-5).map((item) => `${item.Time || "-"} ${item.Message || "-"}`).join(" / ")}</div>` : null}
        ${activeProcess.ID ? html`<div class="small" style="margin-top:8px;"><a href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: activeProcess.ID, returnTo: window.location.pathname + window.location.search })}>查看任务详情</a></div>` : null}
      </div>` : null}
      ${activeProcessError ? html`<div class="flash error" style="margin-bottom:12px;">${activeProcessError}</div>` : null}
      <div class="action-row" style="margin-bottom:12px;">
        <button class="btn secondary" type="button" disabled=${actionState.busy === "download-pending" || !!activeDownloadId} onClick=${startDownloadPending}>${actionState.busy === "download-pending" || !!activeDownloadId ? "下载中..." : "批量下载待下载原图"}</button>
        <button class="btn secondary" type="button" disabled=${actionState.busy === "process-pending" || !!activeProcessId} onClick=${() => startProcessBatch("/api/pim/admin/source/assets/process-pending", "确认批量处理待处理图片吗？", "process-pending", "图片批量处理任务已启动。")}>${actionState.busy === "process-pending" || (!!activeProcessId && !activeProcessMode.includes("failed")) ? "处理中..." : "批量处理待处理图片"}</button>
        <button class="btn secondary" type="button" disabled=${actionState.busy === "reprocess-failed" || !!activeProcessId} onClick=${() => startProcessBatch("/api/pim/admin/source/assets/reprocess-failed", "确认批量重处理失败图片吗？", "reprocess-failed", "失败图片重处理任务已启动。")}>${actionState.busy === "reprocess-failed" || (!!activeProcessId && activeProcessMode.includes("failed")) ? "处理中..." : "批量重处理失败图片"}</button>
      </div>
      <div class="action-row" style="margin-bottom:12px;">
        <button class="btn secondary" type="button" disabled=${!summary.FilteredAssetCount || actionState.busy === "download-filtered" || !!activeDownloadId} onClick=${startFilteredDownload}>${actionState.busy === "download-filtered" ? "处理中..." : "按当前筛选结果下载原图"}</button>
        <button class="btn secondary" type="button" disabled=${!summary.FilteredAssetCount || actionState.busy === "process-filtered" || !!activeProcessId} onClick=${() => startFilteredProcess(false)}>${actionState.busy === "process-filtered" ? "处理中..." : "按当前筛选结果处理"}</button>
        <button class="btn secondary" type="button" disabled=${!summary.FilteredAssetCount || actionState.busy === "process-filtered-failed" || !!activeProcessId} onClick=${() => startFilteredProcess(true)}>${actionState.busy === "process-filtered-failed" ? "处理中..." : "按当前筛选结果重处理失败项"}</button>
      </div>
      <div class="action-row" style="margin-bottom:12px;">
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "download-selected" || !!activeDownloadId} onClick=${startSelectedDownload}>${actionState.busy === "download-selected" ? "处理中..." : "仅对选中图片下载原图"}</button>
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "process-selected" || !!activeProcessId} onClick=${() => startSelectedProcess(false)}>${actionState.busy === "process-selected" ? "处理中..." : "仅对选中图片处理"}</button>
        <button class="btn secondary" type="button" disabled=${!selectedVisibleIDs.length || actionState.busy === "process-selected-failed" || !!activeProcessId} onClick=${() => startSelectedProcess(true)}>${actionState.busy === "process-selected-failed" ? "处理中..." : "仅对选中失败图片重处理"}</button>
        ${selectedVisibleIDs.length ? html`<button class="btn secondary" type="button" onClick=${() => setSelectedIDs([])}>清空选择</button>` : null}
      </div>
      <div class="table-wrap section"><table><thead><tr><th><input type="checkbox" checked=${allVisibleSelected} onChange=${toggleSelectAll} /></th><th>图片</th><th>商品</th><th>原图</th><th>处理</th><th>错误</th><th>动作</th></tr></thead><tbody>
        ${assets.length ? assets.map((item) => html`
          <tr>
            <td><input type="checkbox" checked=${selectedVisibleIDs.includes(item.ID || "")} onChange=${() => toggleSelected(item.ID || "")} /></td>
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
                <button class="btn secondary" type="button" disabled=${actionState.busy === `download:${item.ID || ""}` || !item.CanDownloadOriginal} title=${item.CanDownloadOriginal ? "" : "该图片资产没有可用源图地址"} onClick=${() => assetAction("/api/pim/admin/source/assets/download", { id: item.ID || "" }, "确认下载这张图片的原图吗？", `download:${item.ID || ""}`, "已下载原图。")}>${actionState.busy === `download:${item.ID || ""}` ? "下载中..." : (item.CanDownloadOriginal ? "下载原图" : "不可下载")}</button>
                <button class="btn secondary" type="button" disabled=${actionState.busy === `asset:${item.ID || ""}` || (!item.CanDownloadOriginal && !item.OriginalImageURL)} title=${(!item.CanDownloadOriginal && !item.OriginalImageURL) ? "该图片资产没有可用源图地址或原图文件" : ""} onClick=${() => assetAction("/api/pim/admin/source/assets/process", { id: item.ID || "" }, "确认处理这张图片吗？", `asset:${item.ID || ""}`, "图片已进入处理流程。")}>${actionState.busy === `asset:${item.ID || ""}` ? "处理中..." : ((!item.CanDownloadOriginal && !item.OriginalImageURL) ? "不可处理" : "处理")}</button>
              </div>
            </td>
          </tr>
        `) : html`<tr><td colspan="7" class="small">当前筛选下没有图片。</td></tr>`}
      </tbody></table></div>
      <div class="action-row" style="align-items:center; justify-content:space-between;">
        <div class="small">第 ${currentAssetPage} / ${assetPages} 页，共 ${summary.FilteredAssetCount || 0} 条当前筛选结果。</div>
        <${Pagination}
          basePath="/_/mrtang-admin/source/assets"
          pageParam="assetPage"
          currentPage=${currentAssetPage}
          totalPages=${assetPages}
          params=${{
            assetStatus: filter.AssetStatus || "",
            originalStatus: filter.OriginalStatus || "",
            assetIds: filter.AssetIDs || "",
            q: filter.Query || "",
            pageSize: filter.PageSize || 24,
          }}
        />
      </div>
    </div></div></section>

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">最近任务</div>
      <h2 class="card-title">原图下载与图片处理</h2>
      ${jobsResource.loading ? html`<div class="small">最近任务加载中...</div>` : null}
      ${jobsResource.error ? html`<div class="flash error">${jobsResource.error}</div>` : null}
      ${!jobsResource.loading && !jobsResource.error ? html`<div class="table-wrap section"><table><thead><tr><th>任务</th><th>状态</th><th>进度</th><th>错误摘要</th><th>动作</th></tr></thead><tbody>
        ${(((jobsResource.data || {}).summary || {}).Items || []).length ? (((jobsResource.data || {}).summary || {}).Items || []).map((item) => html`
          <tr>
            <td><strong>${sourceAssetJobTypeLabel(item.JobType, item.Mode)}</strong><div class="small">${sourceAssetJobModeLabel(item.Mode)} / ${item.StartedAt || item.Created || "-"}</div></td>
            <td><${StatusBadge} label=${syncStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td>
            <td class="small">成功 ${item.Processed || 0} / 总数 ${item.Total || 0}<br />失败 ${item.Failed || 0} / 成功率 ${sourceAssetJobSuccessRate(item)}${sourceAssetJobSelectionCount(item) ? html`<br />范围 ${sourceAssetJobSelectionCount(item)} 张` : null}</td>
            <td class="small">${sourceAssetJobRecentError(item) || "-"}</td>
            <td><div class="action-row"><a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: item.ID || "", returnTo: window.location.pathname + window.location.search })}>详情</a><a class="btn secondary" href=${sourceAssetJobTargetHref(item)}>${sourceAssetJobTargetLabel(item)}</a>${item.Processed ? html`<a class="btn secondary" href=${sourceAssetJobSuccessHref(item)}>${sourceAssetJobSuccessLabel(item)}</a>` : null}</div></td>
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
              <div class="small">范围：${sourceAssetJobModeLabel(item.Mode)}</div>
              <div class="small">${item.ID || "-"}</div>
            </td>
            <td><${StatusBadge} label=${syncStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td>
            <td class="small">成功 ${item.Processed || 0} / 总数 ${item.Total || 0}<br />失败 ${item.Failed || 0} / 成功率 ${sourceAssetJobSuccessRate(item)}${sourceAssetJobSelectionCount(item) ? html`<br />范围 ${sourceAssetJobSelectionCount(item)} 张` : null}</td>
            <td class="small">${item.CurrentItem || "-"}${sourceAssetJobRecentError(item) ? html`<div style="margin-top:8px;">${sourceAssetJobRecentError(item)}</div>` : null}</td>
            <td class="small">${item.StartedAt || item.Created || "-"}<br />${item.FinishedAt || "-"}</td>
            <td>
              <div class="action-row">
                <a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/asset-jobs/detail", { id: item.ID || "", returnTo: window.location.pathname + window.location.search })}>详情</a>
                <a class="btn secondary" href=${sourceAssetJobTargetHref(item)}>${sourceAssetJobTargetLabel(item)}</a>
                ${item.Processed ? html`<a class="btn secondary" href=${sourceAssetJobSuccessHref(item)}>${sourceAssetJobSuccessLabel(item)}</a>` : null}
                ${item.CanRetry ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === (item.ID || "")} onClick=${() => retryJob(item)}>${actionState.busy === (item.ID || "") ? "处理中..." : sourceAssetJobRetryLabel(item)}</button>` : html`<span class="pill">执行中</span>`}
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
          <span class="pill">范围 <code>${sourceAssetJobModeLabel(detail.Mode)}</code></span>
          ${sourceAssetJobSelectionCount(detail) ? html`<span class="pill">涉及图片 <code>${sourceAssetJobSelectionCount(detail)}</code></span>` : null}
          <span class="pill">成功率 <code>${sourceAssetJobSuccessRate(detail)}</code></span>
        </div>
        <div class="action-row" style="margin-top:12px;">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          <a class="btn secondary" href=${sourceAssetJobTargetHref(detail)}>${sourceAssetJobTargetLabel(detail)}</a>
          ${detail.Processed ? html`<a class="btn secondary" href=${sourceAssetJobSuccessHref(detail)}>查看成功项</a>` : null}
          ${detail.Failed ? html`<a class="btn secondary" href=${sourceAssetJobFailureHref(detail)}>查看失败项</a>` : null}
          ${detail.CanRetry ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === "retry"} onClick=${retryCurrent}>${actionState.busy === "retry" ? "处理中..." : sourceAssetJobRetryLabel(detail)}</button>` : html`<span class="pill">任务仍在执行中</span>`}
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">进度</div>
        <h2 class="card-title">执行状态</h2>
        <div class="metric-grid section">
          <${MetricCard} eyebrow="总数" value=${detail.Total || 0} />
          <${MetricCard} eyebrow="成功" value=${detail.Processed || 0} />
          <${MetricCard} eyebrow="失败" value=${detail.Failed || 0} />
          <${MetricCard} eyebrow="涉及图片" value=${sourceAssetJobSelectionCount(detail) || 0} />
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

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">失败图片</div>
      <h2 class="card-title">本次任务失败项</h2>
      <div class="table-wrap section"><table><thead><tr><th>图片</th><th>商品</th><th>角色</th><th>错误</th><th>动作</th></tr></thead><tbody>
        ${(detail.FailedItems || []).length ? (detail.FailedItems || []).map((item) => html`<tr>
          <td><strong>${item.AssetKey || "-"}</strong><div class="small">${item.AssetID || "-"}</div></td>
          <td>${item.Name || "-"}<div class="small">${item.ProductID || "-"}</div></td>
          <td class="small">${item.AssetRole || "-"}</td>
          <td class="small">${item.Error || "-"}</td>
          <td><a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/assets/detail", { id: item.AssetID || "", returnTo: window.location.pathname + window.location.search })}>查看图片</a></td>
        </tr>`) : html`<tr><td colspan="5" class="small">当前任务没有失败图片。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function SourceProductJobsPage() {
  const qs = new URLSearchParams(window.location.search);
  const jobType = qs.get("jobType") || "";
  const status = qs.get("status") || "";
  const q = qs.get("q") || "";
  const page = qs.get("page") || "";
  const pageSize = qs.get("pageSize") || "";
  const apiURL = buildURL("/api/pim/admin/source/product-jobs", { jobType, status, q, page, pageSize });
  const [reloadKey, setReloadKey] = useState(0);
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const resource = useResource(apiURL, [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="商品发布任务" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const summary = payload.summary || {};
  const filter = payload.filter || {};
  const items = summary.Items || [];

  async function retryJob(item) {
    if (!window.confirm(`确认重新执行“${sourceProductJobTypeLabel(item.JobType)}”吗？`)) return;
    setActionState({ busy: item.ID || "", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/product-jobs/retry", { id: item.ID || "" });
      const nextJob = result.job || {};
      if (nextJob.ID) {
        window.location.href = buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: nextJob.ID, returnTo: window.location.pathname + window.location.search });
        return;
      }
      setActionState({ busy: "", message: result.message || "商品发布任务已重新启动。", error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "重新执行商品发布任务失败" });
    }
  }

  return html`
    <section class="section card"><div class="card-body">
      <div class="card-kicker">筛选</div>
      <h2 class="card-title">商品发布任务历史</h2>
      <${ActionNotice} state=${actionState} />
      <form class="action-row" method="get" action="/_/mrtang-admin/source/product-jobs">
        <select name="jobType" defaultValue=${filter.JobType || ""}>
          <option value="">全部任务类型</option>
          <option value="promote">加入发布队列</option>
          <option value="retry_sync">重试发布</option>
          <option value="promote_sync">加入发布队列并发布</option>
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
        <a class="btn secondary" href="/_/mrtang-admin/source/product-jobs">重置</a>
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
            <td><strong>${sourceProductJobTypeLabel(item.JobType)}</strong><div class="small">范围：${sourceProductJobModeLabel(item.Mode)}</div><div class="small">${item.ID || "-"}</div></td>
            <td><${StatusBadge} label=${syncStatusLabel(item.Status)} currentTone=${tone(item.Status)} /></td>
            <td class="small">${sourceProductJobSummaryText(item)}</td>
            <td class="small">${item.CurrentItem || "-"}${sourceProductJobRecentError(item) ? html`<div style="margin-top:8px;">${sourceProductJobRecentError(item)}</div>` : null}</td>
            <td class="small">${item.StartedAt || item.Created || "-"}<br />${item.FinishedAt || "-"}</td>
            <td><div class="action-row"><a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: item.ID || "", returnTo: window.location.pathname + window.location.search })}>详情</a>${item.Failed ? html`<a class="btn secondary" href=${sourceProductJobFailedHref(item)}>查看失败商品</a>` : null}${item.CanRetry ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === (item.ID || "")} onClick=${() => retryJob(item)}>${actionState.busy === (item.ID || "") ? "处理中..." : sourceProductJobRetryLabel(item)}</button>` : html`<span class="pill">执行中</span>`}</div></td>
          </tr>
        `) : html`<tr><td colspan="6" class="small">当前筛选下没有商品发布任务。</td></tr>`}
      </tbody></table></div>
    </div></div></section>
  `;
}

function SourceProductJobDetailPage() {
  const qs = new URLSearchParams(window.location.search);
  const id = qs.get("id") || "";
  const returnTo = qs.get("returnTo") || "/_/mrtang-admin/source/product-jobs";
  const [actionState, setActionState] = useState({ busy: "", message: "", error: "" });
  const [reloadKey, setReloadKey] = useState(0);
  const resource = useResource(buildURL("/api/pim/admin/source/product-jobs/detail", { id, returnTo }), [reloadKey]);
  if (resource.loading) return html`<${LoadingSection} label="商品发布任务详情" />`;
  if (resource.error) return html`<${ErrorSection} error=${resource.error} />`;
  const payload = resource.data || {};
  const detail = payload.detail || {};
  const backHref = payload.returnTo || returnTo;

  async function retryCurrent() {
    if (!window.confirm(`确认重新执行“${sourceProductJobTypeLabel(detail.JobType)}”吗？`)) return;
    setActionState({ busy: "retry", message: "", error: "" });
    try {
      const result = await postForm("/api/pim/admin/source/product-jobs/retry", { id: detail.ID || "" });
      const nextJob = result.job || {};
      if (nextJob.ID) {
        window.location.href = buildURL("/_/mrtang-admin/source/product-jobs/detail", { id: nextJob.ID, returnTo: backHref });
        return;
      }
      setActionState({ busy: "", message: result.message || "商品发布任务已重新启动。", error: "" });
      setReloadKey((value) => value + 1);
    } catch (error) {
      setActionState({ busy: "", message: "", error: error.message || "重新执行商品发布任务失败" });
    }
  }

  return html`
    <section class="section split-grid">
      <section class="card"><div class="card-body">
        <div class="card-kicker">任务详情</div>
        <h2 class="card-title">${sourceProductJobTypeLabel(detail.JobType)}</h2>
        <${ActionNotice} state=${actionState} />
        <div class="inline-pills">
          <span class="pill">任务 ID <code>${detail.ID || "-"}</code></span>
          <${StatusBadge} label=${syncStatusLabel(detail.Status)} currentTone=${tone(detail.Status)} />
          <span class="pill">范围 <code>${sourceProductJobModeLabel(detail.Mode)}</code></span>
          <span class="pill">涉及商品 <code>${((detail.ProductIDs || []).length || 0)}</code></span>
          <span class="pill">结果 <code>${sourceProductJobSummaryText(detail)}</code></span>
        </div>
        <div class="action-row" style="margin-top:12px;">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          ${detail.Failed ? html`<a class="btn secondary" href=${sourceProductJobFailedHref(detail)}>查看失败商品</a>` : null}
          ${detail.CanRetry ? html`<button class="btn secondary" type="button" disabled=${actionState.busy === "retry"} onClick=${retryCurrent}>${actionState.busy === "retry" ? "处理中..." : sourceProductJobRetryLabel(detail)}</button>` : html`<span class="pill">任务仍在执行中</span>`}
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">进度</div>
        <h2 class="card-title">执行状态</h2>
        <div class="metric-grid section">
          <${MetricCard} eyebrow="总数" value=${detail.Total || 0} />
          <${MetricCard} eyebrow="成功" value=${detail.Processed || 0} />
          <${MetricCard} eyebrow="失败" value=${detail.Failed || 0} />
          <${MetricCard} eyebrow="剩余" value=${sourceProductJobRemaining(detail)} />
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

    <section class="section table-card"><div class="table-card"><div class="card-body">
      <div class="card-kicker">失败商品</div>
      <h2 class="card-title">本次任务失败项</h2>
      <div class="table-wrap section"><table><thead><tr><th>商品</th><th>SKU</th><th>状态</th><th>错误</th><th>动作</th></tr></thead><tbody>
        ${(detail.FailedItems || []).length ? (detail.FailedItems || []).map((item) => html`<tr>
          <td><strong>${item.Name || "-"}</strong><div class="small">${item.ProductID || "-"}</div></td>
          <td class="small">${item.SKU || "-"}</td>
          <td class="small">${item.SyncStatus || "-"}</td>
          <td class="small">${item.Error || "-"}</td>
          <td><a class="btn secondary" href=${buildURL("/_/mrtang-admin/source/products/detail", { id: item.RecordID || "", returnTo: window.location.pathname + window.location.search })}>查看商品</a></td>
        </tr>`) : html`<tr><td colspan="5" class="small">当前任务没有失败商品。</td></tr>`}
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
        <div class="small" style="margin-top:12px;"><strong>发布分类：</strong>${detail.CategoryPath || "-"}</div>
        <div class="small" style="margin-top:8px;"><strong>观察到分类路径</strong></div>
        <pre>${detail.ObservedCategoryPaths || "-"}</pre>
        <div class="small" style="margin-top:8px;"><strong>观察到分类键</strong></div>
        <pre>${detail.ObservedCategoryKeys || "-"}</pre>
        <div class="action-row">
          <a class="btn secondary" href=${backHref}>返回上一页</a>
          <input type="text" value=${approveNote} onInput=${(event) => setApproveNote(event.currentTarget.value)} placeholder="审核备注" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "approve"} onClick=${() => detailAction("/api/pim/admin/source/products/status", { id: detail.ID || "", status: "approved", note: approveNote }, "确认将这个商品标记为通过吗？", "approve", "商品审核状态已更新。")}>${actionState.busy === "approve" ? "处理中..." : "通过"}</button>
          <input type="text" value=${rejectNote} onInput=${(event) => setRejectNote(event.currentTarget.value)} placeholder="驳回原因" />
          <button class="btn secondary" type="button" disabled=${actionState.busy === "reject"} onClick=${() => detailAction("/api/pim/admin/source/products/status", { id: detail.ID || "", status: "rejected", note: rejectNote }, "确认拒绝这个商品吗？", "reject", "商品审核状态已更新。")}>${actionState.busy === "reject" ? "处理中..." : "拒绝"}</button>
        </div>
      </div></section>
      <section class="card"><div class="card-body">
        <div class="card-kicker">发布队列状态</div>
        <h2 class="card-title">发布链状态</h2>
        <div class="inline-pills">
          <${StatusBadge} label=${(detail.Bridge && detail.Bridge.SyncStatus) || (detail.Bridge && detail.Bridge.Linked ? "linked" : "unlinked")} currentTone=${tone((detail.Bridge && detail.Bridge.SyncStatus) || (detail.Bridge && detail.Bridge.Linked ? "warning" : "error"))} />
          <span class="pill">supplierRecord: <code>${(detail.Bridge && detail.Bridge.SupplierRecordID) || "-"}</code></span>
          <span class="pill">vendure: <code>${(detail.Bridge && detail.Bridge.VendureProductID) || "-"} / ${(detail.Bridge && detail.Bridge.VendureVariantID) || "-"}</code></span>
        </div>
        ${(detail.Bridge && detail.Bridge.LastSyncError) ? html`<div class="flash error" style="margin-top:14px;">${detail.Bridge.LastSyncError}</div>` : null}
        <div class="action-row" style="margin-top:12px;">
          <button class="btn secondary" type="button" disabled=${actionState.busy === "promote"} onClick=${() => detailAction("/api/pim/admin/source/products/promote", { id: detail.ID || "" }, "确认将这个商品加入发布队列吗？", "promote", "商品已加入发布队列。")}>${actionState.busy === "promote" ? "处理中..." : "加入发布队列"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "promote-sync"} onClick=${() => detailAction("/api/pim/admin/source/products/promote-sync", { id: detail.ID || "" }, "确认将这个商品加入发布队列并发布到 Backend 吗？", "promote-sync", "商品已加入发布队列并发布到 Backend。")}>${actionState.busy === "promote-sync" ? "处理中..." : "加入发布队列并发布"}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "retry-sync"} onClick=${() => detailAction("/api/pim/admin/source/products/retry-sync", { id: detail.ID || "" }, "确认重试这个商品发布到 Backend 吗？", "retry-sync", "已触发商品发布重试。")}>${actionState.busy === "retry-sync" ? "处理中..." : "重试发布到 Backend"}</button>
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
          <button class="btn secondary" type="button" disabled=${actionState.busy === "download" || !detail.CanDownloadOriginal} title=${detail.CanDownloadOriginal ? "" : "该图片资产没有可用源图地址"} onClick=${downloadOriginal}>${actionState.busy === "download" ? "下载中..." : (detail.CanDownloadOriginal ? "下载原图" : "不可下载")}</button>
          <button class="btn secondary" type="button" disabled=${actionState.busy === "process" || (!detail.CanDownloadOriginal && !detail.OriginalImageURL)} title=${(!detail.CanDownloadOriginal && !detail.OriginalImageURL) ? "该图片资产没有可用源图地址或原图文件" : ""} onClick=${processAsset}>${actionState.busy === "process" ? "处理中..." : ((!detail.CanDownloadOriginal && !detail.OriginalImageURL) ? "不可处理" : "处理 / 重处理")}</button>
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
      title=${currentRoute === "backend-release" ? "发布准备" : currentRoute === "target-sync" ? "抓取入库" : currentRoute === "source" ? "源数据" : currentRoute === "source-categories" ? "源数据分类" : currentRoute === "source-products" ? "源数据商品" : currentRoute === "source-product-detail" ? "商品详情" : currentRoute === "source-product-jobs" ? "商品发布任务" : currentRoute === "source-product-job-detail" ? "商品发布任务详情" : currentRoute === "source-assets" ? "源数据图片" : currentRoute === "source-asset-detail" ? "图片详情" : currentRoute === "source-asset-jobs" ? "图片任务" : currentRoute === "source-asset-job-detail" ? "图片任务详情" : currentRoute === "procurement" ? "采购" : currentRoute === "procurement-detail" ? "采购详情" : "总览"}
      subtitle=${currentRoute === "target-sync"
        ? "先开页面，再异步拉抓取摘要、来源矩阵和最近写操作；“当前源站结果”只代表本次实际读到的数据。"
        : currentRoute === "backend-release"
          ? "先看 Vendure 字段准备度、分类映射和商品 payload 预览，再决定何时正式同步。"
        : currentRoute === "source"
          ? "先看 source 模块概览，再分流到商品、图片和日志；数据异步加载，不阻塞整页。"
          : currentRoute === "source-categories"
            ? "分类树抓取入库结果和已落库分类都在这里查看；页面先开壳，再异步加载。"
          : currentRoute === "source-products"
            ? "商品审核、加入发布队列、发布重试改成前端异步列表；现有动作端点继续复用。"
            : currentRoute === "source-product-detail"
              ? "详情页也切到前端异步渲染，动作端点继续复用现有 POST 路由。"
            : currentRoute === "source-product-jobs"
              ? "批量发布和重试发布改成后台任务后，这里可以追踪进度、失败项和重跑。"
              : currentRoute === "source-product-job-detail"
                ? "任务详情会显示进度、错误和失败商品，刷新页面后也能继续追踪。"
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
        : currentRoute === "backend-release"
          ? html`<${BackendReleasePage} />`
        : currentRoute === "source"
          ? html`<${SourceModulePage} />`
        : currentRoute === "source-categories"
          ? html`<${SourceCategoriesPage} />`
        : currentRoute === "source-products"
          ? html`<${SourceProductsPage} />`
        : currentRoute === "source-product-detail"
          ? html`<${SourceProductDetailPage} />`
        : currentRoute === "source-product-jobs"
          ? html`<${SourceProductJobsPage} />`
        : currentRoute === "source-product-job-detail"
          ? html`<${SourceProductJobDetailPage} />`
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
