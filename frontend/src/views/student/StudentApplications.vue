<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api } from "@/api/client";
import { formatNumber, publicityCountdown } from "@/api/format";
import MaterialList from "@/components/MaterialList.vue";
import StatusTag from "@/components/StatusTag.vue";
import { ApplicationStatus } from "@/constants/status";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
onMounted(() => auth.user?.studentId && workspace.loadStudent(auth.user.studentId));

const appealingId = ref("");
const appealReason = ref("");
const appealError = ref("");
const appealing = ref(false);

function openAppeal(id: string) {
  appealingId.value = id;
  appealReason.value = "";
  appealError.value = "";
}

function closeAppeal() {
  appealingId.value = "";
  appealReason.value = "";
  appealError.value = "";
}

async function submitAppeal() {
  if (!appealReason.value.trim()) {
    appealError.value = "请填写申诉理由";
    return;
  }
  appealing.value = true;
  try {
    await api.action(appealingId.value, "appeal", { reason: appealReason.value.trim() });
    workspace.setToast("申诉已提交，等待管理员复核");
    closeAppeal();
    if (auth.user?.studentId) await workspace.loadStudent(auth.user.studentId);
  } catch (e) {
    appealError.value = e instanceof Error ? e.message : "申诉提交失败";
  } finally {
    appealing.value = false;
  }
}

async function withdraw(id: string) {
  await api.action(id, "withdraw", { reason: "学生自行撤回" });
  workspace.setToast("申请已撤回");
  if (auth.user?.studentId) await workspace.loadStudent(auth.user.studentId);
}

function canAppeal(item: { status: string; score: number; claimedScore: number }): boolean {
  // 被驳回 → 可申诉
  if (item.status === ApplicationStatus.Rejected) return true;
  // 分数被修改（不论在公示期还是已通过）→ 可申诉
  if (item.score !== item.claimedScore && ([ApplicationStatus.Publicity, ApplicationStatus.Approved] as string[]).includes(item.status)) return true;
  return false;
}
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead>
        <tr><th>活动</th><th>提交时间</th><th>状态</th><th>申报分</th><th>最终分</th><th>材料</th><th>审核意见</th><th>操作</th></tr>
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
          <td>{{ formatNumber(item.claimedScore) }}</td>
          <td>
            <span :style="{ color: item.score !== item.claimedScore ? 'var(--sdu-red)' : '' }">
              {{ formatNumber(item.score) }}
            </span>
          </td>
          <td><MaterialList :items="item.materials" /></td>
          <td>{{ item.reviewerOpinion || "暂无" }}<br />{{ item.adminOpinion }}</td>
          <td>
            <button v-if="canAppeal(item)" class="secondary" style="padding:4px 10px;font-size:0.8rem;margin-right: 4px;" @click="openAppeal(item.id)">
              申诉
            </button>
            <button
              v-if="item.status === ApplicationStatus.PendingReviewer"
              class="secondary"
              style="padding:4px 10px;font-size:0.8rem;color: var(--red);"
              @click="withdraw(item.id)"
            >
              撤回
            </button>
            <span v-if="item.status === ApplicationStatus.Appealing" class="tag amber">申诉处理中</span>
          </td>
        </tr>
        <tr v-if="workspace.applications.length === 0"><td colspan="8">暂无申请记录</td></tr>
      </tbody>
    </table>
  </div>

  <!-- 申诉弹窗 -->
  <div v-if="appealingId" class="pwd-modal-overlay" @click.self="closeAppeal">
    <div class="pwd-modal-card">
      <div class="pwd-modal-header">
        <div class="pwd-modal-icon">📝</div>
        <div>
          <h3>提出申诉</h3>
          <p>请详细说明申诉理由，管理员将重新复核你的申请。</p>
        </div>
        <button class="pwd-modal-close" @click="closeAppeal" aria-label="关闭">✕</button>
      </div>
      <div v-if="appealError" class="notice-strip error">{{ appealError }}</div>
      <form @submit.prevent="submitAppeal">
        <div class="field" style="margin-bottom: 14px;">
          <label>申诉理由</label>
          <textarea
            v-model="appealReason"
            class="textarea"
            rows="4"
            placeholder="例如：我的分数被错误修改 / 驳回原因与实际情况不符……"
          ></textarea>
        </div>
        <div class="pwd-modal-actions">
          <button type="button" class="secondary" @click="closeAppeal">取消</button>
          <button type="submit" class="primary" :disabled="appealing">
            {{ appealing ? "提交中..." : "提交申诉" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
