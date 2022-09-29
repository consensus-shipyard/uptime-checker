package uptime

import (
	"sync"
	"context"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/go-address"
)

type InnerState = HAMTState

type CacheState struct {
	self ActorID

	inner InnerState
	
	// Local cache of voted checkers. TODO: use LRU cache
	processedCheckers map[ActorID]bool

	rwLock sync.RWMutex
}

func Load(ctx context.Context, api v0api.FullNode, actorAddr address.Address, self ActorID) (CacheState, error)  {
	s, err := LoadHAMTState(ctx, api, actorAddr)
	if err != nil {
		return CacheState{}, err
	}
	return CacheState {
		self: self,
		inner: s,
		processedCheckers: make(map[ActorID]bool),
	}, nil
}

func (c *CacheState) HasRegistered(actor ActorID) (bool, error) {
	return c.inner.HasRegistered(actor)
}

func (c *CacheState) ListReportedCheckerNotVoted() (*map[ActorID]*[]MultiAddr, error) {
	ids := make(map[ActorID]*[]MultiAddr)
	
	l, err := c.inner.GetOfflineCheckers()
	if err != nil {
		return nil, err
	}

	for _, actorID := range(l) {
		hasVoted, err := c.HasVotedReportedPeer(actorID)
		if err != nil {
			return nil, err
		}

		if hasVoted {
			continue
		}
		
		addrList, err := c.inner.ListCheckerMultiAddrs(actorID)
		if err != nil {
			return nil, err
		}

		ids[actorID] = addrList
	}

	log.Debugw("list of not voted offline checkers", "actorIds", ids)

	return &ids, nil
}

func (c *CacheState) ListCheckers() ([]ActorID, error) {
	return c.inner.ListCheckers()
}

func (c *CacheState) ListMembers() ([]ActorID, error) {
	return c.inner.ListMembers()
}

func (c *CacheState) ListMemberMultiAddrs(actorID ActorID) (*[]MultiAddr, error) {
	return c.inner.ListMemberMultiAddrs(actorID)
}

func (c *CacheState) ListCheckerMultiAddrs(actorID ActorID) (*[]MultiAddr, error) {
	return c.inner.ListCheckerMultiAddrs(actorID)
}

func (c *CacheState) HasVotedReportedPeer(targetPeer ActorID) (bool, error) {
	if c.hasVotedReportedPeerLocally(targetPeer) {
		return true, nil
	}
	return c.hasVotedReportedPeerInActor(targetPeer)
}

func (c *CacheState) hasVotedReportedPeerLocally(targetPeer ActorID) (bool) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()
	_, ok := c.processedCheckers[targetPeer];
	return ok
}

func (c *CacheState) hasVotedReportedPeerInActor(reported ActorID) (bool, error) {
	return c.inner.HasVotedForReportedChecker(reported, c.self)
}