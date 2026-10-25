//! Lexical source scopes are independently checked against compiled Rust.
#![forbid(unsafe_code)]

use std::{collections::BTreeSet, fs, path::Path};
use umpire_tla::{Lift, Tla};
use umpire_tlc::{State, Tlc, Value, Verdict};

#[path = "../fixtures/shadowing.rs"]
mod authored;

#[test]
fn nested_blocks_and_effect_helpers_preserve_caller_locals()
-> Result<(), Box<dyn std::error::Error>> {
    let module =
        Tla::emit(&Lift::parse("shadowing.rs", include_str!("../fixtures/shadowing.rs"))?)?;
    assert_eq!(
        module.tla.matches("n' = n").count(),
        4,
        "saved caller binding changed in emitted effects"
    );
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let native = umpire_checker::Checker::explore(&authored::lexical::world(), 100)?;
    assert!(!native.counts.cut);
    native.verify_all()?.assert_verified();
    let expected = native
        .states()
        .map(|(_, state)| {
            State::from([
                ("n".into(), Value::Int(i128::from(state.n))),
                ("done".into(), Value::Bool(state.done)),
            ])
        })
        .collect::<BTreeSet<_>>();
    assert_eq!(expected.len(), 2);
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..");
    let result = Tlc::discover(&root)?.check_module(&module)?;
    let evidence = root.join("build/evidence/tlc-controls");
    fs::create_dir_all(&evidence)?;
    fs::write(evidence.join("shadowing.txt"), &result.output)?;
    assert_eq!(result.states, expected);
    assert_eq!(result.properties["Invariant_saved_value"].verdict, Verdict::Verified);
    Ok(())
}
