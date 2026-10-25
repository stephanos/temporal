#![allow(
    dead_code,
    unused_imports,
    reason = "Standalone fixture includes actual private production graph"
)]
include!("../../lib.rs");
fn main() {
    let owner = cycle::process::CycleOwner::initialize_exclusive().expect("exclusive cycle owner");
    let family = std::env::var("CUSTODY_FAMILY").unwrap_or_else(|_| "source".into());
    match family.as_str() {
        "source" => cycle::custody::run_cases(&owner).expect("source custody cases"),
        "inputs" => cycle::preparation::inputs::run_cases(&owner).expect("preparation input cases"),
        "build" => cycle::build::run_cases(&owner).expect("build role cases"),
        "bindings" => {
            cycle::preparation::bindings::run_cases(&owner).expect("freeze binding cases");
        }
        _ => panic!("unmapped custody fixture family"),
    }
}
