package main

import (
	"fmt"
	"strings"
)

type TypeProfile struct {
	Type       string           `json:"type"`
	Dimensions []DimensionScore `json:"dimensions"`
}

type DimensionScore struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	LeftCode   string `json:"leftCode"`
	LeftLabel  string `json:"leftLabel"`
	LeftScore  int    `json:"leftScore"`
	RightCode  string `json:"rightCode"`
	RightLabel string `json:"rightLabel"`
	RightScore int    `json:"rightScore"`
	Winner     string `json:"winner"`
	Percent    int    `json:"percent"`
}

type dimensionMeta struct {
	Label      string
	LeftCode   string
	LeftLabel  string
	RightCode  string
	RightLabel string
}

var dimensions = []dimensionMeta{
	{Label: "Енергія", LeftCode: "E", LeftLabel: "Екстраверсія", RightCode: "I", RightLabel: "Інтроверсія"},
	{Label: "Сприйняття", LeftCode: "S", LeftLabel: "Сенсорика", RightCode: "N", RightLabel: "Інтуїція"},
	{Label: "Рішення", LeftCode: "T", LeftLabel: "Логіка", RightCode: "F", RightLabel: "Етика"},
	{Label: "Ритм", LeftCode: "J", LeftLabel: "Структура", RightCode: "P", RightLabel: "Гнучкість"},
}

type Question struct {
	A string
	B string
}

var questions = []Question{
	{A: "E", B: "I"},
	{A: "E", B: "I"},
	{A: "E", B: "I"},
	{A: "E", B: "I"},
	{A: "E", B: "I"},
	{A: "E", B: "I"},
	{A: "E", B: "I"},
	{A: "S", B: "N"},
	{A: "S", B: "N"},
	{A: "S", B: "N"},
	{A: "S", B: "N"},
	{A: "S", B: "N"},
	{A: "S", B: "N"},
	{A: "S", B: "N"},
	{A: "T", B: "F"},
	{A: "T", B: "F"},
	{A: "T", B: "F"},
	{A: "T", B: "F"},
	{A: "T", B: "F"},
	{A: "T", B: "F"},
	{A: "T", B: "F"},
	{A: "J", B: "P"},
	{A: "J", B: "P"},
	{A: "J", B: "P"},
	{A: "J", B: "P"},
	{A: "J", B: "P"},
	{A: "J", B: "P"},
	{A: "J", B: "P"},
}

func normalizeAnswers(answers []string) ([]string, error) {
	if len(answers) != len(questions) {
		return nil, fmt.Errorf("очікується %d відповідей", len(questions))
	}

	normalized := make([]string, len(answers))
	for i, answer := range answers {
		answer = strings.ToUpper(strings.TrimSpace(answer))
		question := questions[i]
		if answer != question.A && answer != question.B {
			return nil, fmt.Errorf("недопустима відповідь у питанні %d", i+1)
		}
		normalized[i] = answer
	}
	return normalized, nil
}

func computeType(answers []string) (string, error) {
	profile, err := computeProfile(answers)
	if err != nil {
		return "", err
	}
	return profile.Type, nil
}

func computeProfile(answers []string) (TypeProfile, error) {
	normalized, err := normalizeAnswers(answers)
	if err != nil {
		return TypeProfile{}, err
	}
	return buildProfile(normalized), nil
}

func buildProfile(normalized []string) TypeProfile {
	score := map[string]int{"E": 0, "I": 0, "S": 0, "N": 0, "T": 0, "F": 0, "J": 0, "P": 0}
	for _, answer := range normalized {
		score[answer]++
	}

	profile := TypeProfile{Dimensions: make([]DimensionScore, 0, len(dimensions))}
	for _, dim := range dimensions {
		leftScore := score[dim.LeftCode]
		rightScore := score[dim.RightCode]
		winner := pick(leftScore, rightScore, dim.LeftCode, dim.RightCode)
		winnerScore := leftScore
		if winner == dim.RightCode {
			winnerScore = rightScore
		}

		total := leftScore + rightScore
		percent := 0
		if total > 0 {
			percent = (winnerScore*100 + total/2) / total
		}

		profile.Type += winner
		profile.Dimensions = append(profile.Dimensions, DimensionScore{
			Key:        dim.LeftCode + dim.RightCode,
			Label:      dim.Label,
			LeftCode:   dim.LeftCode,
			LeftLabel:  dim.LeftLabel,
			LeftScore:  leftScore,
			RightCode:  dim.RightCode,
			RightLabel: dim.RightLabel,
			RightScore: rightScore,
			Winner:     winner,
			Percent:    percent,
		})
	}
	return profile
}

func pick(a, b int, left, right string) string {
	if a >= b {
		return left
	}
	return right
}
