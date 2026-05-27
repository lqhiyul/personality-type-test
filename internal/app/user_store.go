package app

import (
	"database/sql"
	"time"
)

const (
	profileVisibilityPublic  = "public"
	profileVisibilityPrivate = "private"
)

type User struct {
	ID                 int64
	Username           string
	Email              string
	DisplayName        string
	Bio                string
	AvatarKey          string
	ProfileVisibility  string
	ShowPrimaryResult  bool
	ShowCompletedCount bool
	ShowFriends        bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type userAuthRecord struct {
	User
	PasswordHash string
}

type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	Bio          string
	AvatarKey    string
}

type UpdateUserProfileParams struct {
	DisplayName        string
	Bio                string
	AvatarKey          string
	ProfileVisibility  string
	ShowPrimaryResult  *bool
	ShowCompletedCount *bool
	ShowFriends        *bool
}

type UserTestResult struct {
	ID              int64
	UserID          int64
	MBTIType        string
	ScoresJSON      string
	AnswersJSON     string
	DurationSeconds int
	IsPrimary       bool
	CreatedAt       time.Time
}

type CreateUserTestResultParams struct {
	UserID          int64
	MBTIType        string
	ScoresJSON      string
	AnswersJSON     string
	DurationSeconds int
	IsPrimary       bool
}

type UserStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{
		db:  db,
		now: func() time.Time { return time.Now().UTC() },
	}
}
