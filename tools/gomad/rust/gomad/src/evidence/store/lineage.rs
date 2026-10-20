use serde::Deserialize;

pub const ENVELOPE: &[u8] = b"{\"format\":\"gomad.artifact-storage/v2\"}";

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Envelope {
    format: String,
}

pub fn validate(bytes: &[u8]) -> Result<(), String> {
    let envelope: Envelope = serde_json::from_slice(bytes)
        .map_err(|_| "unsupported artifact storage format: malformed storage.json")?;
    if envelope.format != "gomad.artifact-storage/v2" {
        return Err(format!(
            "unsupported artifact storage format {:?}",
            envelope.format
        ));
    }
    Ok(())
}
