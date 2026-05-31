const DEFAULT_API_BASE = `${window.location.origin}/saas-api/api/v1`;

const state = {
  apiBase: localStorage.getItem("mu.apiBase") || DEFAULT_API_BASE,
  token: localStorage.getItem("mu.token") || "",
  tenantId: localStorage.getItem("mu.tenantId") || "",
  tenants: [],
  kbs: [],
  files: [],
  documents: [],
  selectedDocument: null,
  documentChunks: [],
  documentJobs: [],
  pendingChunks: [],
  agents: [],
  agentBindings: [],
  conversations: [],
  selectedConversationId: "",
  messages: [],
  licenses: [],
  orders: [],
  payments: [],
  paymentEvents: [],
  usage: [],
  subscription: null,
  health: null,
  analytics: null,
  rateLimitPolicy: null,
  members: [],
  auditLogs: [],
  auditNextCursor: "",
  auditCursorStack: [],
  auditPage: 1,
  auditFilters: {
    action: "",
    resource_type: "",
    actor_user_id: "",
    from: "",
    to: "",
    limit: "50",
  },
  invitations: [],
  webhooks: [],
  webhookDeliverySummary: null,
  webhookDeliveries: [],
  webhookDeliveryFilters: {
    endpoint_id: "",
    event_type: "",
    status: "",
    limit: "50",
  },
};

const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => Array.from(root.querySelectorAll(selector));

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function toast(message) {
  const el = $("#toast");
  el.textContent = message;
  el.classList.remove("hidden");
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => el.classList.add("hidden"), 3600);
}

function pretty(value) {
  return JSON.stringify(value, null, 2);
}

async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
  if (options.body !== undefined && !isFormData) headers["Content-Type"] = "application/json";
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  if (options.tenant !== false && state.tenantId) headers["X-Tenant-ID"] = state.tenantId;

  const response = await fetch(`${state.apiBase}${path}`, {
    ...options,
    headers,
    body: options.body === undefined ? undefined : isFormData ? options.body : JSON.stringify(options.body),
  });

  const text = await response.text();
  let payload;
  try {
    payload = text ? JSON.parse(text) : {};
  } catch {
    payload = { message: text };
  }
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || `HTTP ${response.status}`);
  }
  return payload.data;
}

function currentTenant() {
  return state.tenants.find((item) => item.id === state.tenantId);
}

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (ch) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  })[ch]);
}

function empty(text) {
  return `<div class="item item-meta">${escapeHtml(text)}</div>`;
}

function formData(form) {
  return Object.fromEntries(new FormData(form).entries());
}

function formatDate(value, fallback = "-") {
  if (!value) return fallback;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return fallback;
  return date.toLocaleDateString();
}

function formatDateTime(value, fallback = "-") {
  if (!value) return fallback;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return fallback;
  return date.toLocaleString();
}

function countBy(items, predicate) {
  return (items || []).filter(predicate).length;
}

function findLicenseExpiry() {
  const candidates = state.licenses
    .map((license) => license.expired_at || license.expires_at || license.end_at || license.valid_to)
    .filter(Boolean)
    .map((value) => new Date(value))
    .filter((date) => !Number.isNaN(date.getTime()))
    .sort((a, b) => b.getTime() - a.getTime());
  return candidates[0] ? formatDate(candidates[0]) : "暂无授权";
}

function fillSelect(selector, items, placeholder, allowEmpty = false) {
  const el = $(selector);
  if (!el) return;
  el.innerHTML = "";
  if (allowEmpty || !items.length) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = placeholder;
    el.appendChild(option);
  }
  items.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = item.name || item.license_no || item.id;
    el.appendChild(option);
  });
}

function renderShell() {
  $("#authPanel").classList.toggle("hidden", Boolean(state.token));
  $("#workspace").classList.toggle("hidden", !state.token);
  $("#logoutBtn").classList.toggle("hidden", !state.token);
  $("#refreshBtn").classList.toggle("hidden", !state.token);
  $("#tenantSelect").classList.toggle("hidden", !state.token);
  $("#apiBaseInput").value = state.apiBase;
  renderTenants();
  renderMetrics();
  renderRateLimitPolicy();
}

function renderMetrics() {
  setText("tenantCount", state.tenants.length);
  setText("kbCount", state.kbs.length);
  setText("agentCount", state.agents.length);
  setText("licenseCount", state.licenses.length);
  setText("fileCount", state.files.length);
  setText("documentCount", state.documents.length);
  setText("pendingChunkCount", state.pendingChunks.length);
  setText("memberCount", state.members.length);
  setText("invitationCount", state.invitations.length);
  setText("overviewUpdatedAt", `更新时间：${new Date().toLocaleString()}`);
  const tenant = currentTenant();
  setText("tenantSummaryName", tenant ? tenant.name : "当前租户");
  setText("tenantSummaryRole", tenant?.role_code || tenant?.status || "未选择");
  setText("tenantSummaryCode", tenant?.code || "-");
  setText("tenantSummaryExpiry", findLicenseExpiry());
  renderOverviewInsights();
  renderRankings();
}

function renderAnalytics() {
  const data = state.analytics;
  if (!data) {
    setText("analyticsGeneratedAt", "等待加载");
    return;
  }
  const resource = data.resource || {};
  const business = data.business || {};
  setText("analyticsGeneratedAt", `生成时间：${formatDateTime(data.generated_at)}`);
  setText("analyticsKbCount", Number(resource.knowledge_bases || 0).toLocaleString());
  setText("analyticsDocumentCount", Number(resource.documents || 0).toLocaleString());
  setText("analyticsAgentCount", Number(resource.agents || 0).toLocaleString());
  setText("analyticsRevenue", Number(business.total_revenue_cny || 0).toLocaleString());

  const resourceGrid = $("#analyticsResourceGrid");
  if (resourceGrid) {
    const items = [
      ["文件", resource.files],
      ["Chunk", resource.document_chunks],
      ["待向量化", resource.pending_chunks],
      ["文档任务", resource.document_jobs],
      ["会话", resource.conversations],
      ["消息", resource.messages],
      ["成员", resource.members],
      ["邀请", resource.invitations],
    ];
    resourceGrid.innerHTML = items.map(([label, value]) => `
      <article><span>${escapeHtml(label)}</span><strong>${Number(value || 0).toLocaleString()}</strong></article>
    `).join("");
  }

  const riskList = $("#analyticsRiskList");
  if (riskList) {
    riskList.innerHTML = (data.risks || []).map((item) => `
      <article class="item">
        <div class="item-title">
          <strong>${escapeHtml(item.message)}</strong>
          <span class="badge ${item.level === "warn" ? "warn" : item.level === "ok" ? "" : "secondary"}">${escapeHtml(item.level)}</span>
        </div>
        <div class="item-meta">${escapeHtml(item.code)} / ${Number(item.count || 0).toLocaleString()}</div>
      </article>
    `).join("") || empty("暂无运营风险");
  }

  const businessList = $("#analyticsBusinessList");
  if (businessList) {
    const rows = [
      ...(business.orders || []).map((item) => ({ group: "订单", ...item })),
      ...(business.payments || []).map((item) => ({ group: "支付", ...item })),
      ...(business.licenses || []).map((item) => ({ group: "授权", ...item })),
    ];
    const max = Math.max(1, ...rows.map((item) => Number(item.count || 0)));
    businessList.innerHTML = rows.map((item) => {
      const width = Math.max(6, Math.round((Number(item.count || 0) / max) * 100));
      return `
        <article class="bar-row">
          <div><strong>${escapeHtml(item.group)} · ${escapeHtml(item.status)}</strong><span>${Number(item.count || 0).toLocaleString()}</span></div>
          <i style="width:${width}%"></i>
        </article>
      `;
    }).join("") || empty("暂无订单、支付或授权数据");
  }

  const trend = $("#analyticsUsageTrend");
  if (trend) {
    const rows = data.usage_trend || [];
    const max = Math.max(1, ...rows.map((item) => Number(item.quantity || 0)));
    trend.innerHTML = rows.map((item) => {
      const width = Math.max(6, Math.round((Number(item.quantity || 0) / max) * 100));
      return `
        <article class="bar-row">
          <div><strong>${escapeHtml(item.date)} · ${escapeHtml(item.metric)}</strong><span>${Number(item.quantity || 0).toLocaleString()} ${escapeHtml(item.unit || "")}</span></div>
          <i style="width:${width}%"></i>
        </article>
      `;
    }).join("") || empty("暂无近 7 天用量");
  }

  const actionList = $("#analyticsActionList");
  if (actionList) {
    actionList.innerHTML = (data.recent_actions || []).map((item) => `
      <article class="item">
        <div class="item-title">
          <strong>${escapeHtml(item.action)}</strong>
          <span class="badge">${escapeHtml(formatDateTime(item.created_at))}</span>
        </div>
        <div class="item-meta">${escapeHtml(item.resource_type || "-")} / actor: ${escapeHtml(item.actor_user_id || "-")}</div>
      </article>
    `).join("") || empty("暂无最近操作");
  }
}

function renderHealthSummary(result) {
  const el = $("#healthBox");
  if (!el) return;
  if (!result) {
    el.textContent = "等待检查";
    return;
  }
  if (result.error) {
    el.innerHTML = `
      <article class="status-row danger">
        <strong>检查失败</strong>
        <span>${escapeHtml(result.error)}</span>
      </article>
      <article class="status-row">
        <strong>检查时间</strong>
        <span>${escapeHtml(formatDateTime(result.checked_at))}</span>
      </article>
    `;
    return;
  }
  const readyStatus = result.ready?.status || "unknown";
  const healthStatus = result.health?.status || "unknown";
  const serviceName = result.health?.service || "未返回服务名称";
  const tenant = currentTenant();
  el.innerHTML = `
    <article class="status-row">
      <strong>就绪状态</strong>
      <span>${readyStatus === "ready" ? "已就绪" : escapeHtml(readyStatus)}</span>
    </article>
    <article class="status-row">
      <strong>健康状态</strong>
      <span>${healthStatus === "ok" ? "正常" : escapeHtml(healthStatus)}</span>
    </article>
    <article class="status-row">
      <strong>服务名称</strong>
      <span>${escapeHtml(serviceName)}</span>
    </article>
    <article class="status-row">
      <strong>当前租户</strong>
      <span>${escapeHtml(tenant?.name || "未选择")}</span>
    </article>
    <article class="status-row">
      <strong>检查时间</strong>
      <span>${escapeHtml(formatDateTime(result.checked_at))}</span>
    </article>
  `;
}

