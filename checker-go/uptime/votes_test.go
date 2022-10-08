package uptime

import (
    "testing"
)

func TestHasVoted(t *testing.T) {
	votes := Votes{}
	votes.Votes = append(votes.Votes, 10)
	
	if voted, _ := votes.HasVoted(9); voted {
		t.Fatalf(`Should not have voted for 9`)
	}

	if voted, _ := votes.HasVoted(10); !voted {
		t.Fatalf(`Should have voted for 10`)
	}
}
