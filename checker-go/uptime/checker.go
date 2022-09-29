package uptime

import (
	"context"
	"sync"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/go-address"
	chainTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"

	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p-core/host"
	libp2pMultiaddr "github.com/multiformats/go-multiaddr"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
)

var log = logging.Logger("uptime")

const NEW_CHECKER_METHOD = 2
const NEW_MEMBER_METHOD = 3
const EDIT_CHECKER_METHOD = 4
const EDIT_MEMBER_METHOD = 5
const RM_CHCKER_METHOD = 6
const RM_MEMBER_METHOD = 7
const REPORT_CHECKER_METHOD = 8

const PING_TIMEOUT = 120 * time.Second // 120 seconds
const DEFAULT_SLEEP_SECONDS = 5 * time.Second // 5 seconds

// UptimeChecker maintains the uptime of member nodes
type UptimeChecker struct {
	api v0api.FullNode

	self ActorID
	walletIndex int
	uptimeCheckerAddress address.Address
	
	checkerAddresses []MultiAddr
	nodeAddresses map[ActorID]map[MultiAddr]HealtcheckInfo

	// libp2p ping related
	node host.Host // node is the libp2p node struct of the checker
	ping *ping.PingService // the libp2p ping service

	rwLock sync.RWMutex
	stop bool
}

func NewUptimeChecker(
	api v0api.FullNode,
	uptimeCheckerAddress string,
	checkerAddresses []MultiAddr,
	self ActorID,
	walletIndex int,
	node host.Host,
	ping *ping.PingService,
) (UptimeChecker, error) {
	addr, err := address.NewFromString(uptimeCheckerAddress)
	if err != nil {
		return UptimeChecker{}, err
	}
	return UptimeChecker {
		api: api,

		self: self,
		walletIndex: walletIndex,
		uptimeCheckerAddress: addr,

		checkerAddresses: checkerAddresses,
		nodeAddresses: make(map[ActorID]map[MultiAddr]HealtcheckInfo),

		node: node,
		ping: ping,

		stop: false,
	}, nil
}

func (u *UptimeChecker) Start(ctx context.Context) error {
	hasRegistered, err := u.HasRegistered(ctx)
	if err != nil {
		return err
	}

	if !hasRegistered {
		if err := u.Register(ctx); err != nil {
			return err
		}
	} else {
		log.Infow("already registered with the actor, skip register")
	}

	go u.processReportedCheckers(ctx)

	go u.monitorMemberNodes(ctx)

	go u.monitorCheckerNodes(ctx)

	return nil
}

// HasRegistered checks if the current checker has already registered itself in the actor
func (u *UptimeChecker) HasRegistered(ctx context.Context) (bool, error) {
	s, err := Load(ctx, u.api, u.uptimeCheckerAddress, u.self)
	log.Infow("Self actor id", "actorId", u.self)

	if err != nil {
		return false, err
	}
	return s.HasRegistered(u.self)
}

// Register registers the current checker to actor
func (u *UptimeChecker) Register(ctx context.Context) error {
	log.Infow("has yet to be registered with the actor, register now")

	peerID := u.node.ID();
	log.Infow("register new checker with peer id", "peerID", peerID.String())

	params, err := encodeJson(NodeInfo {
		Id: peerID.String(),
		Addresses: u.checkerAddresses,
	})
	if err != nil {
		return err
	}

	fromAddr, err := u.getWalletAddress(ctx)
	if err != nil {
		return err
	}

	return u.executeMsgAndWait(ctx, NEW_CHECKER_METHOD, fromAddr, params)
}

// Reports to the actor that the checker is down
func (u *UptimeChecker) ReportChecker(ctx context.Context, actor ActorID) error {
	log.Infow("report checker as down", "checker", actor)

	params, err := encodeJson(PeerReportPayload {
		Checker: actor,
	})
	if err != nil {
		return err
	}

	fromAddr, err := u.getWalletAddress(ctx)
	if err != nil {
		return err
	}

	return u.executeMsgAndWait(ctx, REPORT_CHECKER_METHOD, fromAddr, params)
}

// IsStop checks if the up time checker should stop running
func (u *UptimeChecker) IsStop() bool {
	u.rwLock.RLock()
	defer u.rwLock.RUnlock()
	return u.stop
}

