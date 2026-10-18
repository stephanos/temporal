//! Helper-owned storage: the execute request's storage descriptor, recovery
//! image frames, and the terminal storage evidence streams.

use crate::namespace::{Namespace, Stop};
use crate::protocol;
use crate::volume::{self, CrashEnumeration, CrashEnumerationLimits, StorageModel};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::io::BufRead;
use wasmtime::{Error, Result};

const CHUNK: u64 = 64 * 1024;
/// Recovery images carry history and metadata beyond the volume capacities;
/// this bound is added to the filesystem byte limit (Go uses the same).
pub const IMAGE_METADATA_BYTES: u64 = 1 << 20;
pub const STATE_MAGIC: &[u8; 8] = b"GMSTORE\x01";
const STATE_VERSION: u32 = 1;

#[derive(Clone, Debug, PartialEq, Eq, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase", deny_unknown_fields)]
pub struct StorageCrashContext {
    #[serde(rename = "PendingSHA256")]
    pub pending_sha256: String,
    pub operations: Vec<volume::Operation>,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ChunkedDescriptor {
    pub chunks: u64,
    pub bytes: u64,
    pub sha256: String,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StorageDescriptor {
    pub model: StorageModel,
    pub model_identity: String,
    #[serde(default)]
    pub crash_limits: Option<CrashEnumerationLimits>,
    #[serde(default)]
    pub crash_selection: Option<BTreeMap<String, Option<Vec<u64>>>>,
    #[serde(default)]
    pub crash_context: Option<ChunkedDescriptor>,
    #[serde(skip)]
    observed_context: Option<BTreeMap<String, StorageCrashContext>>,
    pub return_image: bool,
    #[serde(default)]
    pub image: Option<ChunkedDescriptor>,
}

impl StorageDescriptor {
    pub fn validate(&self) -> Result<()> {
        if self.model.identity() != self.model_identity {
            return Err(Error::msg("storage model identity mismatch"));
        }
        // Go refuses the same before execution: a selection names only
        // configured volumes.
        if self.crash_selection.as_ref().is_some_and(|selection| {
            selection
                .keys()
                .any(|id| !self.model.volumes().iter().any(|config| &config.id == id))
        }) {
            return Err(Error::new(crate::namespace::Boundary {
                kind: "invalid",
                dimension: "storage crash selection volume",
            }));
        }
        let invalid = |dimension| {
            Error::new(crate::namespace::Boundary {
                kind: "invalid",
                dimension,
            })
        };
        for operations in self
            .crash_selection
            .as_ref()
            .into_iter()
            .flat_map(|selection| selection.values())
        {
            let operations = operations.as_deref().unwrap_or_default();
            let selected: BTreeSet<_> = operations.iter().copied().collect();
            if selected.contains(&0) {
                return Err(invalid("storage crash selection zero operation"));
            }
            if selected.len() != operations.len() {
                return Err(invalid("storage crash selection duplicate operation"));
            }
            if !selected.is_empty() && self.crash_context.is_none() {
                return Err(invalid(
                    "storage crash selection requires prior observation",
                ));
            }
        }
        for (declared, dimension) in [
            (&self.image, "invalid storage image descriptor"),
            (
                &self.crash_context,
                "invalid storage crash context descriptor",
            ),
        ] {
            if let Some(image) = declared
                && (image.bytes == 0
                    || image.chunks != image.bytes.div_ceil(CHUNK)
                    || image.sha256.len() != 64
                    || !image
                        .sha256
                        .bytes()
                        .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte)))
            {
                return Err(Error::msg(dimension));
            }
        }
        Ok(())
    }

