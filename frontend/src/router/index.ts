import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import LoginView from "@/views/LoginView.vue";
import ShellLayout from "@/layouts/ShellLayout.vue";
import StudentActivities from "@/views/student/StudentActivities.vue";
import StudentSubmit from "@/views/student/StudentSubmit.vue";
import StudentApplications from "@/views/student/StudentApplications.vue";
import StudentScores from "@/views/student/StudentScores.vue";
import PublicityWall from "@/views/student/PublicityWall.vue";
import ReviewDesk from "@/views/reviewer/ReviewDesk.vue";
import ClassHistory from "@/views/reviewer/ClassHistory.vue";
import ActivityAdmin from "@/views/admin/ActivityAdmin.vue";
import AppealDesk from "@/views/admin/AppealDesk.vue";
import RecheckDesk from "@/views/admin/RecheckDesk.vue";
import DataAdmin from "@/views/admin/DataAdmin.vue";
import AccountsAdmin from "@/views/admin/AccountsAdmin.vue";

export const rolePages = {
  student: [
    { name: "student.activities", label: "首页 / 活动广场" },
    { name: "student.submit", label: "提交申请" },
    { name: "student.applications", label: "我的申请" },
    { name: "student.scores", label: "我的成绩" },
    { name: "student.publicity", label: "公示墙" },
  ],
  reviewer: [
    { name: "reviewer.desk", label: "审核工作台" },
    { name: "reviewer.history", label: "班级申请历史" },
  ],
  admin: [
    { name: "admin.activities", label: "活动名录管理" },
    { name: "admin.recheck", label: "复审工作台" },
    { name: "admin.appeals", label: "申诉工作台" },
    { name: "admin.data", label: "成绩总览与数据管理" },
    { name: "admin.accounts", label: "账号与班级管理" },
  ],
} as const;

const routes: RouteRecordRaw[] = [
  { path: "/login", name: "login", component: LoginView },
  {
    path: "/",
    component: ShellLayout,
    meta: { requiresAuth: true },
    children: [
      { path: "", redirect: { name: "student.activities" } },
      { path: "student/activities", name: "student.activities", component: StudentActivities, meta: { roles: ["student"] } },
      { path: "student/submit", name: "student.submit", component: StudentSubmit, meta: { roles: ["student"] } },
      { path: "student/applications", name: "student.applications", component: StudentApplications, meta: { roles: ["student"] } },
      { path: "student/scores", name: "student.scores", component: StudentScores, meta: { roles: ["student"] } },
      { path: "student/publicity", name: "student.publicity", component: PublicityWall, meta: { roles: ["student"] } },
      { path: "reviewer/desk", name: "reviewer.desk", component: ReviewDesk, meta: { roles: ["reviewer"] } },
      { path: "reviewer/history", name: "reviewer.history", component: ClassHistory, meta: { roles: ["reviewer"] } },
      { path: "admin/activities", name: "admin.activities", component: ActivityAdmin, meta: { roles: ["admin"] } },
      { path: "admin/recheck", name: "admin.recheck", component: RecheckDesk, meta: { roles: ["admin"] } },
      { path: "admin/appeals", name: "admin.appeals", component: AppealDesk, meta: { roles: ["admin"] } },
      { path: "admin/data", name: "admin.data", component: DataAdmin, meta: { roles: ["admin"] } },
      { path: "admin/accounts", name: "admin.accounts", component: AccountsAdmin, meta: { roles: ["admin"] } },
    ],
  },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (to.name === "login") return true;
  const user = await auth.ensureMe();
  if (!user) return { name: "login" };
  const roles = to.meta.roles as string[] | undefined;
  if (roles && !roles.includes(user.role)) {
    return { name: rolePages[user.role][0].name };
  }
  return true;
});
