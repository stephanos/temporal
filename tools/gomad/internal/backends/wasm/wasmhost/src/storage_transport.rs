//! Helper-owned storage: the execute request's storage descriptor, recovery
//! image frames, and the terminal storage evidence streams.

use crate::namespace::{Namespace, Stop};
use crate::protocol;
use crate::volume::{self, CrashEnumeration, CrashEnumerationLimits, StorageModel};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::io::BufRead;
use wasmtime::{Error, Result};

const CHUNK: u64 = 64 * 1024;
/// Recovery images and evidence carry history and metadata beyond the volume
/// capacities; this bound is added to the filesystem byte limit.
pub const IMAGE_METADATA_BYTES: u64 = 64 << 20;
pub const STATE_MAGIC: &[u8; 8] = b"GMSTORE\x01";
const STATE_VERSION: u32 = 1;

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ImageDescriptor {
    pub chunks: u64,
    pub bytes: u64,
    pub sha256: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StorageDescriptor {
    pub model: StorageModel,
    pub model_identity: String,
    #[serde(default)]
    pub crash_limits: Option<CrashEnumerationLimits>,
    #[serde(default)]
    pub crash_selection: Option<BTreeMap<String, Option<Vec<u64>>>>,
    pub return_image: bool,
    #[serde(default)]
    pub image: Option<ImageDescriptor>,
}

impl StorageDescriptor {
    pub fn validate(&self) -> Result<()> {
        if self.model.identity() != self.model_identity {
            return Err(Error::msg("storage model identity mismatch"));
        }
        if let Some(image) = &self.image
            && (image.bytes == 0
                || image.chunks != image.bytes.div_ceil(CHUNK)
                || image.sha256.len() != 64
                || !image
                    .sha256
                    .bytes()
                    .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte)))
        {
            return Err(Error::msg("invalid storage image descriptor"));
        }
        Ok(())
    }

    pub fn image_limit(filesystem_bytes: u64) -> u64 {
        filesystem_bytes + IMAGE_METADATA_BYTES
    }

    /// Reads the declared recovery image frames after the namespace image.
    pub fn read_image(&self, input: &mut dyn BufRead, maximum: u64) -> Result<Option<Vec<u8>>> {
        let Some(declared) = &self.image else {
            return Ok(None);
        };
        if declared.bytes > maximum {
            return Err(Error::new(crate::namespace::Boundary {
                kind: "capacity",
                dimension: "storage-image-bytes",
            }));
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct Chunk {
            #[serde(rename = "type")]
            kind: String,
            sequence: u64,
            data_base64: String,
        }
        let mut bytes = Vec::new();
        for sequence in 0..declared.chunks {
            let chunk: Chunk =
                serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
            if chunk.kind != "storage_image_input"
                || chunk.sequence != sequence
                || chunk.data_base64.len() as u64 > CHUNK.div_ceil(3) * 4
            {
                return Err(Error::msg("invalid storage image chunk"));
            }
            let data = STANDARD.decode(chunk.data_base64).map_err(Error::new)?;
            if data.len() as u64 != CHUNK.min(declared.bytes - bytes.len() as u64) {
                return Err(Error::msg("invalid storage image chunk length"));
            }
            bytes.extend(data);
        }
        #[derive(Deserialize)]
        #[serde(deny_unknown_fields)]
        struct End {
            #[serde(rename = "type")]
            kind: String,
        }
        let end: End = serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
        if end.kind != "storage_image_inputs_end" {
            return Err(Error::msg("invalid storage image end"));
        }
        if format!("{:x}", Sha256::digest(&bytes)) != declared.sha256 {
            return Err(Error::msg("storage image digest mismatch"));
        }
        Ok(Some(bytes))
    }

    /// Validates the model against the namespace configuration and attaches a
    /// fresh or recovered volume model to it.
    pub fn attach(
        &self,
        namespace: &mut Namespace,
        image: Option<&[u8]>,
        transcript_limit: u64,
    ) -> std::result::Result<(), Stop> {
        let filesystem_bytes = namespace.filesystem_bytes();
        self.model
            .validate(namespace.writable_directories(), filesystem_bytes)
            .map_err(|dimension| {
                Stop::Boundary(crate::namespace::Boundary {
                    kind: "invalid",
                    dimension,
                })
            })?;
        let storage = match (image, &self.image) {
            (Some(bytes), Some(declared)) => volume::Storage::recover(
                self.model.clone(),
                transcript_limit,
                bytes,
                &declared.sha256,
                Self::image_limit(filesystem_bytes),
            ),
            _ => volume::Storage::new(self.model.clone(), transcript_limit),
        }
        .map_err(|stop| match stop {
            volume::StorageStop::Boundary(kind, dimension) => {
                Stop::Boundary(crate::namespace::Boundary { kind, dimension })
            }
            volume::StorageStop::Crash(fault) => Stop::StorageCrash(fault),
            volume::StorageStop::Fatal(message) => Stop::Fatal(message),
        })?;
        namespace.attach_storage(storage)
    }

    fn selections(&self) -> BTreeMap<String, Vec<u64>> {
        self.crash_selection
            .clone()
            .unwrap_or_default()
            .into_iter()
            .map(|(volume, operations)| (volume, operations.unwrap_or_default()))
            .collect()
    }
}

