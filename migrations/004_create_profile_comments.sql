CREATE TABLE IF NOT EXISTS profile_comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile_user_id INTEGER NOT NULL,
  author_user_id INTEGER NOT NULL,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(profile_user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY(author_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_profile_comments_profile_user_id_created_at ON profile_comments(profile_user_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_profile_comments_author_user_id ON profile_comments(author_user_id);
