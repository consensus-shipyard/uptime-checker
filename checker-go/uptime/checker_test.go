package uptime

import (
    "testing"
)

func prepareUpInfos(latency uint64, isOnline bool) *[]UpInfo {
	upInfos := make([]UpInfo, 0)
	upInfos = append(upInfos, UpInfo { isOnline: isOnline, latency: latency, checkedTime: uint64(1000)})
	return &upInfos
}

func prepareMultiAddrs(addr string) *[]MultiAddr {
	addresses := make([]MultiAddr, 0)
	addresses = append(addresses, addr)
	return &addresses
}

func TestRecordMemberHealthInfo_allOnline(t *testing.T) {
	u := UptimeChecker{
		nodeAddresses: make(map[ActorID]map[MultiAddr]HealtcheckInfo),
	}

	actorId := ActorID(1000)

	upInfos := prepareUpInfos(uint64(100), true)
	addresses := prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos = prepareUpInfos(uint64(200), true)
	addresses = prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos = prepareUpInfos(uint64(300), true)
	addresses = prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	if !u.nodeAddresses[actorId]["abc"].IsOnline {
		t.Fatalf(`Should be online`)
	}

	if u.nodeAddresses[actorId]["abc"].AvgLatency != 200 {
		t.Fatalf(`Should have avg latency of 200`)
	}

	if u.nodeAddresses[actorId]["abc"].LatencyCounts != 3 {
		t.Fatalf(`Should have latency count of 3`)
	}
}


func TestRecordMemberHealthInfo_someOffline(t *testing.T) {
	u := UptimeChecker{
		nodeAddresses: make(map[ActorID]map[MultiAddr]HealtcheckInfo),
	}

	actorId := ActorID(1000)

	upInfos := prepareUpInfos(uint64(100), true)
	addresses := prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos = prepareUpInfos(uint64(200), false)
	addresses = prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos = prepareUpInfos(uint64(300), true)
	addresses = prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	if !u.nodeAddresses[actorId]["abc"].IsOnline {
		t.Fatalf(`Should be online`)
	}

	if u.nodeAddresses[actorId]["abc"].AvgLatency != 200 {
		t.Fatalf(`Should have avg latency of 200`)
	}

	if u.nodeAddresses[actorId]["abc"].LatencyCounts != 2 {
		t.Fatalf(`Should have latency count of 3`)
	}
}

func TestRecordMemberHealthInfo_finalOffline(t *testing.T) {
	u := UptimeChecker{
		nodeAddresses: make(map[ActorID]map[MultiAddr]HealtcheckInfo),
	}

	actorId := ActorID(1000)

	upInfos := prepareUpInfos(uint64(100), true)
	addresses := prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos = prepareUpInfos(uint64(200), true)
	addresses = prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	upInfos = prepareUpInfos(uint64(0), false)
	addresses = prepareMultiAddrs("abc")
	u.recordMemberHealthInfo(actorId, upInfos, addresses)

	if u.nodeAddresses[actorId]["abc"].IsOnline {
		t.Fatalf(`Should be offline`)
	}

	if u.nodeAddresses[actorId]["abc"].AvgLatency != 150 {
		t.Fatalf(`Should have avg latency of 150`)
	}

	if u.nodeAddresses[actorId]["abc"].LatencyCounts != 2 {
		t.Fatalf(`Should have latency count of 3`)
	}
}