/// The terminal storage evidence of one incarnation.
pub struct Evidence {
    pub log: Vec<u8>,
    pub state: Vec<u8>,
    pub image: Option<Vec<u8>>,
}

/// Go `Run`'s post-execution storage collection: transcript, snapshots and
/// operations, then each configured volume's enumeration in model order, then
/// the crash image. An enumeration error is `invalid`; a crash error is
/// `infrastructure`.
pub fn evidence(
    namespace: &mut Namespace,
    descriptor: &StorageDescriptor,
) -> std::result::Result<Evidence, (&'static str, String)> {
    let filesystem_bytes = namespace.filesystem_bytes();
    let storage = namespace.storage().expect("attached storage");
    let log = storage.log.clone();
    let snapshots = serde_json::to_vec(&storage.snapshots()).expect("in-memory JSON");
    let operations = serde_json::to_vec(&storage.operations()).expect("in-memory JSON");
    let enumerations = match &descriptor.crash_limits {
        None => b"null".to_vec(),
        Some(limits) => {
            let mut pages: BTreeMap<String, CrashEnumeration> = BTreeMap::new();
            for config in descriptor.model.volumes() {
                let page = storage
                    .enumerate(&config.id, *limits, None)
                    .map_err(|error| ("invalid", volume::storage_error_text(&error)))?;
                pages.insert(config.id.clone(), page);
            }
            serde_json::to_vec(&pages).expect("in-memory JSON")
        }
    };
    let image = namespace
        .crash_storage(
            &descriptor.selections(),
            StorageDescriptor::image_limit(filesystem_bytes),
        )
        .map_err(|error| ("infrastructure", volume::storage_error_text(&error)))?;
    let state = encode_state(&snapshots, &operations, &enumerations, &image.digest);
    Ok(Evidence {
        log,
        state,
        image: descriptor.return_image.then_some(image.bytes),
    })
}

/// Storage-state envelope v1: magic, version, four tagged length-prefixed
/// sections (snapshots JSON, operations JSON, enumerations JSON or `null`,
/// image identity), then the u64 length and SHA-256 of all preceding bytes.
pub fn encode_state(
    snapshots: &[u8],
    operations: &[u8],
    enumerations: &[u8],
    image_identity: &str,
) -> Vec<u8> {
    let mut bytes = STATE_MAGIC.to_vec();
    bytes.extend_from_slice(&STATE_VERSION.to_le_bytes());
    bytes.extend_from_slice(&4u32.to_le_bytes());
    for (tag, section) in [
        snapshots,
        operations,
        enumerations,
        image_identity.as_bytes(),
    ]
    .iter()
    .enumerate()
    {
        bytes.extend_from_slice(&(tag as u32 + 1).to_le_bytes());
        bytes.extend_from_slice(&(section.len() as u64).to_le_bytes());
        bytes.extend_from_slice(section);
    }
    let length = bytes.len() as u64;
    let digest = Sha256::digest(&bytes);
    bytes.extend_from_slice(&length.to_le_bytes());
    bytes.extend_from_slice(&digest);
    bytes
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn state_envelope_layout_and_trailer() {
        let bytes = encode_state(b"[]", b"{}", b"null", &"a".repeat(64));
        assert_eq!(&bytes[..8], STATE_MAGIC);
        assert_eq!(&bytes[8..16], &[1, 0, 0, 0, 4, 0, 0, 0]);
        let body = bytes.len() - 40;
        assert_eq!(
            u64::from_le_bytes(bytes[body..body + 8].try_into().unwrap()),
            body as u64
        );
        assert_eq!(&bytes[body + 8..], Sha256::digest(&bytes[..body]).as_slice());
    }

    #[test]
    fn descriptor_requires_go_identity_and_framed_image() {
        let model = r#"{"volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1024}],"limits":{"PendingOperations":10,"Transitions":10}}"#;
        let parsed: StorageModel = serde_json::from_str(model).unwrap();
        let ok = format!(
            r#"{{"model":{model},"model_identity":"{}","return_image":true}}"#,
            parsed.identity()
        );
        let descriptor: StorageDescriptor = serde_json::from_str(&ok).unwrap();
        descriptor.validate().unwrap();
        let wrong = format!(
            r#"{{"model":{model},"model_identity":"sha256:{}","return_image":true}}"#,
            "0".repeat(64)
        );
        let descriptor: StorageDescriptor = serde_json::from_str(&wrong).unwrap();
        assert!(descriptor.validate().is_err());
        let image = format!(
            r#"{{"model":{model},"model_identity":"{}","return_image":false,"image":{{"chunks":2,"bytes":10,"sha256":"{}"}}}}"#,
            parsed.identity(),
            "0".repeat(64)
        );
        let descriptor: StorageDescriptor = serde_json::from_str(&image).unwrap();
        assert!(descriptor.validate().is_err());
        assert!(
            serde_json::from_str::<StorageDescriptor>(&format!(
                r#"{{"model":{model},"model_identity":"x","return_image":true,"extra":1}}"#
            ))
            .is_err()
        );
    }
}
