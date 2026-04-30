<script setup lang="ts">
import { computed, onMounted, reactive } from "vue";
import { api } from "@/api/client";
import { formatNumber } from "@/api/format";
import StatusTag from "@/components/StatusTag.vue";
import { ApplicationStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
const forms = reactive<Record<string, { score: number; reason: string }>>({});
onMounted(() => workspace.loadAdmin());

const adminReviewable: string[] = [ApplicationStatus.PendingAdmin, ApplicationStatus.Special];
const items = computed(() => workspace.applications.filter((item) => adminReviewable.includes(item.status)));

function formFor(id: string, score: number) {
  if (!forms[id]) forms[id] = { score, reason: "" };
  return forms[id];
}

async function pass(id: string, custom: boolean) {
  const form = forms[id];
  if (custom) {
    const reason = form?.reason?.trim();
    if (!reason) {
      workspace.error = "修改分值需要填写原因";
      return;
    }
    await api.action(id, "admin-pass", { score: Number(form.score), reason });
  } else {
    await api.action(id, "admin-pass", { reason: "管理员复审通过" });
  }
  workspace.setToast("复审操作已提交");
  await workspace.loadAdmin();
}

async function reject(id: string) {
  const reason = window.prompt("请输入驳回原因") || "";
  if (!reason) return;
  await api.action(id, "admin-reject", { reason });
  workspace.setToast("复审操作已提交");
  await workspace.loadAdmin();
}
</script>

<template>
  <div class="grid">
    <article v-for="item in items" :key="item.id" class="card">
      <div class="card-header"><h3>{{ item.student }}</h3><StatusTag :status="item.status" /></div>
      <p>{{ item.activity }}</p>
      <div class="meta">
        <span>班委意见：{{ item.reviewerOpinion || item.specialReason || "暂无" }}</span>
        <span>建议分值 {{ formatNumber(item.score) }} 分</span>
      </div>
      <input v-model.number="formFor(item.id, item.score).score" class="input" type="number" min="0" step="0.5" />
      <textarea v-model="formFor(item.id, item.score).reason" class="textarea" placeholder="修改分值或特殊情况说明"></textarea>
      <div class="button-row">
        <button class="primary" @click="pass(item.id, false)">采纳建议并通过</button>
        <button class="secondary" @click="pass(item.id, true)">修改分值后通过</button>
        <button class="danger" @click="reject(item.id)">驳回</button>
      </div>
    </article>
    <div v-if="items.length === 0" class="empty">暂无待复审申请</div>
  </div>
</template>