// Stop stops the checker
func (u *UptimeChecker) Stop() {
	u.rwLock.Lock()
	defer u.rwLock.Unlock()
	u.stop = true
}

func (u *UptimeChecker) CheckChecker(ctx context.Context, actorID ActorID, addrs *[]MultiAddr) error {
	infos := u.multiAddrsUp(ctx, addrs)
	
	if !allUp(&infos) {
		log.Warnw("actor down, report now", "actorID", actorID)

		state, err := Load(ctx, u.api, u.uptimeCheckerAddress, u.self)
		if err != nil {
			log.Errorw("cannot load state", "err", err)
			return err
		}

		hasVoted, err := state.HasVotedReportedPeer(actorID)
		if !hasVoted {
			return u.ReportChecker(ctx, actorID)
		}
		log.Debugw("has already reported actor", "actor", actorID)
		
	}
	return nil
}

func (u *UptimeChecker) CheckMember(ctx context.Context, actorID ActorID, addrs *[]MultiAddr) error {
	infos := u.multiAddrsUp(ctx, addrs)
	return u.recordMemberHealthInfo(actorID, &infos, addrs)
}

// /// =================== Private Functions ====================

// Records and aggregate on the health info of membership nodes
func (u *UptimeChecker) recordMemberHealthInfo(actorID ActorID, upInfos *[]UpInfo, addrs *[]MultiAddr) error {
	healthInfos, ok := u.nodeAddresses[actorID]
	if !ok {
		healthInfos = make(map[MultiAddr]HealtcheckInfo, len(*upInfos))
	}

	for i, addr := range(*addrs) {
		val, ok := healthInfos[addr]
		if !ok {
			val = HealtcheckInfo{
				HealtcheckAddr: addr,
				
				AvgLatency: (*upInfos)[i].latency,
				LatencyCounts: 1,
				
				IsOnline: (*upInfos)[i].isOnline,
				Latency: (*upInfos)[i].latency,
				LastChecked: (*upInfos)[i].checkedTime,
			}
		} else {
			val.IsOnline = (*upInfos)[i].isOnline
			val.Latency = (*upInfos)[i].latency
			val.LastChecked = (*upInfos)[i].checkedTime

			// moving average calculation
			newCount := val.AvgLatency + 1
			val.AvgLatency = val.AvgLatency * val.AvgLatency / newCount + val.Latency / newCount
			val.AvgLatency = newCount
		}
		healthInfos[addr] = val
	}

	u.nodeAddresses[actorID] = healthInfos

	return nil
}

func (u *UptimeChecker) multiAddrsUp(ctx context.Context, addrs *[]MultiAddr) []UpInfo {
	upInfos := make([]UpInfo, 0)
	for _, addr := range(*addrs) {

		isCheck := true
		for _, selfAddr := range u.checkerAddresses {
			if selfAddr == addr {
				isCheck = false
				break
			}
		}

		if !isCheck {
			continue
		}

		upInfos = append(upInfos, u.isUp(ctx, addr))
	}
	return upInfos
}

func (u *UptimeChecker) processReportedCheckers(ctx context.Context) error {
	for {
		if u.IsStop() {
			break
		}

		state, err := Load(ctx, u.api, u.uptimeCheckerAddress, u.self)
		if err != nil {
			log.Errorw("cannot load state", "err", err)
			continue
		}

		listToCheck, err := state.ListReportedCheckerNotVoted()
		if err != nil {
			log.Errorw("cannot list repored checkers not voted", "err", err)
			continue
		}

		for toCheckPeerID, addrs := range(*listToCheck) {
			u.CheckChecker(ctx, toCheckPeerID, addrs)
		}

		u.sleep(DEFAULT_SLEEP_SECONDS)
	}

	return nil
}

func (u *UptimeChecker) monitorMemberNodes(ctx context.Context) error {
	for {
		if u.IsStop() {
			break
		}

		state, err := Load(ctx, u.api, u.uptimeCheckerAddress, u.self)
		if err != nil {
			log.Errorw("cannot load state", "err", err)
			continue
		}

		listToCheck, err := state.ListMembers()
		if err != nil {
			log.Errorw("cannot list members", "err", err)
			continue
		}

		for _, toCheckActorID := range listToCheck {
			addrs, err := state.ListMemberMultiAddrs(toCheckActorID)
			if err != nil {
				log.Errorw("cannot list member multi addrs", "actor", toCheckActorID, "err", err)
				continue
			}

			log.Debugw("member info", "actor", toCheckActorID, "addrs", addrs)

			u.CheckMember(ctx, toCheckActorID, addrs)
		}

		u.sleep(DEFAULT_SLEEP_SECONDS)
	}

	return nil
}

