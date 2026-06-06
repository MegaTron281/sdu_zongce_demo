<script setup lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { daysUntil } from "@/api/format";
import StatusTag from "@/components/StatusTag.vue";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
const router = useRouter();
const filters = reactive({ q: "", category: "全部", level: "全部" });

onMounted(() => auth.user?.studentId && workspace.loadStudent(auth.user.studentId));

const activities = computed(() =>
  workspace.activities.filter((item) => {
    const keyword = `${item.name}${item.organizer}${item.category}`;
    return (
      (!filters.q || keyword.includes(filters.q)) &&
      (filters.category === "全部" || item.category === filters.category) &&
      (filters.level === "全部" || item.level === filters.level)
    );
  }),
);

function apply(id: string) {
  router.push({ name: "student.submit", query: { activityId: id } });
}
</script>

<template>
  <div class="stats">
    <div class="stat"><span>可申报活动</span><strong>{{ activities.length }}</strong></div>
    <div class="stat"><span>我的申请</span><strong>{{ workspace.applications.length }}</strong></div>
    <div class="stat"><span>当前阶段</span><strong>{{ auth.meta?.summaryPhase ? "汇总期" : "申报期" }}</strong></div>
  </div>

  <div class="toolbar">
    <input v-model="filters.q" class="input" placeholder="搜索活动、主办方或类别" />
    <select v-model="filters.category" class="select">
      <option>全部</option>
      <option>身心素养</option>
      <option>文艺素养</option>
      <option>劳动素养</option>
      <option>创新素养</option>
    </select>
    <select v-model="filters.level" class="select">
      <option>全部</option>
      <option>国家级</option>
      <option>省级</option>
      <option>校级</option>
      <option>院级</option>
    </select>
  </div>

  <div class="grid">
    <article v-for="item in activities" :key="item.id" class="card">
      <div class="card-header">
        <h3>{{ item.name }}</h3>
        <StatusTag :status="item.status" />
      </div>
      <div class="meta">
        <span>{{ item.level }}</span>
        <span>{{ item.category }}</span>
        <span>已申报 {{ item.applications }}</span>
      </div>
      <p>{{ item.rule }}</p>
      <div class="button-row">
        <button class="primary" :disabled="auth.meta?.summaryPhase || item.status !== '已上架'" @click="apply(item.id)">
          申报
        </button>
        <span class="tag" :class="{ red: daysUntil(item.deadline) <= 3 }">截止 {{ item.deadline || "待定" }}</span>
      </div>
    </article>
  </div>
</template>
