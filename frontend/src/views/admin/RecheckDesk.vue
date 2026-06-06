<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { api } from "@/api/client";
import { formatNumber } from "@/api/format";
import type { Objection } from "@/api/types";
import MaterialList from "@/components/MaterialList.vue";
import StatusTag from "@/components/StatusTag.vue";
import { ApplicationStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
const forms = reactive<Record<string, { score: number; reason: string }>>({});
const objections = reactive<Record<string, Objection[]>>({});
onMounted(async () => {
  await workspace.loadAdmin();
  // 加载所有待审核申请的异议
  for (const app of workspace.applications) {
    if (adminReviewable.includes(app.status) || app.status === ApplicationStatus.Publicity) {
      objections[app.id] = await api.objections(app.id);
    }
  }
});

const adminReviewable: string[] = [ApplicationStatus.PendingAdmin, ApplicationStatus.Special];
const items = computed(() => workspace.applications.filter((item) => adminReviewable.includes(item.status)));

// 驳回弹窗
const rejectId = ref("");
const rejectReason = ref("");
const rejectError = ref("");

function formFor(id: string, score: number) {
  if (!forms[id]) forms[id] = { score, reason: "" };
  return forms[id];
}

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
    rejectError.value = "请填写驳回原因";
    return;
  }
  await api.action(rejectId.value, "admin-reject", { reason: rejectReason.value.trim() });
  workspace.setToast("复审操作已提交");
  closeReject();
  await workspace.loadAdmin();
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

</script>

<template>
  <div class="grid">
    <article v-for="item in items" :key="item.id" class="card">
      <div class="card-header">
        <h3>{{ item.student }}</h3>
        <StatusTag :status="item.status" />
      </div>
      <p>{{ item.activity }}</p>
      <MaterialList :items="item.materials" />
      <div class="meta">
        <span>班委意见：{{ item.reviewerOpinion || item.specialReason || "暂无" }}</span>
        <span v-if="item.claimedScore !== item.score" style="color: var(--sdu-red);">
          学生申报 {{ formatNumber(item.claimedScore) }} 分 / 当前 {{ formatNumber(item.score) }} 分
        </span>
        <span v-else>分值 {{ formatNumber(item.score) }} 分</span>
      </div>

      <!-- 异议列表 -->
      <div v-if="objections[item.id]?.length" class="feedback" style="margin-bottom: 10px;">
        <strong style="font-size: 13px;">公示期异议（{{ objections[item.id].length }} 条）：</strong>
        <div v-for="obj in objections[item.id]" :key="obj.id" style="margin-top: 6px; padding-left: 8px; border-left: 3px solid var(--amber); font-size: 13px;">
          <span style="color: var(--muted);">{{ obj.student }}：</span>{{ obj.reason }}
        </div>
      </div>
      <div class="form-grid" style="margin-bottom: 8px;">
          <div class="field">
            <label>分值</label>
            <input v-model.number="formFor(item.id, item.score).score" class="input" type="number" min="0" step="0.5" />
          </div>
          <div class="field">
            <label>意见</label>
            <input v-model="formFor(item.id, item.score).reason" class="input" placeholder="修改分值时必填原因" />
          </div>
        </div>
        <div class="button-row">
          <button class="primary" @click="pass(item.id, false)">采纳并通过</button>
          <button class="secondary" @click="pass(item.id, true)">修改分值后通过</button>
          <button class="danger" @click="openReject(item.id)">驳回</button>
        </div>
    </article>
    <div v-if="items.length === 0" class="empty">暂无待复审申请</div>
  </div>

  <!-- 驳回弹窗 -->
  <div v-if="rejectId" class="pwd-modal-overlay" @click.self="closeReject">
    <div class="pwd-modal-card">
      <div class="pwd-modal-header">
        <div class="pwd-modal-icon">❌</div>
        <div>
          <h3>驳回申请</h3>
          <p>请填写驳回原因，学生将能看到此说明。</p>
        </div>
        <button class="pwd-modal-close" @click="closeReject" aria-label="关闭">✕</button>
      </div>
      <div v-if="rejectError" class="notice-strip error">{{ rejectError }}</div>
      <form @submit.prevent="submitReject">
        <div class="field" style="margin-bottom: 14px;">
          <label>驳回原因</label>
          <textarea v-model="rejectReason" class="textarea" rows="4" placeholder="请详细说明驳回原因……"></textarea>
        </div>
        <div class="pwd-modal-actions">
          <button type="button" class="secondary" @click="closeReject">取消</button>
          <button type="submit" class="danger">确认驳回</button>
        </div>
      </form>
    </div>
  </div>
</template>
