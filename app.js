const pages = {
  student: [
    ["activity", "首页 / 活动广场"],
    ["submit", "提交申请"],
    ["myApplications", "我的申请"],
    ["scores", "我的成绩"],
    ["publicity", "公示墙"],
  ],
  reviewer: [
    ["reviewDesk", "审核工作台"],
    ["classHistory", "班级申请历史"],
  ],
  admin: [
    ["activityAdmin", "活动名录管理"],
    ["rules", "加分规则配置"],
    ["recheck", "复审工作台"],
    ["data", "成绩总览与数据管理"],
    ["accounts", "账号与班级管理"],
  ],
};

const loginProfiles = {
  student: {
    title: "普通学生",
    account: "20230001",
    password: "123456",
    hint: "查看活动、提交申请、追踪个人成绩。",
  },
  reviewer: {
    title: "审核学生",
    account: "20230002",
    password: "123456",
    hint: "处理本班初审、公示流转和历史追溯。",
  },
  admin: {
    title: "管理员",
    account: "辅导员 王老师",
    password: "123456",
    hint: "维护名录、规则复审、汇总导出。",
  },
};

let currentStudent = {
  student: "李明",
  studentId: "20230001",
  className: "软件 2301",
};

const state = {
  user: null,
  role: "student",
  page: "activity",
  summaryPhase: false,
  selectedActivityId: "a2",
  loading: false,
  loginRole: "student",
  loginError: "",
  error: "",
  toast: "",
  filters: {
    q: "",
    category: "全部",
    level: "全部",
  },
  sort: "default",
  draft: {
    awardLevel: "一等奖",
    projectType: "个人项目",
    rank: 1,
    materials: "证书图片、公示截图、官方报道链接",
    note: "已上传奖状图片，并附主办方官网公示链接。",
  },
  data: {
    activities: [],
    applications: [],
    publicity: [],
    score: null,
    scoreRows: [],
    rules: [],
    logs: [],
  },
};

let searchTimer = 0;

function $(selector) {
  return document.querySelector(selector);
}

