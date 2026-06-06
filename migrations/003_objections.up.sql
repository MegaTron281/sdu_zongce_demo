CREATE TABLE IF NOT EXISTS objections (
  id TEXT PRIMARY KEY,
  application_id TEXT NOT NULL,
  student TEXT NOT NULL,
  student_id TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_objections_application ON objections(application_id);