function renderSubscriptionSummary(item, error = "") {
  const el = $("#subscriptionBox");
  if (!el) return;
  if (error) {
    el.innerHTML = `
      <article class="status-row danger">
        <strong>加载失败</strong>
        <span>${escapeHtml(error)}</span>
      </article>
    `;
    return;
  }
  if (!item) {
    el.textContent = "等待加载";
    return;
  }
  const source = item.metadata?.source === "auto_free" ? "系统自动开通免费版" : item.metadata?.source || "系统记录";
  el.innerHTML = `
    <article class="status-row">
      <strong>套餐名称</strong>
      <span>${escapeHtml(item.plan_name || item.plan_code || "未命名套餐")}</span>
    </article>
    <article class="status-row">
      <strong>套餐编码</strong>
      <span>${escapeHtml(item.plan_code || "-")}</span>
    </article>
    <article class="status-row">
      <strong>订阅状态</strong>
      <span>${item.status === "active" ? "生效中" : escapeHtml(item.status || "-")}</span>
    </article>
    <article class="status-row">
      <strong>开始时间</strong>
      <span>${escapeHtml(formatDateTime(item.started_at))}</span>
    </article>
    <article class="status-row">
      <strong>到期时间</strong>
      <span>${escapeHtml(formatDateTime(item.expired_at, "长期有效"))}</span>
    </article>
    <article class="status-row">
      <strong>自动续费</strong>
      <span>${item.auto_renew ? "已开启" : "未开启"}</span>
    </article>
    <article class="status-row">
      <strong>开通来源</strong>
      <span>${escapeHtml(source)}</span>
    </article>
  `;
}

function renderRateLimitPolicy(error = "") {
  const el = $("#rateLimitPolicyGrid");
  if (!el) return;
  if (error) {
    el.innerHTML = `
      <article>
        <strong>加载失败</strong>
        <span>${escapeHtml(error)}</span>
      </article>
    `;
    return;
  }
  const item = state.rateLimitPolicy;
  if (!item) {
    el.innerHTML = `<article><strong>限流策略</strong><span>等待加载</span></article>`;
    return;
  }
  const windowSeconds = Number(item.window_seconds || 60);
  const windowText = windowSeconds % 60 === 0 ? `${windowSeconds / 60} 分钟` : `${windowSeconds} 秒`;
  const backendText = item.backend === "redis" ? "Redis 集中限流" : "内存限流";
  const redisText = item.redis_enabled ? "已启用 Redis，多实例共享计数" : "未启用 Redis，当前实例本地计数";
  el.innerHTML = `
    <article>
      <strong>租户 API</strong>
      <span>同一租户每 ${escapeHtml(windowText)} ${Number(item.tenant_per_window || 0).toLocaleString()} 次</span>
    </article>
    <article>
      <strong>用户 API</strong>
      <span>同一用户每 ${escapeHtml(windowText)} ${Number(item.user_per_window || 0).toLocaleString()} 次</span>
    </article>
    <article>
      <strong>登录注册</strong>
      <span>同一 IP 每 ${escapeHtml(windowText)} ${Number(item.auth_ip_per_window || 0).toLocaleString()} 次</span>
    </article>
    <article>
      <strong>限流后端</strong>
      <span>${escapeHtml(backendText)}；${escapeHtml(redisText)}</span>
    </article>
    <article>
      <strong>故障策略</strong>
      <span>${escapeHtml(item.redis_fallback_label || "异常时保持接口可用优先")}</span>
    </article>
  `;
}

function renderAnswerSummary(selector, result, error = "") {
  const el = $(selector);
  if (!el) return;
  if (error) {
    el.innerHTML = `
      <article class="answer-block danger">
        <h4>请求失败</h4>
        <p>${escapeHtml(error)}</p>
      </article>
    `;
    return;
  }
  if (!result) {
    el.textContent = "等待结果";
    return;
  }
  const references = result.references || [];
  const meta = [
    result.conversation_id ? `会话：${result.conversation_id}` : "",
    result.knowledge_base_id ? `知识库：${result.knowledge_base_id}` : "",
    result.history_used !== undefined ? `使用历史：${Number(result.history_used || 0).toLocaleString()} 条` : "",
    result.generation_model ? `生成模型：${result.generation_model}` : "",
    result.generation_source ? `生成来源：${result.generation_source}` : "",
  ].filter(Boolean);
  el.innerHTML = `
    <article class="answer-block">
      <h4>回答内容</h4>
      <p>${escapeHtml(result.answer || "暂无回答")}</p>
    </article>
    ${meta.length ? `<div class="answer-meta">${meta.map((item) => `<span>${escapeHtml(item)}</span>`).join("")}</div>` : ""}
    <article class="answer-block">
      <h4>引用资料</h4>
      <div class="reference-list">
        ${references.map((item, index) => `
          <section class="reference-item">
            <strong>${index + 1}. ${escapeHtml(item.title || item.document_id || "未命名资料")}</strong>
            <span>匹配分：${Number(item.score || 0).toFixed(4)}</span>
            <p>${escapeHtml((item.content || "").slice(0, 260))}</p>
          </section>
        `).join("") || `<div class="item item-meta">暂无引用资料</div>`}
      </div>
    </article>
  `;
}

function renderAcceptInvitationResult(result, error = "") {
  const el = $("#acceptInvitationResult");
  if (!el) return;
  if (error) {
    el.innerHTML = `
      <article class="status-row danger">
        <strong>接受失败</strong>
        <span>${escapeHtml(error)}</span>
      </article>
    `;
    return;
  }
  if (!result) {
    el.textContent = "等待提交";
    return;
  }
  el.innerHTML = `
    <article class="status-row">
      <strong>接受结果</strong>
      <span>已加入租户</span>
    </article>
    <article class="status-row">
      <strong>租户名称</strong>
      <span>${escapeHtml(result.name || "未命名租户")}</span>
    </article>
    <article class="status-row">
      <strong>租户编码</strong>
      <span>${escapeHtml(result.code || "-")}</span>
    </article>
    <article class="status-row">
      <strong>成员角色</strong>
      <span>${escapeHtml(result.role_code || "-")}</span>
    </article>
    <article class="status-row">
      <strong>租户状态</strong>
      <span>${result.status === "active" ? "正常" : escapeHtml(result.status || "-")}</span>
    </article>
  `;
}

function renderOverviewInsights() {
  const failedJobs = countBy(state.documentJobs, (job) => job.status === "failed");
  const runningJobs = countBy(state.documentJobs, (job) => ["running", "pending", "queued"].includes(job.status));
  const openInvitations = countBy(state.invitations, (item) => !["accepted", "revoked", "expired"].includes(item.status));
  const pendingOrders = countBy(state.orders, (order) => ["pending", "created", "unpaid"].includes(order.status));
  const paidOrders = countBy(state.orders, (order) => ["paid", "completed", "success"].includes(order.status));
  const failedPayments = countBy(state.payments, (payment) => ["failed", "closed", "cancelled"].includes(payment.status));
  const healthText = state.health ? `API ${state.health.status || "ready"}，最近检查 ${formatDateTime(state.health.checked_at)}` : `已连接 ${state.apiBase.replace(/^https?:\/\//, "")}`;
  setText("serviceSummaryText", healthText);
  setText("pendingSummaryText", `${state.pendingChunks.length} 个 Chunk，${runningJobs} 个文档任务，${openInvitations} 个邀请`);
  setText("businessSummaryText", `${state.orders.length} 笔订单，${paidOrders} 已成交，${pendingOrders} 待处理`);

  const activities = [
    ...state.auditLogs.slice(0, 5).map((log) => ({
      type: "审计",
      text: `${log.action || "操作"} · ${log.resource_type || "资源"}`,
      time: log.created_at,
    })),
    ...state.documentJobs.slice(0, 3).map((job) => ({
      type: "文档",
      text: `${job.title || job.job_type || "文档任务"} · ${job.status || "pending"}`,
      time: job.updated_at || job.created_at,
    })),
    ...state.paymentEvents.slice(0, 3).map((event) => ({
      type: "支付",
      text: `${event.pay_no || event.channel || "支付回调"} · ${event.result_status || event.event_status || "received"}`,
      time: event.created_at,
    })),
  ].sort((a, b) => new Date(b.time || 0) - new Date(a.time || 0)).slice(0, 6);

  const activityList = $("#activityFeedList");
  if (activityList) {
    activityList.innerHTML = activities.map((item) => `
      <li>
        <span class="feed-type">${escapeHtml(item.type)}</span>
        <span class="feed-text">${escapeHtml(item.text)}</span>
        <time>${escapeHtml(formatDateTime(item.time, "刚刚"))}</time>
      </li>
    `).join("") || `<li class="muted-line">暂无实时动态</li>`;
  }

  const usageText = state.usage.length ? `${state.usage.length} 项用量指标已加载` : "暂无用量指标";
  const insightList = $("#systemInsightList");
  if (insightList) {
    insightList.innerHTML = [
      { label: "知识资产", value: `${state.kbs.length} 个知识库 / ${state.documents.length} 篇文档 / ${state.files.length} 个文件` },
      { label: "智能体", value: `${state.agents.length} 个智能体 / ${state.conversations.length} 个会话` },
      { label: "授权", value: `${state.licenses.length} 条 License / 到期 ${findLicenseExpiry()}` },
      { label: "账单", value: `${usageText} / ${failedPayments} 个异常支付` },
      { label: "任务风险", value: failedJobs ? `${failedJobs} 个文档任务失败` : "文档任务无失败记录" },
    ].map((item) => `
      <li>
        <span>${escapeHtml(item.label)}</span>
        <strong>${escapeHtml(item.value)}</strong>
      </li>
    `).join("");
  }
}

