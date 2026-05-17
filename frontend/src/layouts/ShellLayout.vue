<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { rolePages } from "@/router";
import { roleLabels, useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import type { Role } from "@/api/types";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
const route = useRoute();
const router = useRouter();

const pages = computed(() => (auth.user ? rolePages[auth.user.role] : []));
const pageTitle = computed(() => pages.value.find((item) => item.name === route.name)?.label || "综测服务大厅");
const role = computed<Role>(() => auth.user?.role || "student");

onMounted(async () => {
  if (!auth.meta) await auth.refreshMeta();
});

async function logout() {
  await auth.logout();
  router.push({ name: "login" });
}
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">
          <img class="brand-logo" src="/sdu_pic.svg" alt="山东大学" />
        </div>
        <div>
          <h1>山东大学 · 政治学与公共管理学院 · 综测服务大厅</h1>
          <p>Comprehensive Assessment Service Hall</p>
        </div>
      </div>
      <div class="role-switch">
        <button
          v-for="candidate in (['student', 'reviewer', 'admin'] as Role[])"
          :key="candidate"
          class="role-button"
          :class="{ active: role === candidate }"
          disabled
        >
          {{ roleLabels[candidate] }}
        </button>
      </div>
      <nav class="nav-list">
        <RouterLink v-for="page in pages" :key="page.name" class="nav-item" :to="{ name: page.name }">
          {{ page.label }}
        </RouterLink>
      </nav>
    </aside>

    <main class="main">
      <header class="topbar">
        <div>
          <p class="eyebrow">{{ auth.meta?.summaryPhase ? "综测汇总期" : "日常申报期" }}</p>
          <h2>{{ pageTitle }}</h2>
        </div>
        <div class="topbar-actions">
          <span class="user-badge">{{ auth.user?.displayName }} · {{ auth.user ? roleLabels[auth.user.role] : "" }}</span>
          <button class="secondary logout-button" @click="logout">退出</button>
        </div>
      </header>
      <section v-if="workspace.error" class="notice-strip error">{{ workspace.error }}</section>
      <section v-if="workspace.toast" class="notice-strip">{{ workspace.toast }}</section>
      <section class="content">
        <RouterView />
      </section>
    </main>
  </div>
</template>
