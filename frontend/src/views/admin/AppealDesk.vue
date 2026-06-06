<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api } from "@/api/client";
import { formatNumber } from "@/api/format";
import MaterialList from "@/components/MaterialList.vue";
import StatusTag from "@/components/StatusTag.vue";
import { ApplicationStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadAdmin());

const items = computed(() =>
  workspace.applications.filter((item) => item.status === ApplicationStatus.Appealing)
);

// 驳回弹窗
const rejectId = ref("");
const rejectReason = ref("");
const rejectError = ref("");

function openReject(id: string) {
  rejectId.value = id;
  rejectReason.value = "";
  rejectError.value = "";
}

function closeReject() {
  rejectId.value = "";
  rejectReason.value = "";
  rejectError.value = "";
}

async function submitReject() {
  if (!rejectReason.value.trim()) {
    rejectError.value = "请填写驳回理由";
    return;
  }
  await api.action(rejectId.value, "appeal-reject", { reason: rejectReason.value.trim() });
  workspace.setToast("已驳回申诉");
  closeReject();
  await workspace.loadAdmin();
}

async function approve(id: string) {
  await api.action(id, "appeal-approve", { reason: "管理员采纳申诉" });
  workspace.setToast("已采纳申诉");
  await workspace.loadAdmin();
}
</script>

<template>
  <div class="stats">
    <div class="stat"><span>待处理申诉</span><strong>{{ items.length }}</strong></div>
  </div>

  <div class="grid">
    <article v-for="item in items" :key="item.id" class="card">
      <div class="card-header">
        <h3>{{ item.student }}</h3>
        <StatusTag :status="item.status" />
      </div>
      <p>{{ item.activity }}</p>
      <MaterialList :items="item.materials" />

      <div class="meta" style="margin-bottom: 8px;">
        <span>学生申报 {{ formatNumber(item.claimedScore) }} 分</span>
        <span v-if="item.claimedScore !== item.score" style="color: var(--sdu-red);">
          → 被改为 {{ formatNumber(item.score) }} 分
        </span>
        <span>| {{ item.className }}</span>
      </div>

      <div class="score-box">
        <strong>申诉理由：</strong>{{ item.specialReason }}
        <br /><span class="meta">原审核意见：{{ item.reviewerOpinion || item.adminOpinion || "暂无" }}</span>
      </div>

      <div class="button-row">
        <button class="primary" @click="approve(item.id)">采纳申诉（恢复申报分）</button>
        <button class="danger" @click="openReject(item.id)">驳回申诉</button>
      </div>
    </article>
    <div v-if="items.length === 0" class="empty">暂无待处理申诉</div>
  </div>

  <!-- 驳回弹窗 -->
  <div v-if="rejectId" class="pwd-modal-overlay" @click.self="closeReject">
    <div class="pwd-modal-card">
      <div class="pwd-modal-header">
        <div class="pwd-modal-icon">❌</div>
        <div>
          <h3>驳回申诉</h3>
          <p>请填写驳回申诉的理由，学生将能看到此说明。</p>
        </div>
        <button class="pwd-modal-close" @click="closeReject" aria-label="关闭">✕</button>
      </div>
      <div v-if="rejectError" class="notice-strip error">{{ rejectError }}</div>
      <form @submit.prevent="submitReject">
        <div class="field" style="margin-bottom: 14px;">
          <label>驳回理由</label>
          <textarea v-model="rejectReason" class="textarea" rows="4" placeholder="请说明为何驳回申诉……"></textarea>
        </div>
        <div class="pwd-modal-actions">
          <button type="button" class="secondary" @click="closeReject">取消</button>
          <button type="submit" class="danger">确认驳回</button>
        </div>
      </form>
    </div>
  </div>
</template>
