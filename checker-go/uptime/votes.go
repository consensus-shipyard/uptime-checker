package uptime

func (v *Votes) HasVoted(voter ActorID) (bool, error) {
	for _, item := range(v.Votes) {
		if item == voter {
			return true, nil
		}
	}
	return false, nil
}