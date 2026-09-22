<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "@/api/client";
import { useAuthStore } from "@/stores/auth";
import { useWorkspaceStore } from "@/stores/workspace";

const auth = useAuthStore();
const workspace = useWorkspaceStore();
const route = useRoute();
const router = useRouter();
const uploadedMaterials = ref<string[]>([]);
const uploading = ref(false);
const submitting = ref(false);
const uploadError = ref("");
const submitError = ref("");
// 从「活动广场」进入时带入的活动 id；为空表示学生自填活动信息（补录）
const selectedActivityId = ref("");

const draft = reactive({
  activityName: "",
  organizer: "",
  level: "校级",
  category: "劳动素养",
  range: "待管理员确认",
  deadline: "",
  source: "",
  rule: "学生自填活动，待管理员复核。",
  awardLevel: "一等奖",
  projectType: "个人项目",
  rank: 1,
  score: 0,
  materials: "证书图片、公示截图、官方报道链接",
});

onMounted(async () => {
  await workspace.loadBase();
  const activityId = String(route.query.activityId || "");
  const activity = workspace.activities.find((item) => item.id === activityId);
  if (activity) {
    selectedActivityId.value = activity.id;
    draft.activityName = activity.name;
    draft.organizer = activity.organizer;
    draft.level = activity.level;
    draft.category = activity.category;
    draft.range = activity.range || "待管理员确认";
    draft.deadline = activity.deadline;
    draft.source = activity.source;
    draft.rule = activity.rule || "学生自填活动，待管理员复核。";
  }
});

const locked = computed(() => Boolean(auth.meta?.summaryPhase));
// 申报的是名录中已有活动：活动信息锁定，提交时回传 activityId，后端据此复用该活动而非新建
const fromCatalog = computed(() => Boolean(selectedActivityId.value));
const fieldsLocked = computed(() => locked.value || fromCatalog.value);
const canSubmit = computed(() => Boolean(draft.activityName.trim() && draft.level && draft.category && !locked.value && !uploading.value && !submitting.value));

