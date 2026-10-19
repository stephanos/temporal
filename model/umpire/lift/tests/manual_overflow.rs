//! Both engines retain a checked overflow before the final next-state assignment.
use std::{fs, path::Path};
use umpire_lift::{Lift, Tla};
use umpire_tlc::Tlc;

#[path = "fixtures/manual_overflow.rs"]
mod authored;

#[test]
fn overwritten_increment_overflow_is_visible_to_native_and_tlc()
-> Result<(), Box<dyn std::error::Error>> {
    let native = umpire_checker::Checker::explore(&authored::World, 10)?;
    let report = native.verify_all()?.to_string();
    assert!(report.contains("overflow"), "{report}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let module = Tla::emit(&Lift::parse(
        "manual_overflow.rs",
        include_str!("fixtures/manual_overflow.rs"),
    )?)?;
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let evidence = root.join(".flow/tmp/tlc-controls");
    fs::create_dir_all(&evidence)?;
    fs::write(evidence.join("manual_overflow_native.txt"), report)?;
    let result = Tlc::discover(&root)?.check_module(&module);
    fs::write(
        evidence.join("manual_overflow_result.txt"),
        match &result {
            Ok(result) => result.output.clone(),
            Err(error) => error.to_string(),
        },
    )?;
    let error =
        result.err().ok_or("TLC silently accepted MAX + 1 followed by resetting the field")?;
    assert!(error.to_string().contains("checked Rust integer overflow"), "{error}");
    fs::write(evidence.join("manual_overflow_refusal.txt"), error.to_string())?;
    Ok(())
}
