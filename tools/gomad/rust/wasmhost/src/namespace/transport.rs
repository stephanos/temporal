use super::{Namespace, image};
use crate::protocol;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::io::{self, BufRead, Read};
use wasmtime::{Error, Result};

const CHUNK: u64 = 64 * 1024;

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct NamespaceDescriptor {
    pub chunks: u64,
    pub files: u64,
    pub bytes: u64,
    pub sha256: String,
}

impl NamespaceDescriptor {
    pub fn validate(&self) -> Result<()> {
        let minimum = self
            .files
            .checked_sub(1)
            .and_then(|entries| entries.checked_mul(20))
            .and_then(|entries| entries.checked_add(98));
        if !(1..=1 << 20).contains(&self.files)
            || minimum.is_none_or(|minimum| self.bytes < minimum)
            || self.chunks != self.bytes.div_ceil(CHUNK)
            || !valid_sha256(&self.sha256)
        {
            return Err(Error::msg("invalid namespace descriptor"));
        }
        Ok(())
    }
}

pub fn read(
    input: &mut dyn BufRead,
    descriptor: &NamespaceDescriptor,
    expected_output: u64,
    expected_epoch: u64,
    working_directory_sha256: &str,
) -> Result<Namespace> {
    Namespace::new(read_image(
        input,
        descriptor,
        expected_output,
        expected_epoch,
        working_directory_sha256,
    )?)
    .map_err(|_| Error::msg("invalid namespace image"))
}

pub(crate) fn read_image(
    input: &mut dyn BufRead,
    descriptor: &NamespaceDescriptor,
    expected_output: u64,
    expected_epoch: u64,
    working_directory_sha256: &str,
) -> Result<crate::namespace::Image> {
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct End {
        #[serde(rename = "type")]
        kind: String,
    }
    descriptor.validate()?;
    if !valid_sha256(working_directory_sha256) {
        return Err(Error::msg("invalid namespace working directory digest"));
    }
    let mut reader = ImageFrames {
        input,
        descriptor,
        sequence: 0,
        bytes: 0,
        chunk: Vec::new(),
        offset: 0,
        hash: Sha256::new(),
    };
    let image = image::decode(
        &mut reader,
        descriptor.files,
        expected_output,
        expected_epoch,
    )
    .map_err(|_| Error::msg("invalid namespace image"))?;
    if reader.bytes != descriptor.bytes
        || reader.sequence != descriptor.chunks
        || format!("{:x}", reader.hash.finalize()) != descriptor.sha256
        || format!("{:x}", Sha256::digest(&image.working_directory)) != working_directory_sha256
    {
        return Err(Error::msg("invalid namespace image digest or length"));
    }
    let end: End = serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
    if end.kind != "namespace_inputs_end" {
        return Err(Error::msg("invalid namespace inputs end"));
    }
    Ok(image)
}

fn valid_sha256(digest: &str) -> bool {
    digest.len() == 64
        && digest
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}

struct ImageFrames<'a> {
    input: &'a mut dyn BufRead,
    descriptor: &'a NamespaceDescriptor,
    sequence: u64,
    bytes: u64,
    chunk: Vec<u8>,
    offset: usize,
    hash: Sha256,
}

impl Read for ImageFrames<'_> {
    fn read(&mut self, buffer: &mut [u8]) -> io::Result<usize> {
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Chunk {
            #[serde(rename = "type")]
            kind: String,
            sequence: u64,
            data_base64: String,
        }
        if buffer.is_empty() {
            return Ok(0);
        }
        if self.offset == self.chunk.len() {
            if self.sequence == self.descriptor.chunks {
                return Ok(0);
            }
            let invalid =
                || io::Error::new(io::ErrorKind::InvalidData, "invalid namespace input chunk");
            let frame = protocol::read_frame(self.input).map_err(|_| invalid())?;
            let chunk: Chunk = serde_json::from_slice(&frame).map_err(|_| invalid())?;
            if chunk.kind != "namespace_input"
                || chunk.sequence != self.sequence
                || chunk.data_base64.len() as u64 > CHUNK.div_ceil(3) * 4
            {
                return Err(invalid());
            }
            let bytes = STANDARD.decode(chunk.data_base64).map_err(|_| invalid())?;
            let remaining = self
                .descriptor
                .bytes
                .checked_sub(self.bytes)
                .ok_or_else(invalid)?;
            if bytes.len() as u64 != remaining.min(CHUNK) {
                return Err(invalid());
            }
            self.bytes = self
                .bytes
                .checked_add(bytes.len() as u64)
                .ok_or_else(invalid)?;
            self.sequence = self.sequence.checked_add(1).ok_or_else(invalid)?;
            self.hash.update(&bytes);
            self.chunk = bytes;
            self.offset = 0;
        }
        let length = buffer.len().min(self.chunk.len() - self.offset);
        buffer[..length].copy_from_slice(&self.chunk[self.offset..self.offset + length]);
        self.offset += length;
        Ok(length)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::{Value, json};
    use sha2::{Digest, Sha256};
    use std::io::Cursor;

    const ROOT_IMAGE: &[u8] = b"GMWN\x01\0\0\0\
        \x01\0\0\0\0\0\0\0\x05\0\0\0\0\0\0\0\
        \x02\0\0\0\0\0\0\0\x03\0\0\0\0\0\0\0\
        \x04\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\
        \0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\
        \0\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0/\
        \x01\0\0\0\0\0\0\0x\0";

    fn descriptor(bytes: &[u8]) -> NamespaceDescriptor {
        NamespaceDescriptor {
            chunks: (bytes.len() as u64).div_ceil(65536),
            files: 1,
            bytes: bytes.len() as u64,
            sha256: format!("{:x}", Sha256::digest(bytes)),
        }
    }

    fn frames(bytes: &[u8]) -> Vec<Value> {
        let mut frames: Vec<_> = bytes.chunks(65536).enumerate().map(|(sequence, bytes)| {
            json!({"type":"namespace_input", "sequence":sequence, "data_base64":STANDARD.encode(bytes)})
        }).collect();
        frames.push(json!({"type":"namespace_inputs_end"}));
        frames
    }

    fn wire(frames: &[Value]) -> Cursor<Vec<u8>> {
        let mut bytes = Vec::new();
        for frame in frames {
            protocol::send_frame(&mut bytes, frame).unwrap();
        }
        Cursor::new(bytes)
    }

    #[test]
    fn descriptor_validation_does_not_allocate_declared_bytes() {
        let mut large = descriptor(ROOT_IMAGE);
        large.bytes = u64::MAX;
        large.chunks = large.bytes.div_ceil(65536);
        assert!(large.validate().is_ok());
    }

    #[test]
    fn read_image_rejects_malformed_working_directory_digest() {
        assert!(
            read(
                &mut wire(&frames(ROOT_IMAGE)),
                &descriptor(ROOT_IMAGE),
                3,
                4,
                "wrong"
            )
            .is_err()
        );
    }
}
