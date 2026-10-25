import { defineStore } from "pinia";
import { api, ApiError, setSessionToken, getSessionToken } from "@/api/client";
import type { AuthUser, MetaResponse, Role } from "@/api/types";

interface AuthState {
  user: AuthUser | null;
  meta: MetaResponse | null;
  loading: boolean;
}

export const roleLabels: Record<Role, string> = {
  student: "普通学生",
  reviewer: "审核学生",
  admin: "辅导员",
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
      const { user, token } = await api.login(role, account, password);
      // 把本标签页独立的会话令牌存入 sessionStorage，使并行多账号操作互不干扰
      setSessionToken(token);
      this.user = user;
      this.meta = await api.meta();
    },
    async ensureMe() {
      if (this.user) return this.user;
      // 页签级身份只认本页签自己的 Bearer 令牌（存于 sessionStorage）。
      // 没有令牌时**不回退**到浏览器共享的 session cookie：否则新建的页签会直接继承
      // "最后一次登录"的账号，表现为"登录另一个账号后原账号被顶下线"，并行多账号操作失效。
      // 每个页签各自登录一次后即互不影响。
      if (!getSessionToken()) {
        this.user = null;
        return null;
      }
      try {
        const { user } = await api.me();
        this.user = user;
        this.meta = await api.meta();
        return user;
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          this.user = null;
          setSessionToken(null);
          return null;
        }
        throw error;
      }
    },
    async refreshMeta() {
      this.meta = await api.meta();
    },
    async logout() {
      try {
        await api.logout();
      } finally {
        setSessionToken(null);
        this.user = null;
        this.meta = null;
      }
    },
  },
});
