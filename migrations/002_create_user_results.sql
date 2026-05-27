CREATE TABLE IF NOT EXISTS user_test_results (
  id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL,
  mbti_type TEXT NOT NULL,
  scores_json TEXT,
  answers_json TEXT,
  duration_seconds INTEGER,
  is_primary INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_test_results_user_id ON user_test_results(user_id);
CREATE INDEX IF NOT EXISTS idx_user_test_results_user_primary ON user_test_results(user_id, is_primary);
