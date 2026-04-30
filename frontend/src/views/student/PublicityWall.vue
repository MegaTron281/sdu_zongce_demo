<script setup lang="ts">
import { onMounted } from "vue";
import { formatNumber, publicityCountdown } from "@/api/format";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadBase());
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead><tr><th>姓名</th><th>活动名称</th><th>申请加分</th><th>剩余公示时间</th><th>异议</th></tr></thead>
      <tbody>
        <tr v-for="item in workspace.publicity" :key="item.id">
          <td>{{ item.student }}</td>
          <td>{{ item.activity }}</td>
          <td>{{ formatNumber(item.score) }}</td>
          <td>{{ publicityCountdown(item.publicityUntil) }}</td>
          <td><button class="danger" @click="workspace.setToast(`已记录异议线索：${item.id}`)">提出异议</button></td>
        </tr>
        <tr v-if="workspace.publicity.length === 0"><td colspan="5">暂无公示中申请</td></tr>
      </tbody>
    </table>
  </div>
</template>
