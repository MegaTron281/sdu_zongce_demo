<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { api } from "@/api/client";
import StatusTag from "@/components/StatusTag.vue";
import { ActivityStatus } from "@/constants/status";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadAdmin());

const pendingActivities = computed(() => workspace.activities.filter((item) => item.status === ActivityStatus.Pending));

async function setStatus(id: string, status: string) {
  await api.patchActivity(id, { status });
  workspace.setToast("活动状态已更新");
  await workspace.loadAdmin();
}

// 新建活动：辅导员/超管创建后默认进「待审核上架」，再点「确认上架」对学生可见
const showCreate = ref(false);
const creating = ref(false);
const createError = ref("");
const form = reactive({
  name: "",
  organizer: "",
  level: "",
  category: "",
  range: "",
  deadline: "",
  source: "",
  rule: "",
});
const canCreate = computed(() => Boolean(form.name.trim() && form.level && form.category));

function resetForm() {
  form.name = "";
  form.organizer = "";
  form.level = "";
  form.category = "";
  form.range = "";
  form.deadline = "";
  form.source = "";
  form.rule = "";
}

async function createActivity() {
  createError.value = "";
  if (!canCreate.value) {
    createError.value = "请至少填写活动名称、级别、类别";
    return;
  }
  creating.value = true;
  try {
    await api.createActivity({
      name: form.name.trim(),
      organizer: form.organizer.trim(),
      level: form.level,
      category: form.category,
      range: form.range.trim(),
      deadline: form.deadline.trim(),
      source: form.source.trim(),
      rule: form.rule.trim(),
      status: ActivityStatus.Pending,
    });
    workspace.setToast("活动已创建，待审核上架");
    await workspace.loadAdmin();
    resetForm();
    showCreate.value = false;
  } catch (err) {
    createError.value = err instanceof Error ? err.message : "创建失败";
  } finally {
    creating.value = false;
  }
}
</script>

<template>
  <div>
    <!-- 新建活动 -->
    <div class="card" style="margin-bottom: 20px;">
      <div style="display: flex; align-items: center; justify-content: space-between;">
        <h3 style="margin: 0;">新建活动</h3>
        <button class="secondary" @click="showCreate = !showCreate">{{ showCreate ? "收起" : "展开新建" }}</button>
      </div>
      <div v-if="showCreate" class="form-grid" style="margin-top: 14px;">
        <label>活动名称 *<input v-model="form.name" class="input" placeholder="例如：山大杯篮球比赛" /></label>
        <label>主办方<input v-model="form.organizer" class="input" placeholder="例如：体育学院" /></label>
        <label>级别 *<input v-model="form.level" class="input" placeholder="省级 / 校级 / 院级" /></label>
        <label>类别 *<input v-model="form.category" class="input" placeholder="创新素养 / 劳动素养 / 文艺素养 / 身心素养" /></label>
        <label>分值范围<input v-model="form.range" class="input" placeholder="例如：2-8 分" /></label>
        <label>截止日期<input v-model="form.deadline" class="input" placeholder="例如：2026-06-01" /></label>
        <label>来源链接<input v-model="form.source" class="input" placeholder="通知公告 URL（可选）" /></label>
        <label>认定规则<textarea v-model="form.rule" class="textarea" placeholder="说明活动性质、获奖/参与情况、对应综测条款" /></label>
        <p v-if="createError" class="meta" style="color: #c0392b;">{{ createError }}</p>
        <div style="grid-column: 1 / -1; display: flex; gap: 10px;">
          <button class="primary" :disabled="creating" @click="createActivity">{{ creating ? "创建中..." : "创建活动" }}</button>
          <button class="secondary" @click="showCreate = false">取消</button>
        </div>
      </div>
    </div>

    <div class="split-panel">
      <div>
        <p class="section-label">活动名录</p>
        <div class="table-wrap">
          <table>
            <thead><tr><th>活动名称</th><th>级别</th><th>类别</th><th>申报人数</th><th>状态</th></tr></thead>
            <tbody>
              <tr v-for="item in workspace.activities" :key="item.id">
                <td>{{ item.name }}</td>
                <td>{{ item.level }}</td>
                <td>{{ item.category }}</td>
                <td>{{ item.applications }}</td>
                <td><StatusTag :status="item.status" /></td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <div>
        <p class="section-label">学生申报待审</p>
        <div v-for="item in pendingActivities" :key="item.id" class="card">
          <h3>{{ item.name }}</h3>
          <div class="meta"><span>{{ item.level }}</span><span>{{ item.category }}</span><StatusTag :status="item.status" /></div>
          <p>{{ item.organizer || "主办方待确认" }} · {{ item.source || "来源待补充" }}</p>
          <div class="button-row">
            <button class="primary" @click="setStatus(item.id, ActivityStatus.Listed)">确认上架</button>
            <button class="danger" @click="setStatus(item.id, ActivityStatus.Rejected)">驳回</button>
          </div>
        </div>
        <div v-if="pendingActivities.length === 0" class="empty">暂无待审核活动</div>
      </div>
    </div>
  </div>
</template>
