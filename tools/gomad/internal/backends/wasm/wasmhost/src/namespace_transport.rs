use crate::{namespace::Namespace, namespace_image, protocol};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::io::{self, BufRead, Read};
use wasmtime::{Error, Result};

const CHUNK: u64 = 64 * 1024;

#[derive(Debug, Deserialize)]
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
    let image = namespace_image::decode(
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
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct End {
        #[serde(rename = "type")]
        kind: String,
    }
    let end: End = serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
    if end.kind != "namespace_inputs_end" {
        return Err(Error::msg("invalid namespace inputs end"));
    }
    Namespace::new(image).map_err(|_| Error::msg("invalid namespace image"))
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
        if buffer.is_empty() {
            return Ok(0);
        }
        if self.offset == self.chunk.len() {
            if self.sequence == self.descriptor.chunks {
                return Ok(0);
            }
            #[derive(Deserialize)]
            #[serde(deny_unknown_fields)]
            struct Chunk {
                #[serde(rename = "type")]
                kind: String,
                sequence: u64,
                data_base64: String,
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
    const ROOT_SHA256: &str = "8a5edab282632443219e051e4ade2d1d5bbc671c781051bf1437897cbdfea0f1";

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
    fn shared_go_literal_preserves_full_reference_state_digest() {
        let mut bytes = ROOT_IMAGE.to_vec();
        for (offset, value) in [(8, 10_u64), (24, 16), (32, 32), (40, 9)] {
            bytes[offset..offset + 8].copy_from_slice(&value.to_le_bytes());
        }
        bytes.truncate(bytes.len() - 10);
        bytes.extend_from_slice(&2_u64.to_le_bytes());
        bytes.extend_from_slice(b"hi\0");
        let namespace = read(
            &mut wire(&frames(&bytes)),
            &descriptor(&bytes),
            32,
            9,
            ROOT_SHA256,
        )
        .unwrap();
        assert_eq!(
            format!("{:x}", Sha256::digest(namespace.canonical_state())),
            "8dfc19c04a503ba4bbee0eb717cac7cfff6ecd0c0436a8d2d71b2fba93632d67"
        );
    }

    #[test]
    fn validates_descriptor_without_allocating_declared_bytes() {
        assert!(descriptor(ROOT_IMAGE).validate().is_ok());
        for (field, value) in [
            ("chunks", json!(0)),
            ("chunks", json!(2)),
            ("files", json!(0)),
            ("files", json!((1 << 20) + 1)),
            ("files", json!(u64::MAX)),
            ("bytes", json!(97)),
            ("sha256", json!("a")),
            ("sha256", json!("A".repeat(64))),
        ] {
            let d = descriptor(ROOT_IMAGE);
            let mut value_json =
                json!({"chunks":d.chunks, "files":d.files, "bytes":d.bytes, "sha256":d.sha256});
            value_json[field] = value;
            let d: NamespaceDescriptor = serde_json::from_value(value_json).unwrap();
            assert!(d.validate().is_err(), "field {field}");
        }
        let mut large = descriptor(ROOT_IMAGE);
        large.bytes = u64::MAX;
        large.chunks = large.bytes.div_ceil(65536);
        assert!(large.validate().is_ok());
    }

    #[test]
    fn builds_namespace_from_validated_frames_and_consumes_only_its_end() {
        let mut input = wire(&frames(ROOT_IMAGE));
        input.get_mut().extend_from_slice(b"next frame\n");
        let mut ns = read(&mut input, &descriptor(ROOT_IMAGE), 3, 4, ROOT_SHA256).unwrap();
        let reply = ns
            .execute(
                crate::namespace::Operation::Read {
                    fd: 0,
                    length: 4,
                    offset: None,
                },
                0,
            )
            .unwrap();
        assert_eq!(
            reply.output,
            Some(crate::namespace::Output::Bytes(b"x".to_vec()))
        );
        assert_eq!(protocol::read_frame(&mut input).unwrap(), b"next frame\n");
    }

    #[test]
    fn rejects_digest_cwd_and_image_identity_mismatches() {
        let mut d = descriptor(ROOT_IMAGE);
        d.sha256 = "0".repeat(64);
        assert!(read(&mut wire(&frames(ROOT_IMAGE)), &d, 3, 4, ROOT_SHA256).is_err());
        for (files, output, epoch, cwd) in [
            (2, 3, 4, ROOT_SHA256),
            (1, 4, 4, ROOT_SHA256),
            (1, 3, 5, ROOT_SHA256),
            (1, 3, 4, "wrong"),
        ] {
            let mut d = descriptor(ROOT_IMAGE);
            d.files = files;
            assert!(read(&mut wire(&frames(ROOT_IMAGE)), &d, output, epoch, cwd).is_err());
        }
    }

    #[test]
    fn rejects_chunk_shape_sequence_encoding_and_length() {
        let cases: &[fn(&mut Vec<Value>)] = &[
            |f| f[0]["type"] = json!("tape"),
            |f| f[0]["sequence"] = json!(1),
            |f| f[0]["extra"] = json!(1),
            |f| f[0]["data_base64"] = json!("!"),
            |f| f[0]["data_base64"] = json!(STANDARD.encode(&ROOT_IMAGE[..ROOT_IMAGE.len() - 1])),
            |f| f[0]["data_base64"] = json!(STANDARD.encode([ROOT_IMAGE, b"x"].concat())),
            |f| f[0]["data_base64"] = json!("A".repeat(87385)),
            |f| f[1]["type"] = json!("other"),
            |f| f[1]["extra"] = json!(0),
            |f| {
                f.remove(1);
            },
            |f| {
                f.insert(1, f[0].clone());
            },
        ];
        for (index, change) in cases.iter().enumerate() {
            let mut f = frames(ROOT_IMAGE);
            change(&mut f);
            assert!(
                read(&mut wire(&f), &descriptor(ROOT_IMAGE), 3, 4, ROOT_SHA256).is_err(),
                "case {index}"
            );
        }
    }

    #[test]
    fn rejects_unknown_or_duplicate_descriptor_fields() {
        for json in [
            r#"{"chunks":1,"files":1,"bytes":100,"sha256":"x","extra":1}"#,
            r#"{"chunks":1,"files":1,"files":2,"bytes":100,"sha256":"x"}"#,
        ] {
            assert!(serde_json::from_str::<NamespaceDescriptor>(json).is_err());
        }
    }

    #[test]
    fn streams_multiple_chunks_and_rejects_short_nonfinal_chunk() {
        let data = vec![7; 2 * 65536 + 3];
        let mut bytes = ROOT_IMAGE[..89].to_vec();
        bytes[24..32].copy_from_slice(&(data.len() as u64).to_le_bytes());
        bytes.extend_from_slice(&(data.len() as u64).to_le_bytes());
        bytes.extend_from_slice(&data);
        bytes.push(0);
        let d = descriptor(&bytes);
        assert_eq!(d.chunks, 3);
        let mut ns = read(&mut wire(&frames(&bytes)), &d, 3, 4, ROOT_SHA256).unwrap();
        let reply = ns
            .execute(
                crate::namespace::Operation::Read {
                    fd: 0,
                    length: data.len() as u32,
                    offset: None,
                },
                0,
            )
            .unwrap();
        assert_eq!(reply.output, Some(crate::namespace::Output::Bytes(data)));
        let mut f = frames(&bytes);
        f[0]["data_base64"] = json!(STANDARD.encode(&bytes[..65535]));
        assert!(read(&mut wire(&f), &d, 3, 4, ROOT_SHA256).is_err());
        let mut f = frames(&bytes);
        f[1]["sequence"] = json!(0);
        assert!(read(&mut wire(&f), &d, 3, 4, ROOT_SHA256).is_err());
    }

    #[test]
    fn rejects_duplicate_chunk_and_end_fields() {
        let encoded = STANDARD.encode(ROOT_IMAGE);
        for text in [
            format!(
                "{{\"type\":\"namespace_input\",\"sequence\":0,\"sequence\":0,\"data_base64\":\"{encoded}\"}}\n{{\"type\":\"namespace_inputs_end\"}}\n"
            ),
            format!(
                "{{\"type\":\"namespace_input\",\"sequence\":0,\"data_base64\":\"{encoded}\"}}\n{{\"type\":\"namespace_inputs_end\",\"type\":\"namespace_inputs_end\"}}\n"
            ),
        ] {
            assert!(
                read(
                    &mut Cursor::new(text),
                    &descriptor(ROOT_IMAGE),
                    3,
                    4,
                    ROOT_SHA256
                )
                .is_err()
            );
        }
    }

    #[test]
    fn rejects_declared_trailing_image_bytes_even_with_matching_digest() {
        let bytes = [ROOT_IMAGE, b"trailing"].concat();
        assert!(
            read(
                &mut wire(&frames(&bytes)),
                &descriptor(&bytes),
                3,
                4,
                ROOT_SHA256
            )
            .is_err()
        );
        let mut d = descriptor(ROOT_IMAGE);
        d.bytes += 1;
        assert!(read(&mut wire(&frames(ROOT_IMAGE)), &d, 3, 4, ROOT_SHA256).is_err());
    }
}
