<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api } from "@/api/client";
import { formatNumber, publicityCountdown } from "@/api/format";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
onMounted(() => workspace.loadBase());

// 异议弹窗
const objectId = ref("");
const objectReason = ref("");
const objectError = ref("");
const objecting = ref(false);

function openObject(id: string) {
  objectId.value = id;
  objectReason.value = "";
  objectError.value = "";
}

function closeObject() {
  objectId.value = "";
  objectReason.value = "";
  objectError.value = "";
}

async function submitObject() {
  if (!objectReason.value.trim()) {
    objectError.value = "请填写异议理由";
    return;
  }
  objecting.value = true;
  try {
    await api.action(objectId.value, "object", { reason: objectReason.value.trim() });
    workspace.setToast("异议已提交，供老师复审参考");
    closeObject();
    await workspace.loadBase();
  } catch (e) {
    objectError.value = e instanceof Error ? e.message : "提交失败";
  } finally {
    objecting.value = false;
  }
}

function isOwnApplication(studentId: string): boolean {
  return auth.user?.studentId === studentId;
}
</script>

<template>
  <div class="table-wrap">
    <table>
      <thead><tr><th>姓名</th><th>活动名称</th><th>申报分</th><th>最终分</th><th>剩余公示时间</th><th>异议</th></tr></thead>
      <tbody>
        <tr v-for="item in workspace.publicity" :key="item.id">
          <td>{{ item.student }}</td>
          <td>{{ item.activity }}</td>
          <td>{{ formatNumber(item.claimedScore) }}</td>
          <td>
            <span :style="{ color: item.score !== item.claimedScore ? 'var(--sdu-red)' : '' }">
              {{ formatNumber(item.score) }}
            </span>
          </td>
          <td>{{ publicityCountdown(item.publicityUntil) }}</td>
          <td>
            <button
              v-if="!isOwnApplication(item.studentId)"
              class="danger"
              @click="openObject(item.id)"
            >
              提出异议
            </button>
            <span v-else class="meta">—</span>
          </td>
        </tr>
        <tr v-if="workspace.publicity.length === 0"><td colspan="6">暂无公示中申请</td></tr>
      </tbody>
    </table>
  </div>

  <!-- 异议弹窗 -->
  <div v-if="objectId" class="pwd-modal-overlay" @click.self="closeObject">
    <div class="pwd-modal-card">
      <div class="pwd-modal-header">
        <div class="pwd-modal-icon">📢</div>
        <div>
          <h3>提出异议</h3>
          <p>对公示中的申请提出异议，管理员将重新复核。</p>
        </div>
        <button class="pwd-modal-close" @click="closeObject" aria-label="关闭">✕</button>
      </div>
      <div v-if="objectError" class="notice-strip error">{{ objectError }}</div>
      <form @submit.prevent="submitObject">
        <div class="field" style="margin-bottom: 14px;">
          <label>异议理由</label>
          <textarea
            v-model="objectReason"
            class="textarea"
            rows="4"
            placeholder="例如：分数计算有误、材料不实、本人排名不符……"
          ></textarea>
        </div>
        <div class="pwd-modal-actions">
          <button type="button" class="secondary" @click="closeObject">取消</button>
          <button type="submit" class="danger" :disabled="objecting">
            {{ objecting ? "提交中..." : "提交异议" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
