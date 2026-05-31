package scoring

import (
	"strings"
	"testing"
)

func TestComputeWeightedProfileAllLeft(t *testing.T) {
	profile, err := ComputeWeightedProfile(repeatAnswers(0))
	if err != nil {
		t.Fatalf("ComputeWeightedProfile() error = %v", err)
	}
	if profile.Type != "ESTJ" {
		t.Fatalf("expected ESTJ, got %q", profile.Type)
	}
	for _, dim := range profile.Dimensions {
		if dim.Winner != dim.LeftCode {
			t.Fatalf("expected left winner for %s, got %s", dim.Key, dim.Winner)
		}
		if dim.LeftPercent != 100 || dim.RightPercent != 0 || dim.Percent != 100 {
			t.Fatalf("expected 100/0 for %s, got %+v", dim.Key, dim)
		}
		if dim.BalanceLevel != "strong" {
			t.Fatalf("expected strong balance level for %s, got %q", dim.Key, dim.BalanceLevel)
		}
	}
}

func TestComputeWeightedProfileAllRight(t *testing.T) {
	profile, err := ComputeWeightedProfile(repeatAnswers(100))
	if err != nil {
		t.Fatalf("ComputeWeightedProfile() error = %v", err)
	}
	if profile.Type != "INFP" {
		t.Fatalf("expected INFP, got %q", profile.Type)
	}
	for _, dim := range profile.Dimensions {
		if dim.Winner != dim.RightCode {
			t.Fatalf("expected right winner for %s, got %s", dim.Key, dim.Winner)
		}
		if dim.LeftPercent != 0 || dim.RightPercent != 100 || dim.Percent != 100 {
			t.Fatalf("expected 0/100 for %s, got %+v", dim.Key, dim)
		}
	}
}

func TestComputeWeightedProfileNeutralTiesAreDeterministic(t *testing.T) {
	profile, err := ComputeWeightedProfile(repeatAnswers(50))
	if err != nil {
		t.Fatalf("ComputeWeightedProfile() error = %v", err)
	}
	if profile.Type != "ESTJ" {
		t.Fatalf("expected stable left-side tie type ESTJ, got %q", profile.Type)
	}
	for _, dim := range profile.Dimensions {
		if dim.LeftPercent != 50 || dim.RightPercent != 50 || dim.Margin != 0 {
			t.Fatalf("expected 50/50 tie for %s, got %+v", dim.Key, dim)
		}
		if dim.BalanceLevel != "balanced" {
			t.Fatalf("expected balanced level for %s, got %q", dim.Key, dim.BalanceLevel)
		}
	}
}

func TestComputeWeightedProfileMixedTotals(t *testing.T) {
	answers := make([]int, len(Questions()))
	pattern := []int{0, 25, 50, 75, 100, 0, 25}
	for i := range answers {
		answers[i] = pattern[i%len(pattern)]
	}
	profile, err := ComputeWeightedProfile(answers)
	if err != nil {
		t.Fatalf("ComputeWeightedProfile() error = %v", err)
	}

	want := map[string]struct {
		left  int
		right int
	}{
		"EI": {left: 425, right: 275},
		"SN": {left: 425, right: 275},
		"TF": {left: 425, right: 275},
		"JP": {left: 425, right: 275},
	}
	for _, dim := range profile.Dimensions {
		expected, ok := want[dim.Key]
		if !ok {
			t.Fatalf("unexpected dimension %s", dim.Key)
		}
		if dim.LeftScore != expected.left || dim.RightScore != expected.right {
			t.Fatalf("%s scores = %d/%d, want %d/%d", dim.Key, dim.LeftScore, dim.RightScore, expected.left, expected.right)
		}
	}
	if profile.Type != "ESTJ" {
		t.Fatalf("expected ESTJ, got %q", profile.Type)
	}
}

func TestNormalizeSliderAnswersRejectsBadLengthAndValues(t *testing.T) {
	if _, err := NormalizeSliderAnswers([]int{50}); err == nil {
		t.Fatal("expected short answer list to fail")
	}
	long := append(repeatAnswers(50), 50)
	if _, err := NormalizeSliderAnswers(long); err == nil {
		t.Fatal("expected long answer list to fail")
	}

	negative := repeatAnswers(50)
	negative[0] = -1
	if _, err := NormalizeSliderAnswers(negative); err == nil {
		t.Fatal("expected negative answer to fail")
	}

	tooLarge := repeatAnswers(50)
	tooLarge[0] = 101
	if _, err := NormalizeSliderAnswers(tooLarge); err == nil {
		t.Fatal("expected answer over 100 to fail")
	}
}

func TestBalanceLevels(t *testing.T) {
	cases := []struct {
		name   string
		values []int
		want   string
	}{
		{name: "balanced", values: []int{40, 50, 50, 50, 50, 50, 50}, want: "balanced"},
		{name: "slight", values: []int{25, 50, 50, 50, 50, 50, 50}, want: "slight"},
		{name: "moderate", values: []int{0, 0, 50, 50, 50, 50, 50}, want: "moderate"},
		{name: "strong", values: []int{0, 0, 0, 0, 50, 50, 50}, want: "strong"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answers := repeatAnswers(50)
			copy(answers[:7], tc.values)
			profile, err := ComputeWeightedProfile(answers)
			if err != nil {
				t.Fatalf("ComputeWeightedProfile() error = %v", err)
			}
			if got := profile.Dimensions[0].BalanceLevel; got != tc.want {
				t.Fatalf("BalanceLevel = %q, want %q; dimension=%+v", got, tc.want, profile.Dimensions[0])
			}
		})
	}
}

func TestNormalizeAnswersKeepsLegacyLetterValidation(t *testing.T) {
	if _, err := NormalizeAnswers([]string{"I"}); err == nil {
		t.Fatal("expected bad answer length to fail")
	}
	answers := legacyAnswersForType("INTJ")
	answers[0] = "X"
	if _, err := NormalizeAnswers(answers); err == nil {
		t.Fatal("expected invalid answer value to fail")
	}
}

func TestNormalizeType(t *testing.T) {
	if got, ok := NormalizeType(" infj "); !ok || got != "INFJ" {
		t.Fatalf("NormalizeType() = %q, %t; want INFJ, true", got, ok)
	}
	for _, value := range []string{"", "BAD", "IIII", "INFJA"} {
		if _, ok := NormalizeType(value); ok {
			t.Fatalf("expected %q to be invalid", value)
		}
	}
}

func repeatAnswers(value int) []int {
	answers := make([]int, len(Questions()))
	for i := range answers {
		answers[i] = value
	}
	return answers
}

func legacyAnswersForType(code string) []string {
	questions := Questions()
	answers := make([]string, len(questions))
	for i, question := range questions {
		if strings.Contains(code, question.A) {
			answers[i] = question.A
			continue
		}
		answers[i] = question.B
	}
	return answers
}
