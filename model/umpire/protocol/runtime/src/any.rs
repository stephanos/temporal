use prost::Name;

use crate::{prost_types::Any, protocol::Refusal};

/// Canonical native Any encoding and strict realization admission.
pub trait AnyMessage: Sized {
    /// Encode a message using its generated canonical type URL.
    fn pack<T: Name>(value: &T) -> Self;
    /// Admit the exact canonical type URL before decoding a message.
    fn unpack<T: Name + Default>(&self) -> Result<T, Refusal>;
    /// Require an explicitly present payload.
    fn required(value: Option<Self>) -> Result<Self, Refusal>;
}

impl AnyMessage for Any {
    fn pack<T: Name>(value: &T) -> Self {
        Self::from_msg(value)
            .unwrap_or_else(|error| panic!("encoding Any into Vec failed: {error}"))
    }

    fn unpack<T: Name + Default>(&self) -> Result<T, Refusal> {
        if self.type_url.is_empty() {
            return Err(Refusal::malformed("Any type URL is missing"));
        }
        if self.type_url != T::type_url() {
            return Err(Refusal::unsupported("unsupported Any type URL"));
        }
        self.to_msg().map_err(|error| Refusal::malformed(error.to_string()))
    }

    fn required(value: Option<Self>) -> Result<Self, Refusal> {
        value.ok_or_else(|| Refusal::malformed("required Any payload is missing"))
    }
}
