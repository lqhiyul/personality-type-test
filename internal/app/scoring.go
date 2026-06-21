package app

import "github.com/lqhiyul/personality-type-test/internal/scoring"

type TypeProfile = scoring.TypeProfile
type DimensionScore = scoring.DimensionScore
type Question = scoring.Question

var questions = scoring.Questions()

func normalizeAnswers(answers []string) ([]string, error) {
	return scoring.NormalizeAnswers(answers)
}

func normalizeSliderAnswers(answers []int) ([]int, error) {
	return scoring.NormalizeSliderAnswers(answers)
}

func computeProfile(answers []string) (TypeProfile, error) {
	return scoring.ComputeProfile(answers)
}

func computeWeightedProfile(answers []int) (TypeProfile, error) {
	return scoring.ComputeWeightedProfile(answers)
}
