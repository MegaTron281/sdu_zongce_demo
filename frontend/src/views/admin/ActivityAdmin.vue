<script setup lang="ts">
import { computed, onMounted } from "vue";
import { api } from "@/api/client";
import StatusTag from "@/components/StatusTag.vue";
import { ActivityStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadAdmin());

const pendingActivities = computed(() => workspace.activities.filter((item) => item.status === ActivityStatus.Pending));

async function setStatus(id: string, status: string) {
  await api.patchActivity(id, { status });
  workspace.setToast("活动状态已更新");
  await workspace.loadAdmin();
}
</script>

<template>
  <div class="split-panel">
    <div>
      <p class="section-label">活动名录</p>
      <div class="table-wrap">
        <table>
          <thead><tr><th>活动名称</th><th>级别</th><th>类别</th><th>申报人数</th><th>状态</th></tr></thead>
          <tbody>
            <tr v-for="item in workspace.activities" :key="item.id">
              <td>{{ item.name }}</td>
              <td>{{ item.level }}</td>
              <td>{{ item.category }}</td>
              <td>{{ item.applications }}</td>
              <td><StatusTag :status="item.status" /></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div>
      <p class="section-label">学生申报待审</p>
      <div v-for="item in pendingActivities" :key="item.id" class="card">
        <h3>{{ item.name }}</h3>
        <div class="meta"><span>{{ item.level }}</span><span>{{ item.category }}</span><StatusTag :status="item.status" /></div>
        <p>{{ item.organizer || "主办方待确认" }} · {{ item.source || "来源待补充" }}</p>
        <div class="button-row">
          <button class="primary" @click="setStatus(item.id, ActivityStatus.Listed)">确认上架</button>
          <button class="danger" @click="setStatus(item.id, ActivityStatus.Rejected)">驳回</button>
        </div>
      </div>
      <div v-if="pendingActivities.length === 0" class="empty">暂无待审核活动</div>
    </div>
  </div>
</template>
