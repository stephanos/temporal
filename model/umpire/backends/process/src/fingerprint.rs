#![forbid(unsafe_code)]
use sha2::{Digest, Sha256};
use std::hash::{Hash, Hasher};

struct StructuralSha(Sha256);
impl Hasher for StructuralSha {
    fn finish(&self) -> u64 {
        let digest = self.0.clone().finalize();
        let mut prefix = [0; 8];
        prefix.copy_from_slice(&digest[..8]);
        u64::from_le_bytes(prefix)
    }
    fn write(&mut self, bytes: &[u8]) {
        self.0.update(bytes);
    }
}

/// SHA-256 of the structural `Hash` encoding, domain separated for disposable cache identity.
#[must_use]
pub fn structural_sha256(value: &impl Hash) -> String {
    let mut hasher = StructuralSha(Sha256::new());
    hasher.write(b"umpire-structural-sha256-v1\0");
    value.hash(&mut hasher);
    hex(&hasher.0.finalize())
}

/// SHA-256 of exact artifact bytes.
#[must_use]
pub fn bytes_sha256(bytes: &[u8]) -> String {
    hex(&Sha256::digest(bytes))
}

fn hex(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut text = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        text.push(char::from(HEX[usize::from(byte >> 4)]));
        text.push(char::from(HEX[usize::from(byte & 15)]));
    }
    text
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn full_sha_and_structural_framing() {
        assert_eq!(
            bytes_sha256(b"abc"),
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
        assert_ne!(structural_sha256(&("a", "bc")), structural_sha256(&("ab", "c")));
        assert_eq!(structural_sha256(&"stable").len(), 64);
    }
}
