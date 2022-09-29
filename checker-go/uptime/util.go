package uptime

import (
	"bytes"
	"strconv"
	"encoding/json"
)

func EncodeJson(payload interface{}) ([]byte, error) {
	return encodeJson(payload)
}

func encodeJson(payload interface{}) ([]byte, error) {
	reqBodyBytes := new(bytes.Buffer)
	err := json.NewEncoder(reqBodyBytes).Encode(payload)
	return reqBodyBytes.Bytes(), err
}

func allUp(infos *[]UpInfo) bool {
	for _, info := range *infos {
		if !info.isOnline {
			return false
		}
	}
	return true
}

func keysOfMap(target *map[PeerID]NodeInfo) []PeerID {
	keys := make([]PeerID, len(*target))

	i := 0
	for k := range *target {
		keys[i] = k
		i++
	}
	return keys
}

func parseActorIDFromString(s string) (ActorID, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return ActorID(0), err
	}
	return ActorID(v), nil
}
