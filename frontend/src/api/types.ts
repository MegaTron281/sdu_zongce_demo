export type Role = "student" | "reviewer" | "admin";

export interface AuthUser {
  id: string;
  role: Role;
  displayName: string;
  studentId?: string;
  className?: string;
}

export interface MetaResponse {
  summaryPhase: boolean;
  applicationStatuses: string[];
  activityStatuses: string[];
  modules: string[];
  activeRule: { id: string; name: string };
}

export interface Activity {
  id: string;
  name: string;
  organizer: string;
  level: string;
  category: string;
  range: string;
  deadline: string;
  source: string;
  rule: string;
  status: string;
  applications: number;
}

export interface Application {
  id: string;
  student: string;
  studentId: string;
  className: string;
  activityId: string;
  activity: string;
  category: string;
  awardLevel: string;
  projectType: string;
  rank: number;
  score: number;
  status: string;
  submittedAt: string;
  publicityUntil?: string;
  reviewerOpinion: string;
  adminOpinion: string;
  specialReason?: string;
  materials: string[];
}

export interface MaterialUpload {
  name: string;
  url: string;
  size: number;
  contentType: string;
}

export interface RuleVersion {
  id: string;
  name: string;
  active: boolean;
  publishedAt: string;
  matrix: Record<string, number>;
  moduleMax: Record<string, number>;
}

export interface AuditLog {
  id: string;
  actor: string;
  actorRole: string;
  action: string;
  target: string;
  note: string;
  createdAt: string;
}

export interface ScoreModule {
  name: string;
  score: number;
  max: number;
}

export interface StudentScore {
  student: string;
  studentId: string;
  className: string;
  academicScore: number;
  qualityScore: number;
  assessmentScore: number;
  modules: ScoreModule[];
  locked: boolean;
}

export interface ScorePreview {
  score: number;
  baseScore: number;
  ruleId: string;
  ruleName: string;
  ruleKey: string;
  activityLevel: string;
  awardLevel: string;
  projectType: string;
  rank: number;
  rankAdjusted: boolean;
}
