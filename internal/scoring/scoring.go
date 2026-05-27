package scoring

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

type DimensionMeta struct {
	Label      string
	LeftCode   string
	LeftLabel  string
	RightCode  string
	RightLabel string
}

type Question struct {
	A string
	B string
}

var dimensions = []DimensionMeta{
	{Label: "Energy", LeftCode: "E", LeftLabel: "Extraversion", RightCode: "I", RightLabel: "Introversion"},
	{Label: "Perception", LeftCode: "S", LeftLabel: "Sensing", RightCode: "N", RightLabel: "Intuition"},
	{Label: "Decision", LeftCode: "T", LeftLabel: "Thinking", RightCode: "F", RightLabel: "Feeling"},
	{Label: "Rhythm", LeftCode: "J", LeftLabel: "Judging", RightCode: "P", RightLabel: "Perceiving"},
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

func Dimensions() []DimensionMeta {
	out := make([]DimensionMeta, len(dimensions))
	copy(out, dimensions)
	return out
}

func Questions() []Question {
	out := make([]Question, len(questions))
	copy(out, questions)
	return out
}

func NormalizeAnswers(answers []string) ([]string, error) {
	if len(answers) != len(questions) {
		return nil, fmt.Errorf("expected %d answers", len(questions))
	}

	normalized := make([]string, len(answers))
	for i, answer := range answers {
		answer = strings.ToUpper(strings.TrimSpace(answer))
		question := questions[i]
		if answer != question.A && answer != question.B {
			return nil, fmt.Errorf("invalid answer for question %d", i+1)
		}
		normalized[i] = answer
	}
	return normalized, nil
}

func ComputeType(answers []string) (string, error) {
	profile, err := ComputeProfile(answers)
	if err != nil {
		return "", err
	}
	return profile.Type, nil
}

func ComputeProfile(answers []string) (TypeProfile, error) {
	normalized, err := NormalizeAnswers(answers)
	if err != nil {
		return TypeProfile{}, err
	}
	return BuildProfile(normalized), nil
}

func BuildProfile(normalized []string) TypeProfile {
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

func NormalizeType(value string) (string, bool) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) != len(dimensions) {
		return "", false
	}
	for i, dim := range dimensions {
		letter := code[i : i+1]
		if letter != dim.LeftCode && letter != dim.RightCode {
			return "", false
		}
	}
	return code, true
}

func pick(a, b int, left, right string) string {
	if a >= b {
		return left
	}
	return right
}
