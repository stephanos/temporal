//! Go `wasi/volume.go`'s storage state: the scheduled fault engine, the
//! storage event log, Go errno/boundary classification and crash/recovery.

use super::config::{StorageModel, go_string};
use super::crash::{CrashEnumeration, CrashEnumerationLimits, CrashExploration};
use super::image::{RecoveryImage, image_digest};
use super::state::{Error, Fs, Handle, OpenFlags, Operation, VolumeSnapshot};
use std::collections::BTreeMap;
use std::fmt::Write as _;

/// Why a storage-routed operation stops the run instead of answering it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum Stop {
    /// A typed Go `WASI <kind>: <dimension>` boundary.
    Boundary(&'static str, &'static str),
    /// A modeled `crash-before` fault, by fault name.
    Crash(String),
    /// A Go error outside the boundary taxonomy (its `Error()` text).
    Fatal(String),
}

pub(crate) type Decision = std::result::Result<(u64, u16), Stop>;

/// The durable recovery state of one discarded incarnation.
pub(crate) struct Image {
    pub(crate) bytes: Vec<u8>,
    pub(crate) digest: String,
}

pub(crate) struct Storage {
    pub(crate) fs: Fs,
    pub(crate) model: StorageModel,
    pub(crate) model_identity: String,
    pub(crate) occurrences: BTreeMap<String, u64>,
    pub(crate) log: Vec<u8>,
    sequence: u64,
    /// Go's shared `volume.Clock`: the `now` of the last storage decision.
    pub(crate) clock: i64,
    transcript_limit: u64,
}

impl Storage {
    /// A fresh storage state, after the model has been validated.
    pub(crate) fn new(model: StorageModel, transcript_limit: u64) -> Result<Self, Stop> {
        let mut fs = Fs::new(0);
        fs.configure_volumes(model.volumes(), model.limits, 0)
            .map_err(|_| Stop::Boundary("invalid", "storage volume configuration"))?;
        Ok(Self::with_fs(fs, model, transcript_limit))
    }

    fn with_fs(fs: Fs, model: StorageModel, transcript_limit: u64) -> Self {
        let model_identity = model.identity();
        Self {
            fs,
            model,
            model_identity,
            occurrences: BTreeMap::new(),
            log: Vec::new(),
            sequence: 0,
            clock: 0,
            transcript_limit,
        }
    }

    /// Recovery state from a self-contained image. The image must bind this
    /// model's identity and the configured volumes; lock paths are refused.
    pub(crate) fn recover(
        model: StorageModel,
        transcript_limit: u64,
        bytes: &[u8],
        digest: &str,
        maximum: u64,
    ) -> Result<Self, Stop> {
        let identity = model.identity();
        let image = RecoveryImage::decode(
            bytes,
            digest,
            &identity,
            model.volumes(),
            model.limits,
            model.lock_databases(),
            maximum,
        )
        .map_err(image_refusal)?;
        let (fs, occurrences) = image.recover().map_err(image_refusal)?;
        let mut storage = Self::with_fs(fs, model, transcript_limit);
        storage.occurrences = occurrences;
        Ok(storage)
    }

    /// Go `storageDecision`: counts the occurrence, applies the first matching
    /// fault and appends the event line. Returns the effective byte count and
    /// errno; `crash-before` returns its stop after the event is recorded.
    pub(crate) fn decision(
        &mut self,
        resource: &str,
        operation: &str,
        mut count: u64,
        now: u64,
    ) -> Decision {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.clock = now.cast_signed();
        let key = format!("{resource}\0{operation}");
        let occurrence = {
            let entry = self.occurrences.entry(key).or_default();
            *entry += 1;
            *entry
        };
        let mut fault_name = None;
        let mut action = "complete";
        let mut bytes = count;
        let mut errno = 0;
        let mut crash = None;
        if let Some(fault) = self.model.faults().iter().find(|fault| {
            fault.resource == resource
                && fault.operation == operation
                && fault.occurrence == occurrence
        }) {
            fault_name = Some(fault.name.as_str());
            action = fault.action.as_str();
            match action {
                "io-error" | "sync-error" => errno = 29,
                "no-space" => errno = 51,
                "short-write" | "torn-write" => {
                    if count <= fault.bytes {
                        return Err(Stop::Boundary(
                            "invalid",
                            "storage write fault must shorten the matched write",
                        ));
                    }
                    count = fault.bytes;
                    bytes = count;
                    if action == "torn-write" {
                        errno = 29;
                    }
                }
                "crash-before" => crash = Some(fault.name.clone()),
                _ => {}
            }
        }
        let line = event_line(
            self.sequence,
            resource,
            operation,
            occurrence,
            fault_name,
            action,
            bytes,
        );
        if self.log.len() as u64 + line.len() as u64 + 1 > self.transcript_limit {
            return Err(Stop::Boundary("capacity", "storage-transcript"));
        }
        self.log.extend_from_slice(line.as_bytes());
        self.log.push(b'\n');
        self.sequence += 1;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        crate::event_order::emit_current(
            "model-transition",
            self.sequence - 1,
            self.clock,
            serde_json::json!({"model": "storage", "line": line}),
        );
        match crash {
            Some(fault) => Err(Stop::Crash(fault)),
            None => Ok((count, errno)),
        }
    }

