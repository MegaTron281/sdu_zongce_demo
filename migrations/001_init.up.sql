CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  role TEXT NOT NULL CHECK (role IN ('student', 'reviewer', 'admin')),
  display_name TEXT NOT NULL,
  student_id TEXT,
  class_name TEXT,
  account TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assessment_cycles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  summary_phase BOOLEAN NOT NULL DEFAULT false,
  active BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS activities (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  organizer TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL,
  category TEXT NOT NULL,
  range TEXT NOT NULL DEFAULT '',
  deadline TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  rule TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '已上架',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rule_versions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT false,
  published_at TEXT NOT NULL,
  matrix JSONB NOT NULL DEFAULT '{}'::jsonb,
  module_max JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS applications (
  id TEXT PRIMARY KEY,
  student TEXT NOT NULL,
  student_id TEXT NOT NULL,
  class_name TEXT NOT NULL,
  activity_id TEXT REFERENCES activities(id) ON DELETE SET NULL,
  activity TEXT NOT NULL,
  category TEXT NOT NULL,
  award_level TEXT NOT NULL DEFAULT '',
  project_type TEXT NOT NULL DEFAULT '',
  rank INTEGER NOT NULL DEFAULT 1,
  score NUMERIC(8,2) NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  submitted_at TEXT NOT NULL,
  publicity_until TIMESTAMPTZ,
  reviewer_opinion TEXT NOT NULL DEFAULT '',
  admin_opinion TEXT NOT NULL DEFAULT '',
  special_reason TEXT NOT NULL DEFAULT '',
  materials JSONB NOT NULL DEFAULT '[]'::jsonb,
  cycle_id TEXT NOT NULL DEFAULT '2026',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS academic_scores (
  student_id TEXT PRIMARY KEY,
  student TEXT NOT NULL,
  class_name TEXT NOT NULL,
  score NUMERIC(8,2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  actor_role TEXT NOT NULL,
  action TEXT NOT NULL,
  target TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_applications_student ON applications(student_id);
CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(status);
CREATE INDEX IF NOT EXISTS idx_applications_class ON applications(class_name);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