    fn validate_context(&self) -> Result<()> {
        let invalid = |dimension| {
            Error::new(crate::namespace::Boundary {
                kind: "invalid",
                dimension,
            })
        };
        if let Some(context) = &self.observed_context {
            for (id, prior) in context {
                if !self.model.volumes().iter().any(|config| &config.id == id)
                    || prior.pending_sha256.len() != 71
                    || !prior.pending_sha256.starts_with("sha256:")
                    || !prior.pending_sha256.as_bytes()[7..]
                        .iter()
                        .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(byte))
                    || prior.operations.len() as u64 > self.model.limits.pending_operations
                {
                    return Err(invalid("storage crash context"));
                }
                let mut previous = 0;
                for operation in &prior.operations {
                    if operation.id <= previous
                        || operation
                            .dependencies
                            .as_deref()
                            .unwrap_or_default()
                            .iter()
                            .any(|dependency| *dependency == 0 || *dependency >= operation.id)
                    {
                        return Err(invalid("storage crash context"));
                    }
                    previous = operation.id;
                }
            }
        }
        for (id, operations) in self.crash_selection.as_ref().into_iter().flatten() {
            let selected: BTreeSet<_> = operations
                .as_deref()
                .unwrap_or_default()
                .iter()
                .copied()
                .collect();
            if selected.is_empty() {
                continue;
            }
            let prior = self
                .observed_context
                .as_ref()
                .and_then(|context| context.get(id))
                .ok_or_else(|| invalid("storage crash selection requires prior observation"))?;
            let pending: BTreeSet<_> = prior
                .operations
                .iter()
                .map(|operation| operation.id)
                .collect();
            if !selected.is_subset(&pending) {
                return Err(invalid("storage crash selection unknown operation"));
            }
            if prior
                .operations
                .iter()
                .filter(|operation| selected.contains(&operation.id))
                .any(|operation| {
                    operation
                        .dependencies
                        .as_deref()
                        .unwrap_or_default()
                        .iter()
                        .any(|dependency| {
                            pending.contains(dependency) && !selected.contains(dependency)
                        })
                })
            {
                return Err(invalid("storage crash selection is not dependency-closed"));
            }
        }
        Ok(())
    }

    pub fn image_limit(filesystem_bytes: u64) -> u64 {
        filesystem_bytes + IMAGE_METADATA_BYTES
    }

    pub fn read_context(&mut self, input: &mut dyn BufRead, maximum: u64) -> Result<()> {
        self.observed_context = match &self.crash_context {
            Some(declared) => {
                let bytes = read_chunks(input, declared, maximum, &CONTEXT_STREAM)?;
                Some(serde_json::from_slice(&bytes).map_err(Error::new)?)
            }
            None => None,
        };
        self.validate_context()
    }

    /// Reads the declared recovery image frames after the namespace image.
    pub fn read_image(&self, input: &mut dyn BufRead, maximum: u64) -> Result<Option<Vec<u8>>> {
        let Some(declared) = &self.image else {
            return Ok(None);
        };
        read_chunks(input, declared, maximum, &IMAGE_STREAM).map(Some)
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

/// One chunked execute-input stream: its frame kinds, error label, and the
/// boundary reported when the declared size exceeds the allowance.
struct ChunkedStream {
    chunk_kind: &'static str,
    end_kind: &'static str,
    label: &'static str,
    oversize_kind: &'static str,
    oversize_dimension: &'static str,
}

const IMAGE_STREAM: ChunkedStream = ChunkedStream {
    chunk_kind: "storage_image_input",
    end_kind: "storage_image_inputs_end",
    label: "storage image",
    oversize_kind: "capacity",
    oversize_dimension: "storage-image-bytes",
};

const CONTEXT_STREAM: ChunkedStream = ChunkedStream {
    chunk_kind: "storage_context_input",
    end_kind: "storage_context_inputs_end",
    label: "storage crash context",
    oversize_kind: "invalid",
    oversize_dimension: "storage-context-bytes",
};

fn read_chunks(
    input: &mut dyn BufRead,
    declared: &ChunkedDescriptor,
    maximum: u64,
    stream: &ChunkedStream,
) -> Result<Vec<u8>> {
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct End {
        #[serde(rename = "type")]
        kind: String,
    }
    #[derive(Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Chunk {
        #[serde(rename = "type")]
        kind: String,
        sequence: u64,
        data_base64: String,
    }
    let label = stream.label;
    if declared.bytes > maximum {
        return Err(Error::new(crate::namespace::Boundary {
            kind: stream.oversize_kind,
            dimension: stream.oversize_dimension,
        }));
    }
    let mut bytes = Vec::new();
    for sequence in 0..declared.chunks {
        let chunk: Chunk =
            serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
        if chunk.kind != stream.chunk_kind
            || chunk.sequence != sequence
            || chunk.data_base64.len() as u64 > CHUNK.div_ceil(3) * 4
        {
            return Err(Error::msg(format!("invalid {label} chunk")));
        }
        let data = STANDARD.decode(chunk.data_base64).map_err(Error::new)?;
        if data.len() as u64 != CHUNK.min(declared.bytes - bytes.len() as u64) {
            return Err(Error::msg(format!("invalid {label} chunk length")));
        }
        bytes.extend(data);
    }
    let end: End = serde_json::from_slice(&protocol::read_frame(input)?).map_err(Error::new)?;
    if end.kind != stream.end_kind {
        return Err(Error::msg(format!("invalid {label} end")));
    }
    if format!("{:x}", Sha256::digest(&bytes)) != declared.sha256 {
        return Err(Error::msg(format!("{label} digest mismatch")));
    }
    Ok(bytes)
}

