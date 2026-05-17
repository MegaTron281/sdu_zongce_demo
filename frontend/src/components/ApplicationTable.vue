<script setup lang="ts">
import type { Application } from "@/api/types";
import { formatNumber } from "@/api/format";
import MaterialList from "./MaterialList.vue";
import StatusTag from "./StatusTag.vue";

defineProps<{ items: Application[] }>();
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>学生</th>
          <th>班级</th>
          <th>活动</th>
          <th>状态</th>
          <th>分值</th>
          <th>材料</th>
          <th>意见</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="item.id">
          <td>{{ item.student }}</td>
          <td>{{ item.className }}</td>
          <td>{{ item.activity }}</td>
          <td><StatusTag :status="item.status" /></td>
          <td>{{ item.score ? formatNumber(item.score) : "待定" }}</td>
          <td><MaterialList :items="item.materials" /></td>
          <td>{{ item.reviewerOpinion || item.adminOpinion || item.specialReason || "暂无" }}</td>
        </tr>
        <tr v-if="items.length === 0">
          <td colspan="7">暂无申请记录</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
