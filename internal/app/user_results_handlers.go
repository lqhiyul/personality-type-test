package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type userTestResultResponse struct {
	ID              int64           `json:"id"`
	MBTIType        string          `json:"mbtiType"`
	Scores          json.RawMessage `json:"scores,omitempty"`
	Answers         []string        `json:"answers,omitempty"`
	DurationSeconds int             `json:"durationSeconds"`
	IsPrimary       bool            `json:"isPrimary"`
	CreatedAt       string          `json:"createdAt"`
}

func (a *App) handleMyResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	userID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	results, err := a.userStore.ListUserTestResults(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load saved results")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": newUserTestResultResponses(results)})
}

func (a *App) handleMyResultByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	resultID, action, ok := parseMyResultPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "result not found")
		return
	}

	switch {
	case r.Method == http.MethodPost && action == "primary":
		result, err := a.userStore.SetPrimaryUserTestResult(r.Context(), userID, resultID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, http.StatusNotFound, "result not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "could not set primary result")
			return
		}
		writeJSON(w, http.StatusOK, newUserTestResultResponse(result))
	case r.Method == http.MethodDelete && action == "":
		if err := a.userStore.DeleteUserTestResult(r.Context(), userID, resultID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, http.StatusNotFound, "result not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "could not delete result")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w, http.MethodPost, http.MethodDelete)
	}
}

func (a *App) saveLoggedInUserResult(r *http.Request, profile TypeProfile, answers []string, duration int) (*userTestResultResponse, error) {
	userID, ok := a.currentUserID(r)
	if !ok {
		return nil, nil
	}

	scoresJSON, err := marshalJSONString(profile.Dimensions)
	if err != nil {
		return nil, err
	}
	answersJSON, err := marshalJSONString(answers)
	if err != nil {
		return nil, err
	}

	result, err := a.userStore.CreateUserTestResult(r.Context(), CreateUserTestResultParams{
		UserID:          userID,
		MBTIType:        profile.Type,
		ScoresJSON:      scoresJSON,
		AnswersJSON:     answersJSON,
		DurationSeconds: duration,
	})
	if err != nil {
		return nil, err
	}
	response := newUserTestResultResponse(result)
	return &response, nil
}

func (a *App) requireCurrentUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := a.currentUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return 0, false
	}
	return userID, true
}

func parseMyResultPath(requestPath string) (int64, string, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/me/results/")
	if suffix == requestPath {
		return 0, "", false
	}
	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	if len(parts) == 1 {
		return id, "", true
	}
	if len(parts) == 2 && parts[1] == "primary" {
		return id, "primary", true
	}
	return 0, "", false
}

func newUserTestResultResponses(results []UserTestResult) []userTestResultResponse {
	out := make([]userTestResultResponse, 0, len(results))
	for _, result := range results {
		out = append(out, newUserTestResultResponse(result))
	}
	return out
}

func newUserTestResultResponse(result UserTestResult) userTestResultResponse {
	return userTestResultResponse{
		ID:              result.ID,
		MBTIType:        result.MBTIType,
		Scores:          rawJSONOrNil(result.ScoresJSON),
		Answers:         answerStringsOrNil(result.AnswersJSON),
		DurationSeconds: result.DurationSeconds,
		IsPrimary:       result.IsPrimary,
		CreatedAt:       result.CreatedAt.Format(time.RFC3339Nano),
	}
}

func marshalJSONString(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func rawJSONOrNil(value string) json.RawMessage {
	value = strings.TrimSpace(value)
	if value == "" || !json.Valid([]byte(value)) {
		return nil
	}
	return json.RawMessage(value)
}

func answerStringsOrNil(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	var answers []string
	if err := json.Unmarshal([]byte(value), &answers); err == nil {
		return answers
	}

	var numericAnswers []int
	if err := json.Unmarshal([]byte(value), &numericAnswers); err != nil {
		return nil
	}
	answers = make([]string, len(numericAnswers))
	for i, answer := range numericAnswers {
		answers[i] = strconv.Itoa(answer)
	}
	return answers
}
