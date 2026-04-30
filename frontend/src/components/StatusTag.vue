<script setup lang="ts">
import { computed } from "vue";
import { statusColor } from "@/constants/status";

const props = defineProps<{ status: string }>();
const color = computed(() => statusColor(props.status));
</script>

<template>
  <span class="tag" :class="color">{{ status }}</span>
</template>
