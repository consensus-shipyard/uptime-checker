mod actor;
mod blockstore;
mod error;
mod hamt_state;
mod traits;
mod types;
mod util;

use crate::actor::Actor;
use crate::error::Error;
use crate::traits::UptimeCheckerActor;
use crate::util::parse_params_or_abort;
use fvm_sdk::NO_DATA_BLOCK_ID;
use crate::hamt_state::HamtState;

/// The actor's WASM entrypoint. It takes the ID of the parameters block,
/// and returns the ID of the return value block, or NO_DATA_BLOCK_ID if no
/// return value.
#[no_mangle]
pub fn invoke(params_block_id: u32) -> u32 {
    let params = match fvm_sdk::message::params_raw(params_block_id) {
        Ok(v) => v.1,
        Err(e) => fvm_sdk::vm::abort(e as u32, Some(e.to_string().as_str())),
    };

    let r = match fvm_sdk::message::method_number() {
        1 => Actor::<HamtState>::init(parse_params_or_abort(&params)),
        2 => Actor::<HamtState>::new_checker(parse_params_or_abort(&params)),
        3 => Actor::<HamtState>::new_member(parse_params_or_abort(&params)),
        4 => Actor::<HamtState>::edit_checker(parse_params_or_abort(&params)),
        5 => Actor::<HamtState>::edit_member(parse_params_or_abort(&params)),
        6 => Actor::<HamtState>::rm_checker(),
        7 => Actor::<HamtState>::rm_member(),
        8 => Actor::<HamtState>::report_checker(parse_params_or_abort(&params)),
        _ => Ok(()),
    };

    match r {
        Ok(_) => NO_DATA_BLOCK_ID,
        Err(e) => fvm_sdk::vm::abort(e.code(), Some(e.msg().as_str())),
    }
}
