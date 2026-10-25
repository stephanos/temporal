//! Cargo input freshness after moving the bootstrap catalogs into features.

#![forbid(unsafe_code)]

use std::{fs, path::Path};

#[test]
fn api_build_tracks_existing_inputs_and_feature_catalogs() -> Result<(), Box<dyn std::error::Error>>
{
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR"));
    let output =
        Path::new(env!("OUT_DIR")).parent().ok_or("missing build directory")?.join("output");
    let mut inputs = Vec::new();
    for line in fs::read_to_string(output)?.lines() {
        if let Some(input) = line.strip_prefix("cargo:rerun-if-changed=") {
            let path = manifest.join(input);
            assert!(path.exists(), "missing API build input: {}", path.display());
            inputs.push(path.canonicalize()?);
        }
    }
    let features = manifest.join("../feature").canonicalize()?;
    assert!(inputs.contains(&features));
    assert!(inputs.contains(&features.join("activity/api.rs")));
    assert!(inputs.contains(&features.join("activity/claims.rs")));
    let shared_claims = manifest.join("../foundation/claims.rs").canonicalize()?;
    assert!(inputs.contains(&shared_claims));
    let table = manifest.join("../foundation/service.rs").canonicalize()?;
    assert!(inputs.contains(&table), "untracked service table: {}", table.display());
    for schema in ["temporal.proto", "probe.proto"] {
        let schema = manifest.join("../proto/umpire/temporal/v1").join(schema).canonicalize()?;
        assert!(inputs.contains(&schema), "untracked schema: {}", schema.display());
    }
    Ok(())
}
