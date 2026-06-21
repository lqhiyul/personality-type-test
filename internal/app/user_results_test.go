package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAnonymousSubmitDoesNotCreateUserTestResult(t *testing.T) {
	app := newTestApp(t)

	rec := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Anonymous",
		"answers":  answersForType("INTJ"),
		"duration": 45,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected anonymous submit 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		SavedToAccount bool `json:"savedToAccount"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	if response.SavedToAccount {
		t.Fatal("anonymous submit must not report account save")
	}

	results, err := app.store.All()
	if err != nil {
		t.Fatalf("store.All() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected anonymous JSON result to still be saved, got %d", len(results))
	}
	if countUserTestResults(t, app) != 0 {
		t.Fatal("anonymous submit must not create user_test_results rows")
	}
}

func TestLoggedInSubmitCreatesUserTestResult(t *testing.T) {
	app := newTestApp(t)
	user, cookie := registerUserAndCookie(t, app, "saved_user", "saved@example.com")

	rec := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Saved User",
		"answers":  answersForType("INFJ"),
		"duration": 72,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected logged-in submit 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Type           string                 `json:"type"`
		SavedToAccount bool                   `json:"savedToAccount"`
		SavedResult    userTestResultResponse `json:"savedResult"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode submit response: %v", err)
	}
	if response.Type != "INFJ" || !response.SavedToAccount {
		t.Fatalf("expected INFJ saved account response, got %+v", response)
	}
	if response.SavedResult.MBTIType != "INFJ" || response.SavedResult.DurationSeconds != 72 {
		t.Fatalf("unexpected saved result response: %+v", response.SavedResult)
	}
	if len(response.SavedResult.Scores) == 0 {
		t.Fatal("expected saved result response to include scores")
	}
	if len(response.SavedResult.Answers) != len(questions) {
		t.Fatalf("expected saved answers in response, got %d", len(response.SavedResult.Answers))
	}

	storedResults, err := app.userStore.ListUserTestResults(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListUserTestResults() error = %v", err)
	}
	if len(storedResults) != 1 {
		t.Fatalf("expected one saved user result, got %d", len(storedResults))
	}
	stored := storedResults[0]
	if stored.MBTIType != "INFJ" || stored.DurationSeconds != 72 {
		t.Fatalf("unexpected stored user result: %+v", stored)
	}
	if !strings.Contains(stored.ScoresJSON, `"winner":"I"`) {
		t.Fatalf("expected scores JSON to contain dimension winners, got %s", stored.ScoresJSON)
	}
	if !strings.Contains(stored.AnswersJSON, `"100"`) {
		t.Fatalf("expected answers JSON to contain normalized answers, got %s", stored.AnswersJSON)
	}

	jsonResults, err := app.store.All()
	if err != nil {
		t.Fatalf("store.All() error = %v", err)
	}
	if len(jsonResults) != 1 {
		t.Fatalf("expected existing JSON storage to still receive result, got %d", len(jsonResults))
	}
}

func TestMyResultsEndpointsRequireAuthAndScopeByUser(t *testing.T) {
	app := newTestApp(t)
	ctx := context.Background()
	firstUser, firstCookie := registerUserAndCookie(t, app, "first_user", "first@example.com")
	secondUser, secondCookie := registerUserAndCookie(t, app, "second_user", "second@example.com")

	firstResult := createUserResultForTest(t, app, firstUser.ID, "INTJ", 60)
	secondResult := createUserResultForTest(t, app, secondUser.ID, "ENFP", 90)

	loggedOut := performJSON(app, http.MethodGet, "/api/me/results", nil)
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out my results 401, got %d", loggedOut.Code)
	}

	listFirst := performJSON(app, http.MethodGet, "/api/me/results", nil, firstCookie)
	if listFirst.Code != http.StatusOK {
		t.Fatalf("expected first user results 200, got %d: %s", listFirst.Code, listFirst.Body.String())
	}
	var firstResponse struct {
		Results []userTestResultResponse `json:"results"`
	}
	if err := json.NewDecoder(listFirst.Body).Decode(&firstResponse); err != nil {
		t.Fatalf("decode first results: %v", err)
	}
	if len(firstResponse.Results) != 1 || firstResponse.Results[0].ID != firstResult.ID || firstResponse.Results[0].ID == secondResult.ID {
		t.Fatalf("expected only first user's result, got %+v", firstResponse.Results)
	}
	if len(firstResponse.Results[0].Answers) != len(questions) {
		t.Fatalf("expected private result answers for owner, got %d", len(firstResponse.Results[0].Answers))
	}

	listSecond := performJSON(app, http.MethodGet, "/api/me/results", nil, secondCookie)
	if listSecond.Code != http.StatusOK {
		t.Fatalf("expected second user results 200, got %d: %s", listSecond.Code, listSecond.Body.String())
	}
	var secondResponse struct {
		Results []userTestResultResponse `json:"results"`
	}
	if err := json.NewDecoder(listSecond.Body).Decode(&secondResponse); err != nil {
		t.Fatalf("decode second results: %v", err)
	}
	if len(secondResponse.Results) != 1 || secondResponse.Results[0].ID != secondResult.ID || secondResponse.Results[0].ID == firstResult.ID {
		t.Fatalf("expected only second user's result, got %+v", secondResponse.Results)
	}

	setOtherPrimary := performJSON(app, http.MethodPost, "/api/me/results/"+strconvID(secondResult.ID)+"/primary", nil, firstCookie)
	if setOtherPrimary.Code != http.StatusNotFound {
		t.Fatalf("expected setting another user's result primary to return 404, got %d", setOtherPrimary.Code)
	}

	deleteOther := performJSON(app, http.MethodDelete, "/api/me/results/"+strconvID(secondResult.ID), nil, firstCookie)
	if deleteOther.Code != http.StatusNotFound {
		t.Fatalf("expected deleting another user's result to return 404, got %d", deleteOther.Code)
	}

	remainingSecond, err := app.userStore.GetUserTestResult(ctx, secondUser.ID, secondResult.ID)
	if err != nil {
		t.Fatalf("expected second user's result to remain after forbidden delete: %v", err)
	}
	if remainingSecond.ID != secondResult.ID {
		t.Fatalf("unexpected second result after forbidden delete: %+v", remainingSecond)
	}
}

