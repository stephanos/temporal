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

#[cfg(test)]
mod crash_failure_corpus;

#[cfg(test)]
mod validation_order_corpus;

#[cfg(test)]
mod preservation_tests;

#[cfg(test)]
mod simulation_tests {
    use super::*;
    use crate::linked_simulation::{Controller, Dimension, Override, Plan};
    use std::sync::{Arc, Mutex};

    fn controller(overrides: Vec<Override>) -> crate::linked_simulation::Shared {
        Arc::new(Mutex::new(
            Controller::new(
                Plan::new(
                    format!("sha256:{}", "a".repeat(64)),
                    format!("sha256:{}", "b".repeat(64)),
                    7,
                    overrides,
                )
                .unwrap(),
                100,
            )
            .unwrap(),
        ))
    }

    fn fixture(descriptor: &StorageDescriptor) -> Namespace {
        let mut namespace = Namespace::new(crate::namespace::Image {
            entries: vec![crate::namespace::ImageEntry {
                path: b"/data".to_vec(),
                directory: true,
                readonly: false,
                data: vec![],
            }],
            writable_directories: vec![b"/data".to_vec()],
            preopens: vec![b"/data".to_vec()],
            working_directory: b"/data".to_vec(),
            stdin: vec![],
            limits: crate::namespace::Limits {
                files: 100,
                descriptors: 100,
                filesystem_bytes: 4096,
                output_bytes: 4096,
            },
            epoch_nanos: 0,
        })
        .unwrap();
        descriptor.attach(&mut namespace, None, 1 << 20).unwrap();
        let storage = namespace.storage_mut().unwrap();
        let inode = storage
            .fs
            .create("/data/file", volume::Kind::File, 0o600, 1)
            .unwrap();
        storage.fs.sync_node(inode).unwrap();
        let root = storage.fs.lookup("/data").unwrap().inode;
        storage.fs.sync_node(root).unwrap();
        storage.fs.write_at(inode, 0, b"survives", 2).unwrap();
        namespace
    }

    #[test]
    fn reference_precrash_image_keeps_actual_pending_effects() {
        let model: StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":[{"ID":"db","Path":"/data","CapacityBytes":4096}],
            "limits":{"PendingOperations":100,"Transitions":1000}
        }))
        .unwrap();
        let mut descriptor: StorageDescriptor = serde_json::from_value(serde_json::json!({
            "model":model,"model_identity":model.identity(),"return_image":true,
            "crash_limits":{"States":16,"Operations":100,"Depth":100,"Bytes":1_048_576,"Work":1000}
        }))
        .unwrap();
        let mut ordinary = fixture(&descriptor);
        assert!(
            recovery_evidence(&mut ordinary, &descriptor)
                .precrash_image
                .is_none()
        );
        descriptor.reference_precrash = true;
        let mut namespace = fixture(&descriptor);
        let storage = namespace.storage().unwrap();
        assert_ne!(
            storage.fs.pending_operations()["db"],
            [] as [crate::volume::Operation; 0]
        );
        let expected = storage
            .reference_image(StorageDescriptor::image_limit(namespace.filesystem_bytes()))
            .unwrap();
        let evidence = recovery_evidence(&mut namespace, &descriptor);
        assert_eq!(evidence.failure, None);
        assert_eq!(evidence.precrash_image.as_ref(), Some(&expected));
        assert_ne!(evidence.precrash_image, evidence.image);
    }

    #[test]
    fn exploration_crash_selection_changes_recovered_bytes() {
        let model: StorageModel = serde_json::from_value(serde_json::json!({
            "volumes":[{"ID":"db","Path":"/data","CapacityBytes":4096}],
            "limits":{"PendingOperations":100,"Transitions":1000}
        }))
        .unwrap();
        let mut descriptor: StorageDescriptor = serde_json::from_value(serde_json::json!({
            "model":model, "model_identity":model.identity(), "return_image":true,
            "crash_limits":{"States":16,"Operations":100,"Depth":100,"Bytes":1_048_576,"Work":1000}
        }))
        .unwrap();
        let mut namespace = fixture(&descriptor);
        let page = namespace
            .storage()
            .unwrap()
            .enumerate("db", descriptor.crash_limits.unwrap(), None)
            .unwrap();
        let target = page
            .states
            .as_ref()
            .unwrap()
            .iter()
            .position(|state| {
                state
                    .entries
                    .iter()
                    .any(|entry| entry.data.as_deref() == Some(b"survives"))
            })
            .unwrap();
        let probe = controller(vec![]);
        let original_descriptor = serde_json::to_vec(&descriptor).unwrap();
        descriptor.set_simulation(probe.clone());
        assert_eq!(
            serde_json::to_vec(&descriptor).unwrap(),
            original_descriptor
        );
        let baseline = recovery_evidence(&mut namespace, &descriptor);
        assert_eq!(baseline.failure, None);
        let decision = probe
            .lock()
            .unwrap()
            .decisions()
            .iter()
            .find(|decision| decision.dimension == Dimension::Crash)
            .expect("missing crash choice")
            .clone();
        descriptor.set_simulation(controller(vec![
            Override::from_decision(&decision, u32::try_from(target).unwrap()).unwrap(),
        ]));
        let mut namespace = fixture(&descriptor);
        let evidence = recovery_evidence(&mut namespace, &descriptor);
        assert_eq!(evidence.failure, None);
        let image = evidence.image.unwrap();
        let recovered = volume::Storage::recover(
            descriptor.model.clone(),
            1 << 20,
            &image,
            &format!("{:x}", Sha256::digest(&image)),
            1 << 20,
        )
        .unwrap();
        let inode = recovered.fs.lookup("/data/file").unwrap().inode;
        assert_eq!(recovered.fs.read_at(inode, 0, 32).unwrap(), b"survives");
    }
}

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
    pub operations: Option<Vec<volume::Operation>>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ChunkedDescriptor {
    pub chunks: u64,
    pub bytes: u64,
    pub sha256: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
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
    #[serde(skip)]
    simulation: Option<crate::linked_simulation::Shared>,
    #[serde(skip)]
    pub(crate) reference_precrash: bool,
    pub return_image: bool,
    #[serde(default)]
    pub image: Option<ChunkedDescriptor>,
}

