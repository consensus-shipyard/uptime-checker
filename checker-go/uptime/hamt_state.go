package uptime

import (
	"context"
	// "fmt"

	"github.com/ipfs/go-cid"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/specs-actors/v7/actors/util/adt"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin"

	cbor "github.com/ipfs/go-ipld-cbor"
)

type HAMTStateInner struct {
    Members cid.Cid
    Checkers cid.Cid
    OfflineCheckers cid.Cid
    TotalCheckers uint64
}

type HAMTState struct {
    inner HAMTStateInner
    store adt.Store
}

func LoadHAMTState(ctx context.Context, api v0api.FullNode, addr address.Address) (HAMTState, error)  {
	act, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
	if err != nil {
		return HAMTState{}, err
	}

	var st HAMTStateInner
	bs := blockstore.NewAPIBlockstore(api)
	cst := cbor.NewCborStore(bs)
	if err := cst.Get(ctx, act.Head, &st); err != nil {
		return HAMTState{}, err
	}

	return HAMTState {
		inner: st,
		store: adt.WrapStore(ctx, cst),
	}, nil
}

func (m *HAMTState) GetOfflineCheckers() ([]ActorID, error) {
	keys := make([]ActorID, 0)

	ccid := m.inner.OfflineCheckers
	checkerMap, err := adt.AsMap(m.store, ccid, builtin.DefaultHamtBitwidth)
	if err != nil {
		return keys, err
	}

	strs, err := checkerMap.CollectKeys()
	if err != nil {
		return keys, err
	}

	for _, s := range strs {
		actorID, _ := parseActorIDFromString(s)
		keys = append(keys, actorID)
	}

	return keys, nil
}

func (m *HAMTState) HasVotedForReportedChecker(reported ActorID, voter ActorID) (bool, error) {
	ccid := m.inner.OfflineCheckers
	checkerMap, err := adt.AsMap(m.store, ccid, builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, err
	}

	v := Votes{}
	found, err := checkerMap.Get(NewWrappedActorKey(reported), &v)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	return v.HasVoted(voter)
}

func (m *HAMTState) HasRegistered(actor ActorID) (bool, error) {
	checkerMap, err := adt.AsMap(m.store, m.inner.Checkers, builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, err
	}

	return checkerMap.Has(NewWrappedActorKey(actor))
}

func (m *HAMTState) ListCheckerMultiAddrs(actorID ActorID) (*[]MultiAddr, error) {
	checkerMap, err := adt.AsMap(m.store, m.inner.Checkers, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	d := NodeInfo{}
	found, err := checkerMap.Get(NewWrappedActorKey(actorID), &d)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, nil
	}

	log.Debugw("addresses", "actorID", actorID, "addresses", d)
	return &d.Addresses, nil
}

func (m *HAMTState) ListMemberMultiAddrs(actorID ActorID) (*[]MultiAddr, error) {
	checkerMap, err := adt.AsMap(m.store, m.inner.Members, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	d := NodeInfo{}
	found, err := checkerMap.Get(NewWrappedActorKey(actorID), &d)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, nil
	}

	return &d.Addresses, nil
}

func (m *HAMTState) ListMembers() ([]ActorID, error) {
	return m.List(m.inner.Members)
}

func (m *HAMTState) ListCheckers() ([]ActorID, error) {
	return m.List(m.inner.Checkers)
}

func (m *HAMTState) List(ccid cid.Cid) ([]ActorID, error) {
	ids := make([]ActorID, 0)

	checkerMap, err := adt.AsMap(m.store, ccid, builtin.DefaultHamtBitwidth)
	if err != nil {
		return ids, err
	}

	strs, err := checkerMap.CollectKeys()
	if err != nil {
		return ids, err
	}

	for _, s := range strs {
		actorID, _ := parseActorIDFromString(s)
		ids = append(ids, actorID)
	}

	return ids, nil
}