    pub(crate) fn open(&mut self, resource: &str, flags: OpenFlags) -> Result<Handle, Error> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        self.fs.open(resource, flags, 0o666, self.clock)
    }

    pub(crate) fn snapshots(&self) -> Vec<VolumeSnapshot> {
        self.fs.volume_snapshots()
    }

    pub(crate) fn operations(&self) -> BTreeMap<String, Vec<Operation>> {
        self.fs.operations()
    }

    pub(crate) fn enumerate(
        &self,
        volume: &str,
        limits: CrashEnumerationLimits,
        exploration: Option<&CrashExploration>,
    ) -> Result<CrashEnumeration, Error> {
        self.fs.enumerate_crash_states(volume, limits, exploration)
    }

    pub(crate) fn reference_image(&self, maximum: u64) -> Result<Vec<u8>, Error> {
        RecoveryImage::encode(&self.fs, &self.model_identity, &self.occurrences, maximum)
    }

    /// Go `CrashStorage`: advances the lifecycle with the selected pending
    /// operations and produces the durable recovery image. The namespace
    /// discards its descriptors separately.
    pub(crate) fn crash(
        &mut self,
        selections: &BTreeMap<String, Vec<u64>>,
        maximum: u64,
    ) -> Result<Image, Error> {
        #[cfg(all(feature = "diagnostic-profile", not(feature = "conformance")))]
        let _owner = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Volume);
        let next = self.fs.advance_lifecycle(false, selections)?;
        let bytes = RecoveryImage::encode(&next, &self.model_identity, &self.occurrences, maximum)?;
        let digest = image_digest(&bytes);
        Ok(Image { bytes, digest })
    }
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Error classification is passed directly to Result::map_err"
)]
fn image_refusal(error: Error) -> Stop {
    match error {
        Error::Capacity { .. } => Stop::Boundary("capacity", "storage-image-bytes"),
        Error::Invalid("durable image contains process-owned SQLite lock") => Stop::Boundary(
            "invalid",
            "durable image contains process-owned SQLite lock",
        ),
        Error::Invalid("recovery storage model identity") => {
            Stop::Boundary("invalid", "recovery storage model identity")
        }
        _ => Stop::Boundary("invalid", "durable storage image"),
    }
}

/// Go `json.Marshal(StorageEvent)`.
pub(crate) fn event_line(
    sequence: u64,
    resource: &str,
    operation: &str,
    occurrence: u64,
    fault: Option<&str>,
    action: &str,
    bytes: u64,
) -> String {
    let mut line = format!("{{\"sequence\":{sequence},\"resource\":");
    go_string(&mut line, resource);
    line.push_str(",\"operation\":");
    go_string(&mut line, operation);
    write!(line, ",\"occurrence\":{occurrence}").unwrap();
    if let Some(fault) = fault.filter(|fault| !fault.is_empty()) {
        line.push_str(",\"fault\":");
        go_string(&mut line, fault);
    }
    line.push_str(",\"action\":");
    go_string(&mut line, action);
    if bytes != 0 {
        write!(line, ",\"bytes\":{bytes}").unwrap();
    }
    line.push('}');
    line
}

/// Go `storageErrno`.
pub(crate) fn errno(error: &Error) -> u16 {
    match error {
        Error::Capacity {
            resource: "volume_bytes",
            ..
        } => 51,
        Error::Capacity { .. } | Error::Invalid(_) => 29,
        Error::Errno(name) => match *name {
            "eof" => 0,
            "read-only-access" => 2,
            "bad-descriptor" => 8,
            "busy" => 10,
            "exists" => 20,
            "file-too-large" => 22,
            "invalid" => 28,
            "is-directory" => 31,
            "too-many-files" => 33,
            "no-entry" => 44,
            "no-memory" => 48,
            "no-space" => 51,
            "not-directory" => 54,
            "not-empty" => 55,
            "permission" => 63,
            "read-only" => 69,
            "stale" => 72,
            "cross-device" => 75,
            _ => 29,
        },
    }
}

/// Go `storageBoundary`: non-byte volume capacities stop the run.
pub(crate) fn boundary(error: &Error) -> Option<Stop> {
    match error {
        Error::Capacity { resource, .. } if *resource != "volume_bytes" => {
            Some(Stop::Boundary("capacity", capacity_dimension(resource)))
        }
        _ => None,
    }
}

fn capacity_dimension(resource: &str) -> &'static str {
    match resource {
        "pending_operations" => "storage-pending_operations",
        "transitions" => "storage-transitions",
        "image-bytes" => "storage-image-bytes",
        _ => "storage-capacity",
    }
}

