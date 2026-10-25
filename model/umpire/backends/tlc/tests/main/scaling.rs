//! Source-lifted TLC remains usable when a native reference exploration is cut.

#![forbid(unsafe_code)]

use std::{collections::BTreeSet, fs, path::Path};

use umpire_tla::{Lift, Tla};
use umpire_tlc::{State, Tlc, Value, Verdict};

#[path = "../fixtures/scaling_model.rs"]
mod authored;

#[test]
fn typed_module_completes_independently_of_a_cut_native_reference()
-> Result<(), Box<dyn std::error::Error>> {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let reference = umpire_checker::Checker::explore(&authored::counter::world(), 1)?;
    assert!(reference.counts.cut);
    assert_eq!(reference.counts.reached, 1);

    let lifted = Lift::parse("scaling_model.rs", include_str!("../fixtures/scaling_model.rs"))?;
    let module = Tla::emit(&lifted)?;
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let tlc = Tlc::discover(&root)?;
    let result = tlc.check_module(&module)?;
    let evidence = root.join(".flow/tmp/tlc-controls");
    fs::create_dir_all(&evidence)?;
    fs::write(
        evidence.join("source_scaling.txt"),
        format!(
            "native reference: cut=true; reached=1\nTLC projected states: {:?}\nTLC raw states: {}\nTLC properties: {:?}\n{}",
            result.states, result.raw_state_count, result.properties, result.output
        ),
    )?;
    let expected = BTreeSet::from([
        State::from([("value".into(), Value::Int(0))]),
        State::from([("value".into(), Value::Int(1))]),
        State::from([("value".into(), Value::Int(2))]),
        State::from([("value".into(), Value::Int(3))]),
    ]);
    assert_eq!(result.states, expected);
    assert_eq!(result.properties["Invariant_within_range"].verdict, Verdict::Verified);
    let claim = &module.metadata.progress[0];
    assert_eq!(result.properties[&claim.operator].verdict, Verdict::Verified);
    Ok(())
}
