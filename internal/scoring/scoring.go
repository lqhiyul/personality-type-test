package scoring

import (
	"fmt"
	"strings"
)

type TypeProfile struct {
	Type       string           `json:"type"`
	Code       string           `json:"code,omitempty"`
	Dimensions []DimensionScore `json:"dimensions"`
}

type DimensionScore struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	LeftCode     string `json:"leftCode"`
	LeftLabel    string `json:"leftLabel"`
	LeftScore    int    `json:"leftScore"`
	LeftPercent  int    `json:"leftPercent"`
	RightCode    string `json:"rightCode"`
	RightLabel   string `json:"rightLabel"`
	RightScore   int    `json:"rightScore"`
	RightPercent int    `json:"rightPercent"`
	Winner       string `json:"winner"`
	Percent      int    `json:"percent"`
	Margin       int    `json:"margin"`
	BalanceLevel string `json:"balanceLevel"`
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

func NormalizeSliderAnswers(values []int) ([]int, error) {
	if len(values) != len(questions) {
		return nil, fmt.Errorf("expected %d answers", len(questions))
	}

	normalized := make([]int, len(values))
	for i, value := range values {
		if value < 0 || value > 100 {
			return nil, fmt.Errorf("answer %d must be between 0 and 100", i+1)
		}
		normalized[i] = value
	}
	return normalized, nil
}

func ComputeWeightedType(values []int) (string, error) {
	profile, err := ComputeWeightedProfile(values)
	if err != nil {
		return "", err
	}
	return profile.Type, nil
}

func ComputeWeightedProfile(values []int) (TypeProfile, error) {
	normalized, err := NormalizeSliderAnswers(values)
	if err != nil {
		return TypeProfile{}, err
	}
	return BuildWeightedProfile(normalized), nil
}

func BuildWeightedProfile(normalized []int) TypeProfile {
	score := map[string]int{"E": 0, "I": 0, "S": 0, "N": 0, "T": 0, "F": 0, "J": 0, "P": 0}
	for i, value := range normalized {
		if i >= len(questions) {
			break
		}
		question := questions[i]
		score[question.A] += 100 - value
		score[question.B] += value
	}

	profile := TypeProfile{Dimensions: make([]DimensionScore, 0, len(dimensions))}
	for _, dim := range dimensions {
		leftScore := score[dim.LeftCode]
		rightScore := score[dim.RightCode]
		total := leftScore + rightScore
		leftPercent := percent(leftScore, total)
		rightPercent := percent(rightScore, total)
		winner := pick(leftScore, rightScore, dim.LeftCode, dim.RightCode)
		winnerPercent := leftPercent
		if winner == dim.RightCode {
			winnerPercent = rightPercent
		}
		margin := absInt(leftPercent - rightPercent)

		profile.Type += winner
		profile.Dimensions = append(profile.Dimensions, DimensionScore{
			Key:          dim.LeftCode + dim.RightCode,
			Label:        dim.Label,
			LeftCode:     dim.LeftCode,
			LeftLabel:    dim.LeftLabel,
			LeftScore:    leftScore,
			LeftPercent:  leftPercent,
			RightCode:    dim.RightCode,
			RightLabel:   dim.RightLabel,
			RightScore:   rightScore,
			RightPercent: rightPercent,
			Winner:       winner,
			Percent:      winnerPercent,
			Margin:       margin,
			BalanceLevel: balanceLevel(margin),
		})
	}
	profile.Code = profile.Type
	return profile
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
		winnerPercent := 0
		if total > 0 {
			winnerPercent = (winnerScore*100 + total/2) / total
		}
		leftPercent := percent(leftScore, total)
		rightPercent := percent(rightScore, total)
		margin := absInt(leftPercent - rightPercent)

		profile.Type += winner
		profile.Dimensions = append(profile.Dimensions, DimensionScore{
			Key:          dim.LeftCode + dim.RightCode,
			Label:        dim.Label,
			LeftCode:     dim.LeftCode,
			LeftLabel:    dim.LeftLabel,
			LeftScore:    leftScore,
			LeftPercent:  leftPercent,
			RightCode:    dim.RightCode,
			RightLabel:   dim.RightLabel,
			RightScore:   rightScore,
			RightPercent: rightPercent,
			Winner:       winner,
			Percent:      winnerPercent,
			Margin:       margin,
			BalanceLevel: balanceLevel(margin),
		})
	}
	profile.Code = profile.Type
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

func percent(score, total int) int {
	if total <= 0 {
		return 0
	}
	return (score*100 + total/2) / total
}

func balanceLevel(marginPercent int) string {
	switch {
	case marginPercent <= 5:
		return "balanced"
	case marginPercent <= 15:
		return "slight"
	case marginPercent <= 30:
		return "moderate"
	default:
		return "strong"
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
