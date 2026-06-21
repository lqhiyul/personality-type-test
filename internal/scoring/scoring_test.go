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
		if dim.LeftScore+dim.RightScore != 800 {
			t.Fatalf("expected 800 weighted points for %s, got %d", dim.Key, dim.LeftScore+dim.RightScore)
		}
		if dim.Percent != 100 {
			t.Fatalf("expected full preference for %s, got %d%%", dim.Key, dim.Percent)
		}
	}
}

func TestQuestionBankV3Mapping(t *testing.T) {
	questions := Questions()
	if len(questions) != 32 {
		t.Fatalf("expected 32 questions, got %d", len(questions))
	}

	want := []string{
		"S/N", "T/F", "E/I", "J/P", "N/S", "F/T", "I/E", "P/J",
		"T/F", "S/N", "J/P", "E/I", "F/T", "N/S", "P/J", "I/E",
		"J/P", "E/I", "S/N", "T/F", "P/J", "I/E", "N/S", "F/T",
		"E/I", "J/P", "T/F", "S/N", "I/E", "P/J", "F/T", "N/S",
	}
	for i, question := range questions {
		if got := question.CodeLeft + "/" + question.CodeRight; got != want[i] || question.Axis != want[i] {
			t.Fatalf("question %d mapping = axis %q codes %q; want %q", i+1, question.Axis, got, want[i])
		}
	}
}

func TestWeightedSliderAnswersRespectMixedPolarity(t *testing.T) {
	answers := answersForType("ENFP")
	profile, err := ComputeProfile(answers)
	if err != nil {
		t.Fatalf("ComputeProfile() error = %v", err)
	}
	if profile.Type != "ENFP" {
		t.Fatalf("expected ENFP, got %q", profile.Type)
	}

	balanced := make([]string, len(Questions()))
	for i := range balanced {
		balanced[i] = "50"
	}
	profile, err = ComputeProfile(balanced)
	if err != nil {
		t.Fatalf("ComputeProfile() balanced error = %v", err)
	}
	if profile.Type != "ESTJ" {
		t.Fatalf("expected tie-breaking to keep left dimension letters ESTJ, got %q", profile.Type)
	}
	for _, dim := range profile.Dimensions {
		if dim.LeftScore != 400 || dim.RightScore != 400 || dim.Percent != 50 {
			t.Fatalf("expected balanced 400/400 for %s, got %d/%d at %d%%", dim.Key, dim.LeftScore, dim.RightScore, dim.Percent)
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
	answers = answersForType("INTJ")
	answers[0] = "101"
	if _, err := NormalizeAnswers(answers); err == nil {
		t.Fatal("expected out-of-range slider value to fail")
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
