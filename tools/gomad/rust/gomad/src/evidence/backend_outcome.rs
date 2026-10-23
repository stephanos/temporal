use super::{canonical, store::Opened};
use gomad_wasmhost::linked::NativeOutputAccounting;
use serde::Deserialize;

pub(crate) fn output(raw: &[u8], claimed: u64, retained: u64, limit: u64) -> Result<(), String> {
    let accounting: NativeOutputAccounting = serde_json::from_slice(raw)
        .map_err(|error| format!("backend outcome output accounting: {error}"))?;
    if accounting.schema != "gomad.native-output-accounting/v1"
        || canonical::encode(&accounting)? != raw
        || accounting.offered_output_bytes != claimed
        || claimed > limit
        || retained > claimed
    {
        return Err("backend outcome output accounting mismatch".into());
    }
    Ok(())
}

#[derive(Deserialize)]
struct Claims {
    schema: String,
    termination: String,
    exit_code: Option<u32>,
    fuel_remaining: u64,
    output_bytes: u64,
    #[serde(default)]
    output_accounting_sha256: Option<String>,
}

pub(crate) fn authenticate(opened: &Opened) -> Result<(), String> {
    let manifest = opened.manifest();
    let name = "evidence/backend-outcome.json";
    if !manifest.files.contains_key(name) {
        return Ok(());
    }
    let claims: Claims = serde_json::from_slice(&opened.payload(name, 128 << 20)?)
        .map_err(|error| format!("backend outcome claims: {error}"))?;
    if claims.fuel_remaining > manifest.record.limits.fuel {
        return Err("backend outcome fuel exceeds admitted limit".into());
    }
    if claims.termination != manifest.record.outcome.termination
        || claims.exit_code != manifest.record.outcome.exit_code
    {
        return Err("backend outcome terminal claim contradicts record".into());
    }
    match claims.schema.as_str() {
        "gomad.backend-outcome/v1" => Ok(()),
        "gomad.backend-outcome/v2" => {
            let name = "evidence/native-output-accounting";
            let raw = opened.payload(name, 4096)?;
            let digest = canonical::digest(&raw);
            if claims.output_accounting_sha256.as_deref() != Some(digest.as_str())
                || manifest.record.streams.get(name).is_none_or(|stream| {
                    stream.sha256 != digest || stream.bytes != raw.len() as u64
                })
            {
                return Err("backend outcome output accounting identity mismatch".into());
            }
            let retained =
                ["output/stdout", "output/stderr"]
                    .into_iter()
                    .try_fold(0_u64, |total, name| {
                        total
                            .checked_add(manifest.files.get(name).map_or(0, |entry| entry.bytes))
                            .ok_or("backend outcome output size overflow")
                    })?;
            output(
                &raw,
                claims.output_bytes,
                retained,
                manifest.record.limits.output_bytes,
            )
        }
        _ => Err("unsupported backend outcome schema".into()),
    }
}
