<script setup lang="ts">
import { computed, onMounted, reactive, watch } from "vue";
import { api } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
const levels = ["国家级", "省级", "校级", "院级"];
const awards = ["一等奖", "二等奖", "三等奖", "其他名次"];
const matrix = reactive<Record<string, number>>({});
const moduleMax = reactive<Record<string, number>>({});

const activeRule = computed(() => workspace.rules.find((item) => item.active) || workspace.rules[0]);
const moduleNames = computed(() => auth.meta?.modules ?? Object.keys(activeRule.value?.moduleMax ?? {}));

function syncFromRule() {
  Object.keys(matrix).forEach((key) => delete matrix[key]);
  Object.keys(moduleMax).forEach((key) => delete moduleMax[key]);
  Object.assign(matrix, activeRule.value?.matrix ?? {});
  Object.assign(moduleMax, activeRule.value?.moduleMax ?? {});
  for (const name of moduleNames.value) {
    if (!(name in moduleMax)) moduleMax[name] = 25;
  }
}

onMounted(async () => {
  await workspace.loadAdmin();
  syncFromRule();
});

watch(activeRule, syncFromRule);

async function publish() {
  await api.createRule({
    name: `规则 ${new Date().toLocaleDateString("zh-CN")}`,
    matrix: { ...matrix },
    moduleMax: { ...moduleMax },
  });
  workspace.setToast("新规则版本已发布");
  await workspace.loadAdmin();
}
</script>

<template>
  <div class="panel">
    <p class="section-label">规则版本：{{ activeRule?.name || "未发布" }}</p>
    <div class="button-row">
      <button class="primary" @click="publish">发布新版本</button>
      <button class="secondary" @click="workspace.setToast(workspace.rules.map((item) => `${item.name} · ${item.publishedAt}`).join('；'))">
        查看历史版本
      </button>
    </div>
  </div>
  <div class="table-wrap">
    <table>
      <thead><tr><th>活动级别</th><th v-for="award in awards" :key="award">{{ award }}</th></tr></thead>
      <tbody>
        <tr v-for="level in levels" :key="level">
          <td>{{ level }}</td>
          <td v-for="award in awards" :key="award">
            <input v-model.number="matrix[`${level}:${award}`]" class="input" type="number" min="0" step="0.5" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
  <div class="panel">
    <p class="section-label">模块上限</p>
    <div class="form-grid">
      <div v-for="name in moduleNames" :key="name" class="field">
        <label>{{ name }}</label>
        <input v-model.number="moduleMax[name]" class="input" type="number" min="0" step="1" />
      </div>
    </div>
  </div>
</template>
