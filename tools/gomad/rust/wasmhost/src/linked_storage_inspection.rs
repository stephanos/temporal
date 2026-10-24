mod state;

use crate::{linked::StorageInput, volume::RecoveryImage};
use std::collections::{BTreeMap, BTreeSet};

pub use state::StateObservation;

/// # Errors
/// Refuses invalid retained storage images without executing guests or host callbacks.
pub fn recovery_volumes(
    input: &StorageInput,
    image: &[u8],
    expected_sha256: &str,
    maximum: u64,
) -> Result<Vec<String>, String> {
    Ok(admit_image(input, image, expected_sha256, maximum)?.volume_ids())
}

/// Admits a recovery image and retains its durable objects and persistence effects.
/// # Errors
/// Refuses invalid images through the same decoder as recovery volume inspection.
pub fn recovery_image(
    input: &StorageInput,
    image: &[u8],
    expected_sha256: &str,
    maximum: u64,
) -> Result<RecoveryObservation, String> {
    Ok(admit_image(input, image, expected_sha256, maximum)?.observation())
}

fn admit_image(
    input: &StorageInput,
    image: &[u8],
    expected_sha256: &str,
    maximum: u64,
) -> Result<RecoveryImage, String> {
    if maximum == 0 || maximum > 256 << 20 || image.len() as u64 > maximum {
        return Err("retained storage image exceeds inspection bound".into());
    }
    let descriptor = input.descriptor();
    descriptor.validate().map_err(|error| error.to_string())?;
    if descriptor.model.volumes().len() > 64 {
        return Err("retained storage volume count exceeds inspection bound".into());
    }
    RecoveryImage::decode(
        image,
        expected_sha256,
        &descriptor.model_identity,
        descriptor.model.volumes(),
        descriptor.model.limits,
        descriptor.model.lock_databases(),
        maximum,
    )
    .map_err(|error| format!("retained storage image: {error:?}"))
}

/// # Errors
/// Refuses unauthenticated storage state or inconsistent pending crash context.
pub fn validate_state(
    input: &StorageInput,
    raw: &[u8],
    log: &[u8],
    maximum: u64,
) -> Result<(), String> {
    state::validate(input, raw, log, maximum)
}

/// Admits authenticated state sections and exposes the validated decoded observations.
/// # Errors
/// Refuses invalid framing, log binding, snapshots, enumerations or crash contexts.
pub fn decode_state<'a>(
    input: &StorageInput,
    raw: &'a [u8],
    log: &[u8],
    maximum: u64,
) -> Result<StateObservation<'a>, String> {
    state::decode(input, raw, log, maximum)
}

#[derive(Debug, PartialEq, Eq)]
pub struct RecoveryObservation {
    pub(crate) model_identity: String,
    pub(crate) identity: String,
    pub(crate) next_inode: u64,
    pub(crate) generation: u64,
    pub(crate) occurrences: BTreeMap<String, u64>,
    pub(crate) volumes: Vec<RecoveryVolume>,
}

