<script setup lang="ts">
import { computed, onMounted } from "vue";
import { formatNumber } from "@/api/format";
import { ApplicationStatus } from "@/constants/status";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
onMounted(() => auth.user?.studentId && workspace.loadStudent(auth.user.studentId));

const included = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Approved));
</script>

<template>
  <div v-if="workspace.score">
    <div class="stats">
      <div class="stat"><span>素质能力总分</span><strong>{{ formatNumber(workspace.score.qualityScore) }}</strong></div>
      <div class="stat"><span>学业成绩</span><strong>{{ formatNumber(workspace.score.academicScore) }}</strong></div>
      <div class="stat"><span>预估综测成绩</span><strong>{{ formatNumber(workspace.score.assessmentScore) }}</strong></div>
      <div class="stat"><span>页面状态</span><strong>{{ workspace.score.locked ? "锁定" : "开放" }}</strong></div>
    </div>
    <div class="panel">
      <p class="section-label">四模块得分</p>
      <div class="stats" style="grid-template-columns: repeat(4, minmax(0, 1fr));">
        <div v-for="module in workspace.score.modules" :key="module.name" class="stat">
          <span>{{ module.name }}</span>
          <strong>{{ formatNumber(module.score) }}</strong>
        </div>
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>已计入条目</th><th>模块</th><th>实际加分</th><th>入账说明</th></tr></thead>
        <tbody>
          <tr v-for="item in included" :key="item.id">
            <td>{{ item.activity }}</td>
            <td>{{ item.category }}</td>
            <td>{{ formatNumber(item.score) }}</td>
            <td>已通过复审</td>
          </tr>
          <tr v-if="included.length === 0"><td colspan="4">暂无计入条目</td></tr>
        </tbody>
      </table>
    </div>
  </div>
  <div v-else class="empty">暂无成绩数据</div>
</template>
