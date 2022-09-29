use anyhow::anyhow;
use crate::Error;
use cid::multihash::Code;
use cid::Cid;
use fvm_ipld_blockstore::Block;
use fvm_ipld_hamt::{BytesKey, Hamt};
use serde::de::DeserializeOwned;
use serde::Serialize;

const HAMT_BIT_WIDTH: u32 = 5;

/// A blockstore that delegates to IPLD syscalls.
pub struct Blockstore;

impl fvm_ipld_blockstore::Blockstore for Blockstore {
    fn get(&self, cid: &Cid) -> Result<Option<Vec<u8>>, anyhow::Error> {
        Ok(Option::from(fvm_sdk::ipld::get(cid)?))
    }

    fn put_keyed(&self, k: &Cid, block: &[u8]) -> Result<(), anyhow::Error> {
        let code = Code::try_from(k.hash().code()).map_err(|e| anyhow!(e.to_string()))?;
        let k2 = self.put(code, &Block::new(k.codec(), block))?;
        if k != &k2 {
            return Err(anyhow!("put block with cid {} but has cid {}", k, k2));
        }
        Ok(())
    }

    fn put<D>(&self, code: Code, block: &Block<D>) -> Result<Cid, anyhow::Error>
    where
        D: AsRef<[u8]>,
    {
        const SIZE: u32 = 32;
        let k = fvm_sdk::ipld::put(code.into(), SIZE, block.codec, block.data.as_ref())
            .map_err(|e| anyhow!("put failed with {:?}", e))?;
        Ok(k)
    }
}

/// Map type to be used within actors. The underlying type is a HAMT.
pub type Map<'bs, BS, V> = Hamt<&'bs BS, V, BytesKey>;

/// Create a hamt with a custom bitwidth.
#[inline]
pub(crate) fn make_empty_map<BS, V>(store: &'_ BS) -> Map<'_, BS, V>
where
    BS: fvm_ipld_blockstore::Blockstore,
    V: DeserializeOwned + Serialize,
{
    Map::<_, V>::new_with_bit_width(store, HAMT_BIT_WIDTH)
}

/// Create a map with a root cid.
#[inline]
pub(crate) fn get_map_from_cid<'bs, BS, V>(
    cid: &Cid,
    store: &'bs BS,
) -> Result<Map<'bs, BS, V>, Error>
where
    BS: fvm_ipld_blockstore::Blockstore,
    V: DeserializeOwned + Serialize,
{
    Ok(Map::<_, V>::load_with_bit_width(
        cid,
        store,
        HAMT_BIT_WIDTH,
    )?)
}