/// The terminal storage evidence of one incarnation.
pub struct Evidence {
    pub log: Vec<u8>,
    pub state: Vec<u8>,
    pub image: Option<Vec<u8>>,
    pub failure: Option<(&'static str, String)>,
}

/// Go `Run`'s post-execution storage collection: transcript, snapshots and
/// operations, then each configured volume's enumeration in model order, then
/// the crash image. An enumeration error is `invalid`; a crash error is
/// `infrastructure`.
pub fn evidence(namespace: &mut Namespace, descriptor: &StorageDescriptor) -> Evidence {
    collect(namespace, descriptor, descriptor.return_image)
}

pub fn recovery_evidence(namespace: &mut Namespace, descriptor: &StorageDescriptor) -> Evidence {
    collect(namespace, descriptor, true)
}

#[expect(
    clippy::too_many_lines,
    reason = "The ordered model transition preserves Go admission, mutation and refusal order"
)]
fn collect(
    namespace: &mut Namespace,
    descriptor: &StorageDescriptor,
    return_image: bool,
) -> Evidence {
    let filesystem_bytes = namespace.filesystem_bytes();
    let storage = namespace.storage().expect("attached storage");
    let snapshots = storage.snapshots();
    let context: BTreeMap<_, _> =
        if descriptor.crash_limits.is_some() || descriptor.observed_context.is_some() {
            let mut pending = storage.fs.pending_operations();
            snapshots
                .iter()
                .map(|snapshot| {
                    (
                        snapshot.id.clone(),
                        StorageCrashContext {
                            pending_sha256: snapshot.pending_sha256.clone(),
                            operations: pending.remove(&snapshot.id).expect("configured volume"),
                        },
                    )
                })
                .collect()
        } else {
            BTreeMap::new()
        };
    let log = storage.log.clone();
    let snapshots = serde_json::to_vec(&snapshots).expect("in-memory JSON");
    let operations = serde_json::to_vec(&storage.operations()).expect("in-memory JSON");
    let partial = |enumerations: &[u8], stage: u8, failure| Evidence {
        state: encode_state_sections(
            &snapshots,
            &operations,
            enumerations,
            "",
            &log,
            Some(stage),
            None,
        ),
        log: log.clone(),
        image: None,
        failure: Some(failure),
    };
    let enumerations = match &descriptor.crash_limits {
        None => b"null".to_vec(),
        Some(limits) => {
            let mut pages: BTreeMap<String, CrashEnumeration> = BTreeMap::new();
            for config in descriptor.model.volumes() {
                let page = match storage.enumerate(&config.id, *limits, None) {
                    Ok(page) => page,
                    Err(error) => {
                        return partial(
                            &serde_json::to_vec(&pages).expect("in-memory JSON"),
                            1,
                            ("invalid", volume::storage_error_text(&error)),
                        );
                    }
                };
                pages.insert(config.id.clone(), page);
            }
            serde_json::to_vec(&pages).expect("in-memory JSON")
        }
    };
    if descriptor
        .observed_context
        .as_ref()
        .is_some_and(|requested| {
            requested
                .iter()
                .any(|(id, prior)| context.get(id) != Some(prior))
        })
    {
        return partial(
            &enumerations,
            2,
            (
                "invalid",
                "storage crash context diverged from prior observation".into(),
            ),
        );
    }
    let image = match namespace.crash_storage(
        &descriptor.selections(),
        StorageDescriptor::image_limit(filesystem_bytes),
    ) {
        Ok(image) => image,
        Err(error) => return partial(&enumerations, 2, crash_failure(error)),
    };
    let context_bytes = descriptor
        .crash_limits
        .as_ref()
        .map(|_| serde_json::to_vec(&context).expect("in-memory JSON"));
    let state = match context_bytes {
        Some(context) => encode_state_sections(
            &snapshots,
            &operations,
            &enumerations,
            &image.digest,
            &log,
            None,
            Some(&context),
        ),
        None => encode_state(&snapshots, &operations, &enumerations, &image.digest, &log),
    };
    Evidence {
        log,
        state,
        image: return_image.then_some(image.bytes),
        failure: None,
    }
}

