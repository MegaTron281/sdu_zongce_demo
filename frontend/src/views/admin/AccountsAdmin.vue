<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { api } from "@/api/client";
import type { UserListItem, UserImportResult } from "@/api/types";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";
import { roleLabels } from "@/stores/auth";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
const users = ref<UserListItem[]>([]);
const importResult = ref<UserImportResult | null>(null);
const uploading = ref(false);
const classEdits = reactive<Record<string, string>>({});
// 只有系统管理员能指定辅导员的管辖班级；辅导员不可自改
const isSuper = computed(() => auth.user?.isSuperAdmin === true);

onMounted(async () => {
  await workspace.loadAdmin();
  await loadUsers();
});

async function loadUsers() {
  users.value = await api.listUsers();
  for (const user of users.value) {
    if (user.role === "admin" && classEdits[user.id] === undefined) {
      classEdits[user.id] = (user.managedClasses ?? []).join("、");
    }
  }
}

async function handleFileChange(e: Event) {
  const target = e.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  uploading.value = true;
  importResult.value = null;
  try {
    importResult.value = await api.importUsers(file);
    if (importResult.value.created > 0) await loadUsers();
  } catch (err) {
    importResult.value = {
      created: 0,
      skipped: 0,
      errors: [err instanceof Error ? err.message : "导入失败"],
    };
  } finally {
    uploading.value = false;
    target.value = "";
  }
}

async function changeRole(user: UserListItem, newRole: "student" | "reviewer") {
  await api.updateUserRole(user.id, newRole);
  const item = users.value.find((u) => u.id === user.id);
  if (item) item.role = newRole;
  workspace.setToast(user.role === "student" ? `已将 ${user.displayName} 设为班委` : `已取消 ${user.displayName} 的班委角色`);
}

async function saveClasses(user: UserListItem) {
  const classes = (classEdits[user.id] ?? "")
    .split(/[、,，\s]+/)
    .map((item) => item.trim())
    .filter(Boolean);
  const saved = await api.updateUserClasses(user.id, classes);
  const item = users.value.find((u) => u.id === user.id);
  if (item) item.managedClasses = saved.user.managedClasses ?? classes;
  workspace.setToast(`已保存管辖班级：${(item?.managedClasses ?? []).join("、") || "（空）"}`);
}

async function togglePhase() {
  if (!auth.meta) return;
  const next = await api.setSummaryPhase(!auth.meta.summaryPhase);
  auth.meta = next;
  workspace.setToast(next.summaryPhase ? "已进入综测汇总期" : "已恢复日常申报期");
}
</script>

<template>
  <div class="grid" style="grid-template-columns: 1fr 1fr 1fr; gap: 20px; margin-bottom: 20px;">
    <!-- 学生导入 -->
    <div class="card">
      <h3>学生导入</h3>
      <p style="margin: 8px 0; font-size: 0.85rem; color: #666;">
        CSV 格式：学号,姓名,班级<br />第一行为标题行，编码 UTF-8
      </p>
      <label class="field" style="margin-bottom: 8px;">
        <input type="file" accept=".csv,text/csv" :disabled="uploading" @change="handleFileChange" />
      </label>
      <p v-if="uploading" style="color: #666;">导入中...</p>
      <div v-if="importResult" style="font-size: 0.85rem;">
        <p v-if="importResult.created" style="color: green;">新建 {{ importResult.created }} 人</p>
        <p v-if="importResult.skipped" style="color: #999;">跳过 {{ importResult.skipped }} 人（已存在）</p>
        <p v-for="(err, i) in importResult.errors" :key="i" style="color: red;">{{ err }}</p>
      </div>
    </div>

    <!-- 班委授权 -->
    <div class="card">
      <h3>班委授权</h3>
      <p style="margin: 8px 0; font-size: 0.85rem; color: #666;">
        在下方账号列表中点击操作按钮，可将学生设为班委或取消班委。
      </p>
    </div>

    <!-- 综测周期 -->
    <div class="card">
      <h3>综测周期</h3>
      <p style="margin: 8px 0; font-size: 0.85rem;">
        当前阶段：{{ auth.meta?.summaryPhase ? "综测汇总期" : "日常申报期" }}
      </p>
      <button class="secondary" @click="togglePhase">切换阶段</button>
    </div>
  </div>

  <!-- 账号列表 -->
  <div class="card">
    <h3 style="margin-bottom: 12px;">账号列表（{{ users.length }} 人）</h3>
    <div style="overflow-x: auto;">
      <table style="width: 100%; border-collapse: collapse;" v-if="users.length">
        <thead>
          <tr>
            <th style="text-align: left; padding: 8px; border-bottom: 1px solid #eee;">姓名</th>
            <th style="text-align: left; padding: 8px; border-bottom: 1px solid #eee;">学号</th>
            <th style="text-align: left; padding: 8px; border-bottom: 1px solid #eee;">班级</th>
            <th style="text-align: left; padding: 8px; border-bottom: 1px solid #eee;">角色</th>
            <th style="text-align: left; padding: 8px; border-bottom: 1px solid #eee;">操作</th>
            <th style="text-align: left; padding: 8px; border-bottom: 1px solid #eee;">管辖班级</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td style="padding: 8px; border-bottom: 1px solid #f5f5f5;">{{ user.displayName }}</td>
            <td style="padding: 8px; border-bottom: 1px solid #f5f5f5;">{{ user.studentId || "-" }}</td>
            <td style="padding: 8px; border-bottom: 1px solid #f5f5f5;">{{ user.className || "-" }}</td>
            <td style="padding: 8px; border-bottom: 1px solid #f5f5f5;">
              <span v-if="user.isSuperAdmin" class="tag" style="background: #6b21a8; color: #fff;">系统管理员</span>
              <span v-else class="tag" :class="user.role === 'admin' ? 'red' : user.role === 'reviewer' ? 'blue' : 'green'">
                {{ roleLabels[user.role] }}
              </span>
            </td>
            <td style="padding: 8px; border-bottom: 1px solid #f5f5f5;">
              <template v-if="user.role === 'student'">
                <button class="secondary" style="padding: 4px 10px; font-size: 0.8rem;" @click="changeRole(user, 'reviewer')">设为班委</button>
              </template>
              <template v-else-if="user.role === 'reviewer'">
                <button class="secondary" style="padding: 4px 10px; font-size: 0.8rem;" @click="changeRole(user, 'student')">取消班委</button>
              </template>
              <span v-else class="meta">—</span>
            </td>
            <td style="padding: 8px; border-bottom: 1px solid #f5f5f5;">
              <template v-if="user.role === 'admin' && !user.isSuperAdmin">
                <template v-if="isSuper">
                  <div style="display: flex; gap: 6px; align-items: center;">
                    <input
                      v-model="classEdits[user.id]"
                      class="input"
                      style="max-width: 200px; font-size: 0.8rem;"
                      placeholder="如：公信23、公信24"
                    />
                    <button class="secondary" style="padding: 4px 10px; font-size: 0.8rem;" @click="saveClasses(user)">保存</button>
                  </div>
                  <p class="meta" style="margin: 4px 0 0;">当前：{{ (user.managedClasses ?? []).join("、") || "未配置（看不到任何班级数据）" }}</p>
                </template>
                <span v-else class="meta">{{ (user.managedClasses ?? []).join("、") || "未配置（需系统管理员分配）" }}</span>
              </template>
              <span v-else class="meta">—</span>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else style="color: #999;">暂无账号</p>
    </div>
  </div>
</template>
