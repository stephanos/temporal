//! A small result must not conceal narrow signed remainder overflow.
use std::{fs, path::Path};
use umpire_dsl::Machine;
use umpire_lift::{Lift, Tla};
use umpire_tlc::Tlc;

#[path = "fixtures/remainder_i16.rs"]
mod signed_i16;
#[path = "fixtures/remainder_i8.rs"]
mod signed_i8;

fn overflow<M: Machine>(
    world: &M,
    name: &str,
    source: &str,
) -> Result<(), Box<dyn std::error::Error>> {
    let native = umpire_checker::Checker::explore(world, 10)?.verify_all()?.to_string();
    assert!(native.contains("remainder with overflow"), "{native}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let module = Tla::emit(&Lift::parse(name, source)?)?;
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let result = Tlc::discover(&root)?.check_module(&module);
    let evidence = root.join(".flow/tmp/tlc-controls");
    fs::create_dir_all(&evidence)?;
    fs::write(
        evidence.join(format!("narrow_remainder_{name}.txt")),
        format!(
            "native:\n{native}\nbackend:\n{}",
            match &result {
                Ok(result) => result.output.clone(),
                Err(error) => error.to_string(),
            }
        ),
    )?;
    let error = result.err().ok_or("TLC accepted narrow MIN % -1 despite native overflow")?;
    let error = error.to_string();
    assert!(error.contains("TLC evaluation error in the next-state relation"), "{error}");
    assert!(error.contains("The first argument of Assert evaluated to FALSE"), "{error}");
    assert!(error.contains("Rust signed division/remainder overflow"), "{error}");
    Ok(())
}

#[test]
fn i8_min_remainder_negative_one_refuses() -> Result<(), Box<dyn std::error::Error>> {
    overflow(&signed_i8::remainder::world(), "i8", include_str!("fixtures/remainder_i8.rs"))
}

#[test]
fn i16_min_remainder_negative_one_refuses() -> Result<(), Box<dyn std::error::Error>> {
    overflow(&signed_i16::remainder::world(), "i16", include_str!("fixtures/remainder_i16.rs"))
}
