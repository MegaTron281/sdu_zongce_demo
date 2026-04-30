<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "@/api/client";
import { formatNumber } from "@/api/format";

const props = defineProps<{
  activityId: string;
  awardLevel: string;
  projectType: string;
  rank: number;
}>();

const score = ref(0);
const ruleName = ref("当前规则");
const loading = ref(false);

const payload = computed(() => ({
  activityId: props.activityId,
  awardLevel: props.awardLevel,
  projectType: props.projectType,
  rank: props.rank,
}));

watch(
  payload,
  async (value) => {
    if (!value.activityId) return;
    loading.value = true;
    try {
      const preview = await api.preview(value);
      score.value = preview.score;
      ruleName.value = preview.ruleName;
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);
</script>

<template>
  <div class="score-box">
    <div class="score-value">{{ loading ? "..." : formatNumber(score) }} 分</div>
    <p>规则版本：{{ ruleName }} · 最终分值以后端规则计算为准</p>
  </div>
</template>
