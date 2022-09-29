use cid::Cid;
use fvm_shared::ActorID;
use fvm_shared::clock::ChainEpoch;
use crate::error::Error;
use crate::types::{InitParams, NodeInfo, NodeInfoPayload, ReportPayload};

pub trait UptimeCheckerActor {
    /// Initializes the state of the uptime actor. It accepts
    /// an initial list of checkers to populate the list in
    /// the constructor. IPC subnets will potentially pre-populate
    /// this list with the initial validators of the subnet
    ///
    /// - methodNum: 1
    /// - allowed callers: any account.
    /// - impacted state: State for the uptime actor
    /// is initialized.
    fn init(params: InitParams) -> Result<(), Error>;

    /// Adds a new checker to the list of checkers.
    /// This method checks that a checker for that
    /// peerID is not registered yet, and that the owner
    /// specified in CheckInfo is the message signer.
    ///
    /// - methodNum: 2
    /// - allowed callers: any account.
    /// - impacted state: checkers HAMT is updated.
    fn new_checker(params: NodeInfoPayload) -> Result<(), Error>;

    /// Adds a new member to the list of nodes to be checked.
    /// This method checks that a member for that
    /// peerID is not registered yet, and that the owner
    /// specified in CheckInfo is the message signer.
    ///
    /// - methodNum: 3
    /// - allowed callers: any account.
    /// - impacted state: members HAMT is updated.
    fn new_member(params: NodeInfoPayload) -> Result<(), Error>;

    /// Edits the node information of a checker. The method
    /// checks that the owner of the peer is the one signing
    /// the transaction. Owners are allowed to any information
    /// for the peer (including the owner and the peerID).
    ///
    /// - methodNum: 4
    /// - allowed callers: owner of the peerID.
    /// - impacted state: edits the CheckInfo for the peerID
    /// in checkers.
    fn edit_checker(params: NodeInfoPayload) -> Result<(), Error>;

    /// Edits the node information of a member. The method
    /// checks that the owner of the peer is the one signing
    /// the transaction. Owners are allowed to any information
    /// for the peer (including the owner and the peerID).
    ///
    /// - methodNum: 5
    /// - allowed callers: owner of the peerID.
    /// - impacted state: edits the NodeInfo for the peerID
    /// in members.
    fn edit_member(params: NodeInfoPayload) -> Result<(), Error>;

    /// Removes a checker from the list. Only the owner of
    /// the PeerID is allowed to remove themselves from the list.
    ///
    /// - methodNum: 6
    /// - allowed callers: owner of the peerID.
    /// - impacted state: removes peerID from the checkers HAMT.
    fn rm_checker() -> Result<(), Error>;

    /// Removes a member from the list. Only the owner of
    /// the PeerID is allowed to remove themselves from the list.
    ///
    /// - methodNum: 7
    /// - allowed callers: owner of the peerID.
    /// - impacted state: removes peerID from the members HAMT.
    fn rm_member() -> Result<(), Error>;

    /// Reports a checker for being offline. This registers
    /// a new offline vote for the checker with the specified
    /// peerID and removes the peer from checkers if there are
    /// > 2/3 votes. If the last offline vote is older than
    /// > OFFLINE_COUNT_RESTART the previous votes are not
    /// considered and conveniently cleaned, and the new one is added
    /// as the first one (it would be unfair to collect votes for
    /// the whole history of the checker). Only checkers are allowed
    /// to report other checkers for being offline. Checkers are
    /// allowed to vote as many times as they want to update the
    /// valule of last_vote (despite a single vote being registred
    /// per peerID).
    ///
    /// Before removing a checker from the checkers list,
    /// a sanity-check is performed verifying that the voters
    /// are still in the checkers list. This prevents from peers
    /// being able to abuse the protocol changing their peerIDs or
    /// removing and adding their membership to forge new votes
    /// to force the removal of a specific checker.
    ///
    /// - methodNum: 8
    /// - allowed callers: checkers.
    /// - impacted state: offline_checkers is updated with either
    /// a new peerID and a vote, or a new vote for a PeerID, and
    /// it removes PeerID from checkers if the number of
    /// votes > 2/3 checkers
    fn report_checker(param: ReportPayload) -> Result<(), Error>;
}

pub trait LoadableState {
    fn new(nodes: Vec<NodeInfo>, voting_duration: &Option<ChainEpoch>) -> Result<Self, Error> where Self: Sized;

    fn upsert_node(&mut self, node: NodeInfo) -> Result<(), Error>;

    fn remove_node(&mut self, id: &ActorID) -> Result<(), Error>;

    fn is_checker(&self, caller: &ActorID) -> Result<bool, Error>;

    fn upsert_checker(&mut self, node: NodeInfo) -> Result<(), Error>;

    fn remove_checker(&mut self, id: &ActorID) -> Result<(), Error>;

    /// Removes the checker without performing owner check. Use with care.
    fn remove_checker_unchecked(&mut self, id: &ActorID) -> Result<(), Error>;

    fn has_voted(&self, reported: &ActorID, voter: &ActorID) -> Result<bool, Error>;

    fn record_voted(&mut self, reported: &ActorID, voter: &ActorID) -> Result<usize, Error>;

    fn total_checkers(&self) -> usize;

    fn vote_duration_threshold(&self) -> ChainEpoch;

    fn load() -> Result<Self, Error> where Self: Sized;

    fn save(&self) -> Result<Cid, Error>;
}