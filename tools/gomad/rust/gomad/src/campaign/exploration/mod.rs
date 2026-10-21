pub(crate) mod choice;
#[cfg(feature = "determinism-fixture")]
pub(crate) mod choice_corpus;
pub(crate) mod plan;
pub(crate) mod round;
pub(crate) mod simulation;
#[cfg(feature = "determinism-fixture")]
pub(crate) mod simulation_corpus;
pub(crate) mod simulation_record;
pub(crate) mod simulation_record_wire;
pub(crate) mod site_scope;

use crate::evidence::canonical;
use serde::Serialize;
use sha2::{Digest as _, Sha256};

pub(crate) fn domain_hash(domain: &str, bytes: &[u8]) -> String {
    let mut hash = Sha256::new();
    hash.update(domain.as_bytes());
    hash.update([0]);
    hash.update(bytes);
    format!("sha256:{:x}", hash.finalize())
}

pub(crate) fn hash<T: Serialize>(domain: &str, value: &T) -> Result<String, String> {
    Ok(domain_hash(domain, &canonical::encode(value)?))
}

pub(crate) fn valid_hash(value: &str) -> bool {
    value
        .strip_prefix("sha256:")
        .is_some_and(canonical::valid_digest)
}

pub(crate) fn base64(bytes: &[u8]) -> String {
    const ALPHABET: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut result = String::with_capacity(bytes.len().div_ceil(3) * 4);
    for chunk in bytes.chunks(3) {
        let word = u32::from(chunk[0]) << 16
            | u32::from(*chunk.get(1).unwrap_or(&0)) << 8
            | u32::from(*chunk.get(2).unwrap_or(&0));
        for shift in [18, 12] {
            result.push(char::from(ALPHABET[((word >> shift) & 63) as usize]));
        }
        result.push(if chunk.len() > 1 {
            char::from(ALPHABET[((word >> 6) & 63) as usize])
        } else {
            '='
        });
        result.push(if chunk.len() > 2 {
            char::from(ALPHABET[(word & 63) as usize])
        } else {
            '='
        });
    }
    result
}