function materialList(value: string) {
  return value
    .split(/[、,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function materialName(item: string) {
  const separator = item.indexOf("|");
  return separator >= 0 ? item.slice(0, separator) : item;
}

function combinedMaterials() {
  return [...materialList(draft.materials), ...uploadedMaterials.value];
}

async function uploadMaterialFiles(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files || []);
  if (files.length === 0) return;

  uploadError.value = "";
  uploading.value = true;
  try {
    for (const file of files) {
      const material = await api.uploadMaterial(file);
      uploadedMaterials.value.push(`${material.name}|${material.url}`);
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : "材料上传失败";
    uploadError.value = message === "not found" ? "材料上传接口未生效，请重启 Gin 后端后再上传。" : message;
  } finally {
    uploading.value = false;
    input.value = "";
  }
}

function removeUploadedMaterial(index: number) {
  uploadedMaterials.value.splice(index, 1);
}

async function submitApplication() {
  submitError.value = "";
  submitting.value = true;
  try {
    await api.createApplication({
      activityId: selectedActivityId.value || undefined,
      activityName: draft.activityName,
      organizer: draft.organizer,
      level: draft.level,
      category: draft.category,
      range: draft.range,
      deadline: draft.deadline,
      source: draft.source,
      rule: draft.rule,
      awardLevel: draft.awardLevel,
      projectType: draft.projectType,
      rank: Number(draft.rank || 1),
      score: Number(draft.score || 0),
      materials: combinedMaterials(),
    });
    workspace.setToast("申请已提交，等待班委审核");
    router.push({ name: "student.applications" });
  } catch (error) {
    const message = error instanceof Error ? error.message : "提交申请失败";
    submitError.value = message === "not found" ? "提交接口未生效，请重启 Gin 后端后再试。" : message;
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="split-panel">
    <div>
      <p class="section-label">{{ fromCatalog ? "申报活动（活动信息取自活动名录）" : "学生自填活动信息并申报" }}</p>
      <p v-if="fromCatalog" class="feedback">
        你正在申报活动名录中已有的活动「{{ draft.activityName }}」，活动信息已锁定；只需填写获奖情况与申报分数。
      </p>
      <p v-else class="feedback">
        活动名录中没有你要申报的活动？可直接在此自填活动信息申报，提交后由辅导员审核上架（不会重复建活动）。
      </p>
      <form class="form-grid" @submit.prevent="submitApplication">
        <div class="field full">
          <label>活动名称</label>
          <input v-model="draft.activityName" class="input" placeholder="例如：山大杯篮球比赛" :disabled="fieldsLocked" required />
        </div>
        <div class="field">
          <label>主办方</label>
          <input v-model="draft.organizer" class="input" placeholder="例如：体育学院" :disabled="fieldsLocked" />
        </div>
        <div class="field">
          <label>活动级别</label>
          <select v-model="draft.level" class="select" :disabled="fieldsLocked">
            <option>国家级</option>
            <option>省级</option>
            <option>校级</option>
            <option>院级</option>
          </select>
        </div>
        <div class="field">
          <label>综测类别</label>
          <select v-model="draft.category" class="select" :disabled="fieldsLocked">
            <option>身心素养</option>
            <option>文艺素养</option>
            <option>劳动素养</option>
            <option>创新素养</option>
          </select>
        </div>
        <div class="field">
          <label>活动日期 / 截止日期</label>
          <input v-model="draft.deadline" class="input" type="date" :disabled="fieldsLocked" />
        </div>
        <div class="field full">
          <label>官方来源 / 公示链接</label>
          <input v-model="draft.source" class="input" placeholder="官网通知、公众号推文或成绩册链接" :disabled="fieldsLocked" />
        </div>
        <div class="field full">
          <label>计分依据说明</label>
          <textarea v-model="draft.rule" class="textarea" placeholder="说明活动性质、获奖/参与情况、对应综测条款" :disabled="fieldsLocked"></textarea>
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
          <input v-model.number="draft.rank" class="input" min="1" type="number" :disabled="locked || draft.projectType !== '集体项目'" />
        </div>
        <div class="field">
          <label>申报加分</label>
          <input v-model.number="draft.score" class="input" type="number" min="0" step="0.5" placeholder="自行填写申报分数" :disabled="locked" required />
        </div>
        <div class="field full">
          <label>证明材料</label>
          <input v-model="draft.materials" class="input" placeholder="可填写证书说明、公示链接；多项用逗号或换行分隔" :disabled="locked" />
          <label class="upload-box">
            <input
              type="file"
              multiple
              accept=".pdf,.png,.jpg,.jpeg,application/pdf,image/png,image/jpeg"
              :disabled="locked || uploading"
              @change="uploadMaterialFiles"
            />
            <span>
              <strong>{{ uploading ? "上传中..." : "上传 PDF / PNG / JPG" }}</strong>
              <small>支持多选，单个文件不超过 10MB</small>
            </span>
          </label>
          <p v-if="uploadError" class="feedback error">{{ uploadError }}</p>
          <ul v-if="uploadedMaterials.length" class="upload-list">
            <li v-for="(item, index) in uploadedMaterials" :key="item">
              <span>{{ materialName(item) }}</span>
              <button type="button" class="text-button danger-text" @click="removeUploadedMaterial(index)">移除</button>
            </li>
          </ul>
        </div>
        <p v-if="submitError" class="feedback error full">{{ submitError }}</p>
        <button class="primary" type="submit" :disabled="!canSubmit">{{ submitting ? "提交中..." : "提交申请" }}</button>
      </form>
    </div>
    <div>
      <div class="panel">
        <p class="section-label">申报说明</p>
        <p>学生自行填写申报分数，提交后由班委初审。班委和老师审核时可直接通过、修改分数后通过、或驳回。</p>
      </div>
    </div>
  </div>
</template>
