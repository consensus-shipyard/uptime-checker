use crate::blockstore::{make_empty_map, Blockstore, get_map_from_cid};
use crate::types::{NodeInfo, Votes};
use crate::Error;
use cid::Cid;
use fvm_ipld_encoding::{to_vec, CborStore, DAG_CBOR};
use fvm_ipld_hamt::BytesKey;
use fvm_shared::ActorID;
use fvm_shared::clock::ChainEpoch;
use multihash::Code;
use serde::{Deserialize, Serialize};
use crate::traits::LoadableState;

const DEFAULT_VOTING_DURATION: ChainEpoch = 200;

/// The state object.
#[derive(Debug, Serialize, Deserialize)]
pub struct HamtState {
    /// The list of node members in the registry
    members: Cid, // HAMT<BytesKey from ActorID, NodeInfo>
    /// List of checkers registered in the system.
    checkers: Cid, // HAMT<BytesKey from ActorID, NodeInfo>
    /// Data structure used to signal offline checkers.
    offline_checkers: Cid, // HAMT<BytesKey, Votes>
    /// The total number of checkers
    total_checkers: usize,
    /// The voting duration threshold
    voting_duration: ChainEpoch,
}

impl HamtState {
    fn ensure_owner(b: &NodeInfo) -> Result<(), Error> {
        if fvm_sdk::message::caller() != *b.creator() {
            Err(Error::NotOwner)
        } else {
            Ok(())
        }
    }

    fn upsert(store: &Blockstore, map_cid: &Cid, node: NodeInfo) -> Result<Cid, Error> {
        let mut map = get_map_from_cid(map_cid, store)?;

        let str = &node.creator().to_string()[..];
        let id = BytesKey::from(str);
        let n = map.get(&id)?.unwrap_or(&node);

        Self::ensure_owner(n)?;

        map.set(id, node)?;
        Ok(map.flush()?)
    }

    fn remove(store: &Blockstore, map_cid: &Cid, id: &ActorID) -> Result<Cid, Error> {
        let mut map = get_map_from_cid(map_cid, store)?;

        let key = BytesKey::from(&id.to_string()[..]);
        let n = map.get(&key)?.ok_or(Error::NotExists)?;

        Self::ensure_owner(n)?;

        map.delete(&key)?;
        Ok(map.flush()?)
    }
}

impl LoadableState for HamtState {
    fn new(nodes: Vec<NodeInfo>, voting_duration: &Option<ChainEpoch>) -> Result<Self, Error> {
        let total_checkers = nodes.len();
        let mut checker_map = make_empty_map::<_, NodeInfo>(&Blockstore);
        for n in nodes {
            let str = &n.creator().to_string()[..];
            checker_map.set(BytesKey::from(str), n)?;
        }
        Ok(HamtState {
            members: make_empty_map::<_, NodeInfo>(&Blockstore).flush()?,
            checkers: checker_map.flush()?,
            offline_checkers: make_empty_map::<_, Votes>(&Blockstore).flush()?,
            total_checkers,
            voting_duration: voting_duration.unwrap_or(DEFAULT_VOTING_DURATION)
        })
    }

    fn upsert_node(&mut self, node: NodeInfo) -> Result<(), Error> {
        self.members = Self::upsert(&Blockstore{}, &self.members, node)?;
        Ok(())
    }

    fn remove_node(&mut self, id: &ActorID) -> Result<(), Error> {
        self.members = Self::remove(&Blockstore{}, &self.members, id)?;
        Ok(())
    }

    fn is_checker(&self, checker: &ActorID) -> Result<bool, Error> {
        let map = get_map_from_cid::<_, NodeInfo>(&self.checkers, &Blockstore{})?;
        let key = BytesKey::from(&checker.to_string()[..]);
        Ok(map.contains_key(&key)?)
    }

    fn upsert_checker(&mut self, node: NodeInfo) -> Result<(), Error> {
        self.checkers = Self::upsert(&Blockstore{}, &self.checkers, node)?;
        Ok(())
    }

    fn remove_checker(&mut self, id: &ActorID) -> Result<(), Error> {
        self.checkers = Self::remove(&Blockstore{}, &self.checkers, id)?;
        Ok(())
    }

    fn remove_checker_unchecked(&mut self, checker: &ActorID) -> Result<(), Error> {
        let mut map = get_map_from_cid::<_, NodeInfo>(&self.checkers, &Blockstore{})?;
        map.delete(&BytesKey::from(&checker.to_string()[..]))?;
        self.checkers = map.flush()?;
        Ok(())
    }

    fn has_voted(&self, reported: &ActorID, voter: &ActorID) -> Result<bool, Error> {
        let map = get_map_from_cid::<_, Votes>(&self.offline_checkers, &Blockstore{})?;
        Ok(
            map.get(&BytesKey::from(&reported.to_string()[..]))?
                .map(|v| v.has_voted(voter))
                .unwrap_or(false)
        )
    }

    fn record_voted(&mut self, reported: &ActorID, voter: &ActorID) -> Result<usize, Error> {
        let mut map = get_map_from_cid::<_, Votes>(&self.offline_checkers, &Blockstore{})?;
        let reported_key = BytesKey::from(&reported.to_string()[..]);

        match map.get(&reported_key)? {
            None => {
                let mut votes = Votes::new(fvm_sdk::network::curr_epoch());
                votes.vote(voter);
                map.set(reported_key, votes)?;
                self.offline_checkers = map.flush()?;
                Ok(1)
            }
            Some(votes) => {
                let t = self.vote_duration_threshold();
                if !votes.within_threshold(fvm_sdk::network::curr_epoch(), t) {
                    // delete the current round and start again
                    let mut votes = Votes::new(fvm_sdk::network::curr_epoch());
                    votes.vote(voter);

                    map.set(reported_key, votes)?;

                    self.offline_checkers = map.flush()?;
                    return Ok(0);
                }

                if votes.has_voted(voter) {
                    return Err(Error::AlreadyVoted(*voter));
                }

                // TODO: make votes HAMT as well
                let mut votes = votes.clone();
                votes.vote(voter);
                let total = votes.total_votes();

                map.set(reported_key, votes)?;
                self.offline_checkers = map.flush()?;

                Ok(total)
            }
        }
    }

    fn total_checkers(&self) -> usize { self.total_checkers }

    fn vote_duration_threshold(&self) -> ChainEpoch { self.voting_duration }

    fn load() -> Result<Self, Error> {
        let root = fvm_sdk::sself::root()?;
        (Blockstore.get_cbor::<Self>(&root)?).ok_or(Error::CannotDeserialize)
    }

    fn save(&self) -> Result<Cid, Error> {
        let serialized = to_vec(self)?;
        let cid = fvm_sdk::ipld::put(Code::Blake2b256.into(), 32, DAG_CBOR, serialized.as_slice())?;
        fvm_sdk::sself::set_root(&cid)?;
        Ok(cid)
    }
}
