package scoring

import (
	"fmt"
	"strconv"
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
	Axis      string
	CodeLeft  string
	CodeRight string
	A         string
	B         string
}

var dimensions = []DimensionMeta{
	{Label: "Energy", LeftCode: "E", LeftLabel: "Extraversion", RightCode: "I", RightLabel: "Introversion"},
	{Label: "Perception", LeftCode: "S", LeftLabel: "Sensing", RightCode: "N", RightLabel: "Intuition"},
	{Label: "Decision", LeftCode: "T", LeftLabel: "Thinking", RightCode: "F", RightLabel: "Feeling"},
	{Label: "Rhythm", LeftCode: "J", LeftLabel: "Judging", RightCode: "P", RightLabel: "Perceiving"},
}

var questions = []Question{
	{Axis: "S/N", CodeLeft: "S", CodeRight: "N", A: "S", B: "N"},
	{Axis: "T/F", CodeLeft: "T", CodeRight: "F", A: "T", B: "F"},
	{Axis: "E/I", CodeLeft: "E", CodeRight: "I", A: "E", B: "I"},
	{Axis: "J/P", CodeLeft: "J", CodeRight: "P", A: "J", B: "P"},
	{Axis: "N/S", CodeLeft: "N", CodeRight: "S", A: "N", B: "S"},
	{Axis: "F/T", CodeLeft: "F", CodeRight: "T", A: "F", B: "T"},
	{Axis: "I/E", CodeLeft: "I", CodeRight: "E", A: "I", B: "E"},
	{Axis: "P/J", CodeLeft: "P", CodeRight: "J", A: "P", B: "J"},
	{Axis: "T/F", CodeLeft: "T", CodeRight: "F", A: "T", B: "F"},
	{Axis: "S/N", CodeLeft: "S", CodeRight: "N", A: "S", B: "N"},
	{Axis: "J/P", CodeLeft: "J", CodeRight: "P", A: "J", B: "P"},
	{Axis: "E/I", CodeLeft: "E", CodeRight: "I", A: "E", B: "I"},
	{Axis: "F/T", CodeLeft: "F", CodeRight: "T", A: "F", B: "T"},
	{Axis: "N/S", CodeLeft: "N", CodeRight: "S", A: "N", B: "S"},
	{Axis: "P/J", CodeLeft: "P", CodeRight: "J", A: "P", B: "J"},
	{Axis: "I/E", CodeLeft: "I", CodeRight: "E", A: "I", B: "E"},
	{Axis: "J/P", CodeLeft: "J", CodeRight: "P", A: "J", B: "P"},
	{Axis: "E/I", CodeLeft: "E", CodeRight: "I", A: "E", B: "I"},
	{Axis: "S/N", CodeLeft: "S", CodeRight: "N", A: "S", B: "N"},
	{Axis: "T/F", CodeLeft: "T", CodeRight: "F", A: "T", B: "F"},
	{Axis: "P/J", CodeLeft: "P", CodeRight: "J", A: "P", B: "J"},
	{Axis: "I/E", CodeLeft: "I", CodeRight: "E", A: "I", B: "E"},
	{Axis: "N/S", CodeLeft: "N", CodeRight: "S", A: "N", B: "S"},
	{Axis: "F/T", CodeLeft: "F", CodeRight: "T", A: "F", B: "T"},
	{Axis: "E/I", CodeLeft: "E", CodeRight: "I", A: "E", B: "I"},
	{Axis: "J/P", CodeLeft: "J", CodeRight: "P", A: "J", B: "P"},
	{Axis: "T/F", CodeLeft: "T", CodeRight: "F", A: "T", B: "F"},
	{Axis: "S/N", CodeLeft: "S", CodeRight: "N", A: "S", B: "N"},
	{Axis: "I/E", CodeLeft: "I", CodeRight: "E", A: "I", B: "E"},
	{Axis: "P/J", CodeLeft: "P", CodeRight: "J", A: "P", B: "J"},
	{Axis: "F/T", CodeLeft: "F", CodeRight: "T", A: "F", B: "T"},
	{Axis: "N/S", CodeLeft: "N", CodeRight: "S", A: "N", B: "S"},
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
		switch answer {
		case question.CodeLeft:
			normalized[i] = "0"
			continue
		case question.CodeRight:
			normalized[i] = "100"
			continue
		}

		value, err := strconv.Atoi(answer)
		if err != nil || value < 0 || value > 100 {
			return nil, fmt.Errorf("invalid answer for question %d", i+1)
		}
		normalized[i] = strconv.Itoa(value)
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

func NormalizeSliderAnswers(answers []int) ([]int, error) {
	if len(answers) != len(questions) {
		return nil, fmt.Errorf("expected %d answers", len(questions))
	}

	normalized := make([]int, len(answers))
	for i, answer := range answers {
		if answer < 0 || answer > 100 {
			return nil, fmt.Errorf("invalid answer for question %d", i+1)
		}
		normalized[i] = answer
	}
	return normalized, nil
}

func ComputeWeightedProfile(answers []int) (TypeProfile, error) {
	normalized, err := NormalizeSliderAnswers(answers)
	if err != nil {
		return TypeProfile{}, err
	}
	return BuildWeightedProfile(normalized), nil
}

func BuildProfile(normalized []string) TypeProfile {
	return buildProfile(len(normalized), func(i int, question Question) (int, bool) {
		return answerValue(normalized[i], question)
	})
}

func BuildWeightedProfile(normalized []int) TypeProfile {
	return buildProfile(len(normalized), func(i int, _ Question) (int, bool) {
		value := normalized[i]
		if value < 0 || value > 100 {
			return 0, false
		}
		return value, true
	})
}

func buildProfile(answerCount int, valueAt func(int, Question) (int, bool)) TypeProfile {
	score := map[string]int{"E": 0, "I": 0, "S": 0, "N": 0, "T": 0, "F": 0, "J": 0, "P": 0}
	limit := answerCount
	if limit > len(questions) {
		limit = len(questions)
	}
	for i := 0; i < limit; i++ {
		question := questions[i]
		value, ok := valueAt(i, question)
		if !ok {
			continue
		}
		score[question.CodeLeft] += 100 - value
		score[question.CodeRight] += value
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

func answerValue(answer string, question Question) (int, bool) {
	answer = strings.ToUpper(strings.TrimSpace(answer))
	switch answer {
	case question.CodeLeft:
		return 0, true
	case question.CodeRight:
		return 100, true
	}
	value, err := strconv.Atoi(answer)
	if err != nil || value < 0 || value > 100 {
		return 0, false
	}
	return value, true
}