func (u *UptimeChecker) monitorCheckerNodes(ctx context.Context) error {
	for {
		if u.IsStop() {
			break
		}

		state, err := Load(ctx, u.api, u.uptimeCheckerAddress, u.self)
		if err != nil {
			log.Errorw("cannot load state", "err", err)
			continue
		}

		listToCheck, err := state.ListCheckers()
		if err != nil {
			log.Errorw("cannot list members", "err", err)
			continue
		}

		log.Infow("list of checkers registered", "checkers", listToCheck)

		for _, toCheckPeerID := range listToCheck {
			addrs, err := state.ListCheckerMultiAddrs(toCheckPeerID)
			if err != nil {
				log.Errorw("cannot list member multi addrs", "peer", toCheckPeerID, "err", err)
				continue
			}

			log.Debugw("toCheckPeerID addrs", "toCheckPeerID", toCheckPeerID, "addrs", addrs)

			err = u.CheckChecker(ctx, toCheckPeerID, addrs)
			if err != nil {
				log.Errorw("cannot check checker", "peer", toCheckPeerID, "err", err)
			}
		}

		u.sleep(DEFAULT_SLEEP_SECONDS)
	}

	return nil
}

func (u *UptimeChecker) sleep(seconds time.Duration) {
	time.Sleep(seconds)
}

// executeMsgAndWait executes the method with given params and waits for the message to be executed
func (u *UptimeChecker) executeMsgAndWait(ctx context.Context, method uint32, from address.Address, params []byte) error {
	smsg, err := u.executeMsg(ctx, method, from, params)
	if err != nil {
		return err
	}

	log.Infow("waiting for message to execute...")
	return u.wait(ctx, smsg)
}

func (u *UptimeChecker) executeMsg(ctx context.Context, method uint32, from address.Address, params []byte) (*chainTypes.SignedMessage, error) {
	msg := &chainTypes.Message{
		To:     u.uptimeCheckerAddress,
		From:   from,
		Value:  big.Zero(),
		Method: abi.MethodNum(method),
		Params: params,
	}
	return u.api.MpoolPushMessage(ctx, msg, nil)
}

func (u *UptimeChecker) wait(ctx context.Context, smsg *chainTypes.SignedMessage) (error) {
	wait, err := u.api.StateWaitMsg(ctx, smsg.Cid(), 0)
	if err != nil {
		return err
	}

	// check it executed successfully
	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("actor execution failed")
	}

	return nil
}

// Checks is up and also record the latency
func (u *UptimeChecker) isUp(ctx context.Context, addrStr MultiAddr) UpInfo {
	upInfo := UpInfo{
		isOnline: false,
		latency: uint64(0),
		checkedTime: uint64(time.Now().Unix()),
	}

	addr, err := libp2pMultiaddr.NewMultiaddr(addrStr)
	if err != nil {
		log.Errorw("cannot parse multi addr", "addr", addr)
		return upInfo
	}

	peer, err := peerstore.AddrInfoFromP2pAddr(addr)
	if err != nil {
		log.Errorw("cannot add multi addr", "addr", addr)
		return upInfo
	}

	log.Debugw("addr for peer", "peer", peer)

	now := time.Now()

	cctx, _ := context.WithTimeout(ctx, PING_TIMEOUT)
	if err := u.node.Connect(cctx, *peer); err != nil {
		log.Errorw("cannot connect to multi addr", "peer", peer.ID, "err", err, "addr", addr)
		return upInfo
	}

	ch := u.ping.Ping(cctx, peer.ID)
	res := <-ch
	log.Debugw("got ping response!", "RTT:", res.RTT, "res", res)

	upInfo.isOnline = true
	upInfo.checkedTime = uint64(time.Now().Unix())
	upInfo.latency = uint64(time.Since(now))

	return upInfo
}