function rankBadge(index) {
  return `<span class="rank-badge">${index + 1}</span>`;
}

function renderRankings() {
  const kbBody = $("#kbRankBody");
  if (kbBody) {
    const rows = state.kbs.slice(0, 5).map((kb, index) => {
      const docCount = state.documents.filter((doc) => doc.knowledge_base_id === kb.id || doc.kb_id === kb.id).length;
      return `
        <tr>
          <td>${rankBadge(index)}</td>
          <td><a href="#knowledge" data-view-link="knowledge">${escapeHtml(kb.name)}</a></td>
          <td>${escapeHtml(kb.code || "-")}</td>
          <td>${escapeHtml(kb.status || "active")}</td>
          <td>${Number(docCount).toLocaleString()}</td>
        </tr>
      `;
    }).join("");
    kbBody.innerHTML = rows || `<tr><td colspan="5" class="empty-cell">暂无数据</td></tr>`;
  }
  const agentBody = $("#agentRankBody");
  if (agentBody) {
    const rows = state.agents.slice(0, 5).map((agent, index) => `
      <tr>
        <td>${rankBadge(index)}</td>
        <td><a href="#agents" data-view-link="agents">${escapeHtml(agent.name)}</a></td>
        <td>${escapeHtml(agent.code || "-")}</td>
        <td>${escapeHtml(agent.status || "draft")}</td>
        <td>${Number(state.agentBindings.filter((item) => item.agent_id === agent.id).length).toLocaleString()}</td>
      </tr>
    `).join("");
    agentBody.innerHTML = rows || `<tr><td colspan="5" class="empty-cell">暂无数据</td></tr>`;
  }
}

function renderTenants() {
  const select = $("#tenantSelect");
  select.innerHTML = "";
  state.tenants.forEach((tenant) => {
    const option = document.createElement("option");
    option.value = tenant.id;
    option.textContent = `${tenant.name} (${tenant.code})`;
    option.selected = tenant.id === state.tenantId;
    select.appendChild(option);
  });
  $("#tenantList").innerHTML = state.tenants.map((tenant) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(tenant.name)}</strong>
        <span class="badge">${escapeHtml(tenant.role_code || tenant.status || "active")}</span>
      </div>
      <div class="item-meta">${escapeHtml(tenant.id)} / ${escapeHtml(tenant.code || "")}</div>
    </article>
  `).join("") || empty("暂无租户");
  renderMetrics();
}

function renderKbs() {
  $("#kbList").innerHTML = state.kbs.map((kb) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(kb.name)}</strong>
        <span class="badge">${escapeHtml(kb.status || "active")}</span>
      </div>
      <div class="item-meta">${escapeHtml(kb.id)} / ${escapeHtml(kb.code || "")}</div>
    </article>
  `).join("") || empty("暂无知识库");
  fillSelect("#askForm select[name='kb_id']", state.kbs, "选择知识库");
  fillSelect("#documentJobForm select[name='kb_id']", state.kbs, "选择知识库");
  fillSelect("#pendingChunkFilterForm select[name='kb_id']", state.kbs, "选择知识库");
  fillSelect("#documentListFilterForm select[name='kb_id']", state.kbs, "选择知识库");
  fillSelect("#agentChatForm select[name='knowledge_base_id']", state.kbs, "默认绑定知识库", true);
  fillSelect("#agentBindingForm select[name='knowledge_base_id']", state.kbs, "选择知识库");
  fillSelect("#agentConversationForm select[name='knowledge_base_id']", state.kbs, "默认绑定知识库", true);
  renderMetrics();
}

function renderFiles() {
  $("#fileList").innerHTML = state.files.map((file) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(file.filename)}</strong>
        <span class="badge">${Number(file.size_bytes || 0).toLocaleString()} bytes</span>
      </div>
      <div class="item-meta">${escapeHtml(file.id)} / ${escapeHtml(file.mime_type || "-")}</div>
      <div class="item-actions">
        <button class="button small secondary" data-file-download="${escapeHtml(file.id)}">下载</button>
      </div>
    </article>
  `).join("") || empty("暂无文件");
  fillSelect("#documentJobForm select[name='file_id']", state.files, "选择文件");
  renderMetrics();
}

function renderDocumentJobs() {
  $("#documentJobList").innerHTML = state.documentJobs.map((job) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(job.title || job.job_type)}</strong>
        <span class="badge ${job.status === "failed" ? "danger" : job.status === "running" ? "warn" : ""}">${escapeHtml(job.status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(job.id)} / ${escapeHtml(job.file_id || "-")} / attempts: ${Number(job.attempts || 0).toLocaleString()}</div>
      ${job.last_error ? `<div class="item-meta danger-text">${escapeHtml(job.last_error)}</div>` : ""}
    </article>
  `).join("") || empty("暂无文档任务");
  renderMetrics();
}

function renderPendingChunks() {
  $("#pendingChunkList").innerHTML = state.pendingChunks.map((chunk) => `
    <article class="item">
      <div class="item-title">
        <strong>Chunk ${Number(chunk.chunk_no || 0).toLocaleString()}</strong>
        <span class="badge">${escapeHtml(chunk.embedding_status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(chunk.id)} / ${escapeHtml(chunk.document_id || "-")}</div>
      <div class="item-meta">${escapeHtml((chunk.content || "").slice(0, 180))}</div>
    </article>
  `).join("") || empty("暂无待向量化 Chunk");
  renderMetrics();
}

function renderDocuments() {
  $("#documentList").innerHTML = state.documents.map((doc) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(doc.title)}</strong>
        <span class="badge">${escapeHtml(doc.embedding_status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(doc.id)} / ${escapeHtml(doc.source_type)} / ${escapeHtml(doc.mime_type || "-")}</div>
      <div class="item-meta">parse: ${escapeHtml(doc.parse_status)} / chunk: ${escapeHtml(doc.chunk_status)} / embedding: ${escapeHtml(doc.embedding_status)}</div>
      <div class="item-actions">
        <button class="button small secondary" data-document-detail="${escapeHtml(doc.id)}">查看详情</button>
        <button class="button small secondary" data-document-chunks="${escapeHtml(doc.id)}">查看 Chunk</button>
        <button class="button small danger" data-document-archive="${escapeHtml(doc.id)}">归档</button>
      </div>
    </article>
  `).join("") || empty("暂无文档");
  renderDocumentDetail();
  renderMetrics();
}

function renderDocumentDetail() {
  const detailBox = $("#documentDetailBox");
  const chunkList = $("#documentChunkList");
  if (!detailBox || !chunkList) return;
  detailBox.textContent = state.selectedDocument ? pretty(state.selectedDocument) : "等待选择文档";
  chunkList.innerHTML = state.documentChunks.map((chunk) => `
    <article class="item">
      <div class="item-title">
        <strong>Chunk ${Number(chunk.chunk_no || 0).toLocaleString()}</strong>
        <span class="badge">${escapeHtml(chunk.embedding_status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(chunk.id)} / tokens: ${Number(chunk.content_tokens || 0).toLocaleString()} / ${escapeHtml(chunk.embedding_model || "-")}</div>
      <div class="item-meta">${escapeHtml(chunk.content || "")}</div>
    </article>
  `).join("") || empty("暂无 Chunk");
}

function renderAgents() {
  $("#agentList").innerHTML = state.agents.map((agent) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(agent.name)}</strong>
        <span class="badge">${escapeHtml(agent.status || "draft")}</span>
      </div>
      <div class="item-meta">${escapeHtml(agent.id)} / ${escapeHtml(agent.code || "")}</div>
      <div class="item-actions">
        <button class="button small secondary" data-agent-edit="${agent.id}">编辑</button>
        <button class="button small secondary" data-agent-publish="${agent.id}">发布</button>
        <button class="button small secondary" data-agent-rollback="${agent.id}">回滚</button>
        <button class="button small danger" data-agent-archive="${agent.id}">归档</button>
      </div>
    </article>
  `).join("") || empty("暂无智能体");
  fillSelect("#agentChatForm select[name='agent_id']", state.agents, "选择智能体");
  fillSelect("#agentBindingForm select[name='agent_id']", state.agents, "选择智能体");
  fillSelect("#agentConversationForm select[name='agent_id']", state.agents, "选择智能体");
  renderMetrics();
}

function resetAgentForm() {
  const form = $("#agentForm");
  if (!form) return;
  form.reset();
  form.elements.agent_id.value = "";
  form.elements.code.disabled = false;
  setText("agentFormTitle", "创建智能体");
  setText("saveAgentBtn", "创建智能体");
}

function fillAgentForm(agentID) {
  const agent = state.agents.find((item) => item.id === agentID);
  const form = $("#agentForm");
  if (!agent || !form) return;
  form.elements.agent_id.value = agent.id;
  form.elements.name.value = agent.name || "";
  form.elements.code.value = agent.code || "";
  form.elements.code.disabled = true;
  form.elements.description.value = agent.description || "";
  form.elements.system_prompt.value = agent.system_prompt || "";
  setText("agentFormTitle", "编辑智能体");
  setText("saveAgentBtn", "保存智能体");
}

function renderAgentBindings() {
  const el = $("#agentBindingList");
  if (!el) return;
  el.innerHTML = state.agentBindings.map((binding) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(binding.knowledge_base || binding.knowledge_base_id)}</strong>
        <span class="badge">${escapeHtml(binding.status || "active")}</span>
      </div>
      <div class="item-meta">${escapeHtml(binding.knowledge_base_id)} / ${escapeHtml(binding.id)}</div>
      <div class="item-actions">
        <button class="button small danger" data-agent-kb-unbind="${escapeHtml(binding.knowledge_base_id)}">解绑</button>
      </div>
    </article>
  `).join("") || empty("暂无绑定知识库");
  renderMetrics();
}

function renderConversations() {
  const el = $("#conversationList");
  if (!el) return;
  el.innerHTML = state.conversations.map((conversation) => {
    const active = conversation.id === state.selectedConversationId ? " active" : "";
    return `
      <article class="item selectable${active}" data-conversation-select="${escapeHtml(conversation.id)}">
        <div class="item-title">
          <strong>${escapeHtml(conversation.title || "未命名会话")}</strong>
          <span class="badge">${escapeHtml(conversation.status || "active")}</span>
        </div>
        <div class="item-meta">${escapeHtml(conversation.id)}</div>
        <div class="item-meta">${new Date(conversation.updated_at || conversation.created_at).toLocaleString()}</div>
      </article>
    `;
  }).join("") || empty("暂无会话");
  renderMetrics();
}

function renderMessages() {
  const el = $("#messageList");
  if (!el) return;
  el.innerHTML = state.messages.map((message) => `
    <article class="item message ${escapeHtml(message.role)}">
      <div class="item-title">
        <strong>${escapeHtml(message.role)}</strong>
        <span class="badge">${new Date(message.created_at).toLocaleString()}</span>
      </div>
      <div class="item-meta message-content">${escapeHtml(message.content)}</div>
    </article>
  `).join("") || empty(state.selectedConversationId ? "暂无消息" : "请选择会话");
}

function renderLicenses() {
  $("#licenseList").innerHTML = state.licenses.map((license) => {
    const statusClass = license.status === "revoked" ? "danger" : license.status === "inactive" ? "warn" : "";
    return `
      <article class="item">
        <div class="item-title">
          <strong>${escapeHtml(license.license_no)}</strong>
          <span class="badge ${statusClass}">${escapeHtml(license.status)}</span>
        </div>
        <div class="item-meta">${escapeHtml(license.id)} / ${escapeHtml(license.license_type)}</div>
        <div class="item-actions">
          <button class="button small secondary" data-license-verify="${license.id}">验证</button>
          <button class="button small secondary" data-license-activate="${license.id}">激活</button>
          <button class="button small danger" data-license-revoke="${license.id}">吊销</button>
        </div>
      </article>
    `;
  }).join("") || empty("暂无 License");
  renderMetrics();
}

function renderUsage(items) {
  state.usage = items || [];
  $("#usageList").innerHTML = (items || []).map((item) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(item.metric)}</strong>
        <span class="badge">${escapeHtml(item.unit)}</span>
      </div>
      <div class="item-meta">${Number(item.quantity).toLocaleString()}</div>
    </article>
  `).join("") || empty("暂无用量");
  renderMetrics();
}

