ALTER TABLE applications DROP COLUMN IF EXISTS claimed_score;
CREATE TABLE IF NOT EXISTS rule_versions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  active BOOLEAN NOT NULL DEFAULT false,
  published_at TEXT NOT NULL DEFAULT '',
  matrix JSONB NOT NULL DEFAULT '{}',
  module_max JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
