package scoring

import (
	"strings"
	"testing"
)

func TestComputeProfileBreakdown(t *testing.T) {
	profile, err := ComputeProfile(answersForType("INTJ"))
	if err != nil {
		t.Fatalf("ComputeProfile() error = %v", err)
	}
	if profile.Type != "INTJ" {
		t.Fatalf("expected INTJ, got %q", profile.Type)
	}
	if len(profile.Dimensions) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(profile.Dimensions))
	}
	for _, dim := range profile.Dimensions {
		if dim.Percent != 100 {
			t.Fatalf("expected full preference for %s, got %d%%", dim.Key, dim.Percent)
		}
	}
}

func TestNormalizeAnswersRejectsBadLengthAndValues(t *testing.T) {
	if _, err := NormalizeAnswers([]string{"I"}); err == nil {
		t.Fatal("expected bad answer length to fail")
	}
	answers := answersForType("INTJ")
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

func answersForType(code string) []string {
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
