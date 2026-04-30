<script setup lang="ts">
import { onMounted } from "vue";
import ApplicationTable from "@/components/ApplicationTable.vue";
import { useWorkspaceStore } from "@/stores/workspace";

const workspace = useWorkspaceStore();
onMounted(() => workspace.loadReviewer());
</script>

<template>
  <ApplicationTable :items="workspace.applications" />
</template>