/// Go `CrashStorage` failures: image capacity keeps its classification and
/// message; anything else is infrastructure.
fn crash_failure(error: volume::Error) -> (&'static str, String) {
    match error {
        volume::Error::Capacity {
            resource: "image-bytes",
            ..
        } => ("capacity", "WASI capacity: storage-image-bytes".into()),
        other => ("infrastructure", volume::storage_error_text(&other)),
    }
}

/// Storage-state envelope v1: magic, version, five tagged length-prefixed
/// sections (snapshots JSON, operations JSON, enumerations JSON or `null`,
/// image identity, and the storage log's u64 length plus SHA-256), then the
/// u64 length and SHA-256 of all preceding bytes.
pub fn encode_state(
    snapshots: &[u8],
    operations: &[u8],
    enumerations: &[u8],
    image_identity: &str,
    log: &[u8],
) -> Vec<u8> {
    encode_state_sections(
        snapshots,
        operations,
        enumerations,
        image_identity,
        log,
        None,
        None,
    )
}

fn encode_state_sections(
    snapshots: &[u8],
    operations: &[u8],
    enumerations: &[u8],
    image_identity: &str,
    log: &[u8],
    failed_stage: Option<u8>,
    crash_context: Option<&[u8]>,
) -> Vec<u8> {
    let mut log_binding = (log.len() as u64).to_le_bytes().to_vec();
    log_binding.extend_from_slice(&Sha256::digest(log));
    let mut bytes = STATE_MAGIC.to_vec();
    bytes.extend_from_slice(
        &(if failed_stage.is_some() {
            2
        } else if crash_context.is_some() {
            3
        } else {
            STATE_VERSION
        })
        .to_le_bytes(),
    );
    bytes.extend_from_slice(
        &(if failed_stage.is_some() || crash_context.is_some() {
            6u32
        } else {
            5u32
        })
        .to_le_bytes(),
    );
    for (tag, section) in [
        snapshots,
        operations,
        enumerations,
        image_identity.as_bytes(),
        log_binding.as_slice(),
    ]
    .iter()
    .enumerate()
    {
        bytes.extend_from_slice(&(u32::try_from(tag).unwrap() + 1).to_le_bytes());
        bytes.extend_from_slice(&(section.len() as u64).to_le_bytes());
        bytes.extend_from_slice(section);
    }
    if let Some(stage) = failed_stage {
        bytes.extend_from_slice(&6u32.to_le_bytes());
        bytes.extend_from_slice(&1u64.to_le_bytes());
        bytes.push(stage);
    } else if let Some(context) = crash_context {
        bytes.extend_from_slice(&6u32.to_le_bytes());
        bytes.extend_from_slice(&(context.len() as u64).to_le_bytes());
        bytes.extend_from_slice(context);
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
    fn crash_image_failures_are_classified_with_their_message() {
        assert_eq!(
            crash_failure(volume::Error::Capacity {
                resource: "image-bytes",
                required: 3,
                maximum: 2
            }),
            ("capacity", "WASI capacity: storage-image-bytes".to_string())
        );
        assert_eq!(
            crash_failure(volume::Error::Invalid(
                "crash selection contains an unknown operation"
            )),
            (
                "infrastructure",
                "crash selection contains an unknown operation".to_string()
            )
        );
    }

    #[test]
    fn state_envelope_layout_and_trailer() {
        let bytes = encode_state(b"[]", b"{}", b"null", &"a".repeat(64), b"x\n");
        assert_eq!(&bytes[..8], STATE_MAGIC);
        assert_eq!(&bytes[8..16], &[1, 0, 0, 0, 5, 0, 0, 0]);
        let body = bytes.len() - 40;
        assert_eq!(
            u64::from_le_bytes(bytes[body..body + 8].try_into().unwrap()),
            body as u64
        );
        assert_eq!(
            &bytes[body + 8..],
            Sha256::digest(&bytes[..body]).as_slice()
        );
    }

    #[test]
    fn pending_context_larger_than_one_mib_uses_chunked_input() {
        let model: StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1024}],
            "limits":{"PendingOperations":1000,"Transitions":1000}
        }))
        .unwrap();
        let operations: Vec<_> = (1..=400).map(|id| serde_json::json!({
            "ID":id,"Kind":"namespace","Dependencies":null,"Inode":id + 1,"Offset":0,"Bytes":0,
            "PayloadSHA256":"","EffectSHA256":"","Path":format!("/{}", "f".repeat(3000)),"Destination":""
        })).collect();
        let bytes = serde_json::to_vec(&serde_json::json!({"data":{
            "PendingSHA256":format!("sha256:{}", "a".repeat(64)),"Operations":operations
        }}))
        .unwrap();
        assert!(bytes.len() > 1 << 20);
        let mut descriptor: StorageDescriptor = serde_json::from_value(serde_json::json!({
            "model":serde_json::from_str::<serde_json::Value>(&model.go_json()).unwrap(),
            "model_identity":model.identity(),"return_image":false,"crash_context":{
                "chunks":bytes.len().div_ceil(usize::try_from(CHUNK).unwrap()),"bytes":bytes.len(),
                "sha256":format!("{:x}", Sha256::digest(&bytes))
            }
        }))
        .unwrap();
        descriptor.validate().unwrap();
        let mut framed = Vec::new();
        for (sequence, chunk) in bytes.chunks(usize::try_from(CHUNK).unwrap()).enumerate() {
            serde_json::to_writer(
                &mut framed,
                &serde_json::json!({"type":"storage_context_input",
                "sequence":sequence,"data_base64":STANDARD.encode(chunk)}),
            )
            .unwrap();
            framed.push(b'\n');
        }
        let oversized = descriptor
            .read_context(&mut std::io::Cursor::new(&framed), 1)
            .unwrap_err();
        assert_eq!(oversized.to_string(), "WASI invalid: storage-context-bytes");
        let mut truncated = std::io::Cursor::new(framed.clone());
        assert!(
            descriptor
                .read_context(&mut truncated, 18 * StorageDescriptor::image_limit(1024))
                .is_err()
        );
        framed.extend_from_slice(b"{\"type\":\"storage_context_inputs_end\"}\n");
        descriptor
            .read_context(
                &mut std::io::Cursor::new(framed),
                18 * StorageDescriptor::image_limit(1024),
            )
            .unwrap();
        assert_eq!(
            descriptor.observed_context.unwrap()["data"]
                .operations
                .len(),
            400
        );
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
        let unknown = format!(
            r#"{{"model":{model},"model_identity":"{}","return_image":true,"crash_selection":{{"other":[1]}}}}"#,
            parsed.identity()
        );
        let descriptor: StorageDescriptor = serde_json::from_str(&unknown).unwrap();
        assert_eq!(
            descriptor.validate().unwrap_err().to_string(),
            "WASI invalid: storage crash selection volume"
        );
        assert!(
            serde_json::from_str::<StorageDescriptor>(&format!(
                r#"{{"model":{model},"model_identity":"x","return_image":true,"extra":1}}"#
            ))
            .is_err()
        );
    }
}
