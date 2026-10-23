//! Qualification of one WASM cell on the linked Rust host and the
//! installation-free retained Result reader.

pub(crate) mod cells;
pub(crate) mod cost;
#[cfg(feature = "determinism-fixture")]
pub(crate) mod fixture;
pub(crate) mod report;
pub(crate) mod result;
pub(crate) mod spec;
pub(crate) mod support;
pub(crate) mod usage;

pub(crate) mod workloads;

pub(crate) fn require_release_role() -> Result<(), String> {
    if gomad_wasmhost::linked::Conditions::compiled()
        != gomad_wasmhost::linked::Conditions::Ordinary
    {
        return Err("qualification refuses a stress-jitter or diagnostic-role binary".into());
    }
    Ok(())
}
