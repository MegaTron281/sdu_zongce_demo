<script setup lang="ts">
import { onMounted } from "vue";
import { api } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
onMounted(() => workspace.loadAdmin());

async function togglePhase() {
  if (!auth.meta) return;
  const next = await api.setSummaryPhase(!auth.meta.summaryPhase);
  auth.meta = next;
  workspace.setToast(next.summaryPhase ? "已进入综测汇总期" : "已恢复日常申报期");
}
</script>

<template>
  <div class="grid">
    <div class="card"><h3>学生账号</h3><p>账号已迁移到 PostgreSQL，密码以 bcrypt 哈希保存。</p><button class="primary">批量导入</button></div>
    <div class="card"><h3>班委授权</h3><p>审核学生权限由服务端 RBAC 控制，限定在本班级。</p><button class="secondary">设置角色</button></div>
    <div class="card">
      <h3>综测周期</h3>
      <p>当前阶段：{{ auth.meta?.summaryPhase ? "综测汇总期" : "日常申报期" }}</p>
      <button class="secondary" @click="togglePhase">切换阶段</button>
    </div>
  </div>
</template>
