package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultMessageLimit   = 100
	maxMessageLimit       = 200
	maxMessageBodyLength  = 1000
	directParticipantSize = 2
)

var (
	ErrMessageSelf         = errors.New("cannot message yourself")
	ErrMessageForbidden    = errors.New("message action is not allowed")
	ErrMessageBodyRequired = errors.New("message cannot be empty")
	ErrMessageBodyTooLong  = errors.New("message is too long")
	ErrMessageBodyInvalid  = errors.New("message contains invalid characters")
)

type MessageParticipant struct {
	ID          int64
	Username    string
	DisplayName string
	AvatarKey   string
}

type ConversationMessagePreview struct {
	ID             int64
	ConversationID int64
	Sender         MessageParticipant
	Body           string
	CreatedAt      time.Time
}

type Conversation struct {
	ID           int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Participants []MessageParticipant
	LastMessage  *ConversationMessagePreview
}

type Message struct {
	ID             int64
	ConversationID int64
	SenderID       int64
	Sender         MessageParticipant
	Body           string
	CreatedAt      time.Time
}

func (s *UserStore) FindConversationBetween(ctx context.Context, userAID, userBID int64) (Conversation, error) {
	if userAID == userBID {
		return Conversation{}, ErrMessageSelf
	}

	conversation, err := s.queryConversation(ctx, `
		c.id IN (
			SELECT cp.conversation_id
			FROM conversation_participants cp
			WHERE cp.user_id IN (?, ?)
			GROUP BY cp.conversation_id
			HAVING COUNT(DISTINCT cp.user_id) = 2
			   AND (
				   SELECT COUNT(*)
				   FROM conversation_participants all_cp
				   WHERE all_cp.conversation_id = cp.conversation_id
			   ) = ?
		)
		ORDER BY c.created_at ASC, c.id ASC
	`, userAID, userBID, directParticipantSize)
	if err != nil {
		return Conversation{}, err
	}
	return s.hydrateConversation(ctx, conversation)
}

func (s *UserStore) CreateConversation(ctx context.Context, userAID, userBID int64) (Conversation, error) {
	if userAID == userBID {
		return Conversation{}, ErrMessageSelf
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Conversation{}, fmt.Errorf("begin create conversation: %w", err)
	}
	defer tx.Rollback()

	now := formatDBTime(s.now())
	res, err := tx.ExecContext(ctx, `
		INSERT INTO conversations (created_at, updated_at)
		VALUES (?, ?)
	`, now, now)
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Conversation{}, fmt.Errorf("read created conversation id: %w", err)
	}

	for _, userID := range []int64{userAID, userBID} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO conversation_participants (conversation_id, user_id)
			VALUES (?, ?)
		`, id, userID); err != nil {
			return Conversation{}, fmt.Errorf("create conversation participant: %w", err)
		}
	}

	conversation, err := queryConversation(ctx, tx, "c.id = ?", id)
	if err != nil {
		return Conversation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Conversation{}, fmt.Errorf("commit create conversation: %w", err)
	}
	return s.hydrateConversation(ctx, conversation)
}

func (s *UserStore) GetOrCreateConversation(ctx context.Context, userAID, userBID int64) (Conversation, error) {
	conversation, err := s.FindConversationBetween(ctx, userAID, userBID)
	if err == nil {
		return conversation, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, err
	}
	return s.CreateConversation(ctx, userAID, userBID)
}

func (s *UserStore) ListConversationsForUser(ctx context.Context, userID int64) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.created_at,
			c.updated_at
		FROM conversations c
		JOIN conversation_participants cp ON cp.conversation_id = c.id
		WHERE cp.user_id = ?
		ORDER BY c.updated_at DESC, c.id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	bases := make([]Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversation(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		bases = append(bases, conversation)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("list conversations rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close conversations rows: %w", err)
	}

	conversations := make([]Conversation, 0, len(bases))
	for _, conversation := range bases {
		conversation, err = s.hydrateConversation(ctx, conversation)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, nil
}

func (s *UserStore) GetConversationForUser(ctx context.Context, conversationID, userID int64) (Conversation, error) {
	conversation, err := s.queryConversation(ctx, `
		c.id = ?
		AND EXISTS (
			SELECT 1
			FROM conversation_participants cp
			WHERE cp.conversation_id = c.id AND cp.user_id = ?
		)
	`, conversationID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Conversation{}, ErrMessageForbidden
		}
		return Conversation{}, err
	}
	return s.hydrateConversation(ctx, conversation)
}

func (s *UserStore) ListMessages(ctx context.Context, conversationID, userID int64, limit int) ([]Message, error) {
	if _, err := s.GetConversationForUser(ctx, conversationID, userID); err != nil {
		return nil, err
	}

	limit = normalizedMessageLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			m.id,
			m.conversation_id,
			m.sender_id,
			m.body,
			m.created_at,
			u.id,
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_key, '')
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.conversation_id = ? AND m.deleted_at IS NULL
		ORDER BY m.created_at ASC, m.id ASC
		LIMIT ?
	`, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list messages rows: %w", err)
	}
	return messages, nil
}

