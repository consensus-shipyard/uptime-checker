use fvm_shared::ActorID;

/// All the error from the actor crate
pub enum Error {
    AlreadyVoted(ActorID),
    CannotDeserialize,
    FVMIpldHamt(fvm_ipld_hamt::Error),
    Anyhow(anyhow::Error),
    FVMSharedErrorNum(fvm_shared::error::ErrorNumber),
    FVMSDKNoState(fvm_sdk::error::NoStateError),
    FVMEncoding(fvm_ipld_encoding::Error),
    FVMSharedAddress(fvm_shared::address::Error),
    NotOwner,
    NotExists,
    NotCaller,
}

impl Error {
    pub fn code(&self) -> u32 {
        match self {
            Error::AlreadyVoted(_) => 10002,
            Error::CannotDeserialize => 10003,
            Error::FVMIpldHamt(_) => 10004,
            Error::Anyhow(_) => 10005,
            Error::FVMSharedErrorNum(e) => *e as u32,
            Error::FVMSDKNoState(_) => 10006,
            Error::FVMEncoding(_) => 10007,
            Error::FVMSharedAddress(_) => 10008,
            Error::NotOwner => 10009,
            Error::NotExists => 10010,
            Error::NotCaller => 10011,
        }
    }

    pub fn msg(&self) -> String {
        match self {
            Error::FVMSharedAddress(e) => format!("{:?}", e),
            Error::AlreadyVoted(a) => format!("actor {:?} already voted", a),
            _ => String::from("")
        }
    }
}

impl From<fvm_ipld_hamt::Error> for Error {
    fn from(e: fvm_ipld_hamt::Error) -> Self {
        Error::FVMIpldHamt(e)
    }
}

impl From<anyhow::Error> for Error {
    fn from(e: anyhow::Error) -> Self {
        Error::Anyhow(e)
    }
}

impl From<fvm_shared::error::ErrorNumber> for Error {
    fn from(e: fvm_shared::error::ErrorNumber) -> Self {
        Error::FVMSharedErrorNum(e)
    }
}

impl From<fvm_sdk::error::NoStateError> for Error {
    fn from(e: fvm_sdk::error::NoStateError) -> Self {
        Error::FVMSDKNoState(e)
    }
}

impl From<fvm_ipld_encoding::Error> for Error {
    fn from(e: fvm_ipld_encoding::Error) -> Self {
        Error::FVMEncoding(e)
    }
}

impl From<fvm_shared::address::Error> for Error {
    fn from(e: fvm_shared::address::Error) -> Self {
        Error::FVMSharedAddress(e)
    }
}
