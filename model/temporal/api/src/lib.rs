//! Generated upstream Temporal API messages and the three extension envelopes.

#[allow(missing_docs, unreachable_pub, clippy::all, clippy::pedantic)]
mod generated {
    include!(concat!(env!("OUT_DIR"), "/protocol.rs"));
    include!(concat!(env!("OUT_DIR"), "/services.rs"));
}

pub use generated::*;