function renderOrders() {
  $("#orderList").innerHTML = state.orders.map((order) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(order.order_no)}</strong>
        <span class="badge ${order.status === "cancelled" || order.status === "closed" ? "warn" : ""}">${escapeHtml(order.status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(order.id)} / ${escapeHtml(order.order_type)} / ${Number(order.amount_cents).toLocaleString()} ${escapeHtml(order.currency)}</div>
      <div class="item-actions">
        <button class="button small secondary" data-order-pay="${order.id}">mock 支付</button>
        <button class="button small secondary" data-order-close="${order.id}">关闭</button>
        <button class="button small danger" data-order-cancel="${order.id}">取消</button>
      </div>
    </article>
  `).join("") || empty("暂无订单");
  renderMetrics();
}

function renderPayments() {
  $("#paymentList").innerHTML = state.payments.map((payment) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(payment.pay_no)}</strong>
        <span class="badge ${payment.status === "failed" || payment.status === "closed" ? "warn" : ""}">${escapeHtml(payment.status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(payment.id)} / ${escapeHtml(payment.channel)} / ${Number(payment.amount_cents).toLocaleString()} ${escapeHtml(payment.currency)}</div>
      <div class="item-actions">
        <button class="button small secondary" data-payment-query="${payment.id}">查单</button>
        <button class="button small danger" data-payment-close="${payment.id}">关闭</button>
      </div>
    </article>
  `).join("") || empty("暂无支付单");
  renderMetrics();
}

function renderPaymentEvents() {
  $("#paymentEventList").innerHTML = state.paymentEvents.map((event) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(event.pay_no)}</strong>
        <span class="badge">${escapeHtml(event.result_status || event.event_status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(event.channel)} / ${escapeHtml(event.transaction_id || "-")} / ${new Date(event.created_at).toLocaleString()}</div>
      ${event.error_message ? `<div class="item-meta danger-text">${escapeHtml(event.error_message)}</div>` : ""}
    </article>
  `).join("") || empty("暂无回调事件");
  renderMetrics();
}

function renderMembers() {
  const el = $("#memberList");
  if (!el) return;
  el.innerHTML = state.members.map((member) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(member.email)}</strong>
        <span class="badge">${escapeHtml(member.role_code)}</span>
      </div>
      <div class="item-meta">${escapeHtml(member.id)} / ${escapeHtml(member.nickname || "-")} / ${escapeHtml(member.status)}</div>
      <div class="item-actions">
        <button class="button small secondary" data-member-role="${member.id}" data-role-code="admin">设为 admin</button>
        <button class="button small secondary" data-member-role="${member.id}" data-role-code="member">设为 member</button>
        <button class="button small secondary" data-member-role="${member.id}" data-role-code="viewer">设为 viewer</button>
        <button class="button small danger" data-member-remove="${member.id}">移除</button>
      </div>
    </article>
  `).join("") || empty("暂无成员");
  renderMetrics();
}

function renderAuditLogs() {
  const el = $("#auditLogList");
  if (!el) return;
  const prev = $("#auditPrevBtn");
  const next = $("#auditNextBtn");
  const info = $("#auditPagerInfo");
  if (prev) prev.disabled = state.auditCursorStack.length === 0;
  if (next) next.disabled = !state.auditNextCursor;
  if (info) info.textContent = `第 ${state.auditPage} 页 / ${state.auditLogs.length} 条`;
  el.innerHTML = state.auditLogs.map((log) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(log.action)}</strong>
        <span class="badge">${new Date(log.created_at).toLocaleString()}</span>
      </div>
      <div class="item-meta">${escapeHtml(log.resource_type || "-")} / ${escapeHtml(log.resource_id || "-")}</div>
      <div class="item-meta">actor: ${escapeHtml(log.actor_user_id || "-")} / ip: ${escapeHtml(log.ip || "-")}</div>
    </article>
  `).join("") || empty("暂无审计日志");
  renderMetrics();
}

function renderInvitations() {
  const el = $("#invitationList");
  if (!el) return;
  el.innerHTML = state.invitations.map((item) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(item.email)}</strong>
        <span class="badge">${escapeHtml(item.status)}</span>
      </div>
      <div class="item-meta">${escapeHtml(item.id)} / ${escapeHtml(item.role_code)} / ${new Date(item.expired_at).toLocaleString()}</div>
      ${item.token ? `<pre class="codebox">${escapeHtml(item.token)}</pre>` : ""}
      <div class="item-actions">
        <button class="button small danger" data-invitation-revoke="${item.id}">撤销</button>
      </div>
    </article>
  `).join("") || empty("暂无邀请");
  renderMetrics();
}

function eventLabel(eventType) {
  return {
    "webhook.test": "Webhook 测试",
    "order.paid": "订单支付成功",
    "license.activated": "License 激活",
    "license.revoked": "License 吊销",
    "agent.chat.finished": "Agent 会话完成",
  }[eventType] || eventType || "-";
}

function renderWebhooks() {
  const el = $("#webhookList");
  if (!el) return;
  el.innerHTML = state.webhooks.map((item) => `
    <article class="item">
      <div class="item-title">
        <strong>${escapeHtml(item.name)}</strong>
        <span class="badge ${item.status === "disabled" ? "warn" : ""}">${item.status === "active" ? "启用" : "停用"}</span>
      </div>
      <div class="item-meta">${escapeHtml(item.url)}</div>
      <div class="item-meta">订阅事件：${(item.events || []).map(eventLabel).map(escapeHtml).join("、") || "未配置"}</div>
      <div class="item-actions">
        <button class="button small secondary" data-webhook-test="${escapeHtml(item.id)}">测试发送</button>
        <button class="button small secondary" data-webhook-toggle="${escapeHtml(item.id)}">${item.status === "active" ? "停用" : "启用"}</button>
        <button class="button small danger" data-webhook-delete="${escapeHtml(item.id)}">删除</button>
      </div>
    </article>
  `).join("") || empty("暂无 Webhook 配置");
  fillSelect("#webhookDeliveryFilterForm select[name='endpoint_id']", state.webhooks, "全部 Webhook", true);
  const endpointFilter = $("#webhookDeliveryFilterForm select[name='endpoint_id']");
  if (endpointFilter) endpointFilter.value = state.webhookDeliveryFilters.endpoint_id;
}

function renderWebhookDeliveries() {
  const el = $("#webhookDeliveryList");
  if (!el) return;
  el.innerHTML = state.webhookDeliveries.map((item) => {
    const ok = item.status === "success";
    return `
      <article class="item">
        <div class="item-title">
          <strong>${escapeHtml(eventLabel(item.event_type))}</strong>
          <span class="badge ${ok ? "" : "danger"}">${ok ? "成功" : "失败"}</span>
        </div>
        <div class="item-meta">目标：${escapeHtml(item.target_url)} / HTTP ${item.http_status || "-"}</div>
        <div class="item-meta">耗时：${Number(item.duration_ms || 0).toLocaleString()} ms / 重试：${Number(item.retry_count || 0).toLocaleString()} 次 / ${formatDateTime(item.created_at)}</div>
        ${item.error_message ? `<div class="item-meta danger-text">错误：${escapeHtml(item.error_message)}</div>` : ""}
        ${ok ? "" : `<div class="item-actions"><button class="button small secondary" data-webhook-delivery-retry="${escapeHtml(item.id)}">立即重试</button></div>`}
      </article>
    `;
  }).join("") || empty("暂无投递记录");
}

