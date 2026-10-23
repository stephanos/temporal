mod state;

use crate::{linked::StorageInput, volume::RecoveryImage};

/// # Errors
/// Refuses invalid retained storage images without executing guests or host callbacks.
pub fn recovery_volumes(
    input: &StorageInput,
    image: &[u8],
    expected_sha256: &str,
    maximum: u64,
) -> Result<Vec<String>, String> {
    if maximum == 0 || maximum > 256 << 20 || image.len() as u64 > maximum {
        return Err("retained storage image exceeds inspection bound".into());
    }
    let descriptor = input.descriptor();
    descriptor.validate().map_err(|error| error.to_string())?;
    if descriptor.model.volumes().len() > 64 {
        return Err("retained storage volume count exceeds inspection bound".into());
    }
    let recovered = RecoveryImage::decode(
        image,
        expected_sha256,
        &descriptor.model_identity,
        descriptor.model.volumes(),
        descriptor.model.limits,
        descriptor.model.lock_databases(),
        maximum,
    )
    .map_err(|error| format!("retained storage image: {error:?}"))?;
    Ok(recovered.volume_ids())
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
