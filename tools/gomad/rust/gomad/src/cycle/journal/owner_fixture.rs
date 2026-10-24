#![allow(
    dead_code,
    unused_imports,
    reason = "Standalone fixture includes the actual private production module graph"
)]
include!("../../lib.rs");

fn main() {
    if std::env::args().nth(1).as_deref() == Some("--reopen") {
        cycle::journal::tests::restart_case(std::env::args_os().nth(2).unwrap().into());
        return;
    }
    let owner = cycle::process::CycleOwner::initialize_exclusive().unwrap();
    cycle::journal::tests::owner_cases(&owner);
    cycle::journal::tests::floor_cases(&owner);
    println!("JOURNAL_OWNER_CASES 2 PASS");
}
