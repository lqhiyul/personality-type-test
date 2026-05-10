package main

import "sort"

type friendCompatibilityResponse struct {
	Available    bool   `json:"available"`
	Reason       string `json:"reason,omitempty"`
	Friendship   int    `json:"friendship,omitempty"`
	Relationship int    `json:"relationship,omitempty"`
	Work         int    `json:"work,omitempty"`
}

func buildFriendCompatibility(currentPrimary, friendPrimary string) friendCompatibilityResponse {
	currentType, okA := normalizedResultType(currentPrimary)
	friendType, okB := normalizedResultType(friendPrimary)
	if currentType == "" || friendType == "" {
		return friendCompatibilityResponse{
			Available: false,
			Reason:    "Both users need a primary result",
		}
	}
	if !okA || !okB {
		return friendCompatibilityResponse{
			Available: false,
			Reason:    "Compatibility data unavailable",
		}
	}

	return friendCompatibilityResponse{
		Available:    true,
		Friendship:   compatibilityScore(currentType, friendType, "friendship"),
		Relationship: compatibilityScore(currentType, friendType, "relationship"),
		Work:         compatibilityScore(currentType, friendType, "work"),
	}
}

func compatibilityScore(typeA, typeB, context string) int {
	baseScores := map[string]int{"friendship": 60, "relationship": 58, "work": 62}
	score := baseScores[context]
	if score == 0 {
		score = 60
	}

	axisScores := map[string]struct {
		Same      map[string]int
		Different map[string]int
	}{
		"energy": {
			Same:      map[string]int{"friendship": 5, "relationship": 3, "work": 1},
			Different: map[string]int{"friendship": 1, "relationship": -1, "work": 2},
		},
		"information": {
			Same:      map[string]int{"friendship": 5, "relationship": 4, "work": 5},
			Different: map[string]int{"friendship": 2, "relationship": 1, "work": 2},
		},
		"decision": {
			Same:      map[string]int{"friendship": 3, "relationship": 2, "work": 3},
			Different: map[string]int{"friendship": 3, "relationship": 2, "work": 4},
		},
		"rhythm": {
			Same:      map[string]int{"friendship": 4, "relationship": 3, "work": 4},
			Different: map[string]int{"friendship": 2, "relationship": 0, "work": 1},
		},
	}

	axes := []struct {
		Key   string
		Index int
	}{
		{Key: "energy", Index: 0},
		{Key: "information", Index: 1},
		{Key: "decision", Index: 2},
		{Key: "rhythm", Index: 3},
	}

	for _, axis := range axes {
		scores := axisScores[axis.Key]
		if typeA[axis.Index] == typeB[axis.Index] {
			score += scores.Same[context]
		} else {
			score += scores.Different[context]
		}
	}

	if typeA == typeB {
		if context == "relationship" {
			score += 6
		} else {
			score += 7
		}
	}
	if context == "work" && typeA[2] != typeB[2] {
		score += 2
	}
	if context == "relationship" && typeA[0] != typeB[0] {
		score--
	}

	score += curatedCompatibilityBoost(typeA, typeB, context)
	return clampInt(score, 45, 96)
}

func curatedCompatibilityBoost(typeA, typeB, context string) int {
	boosts := map[string]map[string]int{
		"ENFJ|INTJ": {"friendship": 3, "relationship": 13, "work": -3},
		"INFJ|INTJ": {"friendship": 7, "relationship": 16, "work": 3},
	}
	return boosts[compatibilityPairKey(typeA, typeB)][context]
}

func compatibilityPairKey(typeA, typeB string) string {
	pair := []string{typeA, typeB}
	sort.Strings(pair)
	return pair[0] + "|" + pair[1]
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
