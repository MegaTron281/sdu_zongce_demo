import { defineStore } from "pinia";
import { api, ApiError } from "@/api/client";
import type { AuthUser, MetaResponse, Role } from "@/api/types";

interface AuthState {
  user: AuthUser | null;
  meta: MetaResponse | null;
  loading: boolean;
}

export const roleLabels: Record<Role, string> = {
  student: "普通学生",
  reviewer: "审核学生",
  admin: "管理员",
};

export const useAuthStore = defineStore("auth", {
  state: (): AuthState => ({
    user: null,
    meta: null,
    loading: false,
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.user),
  },
  actions: {
    async login(role: Role, account: string, password: string) {
      const { user } = await api.login(role, account, password);
      this.user = user;
      this.meta = await api.meta();
    },
    async ensureMe() {
      if (this.user) return this.user;
      try {
        const { user } = await api.me();
        this.user = user;
        this.meta = await api.meta();
        return user;
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          this.user = null;
          return null;
        }
        throw error;
      }
    },
    async refreshMeta() {
      this.meta = await api.meta();
    },
    async logout() {
      await api.logout();
      this.user = null;
      this.meta = null;
    },
  },
});
