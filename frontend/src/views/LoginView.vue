<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { rolePages } from "@/router";
import { roleLabels, useAuthStore } from "@/stores/auth";
import type { Role } from "@/api/types";

const profiles: Record<Role, { account: string; hint: string }> = {
  student: { account: "202300221077", hint: "查看活动、提交申请、追踪个人成绩。" },
  reviewer: { account: "202300202018", hint: "处理本班初审、公示流转和历史追溯。" },
  admin: { account: "zhangmiao", hint: "维护名录、规则复审、汇总导出。" },
};

const router = useRouter();
const auth = useAuthStore();
const selected = ref<Role>("student");
const form = reactive({ account: profiles.student.account, password: "" });
const error = ref("");
const loading = ref(false);

function choose(role: Role) {
  selected.value = role;
  form.account = profiles[role].account;
  form.password = "";
  error.value = "";
}

async function submit() {
  loading.value = true;
  error.value = "";
  try {
    await auth.login(selected.value, form.account, form.password);
    router.push({ name: rolePages[selected.value][0].name });
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
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
      <div class="login-copy">
        <p class="eyebrow">统一身份入口</p>
        <h2>按角色进入申报、审核和管理工作台</h2>
        <p>演示账号使用统一密码 123456。登录后由服务端 session 判断身份，审核日志不再信任前端传入的操作者。</p>
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
          <input v-model="form.account" class="input" />
        </div>
        <div class="field">
          <label>密码</label>
          <input v-model="form.password" class="input" type="password" placeholder="测试密码：123456" />
        </div>
        <div v-if="error" class="feedback error">{{ error }}</div>
        <button class="primary login-submit" type="submit" :disabled="loading">
          {{ loading ? "登录中..." : `登录 ${roleLabels[selected]}` }}
        </button>
      </form>
    </section>
  </div>
</template>
