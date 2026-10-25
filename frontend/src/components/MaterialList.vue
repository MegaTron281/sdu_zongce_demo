<script setup lang="ts">
import { authFetchBlob } from "@/api/client";

const props = defineProps<{ items?: string[] }>();

function materialInfo(item: string) {
  const separator = item.indexOf("|");
  const name = separator >= 0 ? item.slice(0, separator).trim() : item.trim();
  const target = separator >= 0 ? item.slice(separator + 1).trim() : item.trim();
  const isSafeLink = target.startsWith("/api/materials/") || target.startsWith("http://") || target.startsWith("https://");
  // 站外链接不需要本站鉴权，保持普通跳转；站内 /api/materials/ 需要带令牌拉取
  const isExternal = target.startsWith("http://") || target.startsWith("https://");
  return { name: name || target, target, isSafeLink, isExternal };
}

// 证明材料预览：用本页签的 Bearer 令牌拉取后以 blob 打开，
// 不再依赖浏览器共享 cookie，避免并行多账号时预览到别的账号的材料（或 401）
async function openMaterial(item: string) {
  const info = materialInfo(item);
  if (!info.isSafeLink) return;
  if (info.isExternal) {
    window.open(info.target, "_blank", "noopener,noreferrer");
    return;
  }
  try {
    const blob = await authFetchBlob(info.target);
    const url = URL.createObjectURL(blob);
    window.open(url, "_blank");
    // 交给浏览器后延迟回收，避免新标签页还没加载就被撤销
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch {
    window.alert("材料加载失败，请确认登录状态后重试");
  }
}
</script>

<template>
  <div v-if="props.items?.length" class="material-list">
    <span class="material-title">证明材料</span>
    <template v-for="item in props.items" :key="item">
      <a
        v-if="materialInfo(item).isSafeLink"
        class="material-link"
        href="javascript:void(0)"
        @click.prevent="openMaterial(item)"
      >
        {{ materialInfo(item).name }}
      </a>
      <span v-else class="material-link">{{ materialInfo(item).name }}</span>
    </template>
  </div>
</template>