function renderWebhookDeliverySummary(error = "") {
  const el = $("#webhookDeliverySummaryGrid");
  if (!el) return;
  if (error) {
    el.innerHTML = `
      <article>
        <strong>加载失败</strong>
        <span>${escapeHtml(error)}</span>
      </article>
    `;
    return;
  }
  const item = state.webhookDeliverySummary;
  if (!item) {
    el.innerHTML = `<article><strong>投递状态</strong><span>等待加载</span></article>`;
    return;
  }
  const total = Number(item.total || 0);
  const success = Number(item.success || 0);
  const failed = Number(item.failed || 0);
  const rate = total > 0 ? `${Math.round((success / total) * 100)}%` : "暂无数据";
  el.innerHTML = `
    <article>
      <strong>总投递</strong>
      <span>${total.toLocaleString()} 次，成功率 ${escapeHtml(rate)}</span>
    </article>
    <article>
      <strong>成功</strong>
      <span>${success.toLocaleString()} 次已完成</span>
    </article>
    <article>
      <strong>失败</strong>
      <span>${failed.toLocaleString()} 次失败记录</span>
    </article>
    <article>
      <strong>自动重试</strong>
      <span>${Number(item.retry_scheduled || 0).toLocaleString()} 条已排队，${Number(item.retry_due || 0).toLocaleString()} 条已到期</span>
    </article>
    <article>
      <strong>人工处理</strong>
      <span>${Number(item.manual_review || 0).toLocaleString()} 条需要人工确认或手动重试</span>
    </article>
    <article>
      <strong>最近尝试</strong>
      <span>${escapeHtml(formatDateTime(item.last_attempt_at, "暂无投递"))}</span>
    </article>
  `;
}

async function loadTenants() {
  const data = await api("/tenants", { tenant: false });
  state.tenants = data.items || [];
  if (!state.tenants.some((item) => item.id === state.tenantId)) {
    state.tenantId = state.tenants[0]?.id || "";
    localStorage.setItem("mu.tenantId", state.tenantId);
  }
  renderTenants();
}

async function loadKbs() {
  if (!state.tenantId) return;
  const data = await api("/kbs");
  state.kbs = data.items || [];
  renderKbs();
}

async function loadFiles() {
  if (!state.tenantId) return;
  const data = await api("/files");
  state.files = data.items || [];
  renderFiles();
  renderMetrics();
}

async function downloadFile(fileID) {
  const file = state.files.find((item) => item.id === fileID);
  const headers = {};
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  if (state.tenantId) headers["X-Tenant-ID"] = state.tenantId;
  const response = await fetch(`${state.apiBase}/files/${fileID}/download`, { headers });
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.message || message;
    } catch {}
    throw new Error(message);
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = file?.filename || `file-${fileID}`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function selectedJobKbID() {
  return $("#documentJobForm select[name='kb_id']")?.value || state.kbs[0]?.id || "";
}

function selectedPendingKbID() {
  return $("#pendingChunkFilterForm select[name='kb_id']")?.value || selectedJobKbID();
}

function selectedDocumentKbID() {
  return $("#documentListFilterForm select[name='kb_id']")?.value || selectedJobKbID();
}

async function loadDocumentJobs() {
  const kbID = selectedJobKbID();
  if (!state.tenantId || !kbID) return;
  const data = await api(`/kbs/${kbID}/document-jobs`);
  state.documentJobs = data.items || [];
  renderDocumentJobs();
  renderMetrics();
}

async function loadPendingChunks() {
  const kbID = selectedPendingKbID();
  if (!state.tenantId || !kbID) return;
  const data = await api(`/kbs/${kbID}/chunks/pending`);
  state.pendingChunks = data.items || [];
  renderPendingChunks();
  renderMetrics();
}

async function loadDocuments() {
  const kbID = selectedDocumentKbID();
  if (!state.tenantId || !kbID) return;
  const data = await api(`/kbs/${kbID}/documents`);
  state.documents = data.items || [];
  state.selectedDocument = null;
  state.documentChunks = [];
  renderDocuments();
  renderMetrics();
}

async function loadDocumentDetail(documentID, includeChunks = false) {
  const kbID = selectedDocumentKbID();
  if (!state.tenantId || !kbID || !documentID) return;
  const data = await api(`/kbs/${kbID}/documents/${documentID}`);
  state.selectedDocument = data;
  if (includeChunks) {
    const chunks = await api(`/kbs/${kbID}/documents/${documentID}/chunks`);
    state.documentChunks = chunks.items || [];
  } else {
    state.documentChunks = [];
  }
  renderDocumentDetail();
}

async function loadAgents() {
  if (!state.tenantId) return;
  const data = await api("/agents");
  state.agents = data.items || [];
  renderAgents();
}

function selectedBindingAgentID() {
  return $("#agentBindingForm select[name='agent_id']")?.value || state.agents[0]?.id || "";
}

async function loadAgentBindings() {
  const agentID = selectedBindingAgentID();
  if (!state.tenantId || !agentID) return;
  const data = await api(`/agents/${agentID}/knowledge-bases`);
  state.agentBindings = data.items || [];
  renderAgentBindings();
}

function selectedConversationAgentID() {
  return $("#agentConversationForm select[name='agent_id']")?.value || state.agents[0]?.id || "";
}

async function loadConversations() {
  const agentID = selectedConversationAgentID();
  if (!state.tenantId || !agentID) return;
  const data = await api(`/agents/${agentID}/conversations`);
  state.conversations = data.items || [];
  if (!state.conversations.some((item) => item.id === state.selectedConversationId)) {
    state.selectedConversationId = state.conversations[0]?.id || "";
  }
  renderConversations();
  if (state.selectedConversationId) {
    await loadMessages(state.selectedConversationId);
  } else {
    state.messages = [];
    renderMessages();
  }
}

async function loadMessages(conversationID = state.selectedConversationId) {
  const agentID = selectedConversationAgentID();
  if (!state.tenantId || !agentID || !conversationID) return;
  const data = await api(`/agents/${agentID}/conversations/${conversationID}/messages`);
  state.selectedConversationId = conversationID;
  state.messages = data.items || [];
  renderConversations();
  renderMessages();
}

async function loadLicenses() {
  if (!state.tenantId) return;
  const data = await api("/licenses");
  state.licenses = data.items || [];
  renderLicenses();
}

async function loadUsage() {
  if (!state.tenantId) return;
  const data = await api("/billing/usage/summary");
  renderUsage(data.items || []);
}

async function loadSubscription() {
  state.subscription = await api("/billing/subscription");
  renderSubscriptionSummary(state.subscription);
  renderMetrics();
}

async function loadRateLimitPolicy() {
  state.rateLimitPolicy = await api("/settings/rate-limit", { tenant: false });
  renderRateLimitPolicy();
}

async function loadOrders() {
  if (!state.tenantId) return;
  const data = await api("/orders");
  state.orders = data.items || [];
  renderOrders();
}

async function loadPayments() {
  if (!state.tenantId) return;
  const data = await api("/payment-orders");
  state.payments = data.items || [];
  renderPayments();
}

async function loadPaymentEvents() {
  if (!state.tenantId) return;
  const data = await api("/payment-callback-events");
  state.paymentEvents = data.items || [];
  renderPaymentEvents();
}

async function loadAnalytics() {
  if (!state.tenantId) return;
  state.analytics = await api("/analytics/summary");
  renderAnalytics();
}

async function loadMembers() {
  if (!state.tenantId) return;
  const data = await api("/tenant/members");
  state.members = data.items || [];
  renderMembers();
  renderMetrics();
}

function auditFiltersFromForm() {
  const form = $("#auditFilterForm");
  if (!form) return state.auditFilters;
  const data = formData(form);
  state.auditFilters = {
    action: (data.action || "").trim(),
    resource_type: (data.resource_type || "").trim(),
    actor_user_id: (data.actor_user_id || "").trim(),
    from: data.from ? new Date(data.from).toISOString() : "",
    to: data.to ? new Date(data.to).toISOString() : "",
    limit: data.limit || "50",
  };
  return state.auditFilters;
}

