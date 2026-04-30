<script setup lang="ts">
import { computed, onMounted } from "vue";
import { api } from "@/api/client";
import { formatNumber, publicityCountdown } from "@/api/format";
import StatusTag from "@/components/StatusTag.vue";
import { ApplicationStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadReviewer());

const pending = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.PendingReviewer));
const publicity = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Publicity));
const approvedCount = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Approved).length);
const specialCount = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Special).length);

async function action(id: string, name: string, reason = "") {
  if (["review-reject", "mark-special"].includes(name) && !reason) return;
  await api.action(id, name, { reason });
  workspace.setToast("操作已提交");
  await workspace.loadReviewer();
}

function reject(id: string) {
  action(id, "review-reject", window.prompt("请输入驳回原因") || "");
}

function markSpecial(id: string) {
  action(id, "mark-special", window.prompt("请输入特殊情况原因") || "");
}
</script>

<template>
  <div class="stats">
    <div class="stat"><span>本班待审核</span><strong>{{ pending.length }}</strong></div>
    <div class="stat"><span>公示中</span><strong>{{ publicity.length }}</strong></div>
    <div class="stat"><span>本周通过</span><strong>{{ approvedCount }}</strong></div>
    <div class="stat"><span>特殊情况</span><strong>{{ specialCount }}</strong></div>
  </div>
  <div class="grid">
    <article v-for="item in pending" :key="item.id" class="card">
      <div class="card-header"><h3>{{ item.student }}</h3><StatusTag :status="item.status" /></div>
      <div class="meta"><span>{{ item.className }}</span><span>{{ item.submittedAt }}</span><span>{{ item.category }}</span></div>
      <p>{{ item.activity }}</p>
      <div class="score-box"><strong>建议分值：{{ formatNumber(item.score) }} 分</strong></div>
      <div class="button-row">
        <button class="primary" @click="action(item.id, 'review-pass', '材料齐全，建议通过。')">通过</button>
        <button class="danger" @click="reject(item.id)">驳回</button>
        <button class="secondary" @click="markSpecial(item.id)">标记特殊情况</button>
      </div>
    </article>
    <article v-for="item in publicity" :key="item.id" class="card">
      <div class="card-header"><h3>{{ item.student }}</h3><StatusTag :status="item.status" /></div>
      <p>{{ item.activity }}</p>
      <div class="meta"><span>{{ publicityCountdown(item.publicityUntil) }}</span><span>建议分值 {{ formatNumber(item.score) }} 分</span></div>
      <div class="button-row">
        <button class="secondary" @click="action(item.id, 'publicity-finish', '公示期结束，无有效异议。')">结束公示并送复审</button>
      </div>
    </article>
    <div v-if="pending.length === 0 && publicity.length === 0" class="empty">暂无待处理申请</div>
  </div>
</template>
