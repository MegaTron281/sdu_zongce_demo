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

const state = {
  role: "student",
  page: "activity",
  summaryPhase: false,
  selectedActivityId: "a1",
};

const activities = [
  {
    id: "a1",
    name: "省级大学生创新创业训练计划",
    organizer: "省教育厅",
    level: "省级",
    category: "创新素养",
    range: "6-12 分",
    deadline: "2026-04-28",
    source: "https://example.edu/notice/innovation",
    applications: 42,
    rule: "按获奖等级计入成果性评价，集体项目按位次递减。",
  },
  {
    id: "a2",
    name: "校级劳动实践示范项目",
    organizer: "学生工作部",
    level: "校级",
    category: "劳动素养",
    range: "3-8 分",
    deadline: "2026-05-12",
    source: "https://example.edu/notice/labor",
    applications: 86,
    rule: "需提交实践证明和学院认定材料，模块上限 25 分。",
  },
  {
    id: "a3",
    name: "学院文艺展演优秀节目评选",
    organizer: "信息工程学院",
    level: "院级",
    category: "文艺素养",
    range: "1-5 分",
    deadline: "2026-04-27",
    source: "https://example.edu/notice/art",
    applications: 31,
    rule: "个人项目按等级计分，集体项目需填写本人位次。",
  },
  {
    id: "a4",
    name: "校运会阳光体育竞赛",
    organizer: "体育教学部",
    level: "校级",
    category: "身心素养",
    range: "2-8 分",
    deadline: "2026-06-01",
    source: "https://example.edu/notice/sports",
    applications: 57,
    rule: "比赛成绩需可在官方成绩册中核验。",
  },
];

const applications = [
  {
    id: "s1",
    student: "李明",
    className: "软件 2301",
    activity: "省级大学生创新创业训练计划",
    category: "创新素养",
    score: 10,
    status: "公示中",
    submittedAt: "2026-04-22",
    countdown: "剩余 1 天 8 小时",
    reviewerOpinion: "材料齐全，建议通过。",
    adminOpinion: "待复审",
  },
  {
    id: "s2",
    student: "陈欣",
    className: "软件 2301",
    activity: "校级劳动实践示范项目",
    category: "劳动素养",
    score: 6,
    status: "待班委审核",
    submittedAt: "2026-04-24",
    countdown: "",
    reviewerOpinion: "",
    adminOpinion: "",
  },
  {
    id: "s3",
    student: "王悦",
    className: "软件 2301",
    activity: "学院文艺展演优秀节目评选",
    category: "文艺素养",
    score: 4,
    status: "待管理员审核",
    submittedAt: "2026-04-20",
    countdown: "",
    reviewerOpinion: "已核验官网公示。",
    adminOpinion: "等待复审",
  },
  {
    id: "s4",
    student: "赵然",
    className: "软件 2302",
    activity: "校运会阳光体育竞赛",
    category: "身心素养",
    score: 5,
    status: "已通过",
    submittedAt: "2026-04-10",
    countdown: "",
    reviewerOpinion: "通过。",
    adminOpinion: "计分入账。",
  },
  {
    id: "s5",
    student: "李明",
    className: "软件 2301",
    activity: "学院志愿服务活动补录",
    category: "劳动素养",
    score: 0,
    status: "已驳回",
    submittedAt: "2026-04-18",
    countdown: "",
    reviewerOpinion: "缺少主办方盖章证明。",
    adminOpinion: "",
  },
];

const scoreModules = [
  ["身心素养", 18, 25],
  ["文艺素养", 14, 25],
  ["劳动素养", 21, 25],
  ["创新素养", 23, 25],
];

function $(selector) {
  return document.querySelector(selector);
}

function daysUntil(dateText) {
  const now = new Date("2026-04-26T00:00:00+08:00");
  const deadline = new Date(`${dateText}T23:59:59+08:00`);
  return Math.ceil((deadline - now) / 86400000);
}

function statusTag(status) {
  const colorMap = {
    "待班委审核": "amber",
    "公示中": "blue",
    "待管理员审核": "teal",
    "已通过": "green",
    "已驳回": "red",
    "特殊情况处理中": "amber",
  };
  return `<span class="tag ${colorMap[status] || ""}">${status}</span>`;
}

