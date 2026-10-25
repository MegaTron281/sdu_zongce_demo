<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { formatDateTime, formatNumber } from "@/api/format";
import { authFetchBlob } from "@/api/client";
import { ApplicationStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadAdmin());

// 导出成绩册：改用带本页签 Bearer 令牌的下载，避免走浏览器共享 cookie
// （并行多账号时 cookie 会指向最后一次登录的账号，导致导出串味）
const exporting = ref(false);
const exportError = ref("");

async function exportScores() {
  if (exporting.value) return;
  exporting.value = true;
  exportError.value = "";
  try {
    const blob = await authFetchBlob("/api/admin/export/scores.csv");
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "scores.csv";
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  } catch (error) {
    exportError.value = error instanceof Error ? error.message : "导出失败，请稍后重试";
  } finally {
    exporting.value = false;
  }
}

const rows = computed(() => [...workspace.scoreRows].sort((a, b) => b.assessmentScore - a.assessmentScore));
const average = computed(() => (rows.value.length ? rows.value.reduce((sum, item) => sum + item.qualityScore, 0) / rows.value.length : 0));
const specialCount = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Special).length);
</script>

<template>
  <div class="stats">
    <div class="stat"><span>已有申请学生</span><strong>{{ rows.length }}</strong></div>
    <div class="stat"><span>平均素质分</span><strong>{{ formatNumber(average) }}</strong></div>
    <div class="stat"><span>异常预警</span><strong>{{ specialCount }}</strong></div>
    <div class="stat"><span>待导出册</span><strong>1</strong></div>
  </div>
  <div class="table-wrap">
    <table>
      <thead><tr><th>排名</th><th>学生</th><th>班级</th><th>四模块</th><th>素质总分</th><th>综测成绩</th></tr></thead>
      <tbody>
        <tr v-for="(item, index) in rows" :key="item.studentId">
          <td>{{ index + 1 }}</td>
          <td>{{ item.student || item.studentId }}</td>
          <td>{{ item.className || "未登记" }}</td>
          <td>{{ item.modules.map((module) => formatNumber(module.score)).join(" / ") }}</td>
          <td>{{ formatNumber(item.qualityScore) }}</td>
          <td>{{ formatNumber(item.assessmentScore) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div class="grid">
    <div class="card">
      <h3>导出综测成绩册</h3>
      <p>包含每位学生四模块分值、素质总分、学业成绩和综测成绩。</p>
      <button class="primary link-button" type="button" :disabled="exporting" @click="exportScores">
        {{ exporting ? "导出中..." : "导出成绩册" }}
      </button>
      <p v-if="exportError" class="feedback error">⚠ {{ exportError }}</p>
    </div>
    <div class="card"><h3>操作日志</h3><p v-for="item in workspace.logs.slice(-5)" :key="item.id">{{ formatDateTime(item.createdAt) }} {{ item.actor }} {{ item.action }}：{{ item.target }}</p></div>
  </div>
</template>