/// Go's `err.Error()` text for errors that surface outside errno replies.
pub(crate) fn error_text(error: &Error) -> String {
    match error {
        Error::Invalid(message) => (*message).into(),
        Error::Capacity {
            resource,
            required,
            maximum,
        } => format!(
            "simulation volume capacity exhausted: resource={resource} required={required} maximum={maximum}"
        ),
        Error::Errno(name) => match *name {
            "bad-descriptor" => "bad file descriptor",
            "busy" => "device or resource busy",
            "exists" => "file exists",
            "file-too-large" => "file too large",
            "invalid" => "invalid argument",
            "is-directory" => "is a directory",
            "too-many-files" => "too many open files",
            "no-entry" => "no such file or directory",
            "no-space" => "no space left on device",
            "not-directory" => "not a directory",
            "not-empty" => "directory not empty",
            "read-only" => "read-only file system",
            "stale" => "stale NFS file handle",
            "cross-device" => "invalid cross-device link",
            "eof" => "EOF",
            other => other,
        }
        .into(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn go_generated_fault_decision_corpus() {
        use base64::{Engine, engine::general_purpose::STANDARD};
        let cases: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../internal/model/wasi/testdata/volume-conformance/vectors/fault-decisions.json"
        ))
        .unwrap();
        for case in cases.as_array().unwrap() {
            let model: StorageModel = serde_json::from_value(case["storage"].clone()).unwrap();
            let mut storage =
                Storage::new(model, case["transcript_limit"].as_u64().unwrap()).unwrap();
            for step in case["steps"].as_array().unwrap() {
                let result = storage.decision(
                    step["resource"].as_str().unwrap(),
                    step["operation"].as_str().unwrap(),
                    step["count"].as_u64().unwrap(),
                    step["now"].as_u64().unwrap(),
                );
                let (written, errno, reply) = match result {
                    Ok((written, errno)) => (written, errno, serde_json::json!({})),
                    Err(Stop::Boundary(kind, detail)) => (
                        0,
                        0,
                        serde_json::json!({"boundary":format!("{kind}:{detail}")}),
                    ),
                    Err(Stop::Crash(fault)) => (0, 0, serde_json::json!({"storage_crash":fault})),
                    Err(error) => panic!("unexpected decision stop {error:?}"),
                };
                assert_eq!(
                    written,
                    step["written"].as_u64().unwrap(),
                    "{}",
                    case["name"]
                );
                assert_eq!(u64::from(errno), step["errno"].as_u64().unwrap());
                assert_eq!(
                    serde_json::to_vec(&reply).unwrap(),
                    STANDARD
                        .decode(step["reply_base64"].as_str().unwrap())
                        .unwrap()
                );
                assert_eq!(
                    storage.log,
                    STANDARD
                        .decode(step["events_base64"].as_str().unwrap_or_default())
                        .unwrap()
                );
                assert_eq!(
                    serde_json::to_value(&storage.occurrences).unwrap(),
                    step["occurrences"]
                );
                assert_eq!(storage.sequence, step["sequence"].as_u64().unwrap());
                assert_eq!(storage.clock, step["clock"].as_i64().unwrap());
            }
        }
    }

    #[test]
    fn go_generated_errno_mapping_corpus() {
        let cases: serde_json::Value = serde_json::from_str(include_str!(
            "../../../../internal/model/wasi/testdata/volume-conformance/vectors/error-mapping.json"
        ))
        .unwrap();
        for case in cases.as_array().unwrap() {
            let name = match case["name"].as_str().unwrap() {
                "nil" => "nil",
                "eof" => "eof",
                "closed" | "bad-descriptor" => "bad-descriptor",
                "io" => "io",
                "stale" => "stale",
                "no-entry" => "no-entry",
                "no-space" => "no-space",
                "read-only-access" => "read-only-access",
                "busy" => "busy",
                "exists" => "exists",
                "file-too-large" => "file-too-large",
                "invalid" => "invalid",
                "is-directory" => "is-directory",
                "too-many-files" => "too-many-files",
                "no-memory" => "no-memory",
                "not-directory" => "not-directory",
                "not-empty" => "not-empty",
                "permission" => "permission",
                "read-only" => "read-only",
                "cross-device" => "cross-device",
                "unmapped" => "unmapped",
                "volume_bytes" => "volume_bytes",
                "pending_operations" => "pending_operations",
                "transitions" => "transitions",
                "x" => "x",
                other => panic!("unknown mapping {other}"),
            };
            let error = match case["kind"].as_str().unwrap() {
                "nil" => None,
                "errno" => Some(Error::Errno(name)),
                "invalid" => Some(Error::Invalid(name)),
                "capacity" => Some(Error::Capacity {
                    resource: name,
                    required: 3,
                    maximum: 2,
                }),
                other => panic!("unknown error kind {other}"),
            };
            assert_eq!(
                u64::from(error.as_ref().map_or(0, errno)),
                case["errno"].as_u64().unwrap()
            );
            let observed = match error.as_ref().and_then(boundary) {
                Some(Stop::Boundary(kind, detail)) => format!("WASI {kind}: {detail}"),
                None => String::new(),
                other => panic!("unexpected boundary {other:?}"),
            };
            assert_eq!(observed, case["boundary"].as_str().unwrap());
        }
    }
}
