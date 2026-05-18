<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "@/api/client";
import { formatNumber } from "@/api/format";
import type { ScorePreview } from "@/api/types";

const props = defineProps<{
  activityId?: string;
  activityLevel?: string;
  awardLevel: string;
  projectType: string;
  rank: number;
}>();

const preview = ref<ScorePreview | null>(null);
const loading = ref(false);
const error = ref("");
let requestSeq = 0;

const payload = computed(() => ({
  activityId: props.activityId,
  activityLevel: props.activityLevel,
  awardLevel: props.awardLevel,
  projectType: props.projectType,
  rank: Number(props.rank || 1),
}));

const score = computed(() => preview.value?.score ?? 0);
const ruleName = computed(() => preview.value?.ruleName || "当前规则");
const matchText = computed(() => {
  if (!preview.value) return "等待选择活动级别和奖项名次";
  return `${preview.value.activityLevel} · ${preview.value.awardLevel}`;
});
const rankText = computed(() => {
  if (!preview.value) return "";
  if (!preview.value.rankAdjusted) return "个人项目或第一位次，不做位次折减";
  return `集体项目第 ${preview.value.rank} 位，按规则做位次折减`;
});

watch(
  payload,
  async (value) => {
    if (!value.activityId && !value.activityLevel) return;
    const seq = ++requestSeq;
    loading.value = true;
    error.value = "";
    try {
      const data = await api.preview(value);
      if (seq !== requestSeq) return;
      preview.value = data;
    } catch (err) {
      if (seq !== requestSeq) return;
      error.value = err instanceof Error ? err.message : "计分预览失败";
    } finally {
      if (seq === requestSeq) loading.value = false;
    }
  },
  { immediate: true },
);
</script>

<template>
  <div class="score-box">
    <div class="score-value">{{ loading ? "..." : formatNumber(score) }} 分</div>
    <p>规则版本：{{ ruleName }} · 最终分值以后端规则计算为准</p>
    <div v-if="preview" class="score-rule-list">
      <div><span>匹配规则</span><strong>{{ matchText }}</strong></div>
      <div><span>基础分</span><strong>{{ formatNumber(preview.baseScore) }} 分</strong></div>
      <div><span>项目位次</span><strong>{{ rankText }}</strong></div>
    </div>
    <p v-if="error" class="feedback error">{{ error }}</p>
  </div>
</template>