function auditQuery(cursor = "") {
  const params = new URLSearchParams();
  Object.entries(state.auditFilters).forEach(([key, value]) => {
    if (value) params.set(key, value);
  });
  if (cursor) params.set("cursor", cursor);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

function resetAuditPager() {
  state.auditNextCursor = "";
  state.auditCursorStack = [];
  state.auditPage = 1;
}

async function loadAuditLogs(cursor = "") {
  if (!state.tenantId) return;
  const data = await api(`/audit-logs${auditQuery(cursor)}`);
  state.auditLogs = data.items || [];
  state.auditNextCursor = data.next_cursor || "";
  renderAuditLogs();
}

async function exportAuditLogs() {
  if (!state.tenantId) return;
  auditFiltersFromForm();
  const qs = auditQuery("").replace(/^\?/, "");
  const url = `${state.apiBase}/audit-logs/export${qs ? `?${qs}` : ""}`;
  const headers = {};
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  if (state.tenantId) headers["X-Tenant-ID"] = state.tenantId;
  const response = await fetch(url, { headers });
  if (!response.ok) {
    throw new Error(`导出失败：HTTP ${response.status}`);
  }
  const blob = await response.blob();
  const objectURL = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = objectURL;
  link.download = `audit-logs-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, "")}.csv`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(objectURL);
}

async function loadInvitations() {
  if (!state.tenantId) return;
  const data = await api("/tenant/invitations");
  state.invitations = data.items || [];
  renderInvitations();
  renderMetrics();
}

async function loadWebhooks() {
  if (!state.tenantId) return;
  const data = await api("/webhooks");
  state.webhooks = data.items || [];
  renderWebhooks();
}

async function loadWebhookDeliverySummary() {
  if (!state.tenantId) return;
  state.webhookDeliverySummary = await api("/webhook-deliveries/summary");
  renderWebhookDeliverySummary();
}

async function loadWebhookDeliveries() {
  if (!state.tenantId) return;
  const data = await api(`/webhook-deliveries${webhookDeliveryQuery()}`);
  state.webhookDeliveries = data.items || [];
  renderWebhookDeliveries();
}

function webhookDeliveryFiltersFromForm() {
  const form = $("#webhookDeliveryFilterForm");
  if (!form) return state.webhookDeliveryFilters;
  const data = formData(form);
  state.webhookDeliveryFilters = {
    endpoint_id: data.endpoint_id || "",
    event_type: data.event_type || "",
    status: data.status || "",
    limit: data.limit || "50",
  };
  return state.webhookDeliveryFilters;
}

function webhookDeliveryQuery() {
  const params = new URLSearchParams();
  Object.entries(state.webhookDeliveryFilters).forEach(([key, value]) => {
    if (value) params.set(key, value);
  });
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

function streamHeaders() {
  const headers = { "Content-Type": "application/json" };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  if (state.tenantId) headers["X-Tenant-ID"] = state.tenantId;
  return headers;
}

async function streamAgentConversation(agentID, body) {
  const resultEl = $("#agentConversationResult");
  resultEl.innerHTML = `
    <div class="answer-main"><strong>流式回答</strong><p id="streamAnswerText"></p></div>
    <div id="streamReferenceList" class="reference-list"></div>
    <div id="streamErrorText"></div>
  `;
  const answerEl = $("#streamAnswerText");
  const referenceEl = $("#streamReferenceList");
  const errorEl = $("#streamErrorText");
  const response = await fetch(`${state.apiBase}/agents/${agentID}/chat/stream`, {
    method: "POST",
    headers: streamHeaders(),
    body: JSON.stringify(body),
  });
  if (!response.ok || !response.body) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.message || message;
    } catch {}
    throw new Error(message);
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let finalPayload = null;
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split("\n\n");
    buffer = parts.pop() || "";
    for (const part of parts) {
      const event = parseSSE(part);
      if (!event) continue;
      if (event.event === "delta") {
        answerEl.textContent += event.data?.content || "";
      } else if (event.event === "reference") {
        const ref = event.data || {};
        referenceEl.insertAdjacentHTML("beforeend", `<article class="reference-item"><strong>${escapeHtml(ref.title || "引用片段")}</strong><span>相关度：${Number(ref.score || 0).toFixed(3)}</span></article>`);
      } else if (event.event === "done") {
        finalPayload = event.data;
      } else if (event.event === "error") {
        const message = event.data?.message || "流式会话失败";
        errorEl.innerHTML = `<div class="item item-meta danger-text">错误：${escapeHtml(message)}</div>`;
        return { ok: false, message };
      }
    }
  }
  if (finalPayload?.conversation_id) {
    state.selectedConversationId = finalPayload.conversation_id;
  }
  toast("流式会话已完成");
  return { ok: true, message: "流式会话已完成" };
}

function parseSSE(chunk) {
  const lines = chunk.split("\n");
  let event = "message";
  const dataLines = [];
  for (const line of lines) {
    if (line.startsWith("event:")) event = line.slice(6).trim();
    if (line.startsWith("data:")) dataLines.push(line.slice(5).trim());
  }
  if (!dataLines.length) return null;
  try {
    return { event, data: JSON.parse(dataLines.join("\n")) };
  } catch {
    return { event, data: { content: dataLines.join("\n") } };
  }
}

async function refreshAll() {
  if (!state.token) {
    renderShell();
    return;
  }
  await loadTenants();
  await Promise.allSettled([
    loadKbs(),
    loadFiles(),
    loadDocuments(),
    loadAgents(),
    loadAgentBindings(),
    loadConversations(),
    loadLicenses(),
    loadUsage(),
    loadOrders(),
    loadPayments(),
    loadPaymentEvents(),
    loadAnalytics(),
    loadSubscription(),
    loadRateLimitPolicy(),
    loadMembers(),
    loadAuditLogs(),
    loadInvitations(),
    loadWebhooks(),
    loadWebhookDeliverySummary(),
    loadWebhookDeliveries(),
  ]);
  renderShell();
}

function switchView(name) {
  $$(".nav-item").forEach((btn) => btn.classList.toggle("active", btn.dataset.view === name));
  $$(".view").forEach((panel) => panel.classList.toggle("active", panel.dataset.viewPanel === name));
  const titles = {
    overview: ["总览", "租户、知识库、智能体和授权状态"],
    knowledge: ["知识库", "创建知识库并测试 RAG 问答"],
    agents: ["智能体", "创建智能体并发起测试会话"],
    licenses: ["授权", "租户 License 生命周期管理"],
    usage: ["用量与订单", "订阅、配额、支付和回调审计"],
    analytics: ["数据分析", "租户资产、经营状态、用量趋势和风险洞察"],
    webhooks: ["Webhook", "外部系统通知、测试发送和投递记录"],
    settings: ["连接", "管理台 API 连接配置"],
  };
  setText("viewTitle", titles[name][0]);
  setText("viewSubtitle", titles[name][1]);
}