impl StorageDescriptor {
    fn exploration_selections(
        &self,
        pages: &BTreeMap<String, CrashEnumeration>,
        controller: &crate::linked_simulation::Shared,
    ) -> std::result::Result<BTreeMap<String, Vec<u64>>, String> {
        use crate::linked_simulation::{Dimension, domain_hash};
        let mut selections = self.selections();
        for (id, page) in pages {
            let states = page.states.as_deref().unwrap_or_default();
            if states.len() < 2 {
                continue;
            }
            let baseline = selections.get(id).map(Vec::as_slice).unwrap_or_default();
            let baseline: BTreeSet<_> = baseline.iter().copied().collect();
            let Some(default) = states.iter().position(|state| {
                state
                    .selected_operations
                    .iter()
                    .copied()
                    .collect::<BTreeSet<_>>()
                    == baseline
            }) else {
                continue;
            };
            let site = domain_hash(
                "gomad-storage-crash-site/v1",
                &serde_json::json!({
                    "volume":id,"pending":states[default].pending_sha256
                }),
            )?;
            let selected = controller
                .lock()
                .map_err(|_| "simulation exploration controller poisoned")?
                .choose(
                    Dimension::Crash,
                    &site,
                    states.iter().map(|state| state.identity.clone()).collect(),
                    u32::try_from(default).map_err(|_| "storage crash alternatives capacity")?,
                )?;
            selections.insert(
                id.clone(),
                states[selected as usize].selected_operations.clone(),
            );
        }
        Ok(selections)
    }

    pub(crate) fn set_simulation(&mut self, controller: crate::linked_simulation::Shared) {
        self.simulation = Some(controller);
    }

