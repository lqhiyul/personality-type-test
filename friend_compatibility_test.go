package main

import "testing"

func TestCuratedFriendCompatibilityScores(t *testing.T) {
	tests := []struct {
		name         string
		typeA        string
		typeB        string
		friendship   int
		relationship int
		work         int
	}{
		{name: "INTJ and ENFJ balanced", typeA: "INTJ", typeB: "ENFJ", friendship: 76, relationship: 78, work: 76},
		{name: "INFJ and INTJ slightly higher", typeA: "INFJ", typeB: "INTJ", friendship: 84, relationship: 86, work: 81},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, pair := range [][2]string{{tt.typeA, tt.typeB}, {tt.typeB, tt.typeA}} {
				got := buildFriendCompatibility(pair[0], pair[1])
				if !got.Available {
					t.Fatalf("expected compatibility for %s + %s to be available: %+v", pair[0], pair[1], got)
				}
				if got.Friendship != tt.friendship || got.Relationship != tt.relationship || got.Work != tt.work {
					t.Fatalf("compatibility for %s + %s = friendship %d, relationship %d, work %d; want %d, %d, %d",
						pair[0], pair[1], got.Friendship, got.Relationship, got.Work, tt.friendship, tt.relationship, tt.work)
				}
			}
		})
	}
}
