package app

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
	defaultMessageLimit  = 100
	maxMessageLimit      = 200
	maxMessageBodyLength = 1000
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
	ID        int64
	SenderID  int64
	Body      string
	CreatedAt time.Time
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
	Body           string
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

func (s *UserStore) FindConversationBetween(ctx context.Context, userAID, userBID int64) (Conversation, error) {
	conversation, err := scanConversationBase(s.db.QueryRowContext(ctx, `
		SELECT c.id, c.created_at, c.updated_at
		FROM conversations c
		JOIN conversation_participants a ON a.conversation_id = c.id AND a.user_id = ?
		JOIN conversation_participants b ON b.conversation_id = c.id AND b.user_id = ?
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT 1
	`, userAID, userBID))
	if err != nil {
		return Conversation{}, err
	}
	if err := s.hydrateConversation(ctx, &conversation); err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func (s *UserStore) CreateConversation(ctx context.Context, userAID, userBID int64) (Conversation, error) {
	if userAID == userBID {
		return Conversation{}, ErrMessageSelf
	}
	blocked, err := s.IsBlockedBetween(ctx, userAID, userBID)
	if err != nil {
		return Conversation{}, err
	}
	if blocked {
		return Conversation{}, ErrBlockedInteraction
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
	conversationID, err := res.LastInsertId()
	if err != nil {
		return Conversation{}, fmt.Errorf("read created conversation id: %w", err)
	}
	for _, userID := range []int64{userAID, userBID} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO conversation_participants (conversation_id, user_id)
			VALUES (?, ?)
		`, conversationID, userID); err != nil {
			return Conversation{}, fmt.Errorf("create conversation participant: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Conversation{}, fmt.Errorf("commit create conversation: %w", err)
	}
	return s.GetConversationForUser(ctx, conversationID, userAID)
}

func (s *UserStore) GetOrCreateConversation(ctx context.Context, userAID, userBID int64) (Conversation, error) {
	if userAID == userBID {
		return Conversation{}, ErrMessageSelf
	}
	blocked, err := s.IsBlockedBetween(ctx, userAID, userBID)
	if err != nil {
		return Conversation{}, err
	}
	if blocked {
		return Conversation{}, ErrBlockedInteraction
	}
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
		SELECT c.id, c.created_at, c.updated_at
		FROM conversations c
		JOIN conversation_participants cp ON cp.conversation_id = c.id
		WHERE cp.user_id = ?
		ORDER BY c.updated_at DESC, c.id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	conversations := make([]Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversationBase(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("list conversations rows: %w", err)
	}
	rows.Close()

	for i := range conversations {
		if err := s.hydrateConversation(ctx, &conversations[i]); err != nil {
			return nil, err
		}
	}
	return conversations, nil
}

func (s *UserStore) GetConversationForUser(ctx context.Context, conversationID, userID int64) (Conversation, error) {
	conversation, err := scanConversationBase(s.db.QueryRowContext(ctx, `
		SELECT c.id, c.created_at, c.updated_at
		FROM conversations c
		JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = ?
		WHERE c.id = ?
		LIMIT 1
	`, userID, conversationID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Conversation{}, ErrMessageForbidden
		}
		return Conversation{}, err
	}
	if err := s.hydrateConversation(ctx, &conversation); err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func (s *UserStore) ListMessages(ctx context.Context, conversationID, userID int64, limit int) ([]Message, error) {
	if ok, err := s.isConversationParticipant(ctx, conversationID, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrMessageForbidden
	}

	limit = normalizedMessageLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at, deleted_at
		FROM messages
		WHERE conversation_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC, id ASC
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
	participantIDs, err := s.listConversationParticipantIDs(ctx, conversationID)
	if err != nil {
		return Message{}, err
	}
	isSenderParticipant := false
	for _, participantID := range participantIDs {
		if participantID == senderID {
			isSenderParticipant = true
			continue
		}
		blocked, err := s.IsBlockedBetween(ctx, senderID, participantID)
		if err != nil {
			return Message{}, err
		}
		if blocked {
			return Message{}, ErrBlockedInteraction
		}
	}
	if !isSenderParticipant {
		return Message{}, ErrMessageForbidden
	}

	now := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body, created_at)
		VALUES (?, ?, ?, ?)
	`, conversationID, senderID, normalizedBody, now)
	if err != nil {
		return Message{}, fmt.Errorf("create message: %w", err)
	}
	messageID, err := res.LastInsertId()
	if err != nil {
		return Message{}, fmt.Errorf("read created message id: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE conversations SET updated_at = ? WHERE id = ?
	`, now, conversationID); err != nil {
		return Message{}, fmt.Errorf("touch conversation: %w", err)
	}
	return s.GetMessageForUser(ctx, messageID, senderID)
}

func (s *UserStore) GetMessageForUser(ctx context.Context, messageID, userID int64) (Message, error) {
	message, err := scanMessage(s.db.QueryRowContext(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at, m.deleted_at
		FROM messages m
		JOIN conversation_participants cp ON cp.conversation_id = m.conversation_id AND cp.user_id = ?
		WHERE m.id = ?
		LIMIT 1
	`, userID, messageID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Message{}, ErrMessageForbidden
		}
		return Message{}, err
	}
	return message, nil
}

func (s *UserStore) DeleteMessage(ctx context.Context, messageID, requesterUserID int64) error {
	message, err := s.GetMessageForUser(ctx, messageID, requesterUserID)
	if err != nil {
		return err
	}
	if message.SenderID != requesterUserID {
		return ErrMessageForbidden
	}
	if message.DeletedAt != nil {
		return nil
	}
	deletedAt := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		UPDATE messages
		SET body = '', deleted_at = ?
		WHERE id = ? AND sender_id = ?
	`, deletedAt, messageID, requesterUserID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted message rows: %w", err)
	}
	if rows == 0 {
		return ErrMessageForbidden
	}
	return nil
}

func (s *UserStore) hydrateConversation(ctx context.Context, conversation *Conversation) error {
	participants, err := s.listConversationParticipants(ctx, conversation.ID)
	if err != nil {
		return err
	}
	conversation.Participants = participants
	lastMessage, err := s.lastConversationMessage(ctx, conversation.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			conversation.LastMessage = nil
			return nil
		}
		return err
	}
	conversation.LastMessage = &lastMessage
	return nil
}

func (s *UserStore) listConversationParticipants(ctx context.Context, conversationID int64) ([]MessageParticipant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, COALESCE(u.display_name, ''), COALESCE(u.avatar_key, '')
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = ?
		ORDER BY LOWER(u.username), u.id
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list conversation participants: %w", err)
	}
	defer rows.Close()

	participants := make([]MessageParticipant, 0)
	for rows.Next() {
		var participant MessageParticipant
		if err := rows.Scan(&participant.ID, &participant.Username, &participant.DisplayName, &participant.AvatarKey); err != nil {
			return nil, err
		}
		participant.DisplayName = strings.TrimSpace(participant.DisplayName)
		if participant.DisplayName == "" {
			participant.DisplayName = participant.Username
		}
		participant.AvatarKey = publicAvatarKey(participant.AvatarKey)
		participants = append(participants, participant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list conversation participants rows: %w", err)
	}
	return participants, nil
}

func (s *UserStore) listConversationParticipantIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id
		FROM conversation_participants
		WHERE conversation_id = ?
		ORDER BY user_id
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list conversation participant ids: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list conversation participant ids rows: %w", err)
	}
	if len(ids) == 0 {
		return nil, ErrMessageForbidden
	}
	return ids, nil
}

func (s *UserStore) isConversationParticipant(ctx context.Context, conversationID, userID int64) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM conversation_participants
			WHERE conversation_id = ? AND user_id = ?
		)
	`, conversationID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check conversation participant: %w", err)
	}
	return exists != 0, nil
}

func (s *UserStore) lastConversationMessage(ctx context.Context, conversationID int64) (ConversationMessagePreview, error) {
	var preview ConversationMessagePreview
	var createdAt string
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, sender_id, body, created_at
		FROM messages
		WHERE conversation_id = ? AND deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, conversationID).Scan(&preview.ID, &preview.SenderID, &preview.Body, &createdAt); err != nil {
		return ConversationMessagePreview{}, err
	}
	parsed, err := parseDBTime(createdAt)
	if err != nil {
		return ConversationMessagePreview{}, err
	}
	preview.CreatedAt = parsed
	return preview, nil
}

func scanConversationBase(row rowScanner) (Conversation, error) {
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
	var deletedAt sql.NullString
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&createdAt,
		&deletedAt,
	); err != nil {
		return Message{}, err
	}
	var err error
	message.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return Message{}, err
	}
	if deletedAt.Valid && strings.TrimSpace(deletedAt.String) != "" {
		parsed, err := parseDBTime(deletedAt.String)
		if err != nil {
			return Message{}, err
		}
		message.DeletedAt = &parsed
	}
	return message, nil
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
