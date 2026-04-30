<script setup lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "@/api/client";
import ScorePreview from "@/components/ScorePreview.vue";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
const route = useRoute();
const router = useRouter();

const draft = reactive({
  activityId: "",
  awardLevel: "一等奖",
  projectType: "个人项目",
  rank: 1,
  materials: "证书图片、公示截图、官方报道链接",
});

const activityDraft = reactive({
  name: "",
  organizer: "",
  level: "校级",
  category: "劳动素养",
  deadline: "",
  source: "",
});

onMounted(async () => {
  await workspace.loadBase();
  draft.activityId = String(route.query.activityId || workspace.activities[0]?.id || "");
});

const selectedActivity = computed(() => workspace.activities.find((item) => item.id === draft.activityId));
const locked = computed(() => Boolean(auth.meta?.summaryPhase));

function materialList(value: string) {
  return value
    .split(/[、,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

async function submitApplication() {
  await api.createApplication({
    activityId: draft.activityId,
    awardLevel: draft.awardLevel,
    projectType: draft.projectType,
    rank: Number(draft.rank || 1),
    materials: materialList(draft.materials),
  });
  workspace.setToast("申请已提交，等待班委审核");
  router.push({ name: "student.applications" });
}

async function submitActivity() {
  await api.createActivity({
    name: activityDraft.name,
    organizer: activityDraft.organizer,
    level: activityDraft.level,
    category: activityDraft.category,
    range: "待管理员确认",
    deadline: activityDraft.deadline,
    source: activityDraft.source,
    rule: "补录活动，待管理员维护规则说明。",
  });
  workspace.setToast("补录活动已提交，等待管理员上架");
  Object.assign(activityDraft, { name: "", organizer: "", level: "校级", category: "劳动素养", deadline: "", source: "" });
  await workspace.loadBase();
}
</script>

<template>
  <div class="split-panel">
    <div>
      <p class="section-label">路径 A：从活动名录发起申请</p>
      <form class="form-grid" @submit.prevent="submitApplication">
        <div class="field full">
          <label>活动名称</label>
          <select v-model="draft.activityId" class="select" :disabled="locked">
            <option v-for="activity in workspace.activities" :key="activity.id" :value="activity.id">
              {{ activity.name }}
            </option>
          </select>
        </div>
        <div class="field">
          <label>获奖等级</label>
          <select v-model="draft.awardLevel" class="select" :disabled="locked">
            <option>一等奖</option>
            <option>二等奖</option>
            <option>三等奖</option>
            <option>其他名次</option>
          </select>
        </div>
        <div class="field">
          <label>项目类型</label>
          <select v-model="draft.projectType" class="select" :disabled="locked">
            <option>个人项目</option>
            <option>集体项目</option>
          </select>
        </div>
        <div class="field">
          <label>集体项目本人位次</label>
          <input v-model.number="draft.rank" class="input" min="1" type="number" :disabled="locked" />
        </div>
        <div class="field full">
          <label>证明材料</label>
          <input v-model="draft.materials" class="input" :disabled="locked" />
        </div>
        <button class="primary" type="submit" :disabled="locked || !selectedActivity">提交申请</button>
      </form>
    </div>
    <div>
      <p class="section-label">实时计分预览</p>
      <ScorePreview
        v-if="selectedActivity"
        :activity-id="draft.activityId"
        :award-level="draft.awardLevel"
        :project-type="draft.projectType"
        :rank="draft.rank"
      />
      <hr />
      <p class="section-label">路径 B：补录新活动</p>
      <form class="compact-form" @submit.prevent="submitActivity">
        <input v-model="activityDraft.name" class="input" placeholder="活动名称" :disabled="locked" required />
        <input v-model="activityDraft.organizer" class="input" placeholder="主办方" :disabled="locked" />
        <select v-model="activityDraft.level" class="select" :disabled="locked">
          <option>国家级</option>
          <option>省级</option>
          <option>校级</option>
          <option>院级</option>
        </select>
        <select v-model="activityDraft.category" class="select" :disabled="locked">
          <option>身心素养</option>
          <option>文艺素养</option>
          <option>劳动素养</option>
          <option>创新素养</option>
        </select>
        <input v-model="activityDraft.deadline" class="input" type="date" :disabled="locked" />
        <input v-model="activityDraft.source" class="input" placeholder="来源链接" :disabled="locked" />
        <button class="ghost" type="submit" :disabled="locked">提交补录</button>
      </form>
    </div>
  </div>
</template>
