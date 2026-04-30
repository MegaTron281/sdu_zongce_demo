<script setup lang="ts">
import { onMounted } from "vue";
import { formatNumber, publicityCountdown } from "@/api/format";
import StatusTag from "@/components/StatusTag.vue";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
onMounted(() => auth.user?.studentId && workspace.loadStudent(auth.user.studentId));
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>活动</th><th>提交时间</th><th>状态</th><th>加分</th><th>审核意见</th></tr>
      </thead>
      <tbody>
        <tr v-for="item in workspace.applications" :key="item.id">
          <td>{{ item.activity }}<br /><span class="tag">{{ item.category }}</span></td>
          <td>{{ item.submittedAt }}</td>
          <td>
            <StatusTag :status="item.status" />
            <br v-if="item.publicityUntil" />
            <span v-if="item.publicityUntil" class="meta">{{ publicityCountdown(item.publicityUntil) }}</span>
          </td>
          <td>{{ item.score ? formatNumber(item.score) : "待定" }}</td>
          <td>{{ item.reviewerOpinion || "暂无" }}<br />{{ item.adminOpinion }}</td>
        </tr>
        <tr v-if="workspace.applications.length === 0"><td colspan="5">暂无申请记录</td></tr>
      </tbody>
    </table>
  </div>
</template>
