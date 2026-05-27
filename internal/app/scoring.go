package app

import "github.com/lqhiyul/personality-type-test/internal/scoring"

type TypeProfile = scoring.TypeProfile
type DimensionScore = scoring.DimensionScore
type Question = scoring.Question

var questions = scoring.Questions()

func normalizeAnswers(answers []string) ([]string, error) {
	return scoring.NormalizeAnswers(answers)
}

func computeProfile(answers []string) (TypeProfile, error) {
	return scoring.ComputeProfile(answers)
}

func buildProfile(normalized []string) TypeProfile {
	return scoring.BuildProfile(normalized)
}