func (s *UserStore) CreateMessage(ctx context.Context, conversationID, senderID int64, body string) (Message, error) {
	normalizedBody, err := normalizeMessageBody(body)
	if err != nil {
		return Message{}, err
	}
	if _, err := s.GetConversationForUser(ctx, conversationID, senderID); err != nil {
		return Message{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, fmt.Errorf("begin create message: %w", err)
	}
	defer tx.Rollback()

	createdAt := formatDBTime(s.now())
	res, err := tx.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body, created_at)
		VALUES (?, ?, ?, ?)
	`, conversationID, senderID, normalizedBody, createdAt)
	if err != nil {
		return Message{}, fmt.Errorf("create message: %w", err)
	}
	messageID, err := res.LastInsertId()
	if err != nil {
		return Message{}, fmt.Errorf("read created message id: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE conversations
		SET updated_at = ?
		WHERE id = ?
	`, createdAt, conversationID); err != nil {
		return Message{}, fmt.Errorf("touch conversation: %w", err)
	}

	message, err := queryMessage(ctx, tx, "m.id = ? AND m.deleted_at IS NULL", messageID)
	if err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, fmt.Errorf("commit create message: %w", err)
	}
	return message, nil
}

func (s *UserStore) DeleteMessage(ctx context.Context, messageID, requesterUserID int64) error {
	message, err := s.queryMessage(ctx, "m.id = ? AND m.deleted_at IS NULL", messageID)
	if err != nil {
		return err
	}
	if _, err := s.GetConversationForUser(ctx, message.ConversationID, requesterUserID); err != nil {
		return err
	}
	if message.SenderID != requesterUserID {
		return ErrMessageForbidden
	}

	deletedAt := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		UPDATE messages
		SET body = '', deleted_at = ?
		WHERE id = ? AND sender_id = ? AND deleted_at IS NULL
	`, deletedAt, messageID, requesterUserID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted message rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *UserStore) queryConversation(ctx context.Context, where string, args ...any) (Conversation, error) {
	return queryConversation(ctx, s.db, where, args...)
}

func queryConversation(ctx context.Context, db queryer, where string, args ...any) (Conversation, error) {
	query := `
		SELECT
			c.id,
			c.created_at,
			c.updated_at
		FROM conversations c
		WHERE ` + where + `
		LIMIT 1
	`
	conversation, err := scanConversation(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return Conversation{}, fmt.Errorf("query conversation: %w", err)
	}
	return conversation, nil
}

func (s *UserStore) queryMessage(ctx context.Context, where string, args ...any) (Message, error) {
	return queryMessage(ctx, s.db, where, args...)
}

func queryMessage(ctx context.Context, db queryer, where string, args ...any) (Message, error) {
	query := `
		SELECT
			m.id,
			m.conversation_id,
			m.sender_id,
			m.body,
			m.created_at,
			u.id,
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_key, '')
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE ` + where + `
		LIMIT 1
	`
	message, err := scanMessage(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return Message{}, fmt.Errorf("query message: %w", err)
	}
	return message, nil
}

func (s *UserStore) hydrateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	participants, err := s.listConversationParticipants(ctx, conversation.ID)
	if err != nil {
		return Conversation{}, err
	}
	lastMessage, err := s.lastConversationMessage(ctx, conversation.ID)
	if err != nil {
		return Conversation{}, err
	}
	conversation.Participants = participants
	conversation.LastMessage = lastMessage
	return conversation, nil
}

func (s *UserStore) listConversationParticipants(ctx context.Context, conversationID int64) ([]MessageParticipant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			u.id,
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_key, '')
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = ?
		ORDER BY LOWER(u.username), u.id
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list conversation participants: %w", err)
	}
	defer rows.Close()

	participants := make([]MessageParticipant, 0, directParticipantSize)
	for rows.Next() {
		participant, err := scanMessageParticipant(rows)
		if err != nil {
			return nil, err
		}
		participants = append(participants, participant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list conversation participants rows: %w", err)
	}
	return participants, nil
}

func (s *UserStore) lastConversationMessage(ctx context.Context, conversationID int64) (*ConversationMessagePreview, error) {
	message, err := s.queryMessage(ctx, `
		m.conversation_id = ?
		AND m.deleted_at IS NULL
		ORDER BY m.created_at DESC, m.id DESC
	`, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ConversationMessagePreview{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Sender:         message.Sender,
		Body:           message.Body,
		CreatedAt:      message.CreatedAt,
	}, nil
}

func scanConversation(row rowScanner) (Conversation, error) {
	var conversation Conversation
	var createdAt string
	var updatedAt string
	if err := row.Scan(&conversation.ID, &createdAt, &updatedAt); err != nil {
		return Conversation{}, err
	}

	var err error
	conversation.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return Conversation{}, err
	}
	conversation.UpdatedAt, err = parseDBTime(updatedAt)
	if err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func scanMessage(row rowScanner) (Message, error) {
	var message Message
	var createdAt string
	var sender MessageParticipant
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&createdAt,
		&sender.ID,
		&sender.Username,
		&sender.DisplayName,
		&sender.AvatarKey,
	); err != nil {
		return Message{}, err
	}

	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return Message{}, err
	}
	message.CreatedAt = parsedCreatedAt
	message.Sender = normalizeMessageParticipant(sender)
	return message, nil
}

func scanMessageParticipant(row rowScanner) (MessageParticipant, error) {
	var participant MessageParticipant
	if err := row.Scan(
		&participant.ID,
		&participant.Username,
		&participant.DisplayName,
		&participant.AvatarKey,
	); err != nil {
		return MessageParticipant{}, err
	}
	return normalizeMessageParticipant(participant), nil
}

func normalizeMessageParticipant(participant MessageParticipant) MessageParticipant {
	participant.DisplayName = strings.TrimSpace(participant.DisplayName)
	if participant.DisplayName == "" {
		participant.DisplayName = participant.Username
	}
	participant.AvatarKey = publicAvatarKey(participant.AvatarKey)
	return participant
}

func normalizeMessageBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", ErrMessageBodyRequired
	}
	if utf8.RuneCountInString(trimmed) > maxMessageBodyLength {
		return "", ErrMessageBodyTooLong
	}
	if containsMessageDisallowedControl(trimmed) {
		return "", ErrMessageBodyInvalid
	}
	return trimmed, nil
}

func containsMessageDisallowedControl(value string) bool {
	for _, r := range value {
		if !unicode.IsControl(r) || r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		return true
	}
	return false
}

func normalizedMessageLimit(limit int) int {
	if limit <= 0 {
		return defaultMessageLimit
	}
	if limit > maxMessageLimit {
		return maxMessageLimit
	}
	return limit
}
