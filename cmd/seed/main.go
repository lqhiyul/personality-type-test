package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lqhiyul/personality-type-test/internal/app"
	"github.com/lqhiyul/personality-type-test/internal/config"
	storagesqlite "github.com/lqhiyul/personality-type-test/internal/storage/sqlite"
)

func main() {
	cfg := config.FromEnv()
	if cfg.Production {
		log.Fatal("refusing to seed data in production mode")
	}

	ctx := context.Background()
	db, err := storagesqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	passwordHash, err := app.HashPassword("DemoPassword123")
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	users := []struct {
		username string
		email    string
		display  string
		mbti     string
	}{
		{username: "demo-alice", email: "demo-alice@example.com", display: "Demo Alice", mbti: "INFJ"},
		{username: "demo-bob", email: "demo-bob@example.com", display: "Demo Bob", mbti: "ENTP"},
	}

	for _, user := range users {
		if _, err := db.ExecContext(ctx, `
			INSERT OR IGNORE INTO users (username, email, password_hash, display_name, bio, avatar_key, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, user.username, user.email, passwordHash, user.display, "Seeded demo account for local development.", "gradient-blue", now, now); err != nil {
			log.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO user_test_results (user_id, mbti_type, scores_json, answers_json, duration_seconds, is_primary, created_at)
			SELECT id, ?, '{}', '[]', 180, 1, ?
			FROM users
			WHERE username = ?
			  AND NOT EXISTS (
				SELECT 1 FROM user_test_results
				WHERE user_id = users.id AND mbti_type = ?
			  )
		`, user.mbti, now, user.username, user.mbti); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("seeded demo users: demo-alice@example.com and demo-bob@example.com (password: DemoPassword123)")
}