impl RecoveryObservation {
    #[must_use]
    pub fn model_identity(&self) -> &str {
        &self.model_identity
    }
    #[must_use]
    pub fn identity(&self) -> &str {
        &self.identity
    }
    #[must_use]
    pub fn next_inode(&self) -> u64 {
        self.next_inode
    }
    #[must_use]
    pub fn generation(&self) -> u64 {
        self.generation
    }
    #[must_use]
    pub fn occurrences(&self) -> &BTreeMap<String, u64> {
        &self.occurrences
    }
    #[must_use]
    pub fn volumes(&self) -> &[RecoveryVolume] {
        &self.volumes
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct RecoveryVolume {
    pub(crate) id: String,
    pub(crate) mount: String,
    pub(crate) capacity_bytes: u64,
    pub(crate) objects: BTreeMap<u64, RecoveryObject>,
    pub(crate) paths: BTreeMap<String, u64>,
    pub(crate) owned: BTreeSet<u64>,
    pub(crate) pending: Vec<RecoveryOperation>,
    pub(crate) history: Vec<RecoveryOperation>,
    pub(crate) next_operation: u64,
    pub(crate) last: BTreeMap<String, u64>,
}

impl RecoveryVolume {
    #[must_use]
    pub fn id(&self) -> &str {
        &self.id
    }
    #[must_use]
    pub fn mount(&self) -> &str {
        &self.mount
    }
    #[must_use]
    pub fn capacity_bytes(&self) -> u64 {
        self.capacity_bytes
    }
    #[must_use]
    pub fn objects(&self) -> &BTreeMap<u64, RecoveryObject> {
        &self.objects
    }
    #[must_use]
    pub fn paths(&self) -> &BTreeMap<String, u64> {
        &self.paths
    }
    #[must_use]
    pub fn owned(&self) -> &BTreeSet<u64> {
        &self.owned
    }
    #[must_use]
    pub fn pending(&self) -> &[RecoveryOperation] {
        &self.pending
    }
    #[must_use]
    pub fn history(&self) -> &[RecoveryOperation] {
        &self.history
    }
    #[must_use]
    pub fn next_operation(&self) -> u64 {
        self.next_operation
    }
    #[must_use]
    pub fn last(&self) -> &BTreeMap<String, u64> {
        &self.last
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct RecoveryObject {
    pub(crate) inode: u64,
    pub(crate) kind: u64,
    pub(crate) mode: u32,
    pub(crate) mod_time: i64,
    pub(crate) data: Vec<u8>,
}

impl RecoveryObject {
    #[must_use]
    pub fn inode(&self) -> u64 {
        self.inode
    }
    #[must_use]
    pub fn kind(&self) -> u64 {
        self.kind
    }
    #[must_use]
    pub fn mode(&self) -> u32 {
        self.mode
    }
    #[must_use]
    pub fn mod_time(&self) -> i64 {
        self.mod_time
    }
    #[must_use]
    pub fn data(&self) -> &[u8] {
        &self.data
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct RecoveryOperation {
    pub(crate) id: u64,
    pub(crate) kind: String,
    pub(crate) dependencies: Vec<u64>,
    pub(crate) stores: Vec<String>,
    pub(crate) effect: RecoveryEffect,
}

impl RecoveryOperation {
    #[must_use]
    pub fn id(&self) -> u64 {
        self.id
    }
    #[must_use]
    pub fn kind(&self) -> &str {
        &self.kind
    }
    #[must_use]
    pub fn dependencies(&self) -> &[u64] {
        &self.dependencies
    }
    #[must_use]
    pub fn stores(&self) -> &[String] {
        &self.stores
    }
    #[must_use]
    pub fn effect(&self) -> &RecoveryEffect {
        &self.effect
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct RecoveryEffect {
    pub(crate) kind: String,
    pub(crate) inode: u64,
    pub(crate) parent: u64,
    pub(crate) parents: Vec<u64>,
    pub(crate) offset: u64,
    pub(crate) size: u64,
    pub(crate) mode: u32,
    pub(crate) mod_time: i64,
    pub(crate) data: Vec<u8>,
    pub(crate) object: Option<RecoveryObject>,
    pub(crate) namespace: Vec<RecoveryNamespace>,
}

impl RecoveryEffect {
    #[must_use]
    pub fn kind(&self) -> &str {
        &self.kind
    }
    #[must_use]
    pub fn inode(&self) -> u64 {
        self.inode
    }
    #[must_use]
    pub fn parent(&self) -> u64 {
        self.parent
    }
    #[must_use]
    pub fn parents(&self) -> &[u64] {
        &self.parents
    }
    #[must_use]
    pub fn offset(&self) -> u64 {
        self.offset
    }
    #[must_use]
    pub fn size(&self) -> u64 {
        self.size
    }
    #[must_use]
    pub fn mode(&self) -> u32 {
        self.mode
    }
    #[must_use]
    pub fn mod_time(&self) -> i64 {
        self.mod_time
    }
    #[must_use]
    pub fn data(&self) -> &[u8] {
        &self.data
    }
    #[must_use]
    pub fn object(&self) -> Option<&RecoveryObject> {
        self.object.as_ref()
    }
    #[must_use]
    pub fn namespace(&self) -> &[RecoveryNamespace] {
        &self.namespace
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct RecoveryNamespace {
    pub(crate) path: String,
    pub(crate) inode: u64,
}

impl RecoveryNamespace {
    #[must_use]
    pub fn path(&self) -> &str {
        &self.path
    }
    #[must_use]
    pub fn inode(&self) -> u64 {
        self.inode
    }
}
