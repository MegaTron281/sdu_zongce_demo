<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { ApiError } from "@/api/client";
import { rolePages } from "@/router";
import { roleLabels, useAuthStore } from "@/stores/auth";
import type { Role } from "@/api/types";

const profiles: Record<Role, { hint: string }> = {
  student: { hint: "查看活动、提交申请、追踪个人成绩。" },
  reviewer: { hint: "处理本班初审、公示流转和历史追溯。" },
  admin: { hint: "维护名录、规则复审、汇总导出。" },
};

const router = useRouter();
const auth = useAuthStore();
const selected = ref<Role>("student");
const form = reactive({ account: "", password: "" });
const error = ref("");
const loading = ref(false);

function choose(role: Role) {
  selected.value = role;
  form.account = "";
  form.password = "";
  error.value = "";
}

function friendlyLoginError(err: unknown): string {
  if (err instanceof ApiError) {
    const msg = err.message?.trim();
    // 后端已区分"账号不存在 / 身份不符 / 密码错误"，直接展示
    if (err.status === 401 || err.status === 403) return msg || "登录失败，请检查账号、密码和登录身份";
    if (err.status >= 500) return "服务器繁忙，请稍后重试";
    return msg || "登录失败，请稍后重试";
  }
  if (err instanceof TypeError) return "无法连接服务器，请确认后端服务已启动";
  return "登录失败，请稍后重试";
}

async function submit() {
  if (!form.account.trim()) {
    error.value = "请输入账号";
    return;
  }
  if (!form.password) {
    error.value = "请输入密码";
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    await auth.login(selected.value, form.account.trim(), form.password);
    router.push({ name: rolePages[selected.value][0].name });
  } catch (err) {
    error.value = friendlyLoginError(err);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-page">
    <section class="login-hero">
      <div class="brand login-brand">
        <div class="brand-mark">
          <img class="brand-logo" src="/sdu_pic.svg" alt="山东大学" />
        </div>
        <div>
          <h1>山东大学 · 政治学与公共管理学院 · 综测服务大厅</h1>
          <p>Comprehensive Assessment Service Hall</p>
        </div>
      </div>
    </section>

    <section class="login-panel">
      <div class="login-role-grid">
        <button
          v-for="(profile, role) in profiles"
          :key="role"
          class="login-role"
          :class="{ active: selected === role }"
          @click="choose(role as Role)"
        >
          <strong>{{ roleLabels[role as Role] }}</strong>
          <span>{{ profile.hint }}</span>
        </button>
      </div>
      <form class="login-form" @submit.prevent="submit">
        <div class="field">
          <label>登录身份</label>
          <input class="input" :value="roleLabels[selected]" readonly />
        </div>
        <div class="field">
          <label>账号</label>
          <input v-model="form.account" class="input" placeholder="请输入学号或账号" autocomplete="username" />
        </div>
        <div class="field">
          <label>密码</label>
          <input v-model="form.password" class="input" type="password" placeholder="默认密码：123456" autocomplete="current-password" />
        </div>
        <div v-if="error" class="feedback error" role="alert">⚠ {{ error }}</div>
        <button class="primary login-submit" type="submit" :disabled="loading">
          {{ loading ? "登录中..." : `登录 ${roleLabels[selected]}` }}
        </button>
      </form>
    </section>
  </div>
</template>
