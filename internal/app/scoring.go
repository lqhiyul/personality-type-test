package app

import "github.com/lqhiyul/personality-type-test/internal/scoring"

type TypeProfile = scoring.TypeProfile
type DimensionScore = scoring.DimensionScore
type Question = scoring.Question

var questions = scoring.Questions()

func normalizeSliderAnswers(answers []int) ([]int, error) {
	return scoring.NormalizeSliderAnswers(answers)
}

func computeProfile(answers []int) (TypeProfile, error) {
	return scoring.ComputeWeightedProfile(answers)
}

func buildProfile(normalized []int) TypeProfile {
	return scoring.BuildWeightedProfile(normalized)
}
