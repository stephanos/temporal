//! Generated upstream Temporal API messages and the three extension envelopes.

#![forbid(unsafe_code)]

#[allow(missing_docs, unreachable_pub, deprecated, clippy::all, clippy::pedantic)]
mod generated {
    include!(concat!(env!("OUT_DIR"), "/protocol.rs"));
    include!(concat!(env!("OUT_DIR"), "/services.rs"));
    include!(concat!(env!("OUT_DIR"), "/wire_schema.rs"));
    include!(concat!(env!("OUT_DIR"), "/claims.rs"));
    include!(concat!(env!("OUT_DIR"), "/mirrors.rs"));
}

pub use generated::*;

/// The exact linked descriptor input used to generate these API catalogs.
pub const RELATION_DESCRIPTOR_BYTES: &[u8] =
    include_bytes!(concat!(env!("OUT_DIR"), "/annotation-api.binpb"));
