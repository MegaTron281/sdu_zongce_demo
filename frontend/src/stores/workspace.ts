import { defineStore } from "pinia";
import { api } from "@/api/client";
import type { Activity, Application, AuditLog, RuleVersion, StudentScore } from "@/api/types";

interface WorkspaceState {
  activities: Activity[];
  applications: Application[];
  publicity: Application[];
  rules: RuleVersion[];
  logs: AuditLog[];
  score: StudentScore | null;
  scoreRows: StudentScore[];
  loading: boolean;
  error: string;
  toast: string;
}

export const useWorkspaceStore = defineStore("workspace", {
  state: (): WorkspaceState => ({
    activities: [],
    applications: [],
    publicity: [],
    rules: [],
    logs: [],
    score: null,
    scoreRows: [],
    loading: false,
    error: "",
    toast: "",
  }),
  actions: {
    async loadBase() {
      this.loading = true;
      this.error = "";
      try {
        const [activities, applications, publicity] = await Promise.all([
          api.activities(),
          api.applications(),
          api.publicity(),
        ]);
        this.activities = activities;
        this.applications = applications;
        this.publicity = publicity;
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      } finally {
        this.loading = false;
      }
    },
    async loadAdmin() {
      await this.loadBase();
      const [rules, logs] = await Promise.all([api.rules(), api.logs()]);
      this.rules = rules;
      this.logs = logs;
      const ids = new Set(this.applications.map((item) => item.studentId));
      this.scoreRows = await Promise.all([...ids].map((id) => api.score(id)));
    },
    async loadStudent(studentId: string) {
      await this.loadBase();
      this.score = await api.score(studentId);
    },
    async loadReviewer() {
      await this.loadBase();
    },
    setToast(message: string) {
      this.toast = message;
      window.setTimeout(() => {
        if (this.toast === message) this.toast = "";
      }, 2600);
    },
  },
});
