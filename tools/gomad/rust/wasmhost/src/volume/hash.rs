use sha2::{Digest, Sha256};
use std::ops::{Deref, DerefMut};
use std::sync::OnceLock;

#[derive(Clone, Debug, Default)]
pub(crate) struct FileBytes {
    bytes: Vec<u8>,
    digest: OnceLock<String>,
}

impl FileBytes {
    pub(super) fn digest(&self) -> &str {
        self.digest.get_or_init(|| bytes_digest(&self.bytes))
    }
}

impl From<Vec<u8>> for FileBytes {
    fn from(bytes: Vec<u8>) -> Self {
        Self {
            bytes,
            digest: OnceLock::new(),
        }
    }
}

impl Deref for FileBytes {
    type Target = Vec<u8>;

    fn deref(&self) -> &Self::Target {
        &self.bytes
    }
}

impl DerefMut for FileBytes {
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.digest.take();
        &mut self.bytes
    }
}

impl AsRef<[u8]> for FileBytes {
    fn as_ref(&self) -> &[u8] {
        &self.bytes
    }
}

impl<T: AsRef<[u8]>> PartialEq<T> for FileBytes {
    fn eq(&self, other: &T) -> bool {
        self.bytes == other.as_ref()
    }
}

impl Eq for FileBytes {}

pub(super) struct Hasher(Sha256);

impl Hasher {
    pub(super) fn new() -> Self {
        Self(Sha256::new())
    }

    pub(super) fn write(&mut self, value: &[u8]) {
        self.0.update(value);
    }

    pub(super) fn bytes(&mut self, value: &[u8]) {
        self.u64(value.len() as u64);
        self.write(value);
    }

    pub(super) fn string(&mut self, value: &str) {
        self.bytes(value.as_bytes());
    }

    pub(super) fn u64(&mut self, value: u64) {
        self.write(&value.to_le_bytes());
    }

    pub(super) fn finish(&self) -> String {
        format!("sha256:{:x}", self.0.clone().finalize())
    }
}

pub(super) fn bytes_digest(value: &[u8]) -> String {
    #[cfg(test)]
    CONTENT_HASHES.with(|count| count.set(count.get() + 1));
    let mut hasher = Hasher::new();
    hasher.write(value);
    hasher.finish()
}

#[cfg(test)]
thread_local! {
    pub(super) static CONTENT_HASHES: std::cell::Cell<usize> = const { std::cell::Cell::new(0) };
}

#[cfg(test)]
mod tests {
    use super::{CONTENT_HASHES, FileBytes, Hasher, bytes_digest};

    #[test]
    fn file_byte_mutable_borrows_invalidate_before_edit_and_clones_keep_independent_caches() {
        let mut bytes = FileBytes::from(b"abc".to_vec());
        let cold = bytes.clone();
        let abc = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad";
        assert_eq!(bytes.digest(), abc);
        assert_eq!(bytes, cold);
        CONTENT_HASHES.with(|count| count.set(0));
        let mut copy = bytes.clone();
        assert_eq!(copy.digest(), abc);
        assert_eq!(CONTENT_HASHES.with(std::cell::Cell::get), 0);
        let raw: &mut Vec<u8> = &mut copy;
        raw[0] = b'z';
        assert_ne!(copy.digest(), abc);
        assert_eq!(bytes.digest(), abc);
        assert_eq!(CONTENT_HASHES.with(std::cell::Cell::get), 1);
        bytes.clear();
        assert_eq!(bytes.digest(), bytes_digest(b""));
        bytes.extend_from_slice(b"abc");
        assert_eq!(bytes.digest(), abc);
    }

    #[test]
    fn raw_writes_preserve_stream_across_nonconsuming_finish() {
        let mut hasher = Hasher::new();
        assert_eq!(
            hasher.finish(),
            "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        );
        hasher.write(b"a");
        hasher.write(b"");
        hasher.write(b"b");
        hasher.write(b"c");
        assert_eq!(
            hasher.finish(),
            "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
        assert_eq!(
            hasher.finish(),
            "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
        );
    }

    #[test]
    fn integers_use_eight_little_endian_bytes() {
        for (value, expected) in [
            (
                0,
                "sha256:af5570f5a1810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc",
            ),
            (
                0x0807_0605_0403_0201,
                "sha256:66840dda154e8a113c31dd0ad32f7f3a366a80e8136979d8f5a101d3d29d6f72",
            ),
            (
                u64::MAX,
                "sha256:12a3ae445661ce5dee78d0650d33362dec29c4f82af05e7e57fb595bbbacf0ca",
            ),
        ] {
            let mut hasher = Hasher::new();
            hasher.u64(value);
            assert_eq!(hasher.finish(), expected);
        }
    }

    #[test]
    fn bytes_prefix_the_eight_byte_length() {
        for (value, expected) in [
            (
                &b""[..],
                "sha256:af5570f5a1810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc",
            ),
            (
                &b"abc"[..],
                "sha256:ce91dc5eec0139adf091900d225971d6ad246a845bad791b5693a9d0d55dd391",
            ),
            (
                &b"\0\xff"[..],
                "sha256:f0160f0e6102feb69a19adad7b99f4be67a486740b7fd9a1bdc79d0817d25874",
            ),
            (
                &[b'a'; 256][..],
                "sha256:2f7a74cced461bd4aff35aea9b77fe965a35d053cbc22a923262ce2f4ee396f0",
            ),
        ] {
            let mut hasher = Hasher::new();
            hasher.bytes(value);
            assert_eq!(hasher.finish(), expected);
        }
    }

    #[test]
    fn strings_frame_utf8_byte_length_including_nul() {
        let mut hasher = Hasher::new();
        hasher.string("é雪\0");
        assert_eq!(
            hasher.finish(),
            "sha256:de1b9114b3667904e6bfda3a7dfa7e321aa99cec31bfaeb7005c2040dcb9511a"
        );
    }

    #[test]
    fn field_boundaries_and_order_change_the_digest() {
        for (first, second, expected) in [
            (
                "ab",
                "c",
                "sha256:43ee655579de01ca739b3f95c1c2d3f46d353b2c0df818064ea594506cdb2617",
            ),
            (
                "a",
                "bc",
                "sha256:9a8acca1b6c6c0befd3fbc756aed625da998c998f7252e738c4ef061906b9b21",
            ),
            (
                "c",
                "ab",
                "sha256:023b16f698511f3efdb0a9905c1030cda37ffcc52fe202d9ef0c17b39fb20d1b",
            ),
        ] {
            let mut hasher = Hasher::new();
            hasher.string(first);
            hasher.string(second);
            assert_eq!(hasher.finish(), expected);
        }
    }
}