func (u *UptimeChecker) NodeInfo() map[ActorID]map[MultiAddr]HealtcheckInfo {
	return u.nodeAddresses
}

func (u *UptimeChecker) NodeInfoJsonString() (string, error) {
	log.Debugw("node map", "nodes",  u.nodeAddresses)

	data := make(map[ActorID]map[MultiAddr]HealtcheckInfo, 0)

	for k, v := range u.nodeAddresses {
		data[k] = v
	}

	bytes, err := encodeJson(data)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (u *UptimeChecker) getWalletAddress(ctx context.Context) (address.Address, error) {
	return getWalletAddressFromIndex(u.api, ctx, u.walletIndex)
}

func getWalletAddressFromIndex(api v0api.FullNode, ctx context.Context, index int) (address.Address, error) {
	walletList, err := api.WalletList(ctx)
	if err != nil {
		return address.Address{}, nil
	}
	return walletList[index], nil
}

func NewMember(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	multiAddresses []string,
	peerId string,
	walletIndex int,
) error {
	return upsert(ctx, api, uptimeCheckerAddress, multiAddresses, peerId, walletIndex, NEW_MEMBER_METHOD)
}

func EditMember(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	multiAddresses []string,
	peerId string,
	walletIndex int,
) error {
	return upsert(ctx, api, uptimeCheckerAddress, multiAddresses, peerId, walletIndex, EDIT_MEMBER_METHOD)
}

func EditChecker(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	multiAddresses []string,
	peerId string,
	walletIndex int,
) error {
	return upsert(ctx, api, uptimeCheckerAddress, multiAddresses, peerId, walletIndex, EDIT_CHECKER_METHOD)
}

func RmChecker(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	walletIndex int,
) error {
	return remove(ctx, api, uptimeCheckerAddress, walletIndex, RM_CHCKER_METHOD)
}

func RmMember(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	walletIndex int,
) error {
	return remove(ctx, api, uptimeCheckerAddress, walletIndex, RM_MEMBER_METHOD)
}

func upsert(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	multiAddresses []string,
	peerId string,
	walletIndex int,
	methodNumber uint32,
) error {
	wallet, err := getWalletAddressFromIndex(api, ctx, walletIndex)
	if err != nil {
		return err
	}

	params, err := encodeJson(NodeInfo {
		Id: peerId,
		Addresses: multiAddresses,
	})
	if err != nil {
		return err
	}

	return executeMsgAndWait(api, uptimeCheckerAddress, ctx, methodNumber, wallet, params)
}

func remove(
	ctx context.Context,
	api v0api.FullNode,
	uptimeCheckerAddress address.Address,
	walletIndex int,
	methodNumber uint32,
) error {
	wallet, err := getWalletAddressFromIndex(api, ctx, walletIndex)
	if err != nil {
		return err
	}
	return executeMsgAndWait(api, uptimeCheckerAddress, ctx, methodNumber, wallet, make([]byte, 0))
}

// executeMsgAndWait executes the method with given params and waits for the message to be executed
func executeMsgAndWait(
	api v0api.FullNode,
	actor address.Address,
	ctx context.Context,
	method uint32,
	from address.Address,
	params []byte,
) error {
	smsg, err := executeMsg(api, actor, ctx, method, from, params)
	if err != nil {
		return err
	}

	log.Infow("waiting for message to execute...")
	return wait(api, ctx, smsg)
}

func executeMsg(
	api v0api.FullNode,
	actor address.Address,
	ctx context.Context,
	method uint32,
	from address.Address,
	params []byte,
) (*chainTypes.SignedMessage, error) {
	msg := &chainTypes.Message{
		To:     actor,
		From:   from,
		Value:  big.Zero(),
		Method: abi.MethodNum(method),
		Params: params,
	}
	return api.MpoolPushMessage(ctx, msg, nil)
}

func wait(
	api v0api.FullNode,
	ctx context.Context,
	smsg *chainTypes.SignedMessage,
) (error) {
	wait, err := api.StateWaitMsg(ctx, smsg.Cid(), 0)
	if err != nil {
		return err
	}

	// check it executed successfully
	if wait.Receipt.ExitCode != 0 {
		return fmt.Errorf("actor execution failed")
	}

	return nil
}