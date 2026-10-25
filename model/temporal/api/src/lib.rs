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