function esc(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function formatNumber(value) {
  const number = Number(value || 0);
  return Number.isInteger(number) ? String(number) : number.toFixed(1);
}

function formatDateTime(value) {
  if (!value) return "暂无";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function daysUntil(dateText) {
  const deadline = new Date(`${dateText}T23:59:59+08:00`);
  if (Number.isNaN(deadline.getTime())) return 999;
  return Math.ceil((deadline - new Date()) / 86400000);
}

function publicityCountdown(value) {
  if (!value) return "待公示";
  const deadline = new Date(value);
  const diff = deadline - new Date();
  if (Number.isNaN(deadline.getTime())) return "时间待确认";
  if (diff <= 0) return "已到期";
  const hours = Math.floor(diff / 3600000);
  const minutes = Math.floor((diff % 3600000) / 60000);
  if (hours >= 24) return `剩余 ${Math.floor(hours / 24)} 天 ${hours % 24} 小时`;
  return `剩余 ${hours} 小时 ${minutes} 分钟`;
}

function statusTag(status) {
  const colorMap = {
    "待班委审核": "amber",
    "公示中": "blue",
    "待管理员审核": "teal",
    "已通过": "green",
    "已驳回": "red",
    "特殊情况处理中": "amber",
    "已上架": "green",
    "待审核上架": "amber",
  };
  return `<span class="tag ${colorMap[status] || ""}">${esc(status)}</span>`;
}

function roleLabel(role) {
  return loginProfiles[role]?.title || role;
}

async function api(path, options = {}) {
  const init = {
    method: options.method || "GET",
    headers: { Accept: "application/json", ...(options.headers || {}) },
  };
  if (options.body !== undefined) {
    init.headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(options.body);
  }
  const response = await fetch(path, init);
  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok) {
    const message = payload && payload.error ? payload.error : `请求失败：${response.status}`;
    throw new Error(message);
  }
  return payload;
}

async function loadActivities() {
  state.data.activities = await api("/api/activities");
  if (!state.data.activities.some((item) => item.id === state.selectedActivityId)) {
    state.selectedActivityId = state.data.activities[0]?.id || "";
  }
}

async function loadApplications() {
  state.data.applications = await api("/api/applications");
  const ids = [...new Set(state.data.applications.map((item) => item.studentId).filter(Boolean))];
  state.data.scoreRows = await Promise.all(ids.map((id) => api(`/api/scores/${encodeURIComponent(id)}`)));
}

async function loadAll() {
  state.loading = true;
  state.error = "";
  $("#appShell").classList.remove("auth-hidden");
  $("#loginRoot").classList.add("auth-hidden");
  render();
  try {
    const meta = await api("/api/meta");
    state.summaryPhase = Boolean(meta.summaryPhase);
    await Promise.all([loadActivities(), loadApplications(), loadRules(), loadLogs()]);
    state.data.publicity = await api("/api/publicity");
    state.data.score = await api(`/api/scores/${currentStudent.studentId}`);
  } catch (error) {
    state.error = error.message;
  } finally {
    state.loading = false;
    render();
  }
}

async function refreshData(message = "") {
  state.error = "";
  try {
    await Promise.all([loadActivities(), loadApplications(), loadRules(), loadLogs()]);
    state.data.publicity = await api("/api/publicity");
    state.data.score = await api(`/api/scores/${currentStudent.studentId}`);
    state.toast = message;
  } catch (error) {
    state.error = error.message;
  }
  render();
}

async function loadRules() {
  state.data.rules = await api("/api/admin/rules");
}

async function loadLogs() {
  state.data.logs = await api("/api/admin/logs");
}

function applyUser(user) {
  state.user = user;
  state.role = user.role;
  state.page = pages[user.role][0][0];
  currentStudent = {
    student: user.displayName,
    studentId: user.studentId || "20230001",
    className: user.className || "软件 2301",
  };
}

async function login(role, account, password) {
  state.loginError = "";
  try {
    const user = await api("/api/auth/login", {
      method: "POST",
      body: { role, account, password },
    });
    applyUser(user);
    await loadAll();
  } catch (error) {
    state.loginError = error.message;
    renderLogin();
  }
}

function logout() {
  state.user = null;
  state.loading = false;
  state.error = "";
  state.toast = "";
  $("#appShell").classList.add("auth-hidden");
  $("#loginRoot").classList.remove("auth-hidden");
  renderLogin();
}

function setRole(role) {
  if (!state.user || role !== state.user.role) return;
  state.role = role;
  state.page = pages[role][0][0];
  render();
}

function setPage(page) {
  state.page = page;
  render();
}

function renderShell() {
  if (!state.user) return;
  document.querySelectorAll(".role-button").forEach((button) => {
    button.classList.toggle("active", button.dataset.role === state.role);
    button.disabled = button.dataset.role !== state.role;
  });

  const nav = $("#navList");
  nav.innerHTML = pages[state.role]
    .map(
      ([id, label]) =>
        `<button class="nav-item ${state.page === id ? "active" : ""}" data-page="${id}">${label}</button>`,
    )
    .join("");

  const current = pages[state.role].find(([id]) => id === state.page);
  $("#pageTitle").textContent = current ? current[1] : "";
  $("#phaseLabel").textContent = state.summaryPhase ? "综测汇总期" : "日常申报期";
  $("#phaseToggle").checked = state.summaryPhase;
  $("#userBadge").textContent = `${roleLabel(state.user.role)} · ${state.user.displayName}`;
  $("#phaseToggle").disabled = state.role !== "admin";

  const pending = state.data.applications.filter((item) =>
    ["待班委审核", "公示中", "待管理员审核", "特殊情况处理中"].includes(item.status),
  ).length;
  const finished = state.data.applications.filter((item) => ["已通过", "已驳回"].includes(item.status)).length;
  const reviewTodo = state.role === "admin"
    ? state.data.applications.filter((item) => ["待管理员审核", "特殊情况处理中"].includes(item.status)).length
    : state.data.applications.filter((item) => item.status === "待班委审核").length;

  $("#noticeStrip").innerHTML = `
    ${state.error ? `<div class="feedback error">${esc(state.error)}</div>` : ""}
    ${state.toast ? `<div class="feedback success">${esc(state.toast)}</div>` : ""}
    <div class="notice task-strip">
      <span class="task-main">正在办理中的事项</span>
      <strong>${pending}</strong>
      <span>办结事项</span>
      <strong>${finished}</strong>
      <span>待我处理</span>
      <strong>${reviewTodo}</strong>
      <span class="task-phase">${state.summaryPhase ? "综测汇总期：复审、异议、导出开放" : "日常申报期：申请、初审、班级公示开放"}</span>
    </div>
  `;
}

function render() {
  if (!state.user) {
    renderLogin();
    return;
  }
  renderShell();
  if (state.loading) {
    $("#content").innerHTML = '<div class="loading-state">正在加载平台数据...</div>';
    return;
  }
  const renderMap = {
    activity: renderActivitySquare,
    submit: renderSubmit,
    myApplications: renderMyApplications,
    scores: renderScores,
    publicity: renderPublicity,
    reviewDesk: renderReviewDesk,
    classHistory: renderClassHistory,
    activityAdmin: renderActivityAdmin,
    rules: renderRules,
    recheck: renderRecheck,
    data: renderData,
    accounts: renderAccounts,
  };
  $("#content").innerHTML = renderMap[state.page]();
  bindPageEvents();
}

function renderLogin() {
  const root = $("#loginRoot");
  const profile = loginProfiles[state.loginRole];
  $("#appShell").classList.add("auth-hidden");
  root.classList.remove("auth-hidden");
  root.innerHTML = `
    <div class="login-page">
      <section class="login-hero">
        <div class="brand login-brand">
          <div class="brand-mark">SDU</div>
          <div>
            <h1>山东大学 · 综测服务大厅</h1>
            <p>Comprehensive Assessment Service Hall</p>
          </div>
        </div>
        <div class="login-copy">
          <p class="eyebrow">统一身份入口</p>
          <h2>按角色进入申报、审核和管理工作台</h2>
          <p>演示密码为 123456。选择身份后会进入对应权限页面，普通学生、审核学生、管理员看到的导航和默认数据各不相同。</p>
        </div>
      </section>

      <section class="login-panel">
        <div class="login-role-grid">
          ${Object.entries(loginProfiles)
            .map(
              ([role, item]) => `
                <button class="login-role ${state.loginRole === role ? "active" : ""}" data-login-role="${role}">
                  <strong>${item.title}</strong>
                  <span>${item.hint}</span>
                </button>
              `,
            )
            .join("")}
        </div>
        <form id="loginForm" class="login-form">
          <div class="field">
            <label>登录身份</label>
            <input class="input" value="${esc(profile.title)}" readonly />
          </div>
          <div class="field">
            <label>账号</label>
            <input class="input" name="account" value="${esc(profile.account)}" />
          </div>
          <div class="field">
            <label>密码</label>
            <input class="input" name="password" type="password" value="${esc(profile.password)}" />
          </div>
          ${state.loginError ? `<div class="feedback error">${esc(state.loginError)}</div>` : ""}
          <button class="primary login-submit" type="submit">登录 ${esc(profile.title)}</button>
        </form>
      </section>
    </div>
  `;
  bindLoginEvents();
}

function renderActivitySquare() {
  const visibleActivities = sortedActivities();
  return `
    <section class="portal-search">
      <div class="search-box">
        <input id="activitySearch" value="${esc(state.filters.q)}" placeholder="搜索您需要的综测活动 / Search Service" />
        <span class="search-icon">⌕</span>
      </div>
      <div class="hot-search">
        <span>最近搜索：</span>
        <button data-search="创新竞赛">创新竞赛</button>
        <button data-search="劳动实践">劳动实践</button>
        <button data-search="公示异议">公示异议</button>
        <button data-search="">全部活动</button>
      </div>
    </section>

    <section class="service-panel">
      <div class="portal-tabs">
        <button class="portal-tab active">服务列表</button>
        <button class="portal-tab">我的收藏</button>
        <button class="portal-tab">推荐服务</button>
      </div>

      ${renderFilterRow("服务分类", "category", ["全部", "身心素养", "文艺素养", "劳动素养", "创新素养"])}
      ${renderFilterRow("活动级别", "level", ["全部", "国家级", "省级", "校级", "院级"])}

      <div class="sort-row">
        <button class="${state.sort === "default" ? "active" : ""}" data-sort="default">默认排序</button>
        <button class="${state.sort === "deadline" ? "active" : ""}" data-sort="deadline">截止时间</button>
        <button class="${state.sort === "applications" ? "active" : ""}" data-sort="applications">申请数</button>
      </div>

      <div class="service-card-grid">
        ${visibleActivities
          .map((activity) => {
            const quick = daysUntil(activity.deadline) <= 3;
            return `
              <article class="service-card">
                <div class="service-fav">
                  ${quick ? '<span class="tag red">快截止</span>' : "设为收藏 ☆"}
                </div>
                <div class="service-body">
                  <div class="service-icon">${esc(activity.category.slice(0, 1))}</div>
                  <div class="service-info">
                    <h3>${esc(activity.name)}</h3>
                    <p>${esc(activity.organizer)} · ${esc(activity.level)} · ${esc(activity.range)}</p>
                    <div class="service-actions">
                      <button class="secondary" data-detail="${esc(activity.id)}">详情</button>
                      <button class="primary" data-apply="${esc(activity.id)}" ${state.summaryPhase ? "disabled" : ""}>申报</button>
                    </div>
                  </div>
                </div>
              </article>
            `;
          })
          .join("") || '<div class="empty">没有匹配的活动</div>'}
      </div>
    </section>
  `;
}

function sortedActivities() {
  const keyword = state.filters.q.toLowerCase();
  const items = state.data.activities.filter((activity) => {
    const matchKeyword =
      !keyword ||
      `${activity.name}${activity.organizer}${activity.category}${activity.level}`.toLowerCase().includes(keyword);
    const matchCategory = state.filters.category === "全部" || activity.category === state.filters.category;
    const matchLevel = state.filters.level === "全部" || activity.level === state.filters.level;
    return matchKeyword && matchCategory && matchLevel;
  });
  return items.sort((a, b) => {
    if (state.sort === "deadline") return String(a.deadline).localeCompare(String(b.deadline));
    if (state.sort === "applications") return Number(b.applications || 0) - Number(a.applications || 0);
    return String(a.id).localeCompare(String(b.id));
  });
}

function renderFilterRow(title, key, options) {
  return `
    <div class="filter-row">
      <div class="filter-title">${title}</div>
      <div class="filter-options">
        ${options
          .map(
            (item) =>
              `<button class="filter-chip ${state.filters[key] === item ? "active" : ""}" data-filter-key="${key}" data-filter-value="${item}">${item}</button>`,
          )
          .join("")}
      </div>
    </div>
  `;
}

function selectedActivity() {
  return state.data.activities.find((item) => item.id === state.selectedActivityId) || state.data.activities[0];
}

function calculatePreviewScore(activity) {
  if (!activity) return 0;
  const matrix = {
    "国家级:一等奖": 16,
    "国家级:二等奖": 12,
    "国家级:三等奖": 8,
    "国家级:其他名次": 5,
    "省级:一等奖": 12,
    "省级:二等奖": 10,
    "省级:三等奖": 7,
    "省级:其他名次": 4,
    "校级:一等奖": 8,
    "校级:二等奖": 6,
    "校级:三等奖": 4,
    "校级:其他名次": 2,
    "院级:一等奖": 5,
    "院级:二等奖": 4,
    "院级:三等奖": 3,
    "院级:其他名次": 1,
  };
  const base = matrix[`${activity.level}:${state.draft.awardLevel}`] || 0;
  if (state.draft.projectType === "集体项目" && Number(state.draft.rank) > 1) {
    return Math.max(base * (1 - (Number(state.draft.rank) - 1) * 0.2), base * 0.3);
  }
  return base;
}

function renderSubmit() {
  const activity = selectedActivity();
  if (!activity) return '<div class="empty">请先在活动广场选择可申报活动</div>';
  const locked = state.summaryPhase;
  const preview = calculatePreviewScore(activity);
  return `
    <div class="split-panel">
      <div>
        <p class="section-label">路径 A：从活动名录发起申请</p>
        <form id="applicationForm">
          <div class="form-grid">
            <div class="field">
              <label>活动名称</label>
              <input class="input" value="${esc(activity.name)}" readonly />
            </div>
            <div class="field">
              <label>级别 / 类别</label>
              <input class="input" value="${esc(activity.level)} · ${esc(activity.category)}" readonly />
            </div>
            <div class="field">
              <label>获奖等级</label>
              <select class="select" name="awardLevel" ${locked ? "disabled" : ""}>${renderOptions(["一等奖", "二等奖", "三等奖", "其他名次"], state.draft.awardLevel)}</select>
            </div>
            <div class="field">
              <label>项目类型</label>
              <select class="select" name="projectType" ${locked ? "disabled" : ""}>${renderOptions(["个人项目", "集体项目"], state.draft.projectType)}</select>
            </div>
            <div class="field">
              <label>集体项目本人位次</label>
              <input class="input" name="rank" type="number" min="1" value="${esc(state.draft.rank)}" ${locked ? "disabled" : ""} />
            </div>
            <div class="field">
              <label>证明材料</label>
              <input class="input" name="materials" value="${esc(state.draft.materials)}" ${locked ? "disabled" : ""} />
            </div>
            <div class="field full">
              <label>补充说明</label>
              <textarea class="textarea" name="note" ${locked ? "disabled" : ""}>${esc(state.draft.note)}</textarea>
            </div>
          </div>
          <div class="button-row" style="margin-top: 14px">
            <button class="primary" type="submit" ${locked ? "disabled" : ""}>提交申请</button>
            <button class="secondary" type="button" data-save-draft>保存草稿</button>
          </div>
        </form>
      </div>
      <div>
        <p class="section-label">实时计分预览</p>
        <div class="score-box">
          <div class="score-value">${formatNumber(preview)} 分</div>
          <p>规则版本：${esc(activeRuleName())} · ${esc(activity.level)}${esc(state.draft.awardLevel)} · ${esc(state.draft.projectType)}</p>
          <span class="tag amber">最终分值以后端规则计算为准</span>
        </div>
        <hr />
        <p class="section-label">路径 B：补录新活动</p>
        <form id="activityForm" class="compact-form">
          <input class="input" name="name" placeholder="活动名称" ${locked ? "disabled" : ""} />
          <input class="input" name="organizer" placeholder="主办方" ${locked ? "disabled" : ""} />
          <select class="select" name="level" ${locked ? "disabled" : ""}>${renderOptions(["国家级", "省级", "校级", "院级"], "校级")}</select>
          <select class="select" name="category" ${locked ? "disabled" : ""}>${renderOptions(["身心素养", "文艺素养", "劳动素养", "创新素养"], "劳动素养")}</select>
          <input class="input" name="deadline" type="date" ${locked ? "disabled" : ""} />
          <input class="input" name="source" placeholder="来源链接" ${locked ? "disabled" : ""} />
          <button class="ghost" type="submit" ${locked ? "disabled" : ""}>提交补录</button>
        </form>
      </div>
    </div>
  `;
}

function renderOptions(options, selected) {
  return options.map((item) => `<option ${item === selected ? "selected" : ""}>${esc(item)}</option>`).join("");
}

function activeRuleName() {
  return state.data.rules.find((item) => item.active)?.name || "当前规则";
}

function renderMyApplications() {
  const mine = state.data.applications.filter((item) => item.studentId === currentStudent.studentId);
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>活动</th><th>提交时间</th><th>状态</th><th>加分</th><th>审核意见</th><th>操作</th></tr></thead>
        <tbody>
          ${mine
            .map(
              (item) => `
                <tr>
                  <td>${esc(item.activity)}<br /><span class="tag">${esc(item.category)}</span></td>
                  <td>${esc(item.submittedAt)}</td>
                  <td>${statusTag(item.status)} ${item.publicityUntil ? `<br /><span class="meta">${publicityCountdown(item.publicityUntil)}</span>` : ""}</td>
                  <td>${item.score ? formatNumber(item.score) : "待定"}</td>
                  <td>${esc(item.reviewerOpinion || "暂无")}<br />${esc(item.adminOpinion || "")}</td>
                  <td>${item.status === "已驳回" ? `<button class="primary" data-apply="${esc(item.activityId || state.selectedActivityId)}">修改后重新提交</button>` : `<button class="secondary" data-app-detail="${esc(item.id)}">查看详情</button>`}</td>
                </tr>
              `,
            )
            .join("") || '<tr><td colspan="6">暂无申请记录</td></tr>'}
        </tbody>
      </table>
    </div>
    <div class="panel">
      <p class="section-label">状态流转</p>
      <div class="timeline">
        <div class="timeline-item done"><span class="dot"></span><span>学生提交申请，系统完成防重复和规则冲突预检</span></div>
        <div class="timeline-item done"><span class="dot"></span><span>班委初审通过，进入 2 天班级公示</span></div>
        <div class="timeline-item active"><span class="dot"></span><span>公示期内可接收异议，学生可补充说明材料</span></div>
        <div class="timeline-item"><span class="dot"></span><span>管理员复审通过后进入学院公示并计分入账</span></div>
      </div>
    </div>
  `;
}

function renderScores() {
  const score = state.data.score;
  if (!score) return '<div class="empty">暂无成绩数据</div>';
  const included = state.data.applications.filter(
    (item) => item.studentId === currentStudent.studentId && ["已通过", "公示中"].includes(item.status),
  );
  return `
    <div class="stats">
      <div class="stat"><span>素质能力总分</span><strong>${formatNumber(score.qualityScore)}</strong></div>
      <div class="stat"><span>学业成绩</span><strong>${formatNumber(score.academicScore)}</strong></div>
      <div class="stat"><span>预估综测成绩</span><strong>${formatNumber(score.assessmentScore)}</strong></div>
      <div class="stat"><span>页面状态</span><strong>${score.locked ? "锁定" : "开放"}</strong></div>
    </div>
    <div class="panel">
      <p class="section-label">四模块进度</p>
      <div class="progress-list">
        ${score.modules
          .map(
            (item) => `
              <div class="progress-row">
                <div class="progress-head"><span>${esc(item.name)}</span><span>${formatNumber(item.score)}/${formatNumber(item.max)} · 距上限 ${formatNumber(item.max - item.score)} 分</span></div>
                <div class="progress-track"><div class="progress-fill" style="width:${Math.min(100, (item.score / item.max) * 100)}%"></div></div>
              </div>
            `,
          )
          .join("")}
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>已计入条目</th><th>模块</th><th>实际加分</th><th>入账说明</th></tr></thead>
        <tbody>
          ${included
            .map((item) => `<tr><td>${esc(item.activity)}</td><td>${esc(item.category)}</td><td>${formatNumber(item.score)}</td><td>${item.status === "公示中" ? "公示期满无异议后入账" : "已通过复审"}</td></tr>`)
            .join("") || '<tr><td colspan="4">暂无计入条目</td></tr>'}
        </tbody>
      </table>
    </div>
  `;
}

function renderPublicity() {
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>姓名</th><th>活动名称</th><th>申请加分</th><th>剩余公示时间</th><th>异议</th></tr></thead>
        <tbody>
          ${state.data.publicity
            .map(
              (item) => `
                <tr>
                  <td>${esc(item.student)}</td>
                  <td>${esc(item.activity)}</td>
                  <td>${formatNumber(item.score)}</td>
                  <td>${publicityCountdown(item.publicityUntil)}</td>
                  <td><button class="danger" data-objection="${esc(item.id)}">提出异议</button></td>
                </tr>
              `,
            )
            .join("") || '<tr><td colspan="5">暂无公示中申请</td></tr>'}
        </tbody>
      </table>
    </div>
  `;
}

function renderReviewDesk() {
  const pending = state.data.applications.filter((item) => item.status === "待班委审核");
  const publicity = state.data.applications.filter((item) => item.status === "公示中");
  return `
    <div class="stats">
      <div class="stat"><span>本班待审核</span><strong>${pending.length}</strong></div>
      <div class="stat"><span>公示中</span><strong>${publicity.length}</strong></div>
      <div class="stat"><span>本周通过</span><strong>${state.data.applications.filter((item) => item.status === "已通过").length}</strong></div>
      <div class="stat"><span>特殊情况</span><strong>${state.data.applications.filter((item) => item.status === "特殊情况处理中").length}</strong></div>
    </div>
    <div class="grid">
      ${pending.map(renderReviewCard).join("") || '<div class="empty">暂无待审核申请</div>'}
      ${publicity.map(renderPublicityFinishCard).join("")}
    </div>
  `;
}

function renderReviewCard(item) {
  return `
    <article class="card">
      <div class="card-header"><h3>${esc(item.student)}</h3>${statusTag(item.status)}</div>
      <div class="meta"><span>${esc(item.className)}</span><span>${esc(item.submittedAt)}</span><span>${esc(item.category)}</span></div>
      <p>${esc(item.activity)}</p>
      <div class="score-box">
        <strong>建议分值：${formatNumber(item.score)} 分</strong>
        <p>规则说明：需核验活动名录、证书图片、官方公示截图或报道链接。</p>
      </div>
      <div class="button-row">
        <button class="primary" data-app-action="review-pass" data-app-id="${esc(item.id)}">通过</button>
        <button class="danger" data-app-action="review-reject" data-app-id="${esc(item.id)}">驳回</button>
        <button class="secondary" data-app-action="mark-special" data-app-id="${esc(item.id)}">标记特殊情况</button>
      </div>
    </article>
  `;
}

function renderPublicityFinishCard(item) {
  return `
    <article class="card">
      <div class="card-header"><h3>${esc(item.student)}</h3>${statusTag(item.status)}</div>
      <p>${esc(item.activity)}</p>
      <div class="meta"><span>${publicityCountdown(item.publicityUntil)}</span><span>建议分值 ${formatNumber(item.score)} 分</span></div>
      <div class="button-row">
        <button class="secondary" data-app-action="publicity-finish" data-app-id="${esc(item.id)}">结束公示并送复审</button>
      </div>
    </article>
  `;
}

function renderClassHistory() {
  return renderApplicationTable(state.data.applications.filter((item) => item.className === currentStudent.className));
}

function renderActivityAdmin() {
  const pendingActivities = state.data.activities.filter((item) => item.status === "待审核上架");
  return `
    <div class="split-panel">
      <div>
        <p class="section-label">活动名录</p>
        ${renderActivityTable()}
      </div>
      <div>
        <p class="section-label">学生申报待审</p>
        ${pendingActivities
          .map(
            (item) => `
              <div class="card">
                <h3>${esc(item.name)}</h3>
                <div class="meta"><span>${esc(item.level)}</span><span>${esc(item.category)}</span>${statusTag(item.status)}</div>
                <p>${esc(item.organizer || "主办方待确认")} · ${esc(item.source || "来源待补充")}</p>
                <div class="button-row">
                  <button class="primary" data-activity-status="已上架" data-activity-id="${esc(item.id)}">确认上架</button>
                  <button class="danger" data-activity-status="已驳回" data-activity-id="${esc(item.id)}">驳回</button>
                </div>
              </div>
            `,
          )
          .join("") || '<div class="empty">暂无待审核活动</div>'}
        <div class="card">
          <h3>爬虫抓取接口</h3>
          <p>预留半自动抓取入口，后续可将公众号链接解析结果放入待确认队列。</p>
          <button class="ghost">导入抓取结果</button>
        </div>
      </div>
    </div>
  `;
}

function renderRules() {
  const activeRule = state.data.rules.find((item) => item.active) || state.data.rules[0];
  const rows = ["国家级", "省级", "校级", "院级"]
    .map((level) => {
      const matrix = activeRule?.matrix || {};
      return `
        <tr>
          <td>${level}</td>
          <td><input class="input" data-rule="${level}:一等奖" value="${matrix[`${level}:一等奖`] ?? ""}" /></td>
          <td><input class="input" data-rule="${level}:二等奖" value="${matrix[`${level}:二等奖`] ?? ""}" /></td>
          <td><input class="input" data-rule="${level}:三等奖" value="${matrix[`${level}:三等奖`] ?? ""}" /></td>
          <td><input class="input" data-rule="${level}:其他名次" value="${matrix[`${level}:其他名次`] ?? ""}" /></td>
        </tr>
      `;
    })
    .join("");
  return `
    <div class="panel">
      <p class="section-label">规则版本：${esc(activeRule?.name || "未发布")}</p>
      <div class="button-row">
        <button class="primary" data-publish-rules>发布新版本</button>
        <button class="secondary" data-show-rules>查看历史版本</button>
        <button class="ghost" data-copy-rule>复制上一年规则</button>
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>活动级别</th><th>一等奖</th><th>二等奖</th><th>三等奖</th><th>其他</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
    </div>
    <div class="grid">
      <div class="card"><h3>集体项目位次递减</h3><p>第一位按满分，之后每位递减 20%，最低不低于 30%。</p></div>
      <div class="card"><h3>模块上限</h3><p>身心、文艺、劳动、创新各 25 分，素质能力总分上限 100 分。</p></div>
      <div class="card"><h3>一事一议</h3><p>无名录活动、证明缺失、跨类别冲突自动标记人工介入。</p></div>
    </div>
  `;
}

function renderRecheck() {
  const items = state.data.applications.filter((item) => ["待管理员审核", "特殊情况处理中"].includes(item.status));
  return `
    <div class="grid">
      ${items
        .map(
          (item) => `
            <article class="card">
              <div class="card-header"><h3>${esc(item.student)}</h3>${statusTag(item.status)}</div>
              <p>${esc(item.activity)}</p>
              <div class="meta"><span>班委意见：${esc(item.reviewerOpinion || item.specialReason || "暂无")}</span><span>建议分值 ${formatNumber(item.score)} 分</span></div>
              <input class="input" data-admin-score="${esc(item.id)}" type="number" min="0" step="0.5" value="${esc(item.score)}" />
              <textarea class="textarea" data-admin-note="${esc(item.id)}" placeholder="修改分值或特殊情况说明"></textarea>
              <div class="button-row">
                <button class="primary" data-app-action="admin-pass" data-app-id="${esc(item.id)}">采纳建议并通过</button>
                <button class="secondary" data-app-action="admin-pass-custom" data-app-id="${esc(item.id)}">修改分值后通过</button>
                <button class="danger" data-app-action="admin-reject" data-app-id="${esc(item.id)}">驳回</button>
              </div>
            </article>
          `,
        )
        .join("") || '<div class="empty">暂无待复审申请</div>'}
      <article class="card">
        <h3>异议处理</h3>
        <p>公示异议会进入一事一议流程，由管理员确认维持原判或重新核查。</p>
        <div class="button-row"><button class="primary">维持原判</button><button class="secondary">重新核查</button></div>
      </article>
    </div>
  `;
}

function renderData() {
  const rows = [...state.data.scoreRows].sort((a, b) => b.assessmentScore - a.assessmentScore);
  const average = rows.length ? rows.reduce((sum, item) => sum + item.qualityScore, 0) / rows.length : 0;
  return `
    <div class="stats">
      <div class="stat"><span>已有申请学生</span><strong>${rows.length}</strong></div>
      <div class="stat"><span>平均素质分</span><strong>${formatNumber(average)}</strong></div>
      <div class="stat"><span>异常预警</span><strong>${state.data.applications.filter((item) => item.status === "特殊情况处理中").length}</strong></div>
      <div class="stat"><span>待导出册</span><strong>${state.summaryPhase ? "1" : "0"}</strong></div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>排名</th><th>学生</th><th>班级</th><th>四模块</th><th>素质总分</th><th>综测成绩</th></tr></thead>
        <tbody>
          ${rows
            .map(
              (item, index) => `
                <tr>
                  <td>${index + 1}</td>
                  <td>${esc(item.student || item.studentId)}</td>
                  <td>${esc(item.className || "未登记")}</td>
                  <td>${item.modules.map((module) => formatNumber(module.score)).join(" / ")}</td>
                  <td>${formatNumber(item.qualityScore)}</td>
                  <td>${formatNumber(item.assessmentScore)}</td>
                </tr>
              `,
            )
            .join("") || '<tr><td colspan="6">暂无成绩数据</td></tr>'}
        </tbody>
      </table>
    </div>
    <div class="grid">
      <div class="card"><h3>导出综测成绩册</h3><p>包含每位学生四模块分值、素质总分、学业成绩和综测成绩。</p><a class="primary link-button" href="/api/admin/export/scores.csv">导出成绩册</a></div>
      <div class="card"><h3>导出申请明细</h3><p>包含申请材料状态、审核意见、公示和异议处理结果。</p><button class="secondary" data-export-detail>导出明细</button></div>
      <div class="card"><h3>导出操作日志</h3><p>用于审计班委与管理员的全部关键操作。</p><button class="secondary" data-show-logs>查看日志</button></div>
    </div>
  `;
}

function renderAccounts() {
  return `
    <div class="grid">
      <div class="card"><h3>学生账号</h3><p>支持创建单个学生账号，或批量导入学生名单。</p><button class="primary">批量导入</button></div>
      <div class="card"><h3>班委授权</h3><p>为指定学生设置审核学生角色，权限限定在本班级。</p><button class="secondary">设置角色</button></div>
      <div class="card"><h3>综测周期</h3><p>配置申报期、公示期、复审期、学院公示期和导出节点。</p><button class="secondary">编辑周期</button></div>
    </div>
    ${renderLogTable()}
  `;
}

function renderActivityTable() {
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>活动名称</th><th>级别</th><th>类别</th><th>申报人数</th><th>状态</th></tr></thead>
        <tbody>
          ${state.data.activities
            .map(
              (item) => `
                <tr>
                  <td>${esc(item.name)}</td>
                  <td>${esc(item.level)}</td>
                  <td>${esc(item.category)}</td>
                  <td>${item.applications}</td>
                  <td>${statusTag(item.status || "已上架")}</td>
                </tr>
              `,
            )
            .join("") || '<tr><td colspan="5">暂无活动</td></tr>'}
        </tbody>
      </table>
    </div>
  `;
}

function renderApplicationTable(items) {
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>学生</th><th>班级</th><th>活动</th><th>状态</th><th>分值</th><th>意见</th></tr></thead>
        <tbody>
          ${items
            .map(
              (item) => `
                <tr>
                  <td>${esc(item.student)}</td>
                  <td>${esc(item.className)}</td>
                  <td>${esc(item.activity)}</td>
                  <td>${statusTag(item.status)}</td>
                  <td>${item.score ? formatNumber(item.score) : "待定"}</td>
                  <td>${esc(item.reviewerOpinion || item.adminOpinion || item.specialReason || "暂无")}</td>
                </tr>
              `,
            )
            .join("") || '<tr><td colspan="6">暂无申请记录</td></tr>'}
        </tbody>
      </table>
    </div>
  `;
}

function renderLogTable() {
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>对象</th><th>说明</th></tr></thead>
        <tbody>
          ${state.data.logs
            .slice()
            .reverse()
            .map(
              (item) => `
                <tr>
                  <td>${formatDateTime(item.createdAt)}</td>
                  <td>${esc(item.actor)}</td>
                  <td>${esc(item.action)}</td>
                  <td>${esc(item.target)}</td>
                  <td>${esc(item.note)}</td>
                </tr>
              `,
            )
            .join("") || '<tr><td colspan="5">暂无日志</td></tr>'}
        </tbody>
      </table>
    </div>
  `;
}

function collectDraft(form) {
  const data = new FormData(form);
  state.draft.awardLevel = data.get("awardLevel") || state.draft.awardLevel;
  state.draft.projectType = data.get("projectType") || state.draft.projectType;
  state.draft.rank = Number(data.get("rank") || 1);
  state.draft.materials = data.get("materials") || "";
  state.draft.note = data.get("note") || "";
}

function materialList(value) {
  return String(value || "")
    .split(/[、,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

async function handleApplicationAction(button) {
  const id = button.dataset.appId;
  const action = button.dataset.appAction;
  let body = { actor: state.role === "admin" ? "管理员" : "班委 陈欣" };
  let endpointAction = action;

  if (["review-reject", "mark-special", "admin-reject"].includes(action)) {
    const reason = prompt("请输入处理原因");
    if (!reason) return;
    body.reason = reason;
  }
  if (action === "review-pass") {
    body.reason = "材料齐全，建议通过。";
  }
  if (action === "publicity-finish") {
    body.reason = "公示期结束，无有效异议。";
  }
  if (action === "admin-pass") {
    body.reason = "管理员复审通过";
  }
  if (action === "admin-pass-custom") {
    endpointAction = "admin-pass";
    const scoreInput = document.querySelector(`[data-admin-score="${CSS.escape(id)}"]`);
    const noteInput = document.querySelector(`[data-admin-note="${CSS.escape(id)}"]`);
    body.score = Number(scoreInput?.value || 0);
    body.reason = noteInput?.value.trim() || prompt("请输入改分原因");
    if (!body.reason) return;
  }

  button.disabled = true;
  try {
    await api(`/api/applications/${encodeURIComponent(id)}/${endpointAction}`, {
      method: "POST",
      body,
    });
    await refreshData("操作已提交");
  } catch (error) {
    state.error = error.message;
    render();
  }
}

function bindPageEvents() {
  document.querySelectorAll("[data-apply]").forEach((button) => {
    button.addEventListener("click", () => {
      state.selectedActivityId = button.dataset.apply;
      setPage("submit");
    });
  });

  document.querySelectorAll("[data-detail]").forEach((button) => {
    button.addEventListener("click", () => {
      const activity = state.data.activities.find((item) => item.id === button.dataset.detail);
      if (!activity) return;
      alert(`${activity.name}\n\n${activity.rule}\n\n官方通知：${activity.source}`);
    });
  });

  document.querySelectorAll("[data-filter-key]").forEach((button) => {
    button.addEventListener("click", () => {
      state.filters[button.dataset.filterKey] = button.dataset.filterValue;
      render();
    });
  });

  document.querySelectorAll("[data-search]").forEach((button) => {
    button.addEventListener("click", () => {
      state.filters.q = button.dataset.search;
      render();
    });
  });

  const searchInput = $("#activitySearch");
  if (searchInput) {
    searchInput.addEventListener("input", () => {
      clearTimeout(searchTimer);
      state.filters.q = searchInput.value.trim();
      searchTimer = window.setTimeout(() => render(), 300);
    });
  }

  document.querySelectorAll("[data-sort]").forEach((button) => {
    button.addEventListener("click", () => {
      state.sort = button.dataset.sort;
      render();
    });
  });

  const applicationForm = $("#applicationForm");
  if (applicationForm) {
    applicationForm.addEventListener("change", () => {
      collectDraft(applicationForm);
      render();
    });
    applicationForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      collectDraft(applicationForm);
      const activity = selectedActivity();
      try {
        await api("/api/applications", {
          method: "POST",
          body: {
            student: currentStudent.student,
            studentId: currentStudent.studentId,
            className: currentStudent.className,
            activityId: activity.id,
            awardLevel: state.draft.awardLevel,
            projectType: state.draft.projectType,
            rank: Number(state.draft.rank || 1),
            materials: materialList(state.draft.materials),
          },
        });
        state.page = "myApplications";
        await refreshData("申请已提交，等待班委审核");
      } catch (error) {
        state.error = error.message;
        render();
      }
    });
  }

  const activityForm = $("#activityForm");
  if (activityForm) {
    activityForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const data = new FormData(activityForm);
      try {
        await api("/api/activities", {
          method: "POST",
          body: {
            name: data.get("name"),
            organizer: data.get("organizer"),
            level: data.get("level"),
            category: data.get("category"),
            range: "待管理员确认",
            deadline: data.get("deadline"),
            source: data.get("source"),
            rule: "补录活动，待管理员维护规则说明。",
          },
        });
        await refreshData("补录活动已提交，等待管理员上架");
      } catch (error) {
        state.error = error.message;
        render();
      }
    });
  }

  document.querySelectorAll("[data-save-draft]").forEach((button) => {
    button.addEventListener("click", () => {
      const form = $("#applicationForm");
      if (form) collectDraft(form);
      state.toast = "草稿已保存在当前页面";
      render();
    });
  });

  document.querySelectorAll("[data-app-action]").forEach((button) => {
    button.addEventListener("click", () => handleApplicationAction(button));
  });

  document.querySelectorAll("[data-activity-status]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await api(`/api/activities/${encodeURIComponent(button.dataset.activityId)}`, {
          method: "PATCH",
          body: { status: button.dataset.activityStatus },
        });
        await refreshData("活动状态已更新");
      } catch (error) {
        state.error = error.message;
        render();
      }
    });
  });

  document.querySelectorAll("[data-publish-rules]").forEach((button) => {
    button.addEventListener("click", async () => {
      const matrix = {};
      document.querySelectorAll("[data-rule]").forEach((input) => {
        matrix[input.dataset.rule] = Number(input.value || 0);
      });
      try {
        await api("/api/admin/rules", {
          method: "POST",
          body: {
            name: `规则 ${new Date().toLocaleDateString("zh-CN")}`,
            matrix,
            moduleMax: { 身心素养: 25, 文艺素养: 25, 劳动素养: 25, 创新素养: 25 },
          },
        });
        await refreshData("新规则版本已发布");
      } catch (error) {
        state.error = error.message;
        render();
      }
    });
  });

  document.querySelectorAll("[data-show-rules]").forEach((button) => {
    button.addEventListener("click", () => {
      alert(state.data.rules.map((item) => `${item.name} · ${item.publishedAt} · ${item.active ? "当前" : "历史"}`).join("\n"));
    });
  });

  document.querySelectorAll("[data-copy-rule]").forEach((button) => {
    button.addEventListener("click", () => {
      state.toast = "已复制当前规则，可直接调整后发布";
      render();
    });
  });

  document.querySelectorAll("[data-objection]").forEach((button) => {
    button.addEventListener("click", () => {
      alert(`已记录异议线索：${button.dataset.objection}`);
    });
  });

  document.querySelectorAll("[data-app-detail]").forEach((button) => {
    button.addEventListener("click", () => {
      const item = state.data.applications.find((app) => app.id === button.dataset.appDetail);
      if (!item) return;
      alert(`${item.activity}\n状态：${item.status}\n分值：${formatNumber(item.score)}\n班委意见：${item.reviewerOpinion || "暂无"}\n管理员意见：${item.adminOpinion || "暂无"}`);
    });
  });

  document.querySelectorAll("[data-show-logs]").forEach((button) => {
    button.addEventListener("click", () => {
      alert(state.data.logs.slice(-8).map((item) => `${formatDateTime(item.createdAt)} ${item.actor} ${item.action} ${item.target}`).join("\n"));
    });
  });
}

function bindLoginEvents() {
  document.querySelectorAll("[data-login-role]").forEach((button) => {
    button.addEventListener("click", () => {
      state.loginRole = button.dataset.loginRole;
      state.loginError = "";
      renderLogin();
    });
  });

  $("#loginForm").addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    await login(state.loginRole, data.get("account"), data.get("password"));
  });
}

document.querySelectorAll(".role-button").forEach((button) => {
  button.addEventListener("click", () => setRole(button.dataset.role));
});

$("#navList").addEventListener("click", (event) => {
  const button = event.target.closest("[data-page]");
  if (button) setPage(button.dataset.page);
});

$("#phaseToggle").addEventListener("change", async (event) => {
  if (state.role !== "admin") {
    event.target.checked = state.summaryPhase;
    return;
  }
  const checked = event.target.checked;
  event.target.disabled = true;
  try {
    const meta = await api("/api/meta", {
      method: "PATCH",
      body: { summaryPhase: checked },
    });
    state.summaryPhase = Boolean(meta.summaryPhase);
    await refreshData("综测周期状态已更新");
  } catch (error) {
    state.error = error.message;
    render();
  } finally {
    event.target.disabled = false;
  }
});

$("#notificationButton").addEventListener("click", () => {
  const latest = state.data.logs.slice(-3).reverse();
  alert(`通知\n${latest.map((item, index) => `${index + 1}. ${item.action}：${item.target}`).join("\n") || "暂无新通知"}`);
});

$("#logoutButton").addEventListener("click", logout);

renderLogin();
