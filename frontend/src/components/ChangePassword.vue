<script setup lang="ts">
import { ref } from "vue";
import { api } from "@/api/client";

const emit = defineEmits<{ close: [] }>();

const oldPassword = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const loading = ref(false);
const error = ref("");
const success = ref("");

async function submit() {
  error.value = "";
  success.value = "";

  if (!oldPassword.value || !newPassword.value) {
    error.value = "请填写新旧密码";
    return;
  }
  if (newPassword.value.length < 6) {
    error.value = "新密码长度至少 6 位";
    return;
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = "两次输入的新密码不一致";
    return;
  }

  loading.value = true;
  try {
    await api.changePassword(oldPassword.value, newPassword.value);
    success.value = "密码修改成功";
    setTimeout(() => emit("close"), 1500);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "修改失败";
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="pwd-modal-overlay" @click.self="emit('close')">
    <div class="pwd-modal-card">
      <div class="pwd-modal-header">
        <div class="pwd-modal-icon">🔒</div>
        <div>
          <h3>修改密码</h3>
          <p>请输入旧密码并设置新密码，长度至少 6 位</p>
        </div>
        <button class="pwd-modal-close" @click="emit('close')" aria-label="关闭">✕</button>
      </div>

      <div v-if="error" class="notice-strip error">{{ error }}</div>
      <div v-if="success" class="notice-strip success">{{ success }}</div>

      <form @submit.prevent="submit" class="pwd-form">
        <div class="field full">
          <label>旧密码</label>
          <input
            class="input"
            v-model="oldPassword"
            type="password"
            placeholder="输入当前密码"
            autocomplete="current-password"
          />
        </div>
        <div class="field full">
          <label>新密码</label>
          <input
            class="input"
            v-model="newPassword"
            type="password"
            placeholder="输入新密码（至少 6 位）"
            autocomplete="new-password"
          />
        </div>
        <div class="field full">
          <label>确认新密码</label>
          <input
            class="input"
            v-model="confirmPassword"
            type="password"
            placeholder="再次输入新密码"
            autocomplete="new-password"
          />
        </div>
        <div class="pwd-modal-actions">
          <button type="button" class="secondary" @click="emit('close')">取消</button>
          <button type="submit" class="primary" :disabled="loading">
            {{ loading ? "提交中..." : "确认修改" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
