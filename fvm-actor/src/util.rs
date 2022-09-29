use crate::Error;

pub fn parse_params_or_abort<'a, T: serde::Deserialize<'a>>(params: &'a [u8]) -> T {
    match serde_json::from_slice::<T>(params) {
        Ok(t) => t,
        Err(e) => fvm_sdk::vm::abort(
            Error::CannotDeserialize.code(),
            Option::from(e.to_string().as_str()),
        ),
    }
}

#[macro_export]
macro_rules! ensure {
    ( $x:expr, $y:expr ) => {{
        if !$x {
            Err($y)
        } else {
            Ok(())
        }
    }};
}
