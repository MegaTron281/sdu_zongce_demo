import type {
  Activity,
  Application,
  AuditLog,
  AuthUser,
  MaterialUpload,
  MetaResponse,
  StudentScore,
  Objection,
  UserListItem,
  UserImportResult,
} from "./types";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

// 每个标签页独立的会话令牌：存于 sessionStorage（随标签页关闭失效，不会像
// localStorage 那样在同源多标签页间共享），配合后端的 Authorization: Bearer
// 头，使得同一浏览器内多个标签页可各自保持不同账号的登录态，互不顶替。
const TOKEN_KEY = "sdu_token";

export function setSessionToken(token: string | null) {
  if (token) {
    sessionStorage.setItem(TOKEN_KEY, token);
  } else {
    sessionStorage.removeItem(TOKEN_KEY);
  }
}

export function getSessionToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY);
}

/**
 * 需要鉴权的二进制资源（导出成绩册、证明材料预览）下载。
 *
 * 这里显式带上本页签的 Bearer 令牌，并且刻意不携带 cookie（credentials: "omit"），
 * 使身份完全由本页签令牌决定：并行多账号操作时，导出/预览只会取"当前页签这个账号"
 * 有权限的数据，不会被浏览器里最后一次登录的账号串味。
 */
export async function authFetchBlob(path: string): Promise<Blob> {
  const headers = new Headers();
  const token = getSessionToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const response = await fetch(path, { headers, credentials: "omit" });
  if (!response.ok) {
    const text = await response.text();
    let message = `加载失败：${response.status}`;
    try {
      const parsed = JSON.parse(text);
      if (parsed?.error) message = parsed.error;
    } catch {
      // 非 JSON 响应（例如错误页）保持默认提示
    }
    throw new ApiError(message, response.status);
  }
  return response.blob();
}

type ApiOptions = Omit<RequestInit, "body"> & { body?: unknown };

async function request<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");
  // 携带本标签页独立保存的会话令牌（并行多账号操作的关键）
  const token = getSessionToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  let body = options.body;
  if (body !== undefined && typeof body !== "string" && !(body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(body);
  }
  const response = await fetch(path, {
    ...options,
    headers,
    body: body as BodyInit | undefined,
    credentials: "include",
  });
  const contentType = response.headers.get("content-type") || "";
  const payload = contentType.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok) {
    const message = typeof payload === "object" && payload?.error ? payload.error : `请求失败：${response.status}`;
    throw new ApiError(message, response.status);
  }
  return payload as T;
}

export const api = {
  login(role: string, account: string, password: string) {
    return request<{ user: AuthUser; token: string }>("/api/auth/login", {
      method: "POST",
      body: { role, account, password },
    });
  },
  me() {
    return request<{ user: AuthUser }>("/api/auth/me");
  },
  logout() {
    return request<{ status: string }>("/api/auth/logout", { method: "POST" });
  },
  meta() {
    return request<MetaResponse>("/api/meta");
  },
  setSummaryPhase(summaryPhase: boolean) {
    return request<MetaResponse>("/api/meta", { method: "PATCH", body: { summaryPhase } });
  },
  activities() {
    return request<Activity[]>("/api/activities");
  },
  createActivity(body: Partial<Activity>) {
    return request<Activity>("/api/activities", { method: "POST", body });
  },
  patchActivity(id: string, body: Partial<Activity>) {
    return request<Activity>(`/api/activities/${encodeURIComponent(id)}`, { method: "PATCH", body });
  },
  applications(params: Record<string, string> = {}) {
    const search = new URLSearchParams(params);
    const suffix = search.toString() ? `?${search}` : "";
    return request<Application[]>(`/api/applications${suffix}`);
  },
  createApplication(body: {
    activityId?: string;
    activityName?: string;
    organizer?: string;
    level?: string;
    category?: string;
    range?: string;
    deadline?: string;
    source?: string;
    rule?: string;
    awardLevel: string;
    projectType: string;
    rank: number;
    score: number;
    materials: string[];
  }) {
    return request<Application>("/api/applications", { method: "POST", body });
  },
  action(id: string, action: string, body: { reason?: string; score?: number | null }) {
    const payload: Record<string, string | number> = {};
    if (body.reason !== undefined) payload.reason = body.reason;
    if (body.score !== undefined && body.score !== null) payload.score = body.score;
    return request<Application>(`/api/applications/${encodeURIComponent(id)}/${action}`, { method: "POST", body: payload });
  },
  uploadMaterial(file: File) {
    const body = new FormData();
    body.append("file", file);
    return request<MaterialUpload>("/api/materials/upload", { method: "POST", body });
  },
  publicity() {
    return request<Application[]>("/api/publicity");
  },
  score(studentId: string) {
    return request<StudentScore>(`/api/scores/${encodeURIComponent(studentId)}`);
  },
  objections(applicationId: string) {
    return request<Objection[]>(`/api/applications/${encodeURIComponent(applicationId)}/objections`);
  },
  logs() {
    return request<AuditLog[]>("/api/admin/logs");
  },
  changePassword(oldPassword: string, newPassword: string) {
    return request<{ status: string }>("/api/auth/change-password", {
      method: "POST",
      body: { oldPassword, newPassword },
    });
  },
  listUsers(limit = 200, offset = 0) {
    return request<UserListItem[]>(`/api/admin/users?limit=${limit}&offset=${offset}`);
  },
  importUsers(file: File) {
    const body = new FormData();
    body.append("file", file);
    return request<UserImportResult>("/api/admin/users/import", { method: "POST", body });
  },
  updateUserRole(id: string, role: string) {
    return request<{ status: string }>(`/api/admin/users/${encodeURIComponent(id)}/role`, {
      method: "PATCH",
      body: { role },
    });
  },
  updateUserClasses(id: string, classes: string[]) {
    return request<{ user: AuthUser }>(`/api/admin/users/${encodeURIComponent(id)}/classes`, {
      method: "PATCH",
      body: { classes },
    });
  },
};