function setRole(role) {
  state.role = role;
  state.page = pages[role][0][0];
  render();
}

function setPage(page) {
  state.page = page;
  render();
}

function renderShell() {
  document.querySelectorAll(".role-button").forEach((button) => {
    button.classList.toggle("active", button.dataset.role === state.role);
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
  $("#noticeStrip").innerHTML = `
    <div class="notice task-strip">
      <span class="task-main">正在办理中的事项</span>
      <strong>${state.summaryPhase ? "12" : "暂无"}</strong>
      <span>办结事项</span>
      <strong>108</strong>
      <span>待我评价</span>
      <strong>108</strong>
      <span class="task-phase">${state.summaryPhase ? "综测汇总期：复审、异议、导出开放" : "日常申报期：申请、初审、班级公示开放"}</span>
    </div>
  `;
}

function render() {
  renderShell();
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

function renderActivitySquare() {
  return `
    <section class="portal-search">
      <div class="search-box">
        <input id="activitySearch" placeholder="搜索您需要的综测活动 / Search Service" />
        <span class="search-icon">⌕</span>
      </div>
      <div class="hot-search">
        <span>最近搜索：</span>
        <span>创新竞赛</span>
        <span>劳动实践</span>
        <span>公示异议</span>
        <span>成绩导出</span>
      </div>
    </section>

    <section class="service-panel">
      <div class="portal-tabs">
        <button class="portal-tab active">服务列表</button>
        <button class="portal-tab">我的收藏</button>
        <button class="portal-tab">推荐服务</button>
      </div>

      <div class="filter-row">
        <div class="filter-title">服务分类</div>
        <div class="filter-options">
          ${["全部", "身心素养", "文艺素养", "劳动素养", "创新素养", "补录申请", "公示异议", "成绩查询"]
            .map((item, index) => `<button class="filter-chip ${index === 0 ? "active" : ""}">${item}</button>`)
            .join("")}
        </div>
      </div>
      <div class="filter-row">
        <div class="filter-title">活动级别</div>
        <div class="filter-options">
          ${["全部", "国家级", "省级", "校级", "院级"]
            .map((item, index) => `<button class="filter-chip ${index === 0 ? "active" : ""}">${item}</button>`)
            .join("")}
        </div>
      </div>
      <div class="filter-row">
        <div class="filter-title">项目名称</div>
        <div class="filter-options">
          ${["全部", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "W", "X", "Y", "Z"]
            .map((item, index) => `<button class="filter-chip ${index === 0 ? "active" : ""}">${item}</button>`)
            .join("")}
        </div>
      </div>

      <div class="sort-row">
        <button>默认排序</button>
        <button>截止时间</button>
        <button>申请数</button>
      </div>

      <div class="service-card-grid">
        ${activities
          .map((activity) => {
            const quick = daysUntil(activity.deadline) <= 3;
            return `
              <article class="service-card">
                <div class="service-fav">
                  ${quick ? '<span class="tag red">快截止</span>' : "设为收藏 ☆"}
                </div>
                <div class="service-body">
                  <div class="service-icon">${activity.category.slice(0, 1)}</div>
                  <div class="service-info">
                    <h3>${activity.name}</h3>
                    <p>${activity.organizer} · ${activity.level} · ${activity.range}</p>
                    <div class="service-actions">
                      <button class="secondary" data-detail="${activity.id}">详情</button>
                      <button class="primary" data-apply="${activity.id}">申报</button>
                    </div>
                  </div>
                </div>
              </article>
            `;
          })
          .join("")}
      </div>
    </div>
  `;
}

function renderSubmit() {
  const activity = activities.find((item) => item.id === state.selectedActivityId) || activities[0];
  const locked = state.summaryPhase;
  return `
    <div class="split-panel">
      <div>
        <p class="section-label">路径 A：从活动名录发起申请</p>
        <div class="form-grid">
          <div class="field">
            <label>活动名称</label>
            <input class="input" value="${activity.name}" ${locked ? "disabled" : "readonly"} />
          </div>
          <div class="field">
            <label>级别 / 类别</label>
            <input class="input" value="${activity.level} · ${activity.category}" ${locked ? "disabled" : "readonly"} />
          </div>
          <div class="field">
            <label>获奖等级</label>
            <select class="select" ${locked ? "disabled" : ""}><option>一等奖</option><option>二等奖</option><option>三等奖</option><option>其他名次</option></select>
          </div>
          <div class="field">
            <label>项目类型</label>
            <select class="select" ${locked ? "disabled" : ""}><option>个人项目</option><option>集体项目</option></select>
          </div>
          <div class="field">
            <label>集体项目本人位次</label>
            <input class="input" value="1" ${locked ? "disabled" : ""} />
          </div>
          <div class="field">
            <label>证明材料</label>
            <input class="input" value="证书图片、公示截图、官方报道链接" ${locked ? "disabled" : ""} />
          </div>
          <div class="field full">
            <label>补充说明</label>
            <textarea class="textarea" ${locked ? "disabled" : ""}>已上传奖状图片，并附主办方官网公示链接。</textarea>
          </div>
        </div>
        <div class="button-row" style="margin-top: 14px">
          <button class="primary" ${locked ? "disabled" : ""}>提交申请</button>
          <button class="secondary">保存草稿</button>
        </div>
      </div>
      <div>
        <p class="section-label">实时计分预览</p>
        <div class="score-box">
          <div class="score-value">10 分</div>
          <p>规则版本：2026 V1 · 省级一等奖 · 个人项目</p>
          <span class="tag amber">与创新类竞赛互斥，最终最高分取一</span>
        </div>
        <hr />
        <p class="section-label">路径 B：补录新活动</p>
        <p>填写活动名称、主办方、活动级别、类别、活动时间和来源链接后，活动先进入“待审核上架”，申请暂挂，管理员通过后自动进入班委初审。</p>
        <button class="ghost" ${locked ? "disabled" : ""}>切换到补录表单</button>
      </div>
    </div>
  `;
}

function renderMyApplications() {
  const mine = applications.filter((item) => item.student === "李明");
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>活动</th><th>提交时间</th><th>状态</th><th>加分</th><th>审核意见</th><th>操作</th></tr></thead>
        <tbody>
          ${mine
            .map(
              (item) => `
                <tr>
                  <td>${item.activity}<br /><span class="tag">${item.category}</span></td>
                  <td>${item.submittedAt}</td>
                  <td>${statusTag(item.status)} ${item.countdown ? `<br /><span class="meta">${item.countdown}</span>` : ""}</td>
                  <td>${item.score || "待定"}</td>
                  <td>${item.reviewerOpinion || "暂无"}<br />${item.adminOpinion || ""}</td>
                  <td>${item.status === "已驳回" ? '<button class="primary">修改后重新提交</button>' : '<button class="secondary">查看详情</button>'}</td>
                </tr>
              `,
            )
            .join("")}
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
  const total = scoreModules.reduce((sum, [, score]) => sum + score, 0);
  return `
    <div class="stats">
      <div class="stat"><span>素质能力总分</span><strong>${total}</strong></div>
      <div class="stat"><span>学业成绩</span><strong>86.4</strong></div>
      <div class="stat"><span>预估综测成绩</span><strong>86.2</strong></div>
      <div class="stat"><span>页面状态</span><strong>${state.summaryPhase ? "锁定" : "开放"}</strong></div>
    </div>
    <div class="panel">
      <p class="section-label">四模块进度</p>
      <div class="progress-list">
        ${scoreModules
          .map(
            ([name, score, max]) => `
              <div class="progress-row">
                <div class="progress-head"><span>${name}</span><span>${score}/${max} · 距上限 ${max - score} 分</span></div>
                <div class="progress-track"><div class="progress-fill" style="width:${(score / max) * 100}%"></div></div>
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
          ${applications
            .filter((item) => item.student === "李明" && item.status === "公示中")
            .map((item) => `<tr><td>${item.activity}</td><td>${item.category}</td><td>${item.score}</td><td>公示期满无异议后入账</td></tr>`)
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderPublicity() {
  const publicItems = applications.filter((item) => item.status === "公示中");
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>姓名</th><th>活动名称</th><th>申请加分</th><th>剩余公示时间</th><th>异议</th></tr></thead>
        <tbody>
          ${publicItems
            .map(
              (item) => `
                <tr>
                  <td>${item.student}</td>
                  <td>${item.activity}</td>
                  <td>${item.score}</td>
                  <td>${item.countdown}</td>
                  <td><button class="danger">提出异议</button></td>
                </tr>
              `,
            )
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderReviewDesk() {
  const pending = applications.filter((item) => item.status === "待班委审核");
  return `
    <div class="stats">
      <div class="stat"><span>本班待审核</span><strong>${pending.length}</strong></div>
      <div class="stat"><span>公示中</span><strong>1</strong></div>
      <div class="stat"><span>本周通过</span><strong>12</strong></div>
      <div class="stat"><span>特殊情况</span><strong>2</strong></div>
    </div>
    <div class="grid">
      ${pending
        .map(
          (item) => `
            <article class="card">
              <div class="card-header"><h3>${item.student}</h3>${statusTag(item.status)}</div>
              <div class="meta"><span>${item.className}</span><span>${item.submittedAt}</span><span>${item.category}</span></div>
              <p>${item.activity}</p>
              <div class="score-box">
                <strong>建议分值：${item.score} 分</strong>
                <p>规则说明：需核验活动名录、证书图片、官方公示截图或报道链接。</p>
              </div>
              <div class="button-row">
                <button class="primary">通过</button>
                <button class="danger">驳回</button>
                <button class="secondary">标记特殊情况</button>
              </div>
            </article>
          `,
        )
        .join("") || '<div class="empty">暂无待审核申请</div>'}
    </div>
  `;
}

function renderClassHistory() {
  return renderApplicationTable(applications.filter((item) => item.className === "软件 2301"), true);
}

function renderActivityAdmin() {
  return `
    <div class="split-panel">
      <div>
        <p class="section-label">活动名录</p>
        ${renderActivityTable()}
      </div>
      <div>
        <p class="section-label">学生申报待审</p>
        <div class="card">
          <h3>学院志愿服务活动补录</h3>
          <div class="meta"><span>学生填报：院级</span><span>劳动素养</span><span>待审核上架</span></div>
          <p>管理员可修改级别、确认上架，或驳回并要求学生补充材料。</p>
          <div class="button-row">
            <button class="primary">确认上架</button>
            <button class="secondary">修改后上架</button>
            <button class="danger">驳回</button>
          </div>
        </div>
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
  const rows = ["国家级", "省级", "校级", "院级"]
    .map(
      (level, index) => `
        <tr>
          <td>${level}</td>
          <td><input class="input" value="${16 - index * 3}" /></td>
          <td><input class="input" value="${12 - index * 2}" /></td>
          <td><input class="input" value="${8 - index}" /></td>
          <td><input class="input" value="${5 - index}" /></td>
        </tr>
      `,
    )
    .join("");
  return `
    <div class="panel">
      <p class="section-label">规则版本：2026 V1</p>
      <div class="button-row">
        <button class="primary">发布新版本</button>
        <button class="secondary">查看历史版本</button>
        <button class="ghost">复制上一年规则</button>
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
  const items = applications.filter((item) => item.status === "待管理员审核");
  return `
    <div class="grid">
      ${items
        .map(
          (item) => `
            <article class="card">
              <div class="card-header"><h3>${item.student}</h3>${statusTag(item.status)}</div>
              <p>${item.activity}</p>
              <div class="meta"><span>班委意见：${item.reviewerOpinion}</span><span>建议分值 ${item.score} 分</span></div>
              <textarea class="textarea" placeholder="修改分值或特殊情况说明"></textarea>
              <div class="button-row">
                <button class="primary">采纳建议并通过</button>
                <button class="secondary">修改分值后通过</button>
                <button class="danger">驳回</button>
              </div>
            </article>
          `,
        )
        .join("")}
      <article class="card">
        <h3>异议处理</h3>
        <p>李明的创新创业训练计划申请收到 1 条异议，需管理员确认维持原判或重新核查。</p>
        <div class="button-row"><button class="primary">维持原判</button><button class="secondary">重新核查</button></div>
      </article>
    </div>
  `;
}

function renderData() {
  return `
    <div class="stats">
      <div class="stat"><span>全年级学生</span><strong>286</strong></div>
      <div class="stat"><span>平均素质分</span><strong>72.4</strong></div>
      <div class="stat"><span>异常预警</span><strong>5</strong></div>
      <div class="stat"><span>待导出册</span><strong>1</strong></div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>排名</th><th>学生</th><th>班级</th><th>四模块</th><th>素质总分</th><th>综测成绩</th></tr></thead>
        <tbody>
          <tr><td>1</td><td>李明</td><td>软件 2301</td><td>18 / 14 / 21 / 23</td><td>76</td><td>86.2</td></tr>
          <tr><td>2</td><td>王悦</td><td>软件 2301</td><td>20 / 16 / 18 / 18</td><td>72</td><td>85.1</td></tr>
          <tr><td>3</td><td>赵然</td><td>软件 2302</td><td>22 / 12 / 17 / 19</td><td>70</td><td>84.7</td></tr>
        </tbody>
      </table>
    </div>
    <div class="grid">
      <div class="card"><h3>导出综测成绩册</h3><p>包含每位学生四模块分值、素质总分、学业成绩和综测成绩。</p><button class="primary">导出成绩册</button></div>
      <div class="card"><h3>导出申请明细</h3><p>包含申请材料状态、审核意见、公示和异议处理结果。</p><button class="secondary">导出明细</button></div>
      <div class="card"><h3>导出操作日志</h3><p>用于审计班委与管理员的全部关键操作。</p><button class="secondary">导出日志</button></div>
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
    <div class="table-wrap">
      <table>
        <thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>对象</th><th>说明</th></tr></thead>
        <tbody>
          <tr><td>2026-04-26 10:18</td><td>辅导员</td><td>修改规则</td><td>2026 V1</td><td>调整省级二等奖分值</td></tr>
          <tr><td>2026-04-25 16:42</td><td>班委 陈欣</td><td>初审通过</td><td>李明申请</td><td>材料齐全</td></tr>
          <tr><td>2026-04-24 09:12</td><td>辅导员</td><td>设置角色</td><td>陈欣</td><td>授权软件 2301 班委</td></tr>
        </tbody>
      </table>
    </div>
  `;
}

function renderActivityTable() {
  return `
    <div class="table-wrap">
      <table>
        <thead><tr><th>活动名称</th><th>级别</th><th>类别</th><th>申报人数</th><th>状态</th></tr></thead>
        <tbody>
          ${activities
            .map(
              (item) => `
                <tr>
                  <td>${item.name}</td>
                  <td>${item.level}</td>
                  <td>${item.category}</td>
                  <td>${item.applications}</td>
                  <td><span class="tag green">已上架</span></td>
                </tr>
              `,
            )
            .join("")}
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
                  <td>${item.student}</td>
                  <td>${item.className}</td>
                  <td>${item.activity}</td>
                  <td>${statusTag(item.status)}</td>
                  <td>${item.score || "待定"}</td>
                  <td>${item.reviewerOpinion || item.adminOpinion || "暂无"}</td>
                </tr>
              `,
            )
            .join("")}
        </tbody>
      </table>
    </div>
  `;
}

function bindPageEvents() {
  document.querySelectorAll("[data-page-jump]").forEach((button) => {
    button.addEventListener("click", () => setPage(button.dataset.pageJump));
  });

  document.querySelectorAll("[data-apply]").forEach((button) => {
    button.addEventListener("click", () => {
      state.selectedActivityId = button.dataset.apply;
      setPage("submit");
    });
  });

  document.querySelectorAll("[data-detail]").forEach((button) => {
    button.addEventListener("click", () => {
      const activity = activities.find((item) => item.id === button.dataset.detail);
      if (!activity) return;
      alert(`${activity.name}\n\n${activity.rule}\n\n官方通知：${activity.source}`);
    });
  });
}

document.querySelectorAll(".role-button").forEach((button) => {
  button.addEventListener("click", () => setRole(button.dataset.role));
});

$("#navList").addEventListener("click", (event) => {
  const button = event.target.closest("[data-page]");
  if (button) setPage(button.dataset.page);
});

$("#phaseToggle").addEventListener("change", (event) => {
  state.summaryPhase = event.target.checked;
  render();
});

$("#notificationButton").addEventListener("click", () => {
  alert("通知\n1. 李明申请进入班级公示\n2. 王悦申请等待管理员复审\n3. 规则版本 2026 V1 已生效");
});

render();
