//! Sequential claim statements retain their independent evaluation obligations.
#![forbid(unsafe_code)]

use std::{fs, path::Path};
use umpire_tla::{Lift, Tla};
use umpire_tlc::Tlc;

#[path = "fixtures/sequential_checks.rs"]
mod authored;

#[test]
fn failed_check_does_not_hide_a_later_division_failure() -> Result<(), Box<dyn std::error::Error>> {
    let native = umpire_checker::Checker::explore(&authored::sequential::world(), 10)?;
    let report = native.verify_all()?.to_string();
    assert!(report.contains("divide by zero"), "{report}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let module = Tla::emit(&Lift::parse(
        "sequential_checks.rs",
        include_str!("fixtures/sequential_checks.rs"),
    )?)?;
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let error = Tlc::discover(&root)?
        .check_module(&module)
        .err()
        .ok_or("a failed check skipped the later division failure")?;
    assert!(error.to_string().contains("division by zero"), "{error}");
    let evidence = root.join(".flow/tmp/tlc-controls");
    fs::create_dir_all(&evidence)?;
    fs::write(evidence.join("sequential_checks.txt"), error.to_string())?;
    Ok(())
}
