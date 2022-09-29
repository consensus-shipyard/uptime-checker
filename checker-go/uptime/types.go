package uptime

import (
	"fmt"
)

// TODO: correct types?
type PeerID = string;
type MultiAddr = string;
type ActorID = uint64;
type ChainEpoch = int64;

type WrappedActorKey struct {
	A ActorID
}

func (a WrappedActorKey) Key() string {
	return fmt.Sprintf("%d",a.A)
}

func NewWrappedActorKey(a ActorID) WrappedActorKey {
	return WrappedActorKey{ A: a }
}

type NodeInfo struct {
	// PeerID of the node
	Id PeerID `json:"id"`
	// The creator of the node. Only creator can modifier other fields of this struct
	Creator ActorID `json:"creator"`
	/// List of multiaddresses exposed by the node
	/// along with the supported healthcheck endpoints.
	///
	/// e.g. [ /ip4/10.1.1.1/quic/8080/p2p/<peer_id>/ping,
	///        /ip4/10.1.1.1/tcp/8081/http/get/healtcheck,
	///      ]
	/// These multiaddresses are signalling that the liveliness
	/// can be checked by using the default libp2p ping protocol
	/// in the first multiaddress, or by sending a GET HTTP
	/// query to the /healtchek endpoint at 10.1.1.1:8081.
	Addresses []MultiAddr `json:"addresses"`
}

type Votes struct {
    // Time of the last offline vote received by a checker.
    LastVote ChainEpoch
    // Checkers that have voted
    Votes []ActorID
}

/// Healthcheck information provided for each peer.
/// NOTE: This is an initial proposal, each checker could
/// include different (an arbitrary types of) information.
type HealtcheckInfo struct {
    HealtcheckAddr MultiAddr

    AvgLatency uint64
    LatencyCounts uint64

    IsOnline bool
    Latency uint64
    LastChecked uint64
}

type UpInfo struct {
    isOnline bool
    latency uint64
    checkedTime uint64
}

type PeerReportPayload struct {
    Checker ActorID `json:"checker"`
}
