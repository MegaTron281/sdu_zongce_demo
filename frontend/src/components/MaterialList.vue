<script setup lang="ts">
defineProps<{ items?: string[] }>();

function materialInfo(item: string) {
  const separator = item.indexOf("|");
  const name = separator >= 0 ? item.slice(0, separator).trim() : item.trim();
  const target = separator >= 0 ? item.slice(separator + 1).trim() : item.trim();
  const isSafeLink = target.startsWith("/api/materials/") || target.startsWith("http://") || target.startsWith("https://");
  return { name: name || target, target, isSafeLink };
}
</script>

<template>
  <div v-if="items?.length" class="material-list">
    <span class="material-title">证明材料</span>
    <template v-for="item in items" :key="item">
      <a v-if="materialInfo(item).isSafeLink" class="material-link" :href="materialInfo(item).target" target="_blank" rel="noreferrer">
        {{ materialInfo(item).name }}
      </a>
      <span v-else class="material-link">{{ materialInfo(item).name }}</span>
    </template>
  </div>
</template>
