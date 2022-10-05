package uptime

import (
    "testing"
)

func TestHasVoted(t *testing.T) {
	votes := Votes{}
	votes.Votes = append(votes.Votes, 10)
	
	if votes.HasVoted(9) {
		t.Fatalf(`Should not have voted for 9`)
	}

	if !votes.HasVoted(10) {
		t.Fatalf(`Should have voted for 10`)
	}
}
