package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type submitRequest struct {
	Name     string        `json:"name"`
	Answers  []answerInput `json:"answers"`
	Duration int           `json:"duration"`
}

type answerInput string

func (a *answerInput) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*a = answerInput(text)
		return nil
	}
	var value int
	if err := json.Unmarshal(data, &value); err == nil {
		*a = answerInput(strconv.Itoa(value))
		return nil
	}
	return errors.New("answer must be a string or integer")
}

func (a *App) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req submitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	name := strings.TrimSpace(req.Name)
	if !validName(name) {
		writeJSONError(w, http.StatusBadRequest, "name must be 1 to 64 characters")
		return
	}

	normalizedAnswers, err := normalizeAnswers(answerInputsToStrings(req.Answers))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	profile := buildProfile(normalizedAnswers)

	result := Result{
		ID:       newID(),
		Name:     name,
		Type:     profile.Type,
		Answers:  strings.Join(normalizedAnswers, ","),
		Duration: clampDuration(req.Duration),
		Created:  time.Now().UTC(),
	}
	if err := a.store.Add(result); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save result")
		return
	}

	response := map[string]any{
		"type":    profile.Type,
		"profile": profile,
		"result":  result,
	}
	if savedResult, err := a.saveLoggedInUserResult(r, profile, normalizedAnswers, result.Duration); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save result to account")
		return
	} else if savedResult != nil {
		response["savedToAccount"] = true
		response["savedResult"] = savedResult
	} else {
		response["savedToAccount"] = false
	}

	writeJSON(w, http.StatusOK, response)
}

func answerInputsToStrings(inputs []answerInput) []string {
	answers := make([]string, len(inputs))
	for i, answer := range inputs {
		answers[i] = string(answer)
	}
	return answers
}

func validName(name string) bool {
	count := utf8.RuneCountInString(name)
	if count < 1 || count > 64 {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func clampDuration(seconds int) int {
	if seconds < 0 {
		return 0
	}
	if seconds > 24*60*60 {
		return 24 * 60 * 60
	}
	return seconds
}