    pub fn validate(&self) -> Result<()> {
        if self.model.identity() != self.model_identity {
            return Err(Error::msg("storage model identity mismatch"));
        }
        // Without a declared crash context every selection check is decidable now;
        // with one, the whole Go-ordered pass runs once the context is attached.
        if self.crash_context.is_none() {
            self.validate_crash_controls(None)?;
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
        self.validate_crash_controls(self.observed_context.as_ref())
    }

    /// Mirrors Go `ValidateStorageCrashSelection`: selected volumes in sorted
    /// order (membership, operations in sequence, prior observation, unknown
    /// operations, dependency closure), then context volumes in sorted order.
    /// The first refusal therefore never depends on map insertion order.
    fn validate_crash_controls(
        &self,
        context: Option<&BTreeMap<String, StorageCrashContext>>,
    ) -> Result<()> {
        let invalid = |dimension| {
            Error::new(crate::namespace::Boundary {
                kind: "invalid",
                dimension,
            })
        };
        let configured = |id: &String| self.model.volumes().iter().any(|config| &config.id == id);
        for (id, operations) in self.crash_selection.iter().flatten() {
            if !configured(id) {
                return Err(invalid("storage crash selection volume"));
            }
            let selected = operations.as_deref().unwrap_or_default();
            let mut seen = BTreeSet::new();
            for operation in selected {
                if *operation == 0 {
                    return Err(invalid("storage crash selection zero operation"));
                }
                if !seen.insert(*operation) {
                    return Err(invalid("storage crash selection duplicate operation"));
                }
            }
            let prior = context.and_then(|context| context.get(id));
            if selected.is_empty() && prior.is_none() {
                continue;
            }
            let prior = prior
                .ok_or_else(|| invalid("storage crash selection requires prior observation"))?;
            let pending: BTreeMap<_, _> = prior
                .operations
                .as_deref()
                .unwrap_or_default()
                .iter()
                .map(|operation| (operation.id, operation))
                .collect();
            for id in selected {
                let operation = pending
                    .get(id)
                    .ok_or_else(|| invalid("storage crash selection unknown operation"))?;
                if operation
                    .dependencies
                    .as_deref()
                    .unwrap_or_default()
                    .iter()
                    .any(|dependency| {
                        pending.contains_key(dependency) && !seen.contains(dependency)
                    })
                {
                    return Err(invalid("storage crash selection is not dependency-closed"));
                }
            }
        }
        for (id, prior) in context.into_iter().flatten() {
            if !configured(id) {
                return Err(invalid("storage crash context volume"));
            }
            if prior.pending_sha256.len() != 71
                || !prior.pending_sha256.starts_with("sha256:")
                || !prior.pending_sha256.as_bytes()[7..]
                    .iter()
                    .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(byte))
                || prior.operations.is_none()
                || prior.operations.as_deref().unwrap_or_default().len() as u64
                    > self.model.limits.pending_operations
            {
                return Err(invalid("storage crash context inventory"));
            }
            let mut previous = 0;
            for operation in prior.operations.as_deref().unwrap_or_default() {
                if operation.id <= previous {
                    return Err(invalid("storage crash context operation order"));
                }
                previous = operation.id;
                if operation
                    .dependencies
                    .as_deref()
                    .unwrap_or_default()
                    .iter()
                    .any(|dependency| *dependency == 0 || *dependency >= operation.id)
                {
                    return Err(invalid("storage crash context dependency"));
                }
            }
        }
        Ok(())
    }

    pub fn image_limit(filesystem_bytes: u64) -> u64 {
        filesystem_bytes + IMAGE_METADATA_BYTES
    }

    pub fn read_context(&mut self, input: &mut dyn BufRead, maximum: u64) -> Result<()> {
        #[cfg(feature = "diagnostic-profile")]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::StorageTransport);
        self.observed_context = match &self.crash_context {
            Some(declared) => {
                let bytes = read_chunks(input, declared, maximum, &CONTEXT_STREAM)?;
                Some(serde_json::from_slice(&bytes).map_err(Error::new)?)
            }
            None => None,
        };
        self.validate_context()
    }

    pub(crate) fn attach_context_bytes(
        &mut self,
        bytes: Option<&[u8]>,
        maximum: u64,
    ) -> Result<()> {
        authenticate_bytes(bytes, self.crash_context.as_ref(), maximum, &CONTEXT_STREAM)?;
        self.observed_context = bytes
            .map(serde_json::from_slice)
            .transpose()
            .map_err(Error::new)?;
        self.validate_context()
    }

    pub(crate) fn authenticate_image_bytes(
        &self,
        bytes: Option<&[u8]>,
        maximum: u64,
    ) -> Result<()> {
        authenticate_bytes(bytes, self.image.as_ref(), maximum, &IMAGE_STREAM)
    }

