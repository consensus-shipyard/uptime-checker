package uptime

import (
    "testing"
)

func prepareUpInfos(latency uint64) *[]UpInfo {
	upInfos := make([]UpInfo, 1)
	upInfos = append(upInfos, UpInfo { isOnline: true, latency: latency, checkedTime: uint64(1000)})
	return &upInfos
}

func prepareMultiAddrs(addr string) *[]MultiAddr {
	addresses := make([]MultiAddr, 0)
	addresses = append(addresses, addr)
	return &addresses
}

func TestRecordMemberHealthInfot(*testing.T) {
	u := UptimeChecker{
		nodeAddresses: make(map[ActorID]map[MultiAddr]HealtcheckInfo)
	}

	actorId := ActorID(1000)

	upInfos := prepareUpInfos(uint64(100))
	addresses := prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos := prepareUpInfos(uint64(200))
	addresses := prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)


	upInfos := prepareUpInfos(uint64(300))
	addresses := prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	if !u.nodeAddresses[actorId].IsOnline {
		t.Fatalf(`Should be online`)
	}

	if u.nodeAddresses[actorId]. {
		t.Fatalf(`Should be online`)
	}
}