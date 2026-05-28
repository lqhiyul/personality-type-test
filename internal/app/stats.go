package app

import (
	"net/http"
	"sort"
	"time"

	"github.com/lqhiyul/personality-type-test/internal/scoring"
)

type typeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type statsResponse struct {
	Total                  int            `json:"total"`
	AverageDurationSeconds int            `json:"averageDurationSeconds"`
	ByType                 map[string]int `json:"byType"`
	TopTypes               []typeCount    `json:"topTypes"`
	AxisDistribution       map[string]int `json:"axisDistribution"`
	LatestResultAt         *time.Time     `json:"latestResultAt,omitempty"`
}

func (a *App) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin authentication required")
		return
	}

	results, err := a.store.All()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not read statistics")
		return
	}
	writeJSON(w, http.StatusOK, buildStats(results))
}

func buildStats(results []Result) statsResponse {
	byType := map[string]int{}
	axisDistribution := map[string]int{"E": 0, "I": 0, "S": 0, "N": 0, "T": 0, "F": 0, "J": 0, "P": 0}
	totalDuration := 0
	var latestResultAt *time.Time

	for _, result := range results {
		duration := result.Duration
		if duration < 0 {
			duration = 0
		}
		totalDuration += duration

		typeCode, ok := normalizedResultType(result.Type)
		if ok {
			byType[typeCode]++
			for _, code := range typeCode {
				axisDistribution[string(code)]++
			}
		}

		if !result.Created.IsZero() {
			created := result.Created.UTC()
			if latestResultAt == nil || created.After(*latestResultAt) {
				latestResultAt = &created
			}
		}
	}

	topTypes := make([]typeCount, 0, len(byType))
	for typeCode, count := range byType {
		topTypes = append(topTypes, typeCount{Type: typeCode, Count: count})
	}
	sort.Slice(topTypes, func(i, j int) bool {
		if topTypes[i].Count == topTypes[j].Count {
			return topTypes[i].Type < topTypes[j].Type
		}
		return topTypes[i].Count > topTypes[j].Count
	})

	averageDuration := 0
	if len(results) > 0 {
		averageDuration = totalDuration / len(results)
	}

	return statsResponse{
		Total:                  len(results),
		AverageDurationSeconds: averageDuration,
		ByType:                 byType,
		TopTypes:               topTypes,
		AxisDistribution:       axisDistribution,
		LatestResultAt:         latestResultAt,
	}
}

func normalizedResultType(value string) (string, bool) {
	return scoring.NormalizeType(value)
}

func sortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Created.After(results[j].Created)
	})
}