    /// Reads the declared recovery image frames after the namespace image.
    pub fn read_image(&self, input: &mut dyn BufRead, maximum: u64) -> Result<Option<Vec<u8>>> {
        #[cfg(feature = "diagnostic-profile")]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::StorageTransport);
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
        #[cfg(feature = "diagnostic-profile")]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::StorageTransport);
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

fn authenticate_bytes(
    bytes: Option<&[u8]>,
    declared: Option<&ChunkedDescriptor>,
    maximum: u64,
    stream: &ChunkedStream,
) -> Result<()> {
    let (bytes, declared) = match (bytes, declared) {
        (None, None) => return Ok(()),
        (Some(bytes), Some(declared)) => (bytes, declared),
        _ => {
            return Err(Error::msg(format!(
                "{} bytes/declaration mismatch",
                stream.label
            )));
        }
    };
    if declared.bytes > maximum {
        return Err(Error::new(crate::namespace::Boundary {
            kind: stream.oversize_kind,
            dimension: stream.oversize_dimension,
        }));
    }
    if bytes.len() as u64 != declared.bytes {
        return Err(Error::msg(format!("invalid {} byte count", stream.label)));
    }
    if format!("{:x}", Sha256::digest(bytes)) != declared.sha256 {
        return Err(Error::msg(format!("{} digest mismatch", stream.label)));
    }
    Ok(())
}

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
    authenticate_bytes(Some(&bytes), Some(declared), maximum, stream)?;
    Ok(bytes)
}

/// The terminal storage evidence of one incarnation.
pub struct Evidence {
    pub log: Vec<u8>,
    pub state: Vec<u8>,
    pub image: Option<Vec<u8>>,
    pub precrash_image: Option<Vec<u8>>,
    pub failure: Option<(&'static str, String)>,
}

/// Go `Run`'s post-execution storage collection: transcript, snapshots and
/// operations, then each configured volume's enumeration in model order, then
/// the crash image. An enumeration error is `invalid`; a crash error is
/// `infrastructure`.
pub fn evidence(namespace: &mut Namespace, descriptor: &StorageDescriptor) -> Evidence {
    #[cfg(feature = "diagnostic-profile")]
    let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::StorageTransport);
    collect(namespace, descriptor, descriptor.return_image)
}

pub fn recovery_evidence(namespace: &mut Namespace, descriptor: &StorageDescriptor) -> Evidence {
    #[cfg(feature = "diagnostic-profile")]
    let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::StorageTransport);
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
    let persistence = descriptor
        .simulation
        .as_ref()
        .filter(|_| descriptor.observed_context.is_none())
        .map(|controller| {
            namespace
                .storage_mut()
                .expect("attached storage")
                .fs
                .explore_persistence(controller)
        })
        .transpose();
    let storage = namespace.storage().expect("attached storage");
    let snapshots = storage.snapshots();
    let context: BTreeMap<_, _> = if descriptor.crash_limits.is_some()
        || descriptor.observed_context.is_some()
    {
        let mut pending = storage.fs.pending_operations();
        snapshots
            .iter()
            .map(|snapshot| {
                (
                    snapshot.id.clone(),
                    StorageCrashContext {
                        pending_sha256: snapshot.pending_sha256.clone(),
                        operations: Some(pending.remove(&snapshot.id).expect("configured volume")),
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
        precrash_image: None,
        failure: Some(failure),
    };
    if let Err(error) = persistence {
        return partial(b"null", 1, ("invalid", error));
    }
    let mut pages: BTreeMap<String, CrashEnumeration> = BTreeMap::new();
    let enumerations = match &descriptor.crash_limits {
        None => b"null".to_vec(),
        Some(limits) => {
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
    let selections = match &descriptor.simulation {
        Some(controller) => match descriptor.exploration_selections(&pages, controller) {
            Ok(selections) => selections,
            Err(error) => return partial(&enumerations, 2, ("invalid", error)),
        },
        None => descriptor.selections(),
    };
    let precrash_image = if descriptor.reference_precrash {
        match storage.reference_image(StorageDescriptor::image_limit(filesystem_bytes)) {
            Ok(bytes) => Some(bytes),
            Err(error) => return partial(&enumerations, 2, crash_failure(error)),
        }
    } else {
        None
    };
    let image = match namespace.crash_storage(
        &selections,
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
        precrash_image,
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

pub(crate) fn encode_state_sections(
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
mod vector_corpus;
