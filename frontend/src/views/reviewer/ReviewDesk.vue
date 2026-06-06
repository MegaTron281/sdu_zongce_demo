<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { api } from "@/api/client";
import { formatNumber, publicityCountdown } from "@/api/format";
import MaterialList from "@/components/MaterialList.vue";
import StatusTag from "@/components/StatusTag.vue";
import { ApplicationStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
const forms = reactive<Record<string, { score: number; reason: string }>>({});
onMounted(() => workspace.loadReviewer());

const pending = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.PendingReviewer));
const publicity = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Publicity));
const approvedCount = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Approved).length);
const specialCount = computed(() => workspace.applications.filter((item) => item.status === ApplicationStatus.Special).length);

// 弹窗状态
const dialogAppId = ref("");
const dialogMode = ref<"" | "reject" | "special">("");
const dialogReason = ref("");
const dialogError = ref("");

function formFor(id: string, score: number) {
  if (!forms[id]) forms[id] = { score, reason: "" };
  return forms[id];
}

function openDialog(id: string, mode: "reject" | "special") {
  dialogAppId.value = id;
  dialogMode.value = mode;
  dialogReason.value = "";
  dialogError.value = "";
}

function closeDialog() {
  dialogAppId.value = "";
  dialogMode.value = "";
  dialogReason.value = "";
  dialogError.value = "";
}

async function submitDialog() {
  if (!dialogReason.value.trim()) {
    dialogError.value = "请填写原因";
    return;
  }
  const action = dialogMode.value === "reject" ? "review-reject" : "mark-special";
  await api.action(dialogAppId.value, action, { reason: dialogReason.value.trim() });
  workspace.setToast("操作已提交");
  closeDialog();
  await workspace.loadReviewer();
}

async function pass(id: string, custom: boolean) {
  const form = forms[id];
  if (custom) {
    const reason = form?.reason?.trim();
    if (!reason) {
      workspace.error = "修改分值需要填写原因";
      return;
    }
    await api.action(id, "review-pass", { score: Number(form.score), reason });
  } else {
    await api.action(id, "review-pass", { reason: "班委初审通过" });
  }
  workspace.setToast("操作已提交");
  await workspace.loadReviewer();
}

async function finishPublicity(id: string) {
  await api.action(id, "publicity-finish", { reason: "公示期结束，无有效异议。" });
  workspace.setToast("操作已提交");
  await workspace.loadReviewer();
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
      <MaterialList :items="item.materials" />
      <div class="score-box"><strong>学生申报分：{{ formatNumber(item.score) }} 分</strong></div>
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
        <button class="primary" @click="pass(item.id, false)">通过</button>
        <button class="secondary" @click="pass(item.id, true)">改分后通过</button>
        <button class="danger" @click="openDialog(item.id, 'reject')">驳回</button>
        <button class="secondary" @click="openDialog(item.id, 'special')">标记特殊情况</button>
      </div>
    </article>
    <article v-for="item in publicity" :key="item.id" class="card">
      <div class="card-header"><h3>{{ item.student }}</h3><StatusTag :status="item.status" /></div>
      <p>{{ item.activity }}</p>
      <MaterialList :items="item.materials" />
      <div class="meta"><span>{{ publicityCountdown(item.publicityUntil) }}</span><span>分值 {{ formatNumber(item.score) }} 分</span></div>
      <div class="button-row">
        <button class="secondary" @click="finishPublicity(item.id)">结束公示并送复审</button>
      </div>
    </article>
    <div v-if="pending.length === 0 && publicity.length === 0" class="empty">暂无待处理申请</div>
  </div>

  <!-- 驳回/特殊情况弹窗 -->
  <div v-if="dialogAppId" class="pwd-modal-overlay" @click.self="closeDialog">
    <div class="pwd-modal-card">
      <div class="pwd-modal-header">
        <div class="pwd-modal-icon">{{ dialogMode === 'reject' ? '❌' : '⚠️' }}</div>
        <div>
          <h3>{{ dialogMode === 'reject' ? '驳回申请' : '标记特殊情况' }}</h3>
          <p>请填写原因，学生将能看到此说明。</p>
        </div>
        <button class="pwd-modal-close" @click="closeDialog" aria-label="关闭">✕</button>
      </div>
      <div v-if="dialogError" class="notice-strip error">{{ dialogError }}</div>
      <form @submit.prevent="submitDialog">
        <div class="field" style="margin-bottom: 14px;">
          <label>{{ dialogMode === 'reject' ? '驳回原因' : '特殊情况说明' }}</label>
          <textarea v-model="dialogReason" class="textarea" rows="4" placeholder="请详细说明原因……"></textarea>
        </div>
        <div class="pwd-modal-actions">
          <button type="button" class="secondary" @click="closeDialog">取消</button>
          <button type="submit" :class="dialogMode === 'reject' ? 'danger' : 'primary'">确认提交</button>
        </div>
      </form>
    </div>
  </div>
</template>