function bindEvents() {
  $$(".nav-item").forEach((btn) => btn.addEventListener("click", () => switchView(btn.dataset.view)));
  $("#refreshBtn").addEventListener("click", () => refreshAll().then(() => toast("已刷新")).catch((err) => toast(err.message)));
  $("#logoutBtn").addEventListener("click", () => {
    state.token = "";
    state.tenantId = "";
    localStorage.removeItem("mu.token");
    localStorage.removeItem("mu.tenantId");
    renderShell();
  });
  $("#tenantSelect").addEventListener("change", async (event) => {
    state.tenantId = event.target.value;
    localStorage.setItem("mu.tenantId", state.tenantId);
    resetAuditPager();
    await Promise.allSettled([
      loadKbs(),
      loadFiles(),
      loadDocuments(),
      loadAgents(),
      loadAgentBindings(),
      loadConversations(),
      loadLicenses(),
      loadUsage(),
      loadOrders(),
      loadPayments(),
      loadPaymentEvents(),
      loadAnalytics(),
      loadSubscription(),
      loadRateLimitPolicy(),
      loadMembers(),
      loadAuditLogs(),
      loadInvitations(),
      loadWebhooks(),
      loadWebhookDeliverySummary(),
      loadWebhookDeliveries(),
    ]);
    renderTenants();
  });

  $("#authForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const action = event.submitter?.dataset.authAction || "login";
    const body = {
      email: $("#emailInput").value.trim(),
      password: $("#passwordInput").value,
    };
    if (action === "register") body.nickname = $("#nicknameInput").value.trim();
    try {
      const data = await api(`/auth/${action}`, { method: "POST", body, tenant: false });
      state.token = data.access_token;
      localStorage.setItem("mu.token", state.token);
      await refreshAll();
      toast(action === "login" ? "登录成功" : "注册成功");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#createTenantBtn").addEventListener("click", async () => {
    const stamp = Date.now().toString().slice(-6);
    try {
      await api("/tenants", { method: "POST", body: { name: `新租户${stamp}`, code: `tenant_${stamp}` }, tenant: false });
      await loadTenants();
      toast("租户已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#checkHealthBtn").addEventListener("click", async () => {
    try {
      const [ready, health] = await Promise.all([api("/ready", { tenant: false }), api("/health", { tenant: false })]);
      state.health = { status: ready?.status || health?.status || "ready", checked_at: new Date().toISOString(), ready, health };
      renderHealthSummary(state.health);
      renderMetrics();
    } catch (err) {
      state.health = { status: "error", checked_at: new Date().toISOString(), message: err.message };
      renderHealthSummary({ error: err.message, checked_at: state.health.checked_at });
      renderMetrics();
    }
  });

  $("#settingsForm").addEventListener("submit", (event) => {
    event.preventDefault();
    state.apiBase = $("#apiBaseInput").value.trim().replace(/\/$/, "");
    localStorage.setItem("mu.apiBase", state.apiBase);
    toast("连接配置已保存");
  });

  $("#loadRateLimitPolicyBtn").addEventListener("click", () => {
    loadRateLimitPolicy().then(() => toast("限流策略已刷新")).catch((err) => renderRateLimitPolicy(err.message));
  });

  $("#kbForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const data = formData(event.currentTarget);
      await api("/kbs", { method: "POST", body: { ...data, embedding_dim: 1536 } });
      event.currentTarget.reset();
      await loadKbs();
      toast("知识库已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#askForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      const result = await api(`/kbs/${data.kb_id}/ask`, {
        method: "POST",
        body: { question: data.question, top_k: 5, candidate_k: 25, min_score: 0 },
      });
      renderAnswerSummary("#askResult", result);
      await loadUsage();
    } catch (err) {
      renderAnswerSummary("#askResult", null, err.message);
    }
  });

  $("#fileUploadForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const input = event.currentTarget.elements.file;
    const file = input.files?.[0];
    if (!file) {
      toast("请选择文件");
      return;
    }
    const body = new FormData();
    body.append("file", file);
    try {
      await api("/files/upload", { method: "POST", body });
      event.currentTarget.reset();
      await loadFiles();
      toast("文件已上传");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#documentJobForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      await api(`/kbs/${data.kb_id}/document-jobs`, {
        method: "POST",
        body: {
          file_id: data.file_id,
          job_type: "parse_chunk",
          title: data.title,
          max_chars: 1200,
          overlap_chars: 120,
        },
      });
      await loadDocumentJobs();
      toast("文档任务已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#runDocumentJobsBtn").addEventListener("click", async () => {
    const kbID = selectedJobKbID();
    if (!kbID) {
      toast("请选择知识库");
      return;
    }
    try {
      const result = await api(`/kbs/${kbID}/document-jobs/run`, { method: "POST", body: { limit: 5 } });
      await Promise.allSettled([loadDocumentJobs(), loadPendingChunks(), loadDocuments()]);
      toast(`任务执行完成：成功 ${result.processed || 0}，失败 ${result.failed || 0}`);
    } catch (err) {
      toast(err.message);
    }
  });

  $("#runEmbeddingBtn").addEventListener("click", async () => {
    const kbID = selectedPendingKbID();
    if (!kbID) {
      toast("请选择知识库");
      return;
    }
    try {
      const result = await api(`/kbs/${kbID}/embedding/run`, { method: "POST", body: { limit: 20 } });
      await Promise.allSettled([loadPendingChunks(), loadUsage()]);
      toast(`向量化完成：成功 ${result.processed || 0}，失败 ${result.failed || 0}`);
    } catch (err) {
      toast(err.message);
    }
  });

  $("#documentJobForm select[name='kb_id']").addEventListener("change", () => loadDocumentJobs().catch((err) => toast(err.message)));
  $("#pendingChunkFilterForm select[name='kb_id']").addEventListener("change", () => loadPendingChunks().catch((err) => toast(err.message)));
  $("#documentListFilterForm select[name='kb_id']").addEventListener("change", () => loadDocuments().catch((err) => toast(err.message)));
  $("#agentBindingForm select[name='agent_id']").addEventListener("change", () => loadAgentBindings().catch((err) => toast(err.message)));
  $("#agentConversationForm select[name='agent_id']").addEventListener("change", () => {
    state.selectedConversationId = "";
    state.messages = [];
    loadConversations().catch((err) => toast(err.message));
  });

  $("#agentForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      const body = {
        name: data.name,
        description: data.description,
        system_prompt: data.system_prompt,
        model_config: {},
        tool_policy: {},
        memory_policy: {},
      };
      if (data.agent_id) {
        await api(`/agents/${data.agent_id}`, { method: "PUT", body });
      } else {
        await api("/agents", { method: "POST", body: { ...body, code: data.code } });
      }
      resetAgentForm();
      await loadAgents();
      toast(data.agent_id ? "智能体已更新" : "智能体已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#resetAgentFormBtn").addEventListener("click", resetAgentForm);

  $("#agentChatForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    const body = { message: data.message, min_score: 0 };
    if (data.knowledge_base_id) body.knowledge_base_id = data.knowledge_base_id;
    try {
      const result = await api(`/agents/${data.agent_id}/test-chat`, { method: "POST", body });
      renderAnswerSummary("#agentChatResult", result);
      await loadUsage();
    } catch (err) {
      renderAnswerSummary("#agentChatResult", null, err.message);
    }
  });

  $("#agentBindingForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      await api(`/agents/${data.agent_id}/knowledge-bases`, {
        method: "POST",
        body: { knowledge_base_id: data.knowledge_base_id, metadata: { priority: 1 } },
      });
      await loadAgentBindings();
      toast("知识库已绑定");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#agentConversationForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    const body = {
      conversation_id: state.selectedConversationId,
      message: data.message,
      history_limit: 20,
      min_score: 0,
    };
    if (data.knowledge_base_id) body.knowledge_base_id = data.knowledge_base_id;
    try {
      const result = await api(`/agents/${data.agent_id}/chat`, { method: "POST", body });
      state.selectedConversationId = result.conversation_id;
      renderAnswerSummary("#agentConversationResult", result);
      event.currentTarget.elements.message.value = "";
      await Promise.allSettled([loadConversations(), loadUsage()]);
      toast(`多轮会话已回复，使用历史 ${result.history_used || 0} 条`);
    } catch (err) {
      renderAnswerSummary("#agentConversationResult", null, err.message);
    }
  });

  $("#streamConversationBtn").addEventListener("click", async () => {
    const form = $("#agentConversationForm");
    const data = formData(form);
    const body = {
      conversation_id: state.selectedConversationId,
      message: data.message,
      history_limit: 20,
      min_score: 0,
    };
    if (data.knowledge_base_id) body.knowledge_base_id = data.knowledge_base_id;
    try {
      if (!data.agent_id || !data.message) {
        throw new Error("请选择智能体并输入消息");
      }
      const streamResult = await streamAgentConversation(data.agent_id, body);
      if (streamResult.ok) {
        form.elements.message.value = "";
        await Promise.allSettled([loadConversations(), loadUsage()]);
      } else {
        toast(streamResult.message);
      }
    } catch (err) {
      renderAnswerSummary("#agentConversationResult", null, err.message);
    }
  });

  $("#newConversationBtn").addEventListener("click", () => {
    state.selectedConversationId = "";
    state.messages = [];
    renderConversations();
    renderMessages();
    $("#agentConversationResult").textContent = "已切换到新会话";
  });

  $("#licenseForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    const body = {
      license_type: data.license_type,
      subject: { tenant_id: state.tenantId },
      limits: {
        rag_requests: Number(data.rag_requests || 0),
        agent_messages: Number(data.agent_messages || 0),
      },
    };
    if (data.expired_at) body.expired_at = new Date(data.expired_at).toISOString();
    try {
      await api("/licenses", { method: "POST", body });
      await loadLicenses();
      toast("License 已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#orderForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      await api("/orders", {
        method: "POST",
        body: {
          order_type: "subscription",
          plan_code: data.plan_code,
          amount_cents: Number(data.amount_cents || 0),
        },
      });
      await loadOrders();
      toast("业务订单已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#memberForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      await api("/tenant/members", { method: "POST", body: data });
      event.currentTarget.reset();
      await Promise.allSettled([loadMembers(), loadAuditLogs()]);
      toast("成员已添加");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#invitationForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      const item = await api("/tenant/invitations", { method: "POST", body: { ...data, ttl_hours: 168 } });
      state.invitations = [item, ...state.invitations];
      renderInvitations();
      await loadAuditLogs();
      toast("邀请已创建，token 仅显示一次");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#acceptInvitationForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    try {
      const result = await api("/tenant-invitations/accept", { method: "POST", body: data, tenant: false });
      renderAcceptInvitationResult(result);
      await loadTenants();
      toast("邀请已接受");
    } catch (err) {
      renderAcceptInvitationResult(null, err.message);
    }
  });

  $("#webhookForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = formData(event.currentTarget);
    const events = Array.from(event.currentTarget.elements.events.selectedOptions).map((option) => option.value);
    try {
      await api("/webhooks", {
        method: "POST",
        body: {
          name: data.name,
          url: data.url,
          secret: data.secret,
          status: data.status,
          events,
        },
      });
      event.currentTarget.reset();
      Array.from(event.currentTarget.elements.events.options).forEach((option) => { option.selected = true; });
      await loadWebhooks();
      toast("Webhook 已创建");
    } catch (err) {
      toast(err.message);
    }
  });

  $("#webhookDeliveryFilterForm").addEventListener("submit", (event) => {
    event.preventDefault();
    webhookDeliveryFiltersFromForm();
    Promise.all([loadWebhookDeliverySummary(), loadWebhookDeliveries()]).catch((err) => toast(err.message));
  });

  $("#resetWebhookDeliveryFilterBtn").addEventListener("click", () => {
    state.webhookDeliveryFilters = {
      endpoint_id: "",
      event_type: "",
      status: "",
      limit: "50",
    };
    const form = $("#webhookDeliveryFilterForm");
    form.reset();
    form.elements.limit.value = "50";
    Promise.all([loadWebhookDeliverySummary(), loadWebhookDeliveries()]).catch((err) => toast(err.message));
  });

  $("#loadKbsBtn").addEventListener("click", () => loadKbs().catch((err) => toast(err.message)));
  $("#loadFilesBtn").addEventListener("click", () => loadFiles().catch((err) => toast(err.message)));
  $("#loadDocumentsBtn").addEventListener("click", () => loadDocuments().catch((err) => toast(err.message)));
  $("#loadDocumentJobsBtn").addEventListener("click", () => loadDocumentJobs().catch((err) => toast(err.message)));
  $("#loadPendingChunksBtn").addEventListener("click", () => loadPendingChunks().catch((err) => toast(err.message)));
  $("#loadAgentsBtn").addEventListener("click", () => loadAgents().catch((err) => toast(err.message)));
  $("#loadAgentBindingsBtn").addEventListener("click", () => loadAgentBindings().catch((err) => toast(err.message)));
  $("#loadConversationsBtn").addEventListener("click", () => loadConversations().catch((err) => toast(err.message)));
  $("#loadLicensesBtn").addEventListener("click", () => loadLicenses().catch((err) => toast(err.message)));
  $("#loadUsageBtn").addEventListener("click", () => loadUsage().catch((err) => toast(err.message)));
  $("#loadOrdersBtn").addEventListener("click", () => loadOrders().catch((err) => toast(err.message)));
  $("#loadPaymentsBtn").addEventListener("click", () => loadPayments().catch((err) => toast(err.message)));
  $("#loadPaymentEventsBtn").addEventListener("click", () => loadPaymentEvents().catch((err) => toast(err.message)));
  $("#loadAnalyticsBtn").addEventListener("click", () => loadAnalytics().then(() => toast("分析已刷新")).catch((err) => toast(err.message)));
  $("#loadMembersBtn").addEventListener("click", () => loadMembers().catch((err) => toast(err.message)));
  $("#loadInvitationsBtn").addEventListener("click", () => loadInvitations().catch((err) => toast(err.message)));
  $("#loadWebhooksBtn").addEventListener("click", () => loadWebhooks().catch((err) => toast(err.message)));
  $("#loadWebhookDeliverySummaryBtn").addEventListener("click", () => {
    loadWebhookDeliverySummary().then(() => toast("投递摘要已刷新")).catch((err) => renderWebhookDeliverySummary(err.message));
  });
  $("#loadWebhookDeliveriesBtn").addEventListener("click", () => {
    webhookDeliveryFiltersFromForm();
    Promise.all([loadWebhookDeliverySummary(), loadWebhookDeliveries()]).catch((err) => toast(err.message));
  });
  $("#loadSubscriptionBtn").addEventListener("click", async () => {
    try {
      await loadSubscription();
    } catch (err) {
      renderSubscriptionSummary(null, err.message);
    }
  });

  $("#loadAuditLogsBtn").addEventListener("click", () => {
    resetAuditPager();
    auditFiltersFromForm();
    loadAuditLogs().catch((err) => toast(err.message));
  });
  $("#auditFilterForm").addEventListener("submit", (event) => {
    event.preventDefault();
    resetAuditPager();
    auditFiltersFromForm();
    loadAuditLogs().catch((err) => toast(err.message));
  });
  $("#exportAuditLogsBtn").addEventListener("click", () => exportAuditLogs().then(() => toast("审计日志已导出")).catch((err) => toast(err.message)));
  $("#auditNextBtn").addEventListener("click", () => {
    if (!state.auditNextCursor) return;
    state.auditCursorStack.push(state.auditNextCursor);
    state.auditPage += 1;
    loadAuditLogs(state.auditNextCursor).catch((err) => toast(err.message));
  });
  $("#auditPrevBtn").addEventListener("click", () => {
    if (!state.auditCursorStack.length) return;
    state.auditCursorStack.pop();
    state.auditPage = Math.max(1, state.auditPage - 1);
    const cursor = state.auditCursorStack[state.auditCursorStack.length - 1] || "";
    loadAuditLogs(cursor).catch((err) => toast(err.message));
  });

  document.addEventListener("click", async (event) => {
    const target = event.target;
    const editAgentId = target.dataset?.agentEdit;
    const publishId = target.dataset?.agentPublish;
    const rollbackId = target.dataset?.agentRollback;
    const archiveAgentId = target.dataset?.agentArchive;
    const verifyId = target.dataset?.licenseVerify;
    const activateId = target.dataset?.licenseActivate;
    const revokeId = target.dataset?.licenseRevoke;
    const orderPayId = target.dataset?.orderPay;
    const orderCloseId = target.dataset?.orderClose;
    const orderCancelId = target.dataset?.orderCancel;
    const paymentQueryId = target.dataset?.paymentQuery;
    const paymentCloseId = target.dataset?.paymentClose;
    const memberRoleId = target.dataset?.memberRole;
    const memberRemoveId = target.dataset?.memberRemove;
    const invitationRevokeId = target.dataset?.invitationRevoke;
    const documentDetailId = target.dataset?.documentDetail;
    const documentChunksId = target.dataset?.documentChunks;
    const fileDownloadId = target.dataset?.fileDownload;
    const documentArchiveId = target.dataset?.documentArchive;
    const webhookTestId = target.dataset?.webhookTest;
    const webhookToggleId = target.dataset?.webhookToggle;
    const webhookDeleteId = target.dataset?.webhookDelete;
    const webhookDeliveryRetryId = target.dataset?.webhookDeliveryRetry;
    const conversationSelectId = target.closest("[data-conversation-select]")?.dataset.conversationSelect;
    const agentKbUnbindId = target.dataset?.agentKbUnbind;
    const viewLink = target.closest("[data-view-link]")?.dataset.viewLink;

    try {
      if (viewLink) {
        event.preventDefault();
        switchView(viewLink);
      }
      if (editAgentId) {
        fillAgentForm(editAgentId);
        toast("智能体已载入编辑表单");
      }
      if (agentKbUnbindId) {
        const agentID = selectedBindingAgentID();
        if (!agentID) throw new Error("请选择智能体");
        await api(`/agents/${agentID}/knowledge-bases/${agentKbUnbindId}`, { method: "DELETE" });
        await loadAgentBindings();
        toast("知识库已解绑");
      }
      if (conversationSelectId) {
        await loadMessages(conversationSelectId);
        toast("会话消息已加载");
      }
      if (fileDownloadId) {
        await downloadFile(fileDownloadId);
        toast("文件下载已开始");
      }
      if (documentArchiveId) {
        const kbID = selectedDocumentKbID();
        if (!kbID) throw new Error("请选择知识库");
        await api(`/kbs/${kbID}/documents/${documentArchiveId}`, { method: "DELETE" });
        state.selectedDocument = null;
        state.documentChunks = [];
        await Promise.allSettled([loadDocuments(), loadPendingChunks(), loadDocumentJobs()]);
        toast("文档已归档");
      }
      if (documentDetailId) {
        await loadDocumentDetail(documentDetailId, false);
        toast("文档详情已加载");
      }
      if (documentChunksId) {
        await loadDocumentDetail(documentChunksId, true);
        toast("文档 Chunk 已加载");
      }
      if (webhookTestId) {
        const delivery = await api(`/webhooks/${webhookTestId}/test`, { method: "POST", body: {} });
        await Promise.allSettled([loadWebhookDeliverySummary(), loadWebhookDeliveries()]);
        toast(delivery.status === "success" ? "Webhook 测试发送成功" : "Webhook 测试发送失败");
      }
      if (webhookToggleId) {
        const item = state.webhooks.find((hook) => hook.id === webhookToggleId);
        if (!item) throw new Error("Webhook 不存在");
        await api(`/webhooks/${webhookToggleId}`, { method: "PUT", body: { status: item.status === "active" ? "disabled" : "active" } });
        await loadWebhooks();
        toast("Webhook 状态已更新");
      }
      if (webhookDeleteId) {
        if (!window.confirm("确认删除该 Webhook 配置？")) return;
        await api(`/webhooks/${webhookDeleteId}`, { method: "DELETE" });
        await Promise.allSettled([loadWebhooks(), loadWebhookDeliverySummary(), loadWebhookDeliveries()]);
        toast("Webhook 已删除");
      }
      if (webhookDeliveryRetryId) {
        const delivery = await api(`/webhook-deliveries/${webhookDeliveryRetryId}/retry`, { method: "POST", body: {} });
        await Promise.allSettled([loadWebhookDeliverySummary(), loadWebhookDeliveries()]);
        toast(delivery.status === "success" ? "Webhook 重试发送成功" : "Webhook 重试发送失败");
      }
      if (publishId) {
        await api(`/agents/${publishId}/publish`, { method: "POST" });
        await loadAgents();
      }
      if (rollbackId) {
        await api(`/agents/${rollbackId}/rollback`, { method: "POST" });
        await loadAgents();
      }
      if (archiveAgentId) {
        const agent = state.agents.find((item) => item.id === archiveAgentId);
        if (!window.confirm(`确认归档智能体 ${agent?.name || archiveAgentId}？`)) return;
        await api(`/agents/${archiveAgentId}`, { method: "DELETE" });
        state.agentBindings = [];
        state.conversations = [];
        state.selectedConversationId = "";
        state.messages = [];
        await Promise.allSettled([loadAgents(), loadAgentBindings(), loadConversations()]);
        renderAgentBindings();
        renderConversations();
        renderMessages();
        toast("智能体已归档");
      }
      if (verifyId) {
        const result = await api(`/licenses/${verifyId}/verify`, { method: "POST" });
        toast(result.valid ? "License 验证通过" : `License 验证失败：${result.message}`);
      }
      if (activateId) {
        await api(`/licenses/${activateId}/activate`, { method: "POST" });
        await loadLicenses();
      }
      if (revokeId) {
        await api(`/licenses/${revokeId}/revoke`, { method: "POST" });
        await loadLicenses();
      }
      if (orderPayId) {
        const pay = await api("/payment-orders", { method: "POST", body: { business_order_id: orderPayId, channel: "mock" } });
        await api("/payment-callbacks/mock", {
          method: "POST",
          body: { pay_no: pay.pay_no, status: "paid", transaction_id: `mock-${Date.now()}` },
        });
        await Promise.allSettled([loadOrders(), loadPayments(), loadPaymentEvents(), loadSubscription()]);
        toast("mock 支付已完成");
      }
      if (orderCloseId) {
        await api(`/orders/${orderCloseId}/close`, { method: "POST", body: { reason: "manual close from console" } });
        await Promise.allSettled([loadOrders(), loadPayments()]);
        toast("订单已关闭");
      }
      if (orderCancelId) {
        await api(`/orders/${orderCancelId}/cancel`, { method: "POST", body: { reason: "manual cancel from console" } });
        await Promise.allSettled([loadOrders(), loadPayments()]);
        toast("订单已取消");
      }
      if (paymentQueryId) {
        const result = await api(`/payments/${paymentQueryId}/query`, { method: "POST" });
        toast(`支付单状态：${result.status}`);
      }
      if (paymentCloseId) {
        await api(`/payments/${paymentCloseId}/close`, { method: "POST", body: { reason: "manual close from console" } });
        await loadPayments();
        toast("支付单已关闭");
      }
      if (memberRoleId) {
        await api(`/tenant/members/${memberRoleId}/role`, { method: "PUT", body: { role_code: target.dataset.roleCode } });
        await Promise.allSettled([loadMembers(), loadAuditLogs()]);
        toast("成员角色已更新");
      }
      if (memberRemoveId) {
        await api(`/tenant/members/${memberRemoveId}`, { method: "DELETE" });
        await Promise.allSettled([loadMembers(), loadAuditLogs()]);
        toast("成员已移除");
      }
      if (invitationRevokeId) {
        await api(`/tenant/invitations/${invitationRevokeId}/revoke`, { method: "POST" });
        await Promise.allSettled([loadInvitations(), loadAuditLogs()]);
        toast("邀请已撤销");
      }
    } catch (err) {
      toast(err.message);
    }
  });
}

bindEvents();
renderShell();
refreshAll().catch(() => renderShell());
