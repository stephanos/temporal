mod config;
mod crash;
mod faults;
mod hash;
mod image;
mod state;

#[cfg(not(feature = "conformance"))]
pub(crate) use config::MAXIMUM_PATH_BYTES;
pub(crate) use config::StorageModel;
#[cfg(any(test, feature = "conformance"))]
pub(crate) use crash::CrashExploration;
pub(crate) use crash::{CrashEnumeration, CrashEnumerationLimits};
pub(crate) use faults::{
    Image, Stop as StorageStop, Storage, boundary as storage_boundary, errno as storage_errno,
    error_text as storage_error_text,
};
#[cfg(any(test, feature = "conformance"))]
pub(crate) use image::IMAGE_VERSION;
#[cfg(not(feature = "conformance"))]
pub(crate) use image::RecoveryImage;
#[cfg(not(feature = "conformance"))]
pub(crate) use state::Operation;
pub(crate) use state::{Error, Handle, Kind, OpenFlags};

#[cfg(not(feature = "conformance"))]
pub(crate) use state::{VolumeSnapshot, volume_snapshot_identity};