func TestMyResultsPrimaryAndDeleteOwnResult(t *testing.T) {
	app := newTestApp(t)
	user, cookie := registerUserAndCookie(t, app, "primary_user", "primary@example.com")

	first := createUserResultForTest(t, app, user.ID, "INTJ", 60)
	second := createUserResultForTest(t, app, user.ID, "ENFP", 90)

	setFirst := performJSON(app, http.MethodPost, "/api/me/results/"+strconvID(first.ID)+"/primary", nil, cookie)
	if setFirst.Code != http.StatusOK {
		t.Fatalf("expected set first primary 200, got %d: %s", setFirst.Code, setFirst.Body.String())
	}
	firstAfter, err := app.userStore.GetUserTestResult(context.Background(), user.ID, first.ID)
	if err != nil {
		t.Fatalf("read first after primary: %v", err)
	}
	secondAfter, err := app.userStore.GetUserTestResult(context.Background(), user.ID, second.ID)
	if err != nil {
		t.Fatalf("read second after primary: %v", err)
	}
	if !firstAfter.IsPrimary || secondAfter.IsPrimary {
		t.Fatalf("expected only first primary, first=%+v second=%+v", firstAfter, secondAfter)
	}

	setSecond := performJSON(app, http.MethodPost, "/api/me/results/"+strconvID(second.ID)+"/primary", nil, cookie)
	if setSecond.Code != http.StatusOK {
		t.Fatalf("expected set second primary 200, got %d: %s", setSecond.Code, setSecond.Body.String())
	}
	firstAfter, _ = app.userStore.GetUserTestResult(context.Background(), user.ID, first.ID)
	secondAfter, _ = app.userStore.GetUserTestResult(context.Background(), user.ID, second.ID)
	if firstAfter.IsPrimary || !secondAfter.IsPrimary {
		t.Fatalf("expected second primary to unset first, first=%+v second=%+v", firstAfter, secondAfter)
	}

	deleteFirst := performJSON(app, http.MethodDelete, "/api/me/results/"+strconvID(first.ID), nil, cookie)
	if deleteFirst.Code != http.StatusOK {
		t.Fatalf("expected delete first 200, got %d: %s", deleteFirst.Code, deleteFirst.Body.String())
	}
	if _, err := app.userStore.GetUserTestResult(context.Background(), user.ID, first.ID); err == nil {
		t.Fatal("expected deleted result to be gone")
	}

	list := performJSON(app, http.MethodGet, "/api/me/results", nil, cookie)
	if list.Code != http.StatusOK {
		t.Fatalf("expected list after delete 200, got %d: %s", list.Code, list.Body.String())
	}
	var response struct {
		Results []userTestResultResponse `json:"results"`
	}
	if err := json.NewDecoder(list.Body).Decode(&response); err != nil {
		t.Fatalf("decode list after delete: %v", err)
	}
	if len(response.Results) != 1 || response.Results[0].ID != second.ID || !response.Results[0].IsPrimary {
		t.Fatalf("expected only second primary result after delete, got %+v", response.Results)
	}
}

func registerUserAndCookie(t *testing.T, app *App, username, email string) (authUserResponse, *http.Cookie) {
	t.Helper()
	rec := registerAccountForTest(t, app, username, email, "StrongPassword123")
	var user authUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode registered user: %v", err)
	}
	cookie := findCookie(rec.Result().Cookies(), userSessionCookieName)
	if cookie == nil {
		t.Fatal("expected user session cookie")
	}
	return user, cookie
}

func createUserResultForTest(t *testing.T, app *App, userID int64, mbtiType string, duration int) UserTestResult {
	t.Helper()
	result, err := app.userStore.CreateUserTestResult(context.Background(), CreateUserTestResultParams{
		UserID:          userID,
		MBTIType:        mbtiType,
		ScoresJSON:      `[{"key":"EI","winner":"` + mbtiType[:1] + `"}]`,
		AnswersJSON:     marshalAnswersForTest(t, answersForType(mbtiType)),
		DurationSeconds: duration,
	})
	if err != nil {
		t.Fatalf("CreateUserTestResult() error = %v", err)
	}
	time.Sleep(time.Nanosecond)
	return result
}

func marshalAnswersForTest(t *testing.T, answers []string) string {
	t.Helper()
	data, err := json.Marshal(answers)
	if err != nil {
		t.Fatalf("marshal answers: %v", err)
	}
	return string(data)
}

func countUserTestResults(t *testing.T, app *App) int {
	t.Helper()
	var count int
	if err := app.userStore.db.QueryRow(`SELECT COUNT(*) FROM user_test_results`).Scan(&count); err != nil {
		t.Fatalf("count user test results: %v", err)
	}
	return count
}

func strconvID(id int64) string {
	return strconv.FormatInt(id, 10)
}
