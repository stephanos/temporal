use prost::Name;

use crate::{google::protobuf::Any, protocol::Refusal};

impl Any {
    /// Encode a message using its generated canonical type URL.
    pub fn pack<T: Name>(value: &T) -> Self {
        Self { type_url: T::type_url(), value: value.encode_to_vec() }
    }

    /// Admit the exact canonical type URL before decoding a message.
    pub fn unpack<T: Name + Default>(&self) -> Result<T, Refusal> {
        if self.type_url.is_empty() {
            return Err(Refusal::malformed("Any type URL is missing"));
        }
        if self.type_url != T::type_url() {
            return Err(Refusal::unsupported("unsupported Any type URL"));
        }
        T::decode(self.value.as_slice()).map_err(|error| Refusal::malformed(error.to_string()))
    }

    /// Require an explicitly present payload.
    pub fn required(value: Option<Self>) -> Result<Self, Refusal> {
        value.ok_or_else(|| Refusal::malformed("required Any payload is missing"))
    }
}
