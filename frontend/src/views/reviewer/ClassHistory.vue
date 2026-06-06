<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { formatNumber } from "@/api/format";
import MaterialList from "@/components/MaterialList.vue";
import StatusTag from "@/components/StatusTag.vue";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
const search = ref("");
onMounted(() => workspace.loadReviewer());

const filtered = computed(() => {
  if (!search.value.trim()) return workspace.applications;
  const keyword = search.value.trim().toLowerCase();
  return workspace.applications.filter(
    (item) =>
      item.student.toLowerCase().includes(keyword) ||
      item.activity.toLowerCase().includes(keyword) ||
      item.category.toLowerCase().includes(keyword)
  );
});
</script>

<template>
  <div class="toolbar" style="margin-bottom: 14px;">
    <input v-model="search" class="input" placeholder="搜索学生、活动或模块……" style="max-width: 320px;" />
  </div>
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>学生</th><th>班级</th><th>活动</th><th>状态</th><th>分值</th><th>材料</th><th>意见</th></tr>
      </thead>
      <tbody>
        <tr v-for="item in filtered" :key="item.id">
          <td>{{ item.student }}</td>
          <td>{{ item.className }}</td>
          <td>{{ item.activity }}<br /><span class="tag">{{ item.category }}</span></td>
          <td><StatusTag :status="item.status" /></td>
          <td>{{ formatNumber(item.score) }}</td>
          <td><MaterialList :items="item.materials" /></td>
          <td>{{ item.reviewerOpinion || item.adminOpinion || "暂无" }}</td>
        </tr>
        <tr v-if="filtered.length === 0"><td colspan="7">暂无申请记录</td></tr>
      </tbody>
    </table>
  </div>
</template>
