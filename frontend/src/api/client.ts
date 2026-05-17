import type {
  Activity,
  Application,
  AuditLog,
  AuthUser,
  MaterialUpload,
  MetaResponse,
  RuleVersion,
  ScorePreview,
  StudentScore,
} from "./types";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

type ApiOptions = Omit<RequestInit, "body"> & { body?: unknown };

async function request<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");
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
    return request<{ user: AuthUser }>("/api/auth/login", {
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
  preview(body: { activityId?: string; activityLevel?: string; awardLevel: string; projectType: string; rank: number }) {
    return request<ScorePreview>("/api/scores/preview", { method: "POST", body });
  },
  rules() {
    return request<RuleVersion[]>("/api/admin/rules");
  },
  createRule(body: { name: string; matrix: Record<string, number>; moduleMax: Record<string, number> }) {
    return request<RuleVersion>("/api/admin/rules", { method: "POST", body });
  },
  logs() {
    return request<AuditLog[]>("/api/admin/logs");
  },
};
