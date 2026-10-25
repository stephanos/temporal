//! Typed persistent-vector source admission controls.
#![forbid(unsafe_code)]

use umpire_tla::{Lift, Tla, Type};

#[path = "fixtures/vector.rs"]
mod authored;

#[path = "fixtures/vector_fallback.rs"]
mod fallback_authored;

#[test]
fn bounded_vector_operations_are_lifted_with_safe_defaulted_get()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("fixtures/vector.rs");
    let module = Tla::emit(&Lift::parse("vector.rs", source)?)?;
    assert_eq!(module.metadata.fields[0].ty, Type::Seq(Box::new(Type::Int { min: 0, max: 255 })));
    assert_eq!(module.metadata.invariants.len(), 2);
    Ok(())
}

/// Rust computes lengths in `usize`, TLC in 32-bit integers: a sum or product of a length,
/// including an index computed from one, refuses at its span; a difference, quotient or
/// remainder is admitted.
#[test]
fn length_sums_and_products_refuse_at_their_span() -> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("fixtures/vector.rs");
    for (claim, refused) in [
        ("self.values.len() + 1 <= 3", true),
        ("1 + self.values.len() <= 3", true),
        ("self.values.len() * 2 <= 4", true),
        ("self.values.get(self.values.len() + 0).is_none()", true),
        ("self.values.len() - self.values.len() / 2 <= 2", false),
        ("self.values.len() % 2 < 2", false),
        ("self.values.get(self.values.len() / 2).copied().unwrap_or(7) == 7", false),
    ] {
        let edited = source.replace(
            "check!(self.values.len() <= 2);",
            &format!("check!(self.values.len() <= 2); check!({claim});"),
        );
        match Lift::parse("vector.rs", &edited) {
            Err(error) if refused => {
                assert!(error.message.contains("`+` and `*` on a Vector length"), "{error}");
                assert_eq!(error.line, 40, "{claim}: {error}");
            }
            Ok(_) if !refused => {}
            other => return Err(format!("{claim}: {:?}", other.map(|_| ())).into()),
        }
    }
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn real_vector_native_and_tlc_states_and_properties_agree() -> Result<(), Box<dyn std::error::Error>>
{
    use std::{collections::BTreeSet, fs, path::Path};
    use umpire_tlc::{State, Value, Verdict};
    let world = authored::collection::world();
    let native = umpire_checker::Checker::explore(&world, 100)?;
    assert!(!native.counts.cut);
    let report = native.verify_all()?;
    let native_properties: Vec<_> = report
        .claims
        .iter()
        .map(|claim| (claim.name.as_str(), format!("{:?}", claim.outcome)))
        .collect();
    assert_eq!(
        native_properties,
        [("contents", "Verified".to_owned()), ("never_two", "Refuted".to_owned())]
    );
    let expected = native
        .states()
        .map(|(_, state)| {
            State::from([(
                "values".into(),
                Value::Seq(
                    state.values.iter().map(|value| Value::Int(i128::from(*value))).collect(),
                ),
            )])
        })
        .collect::<BTreeSet<_>>();
    assert_eq!(expected.len(), 3);
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("Vector native checks passed; no TLC evidence (set UMPIRE_BACKENDS=require)");
        return Ok(());
    }
    let source = include_str!("fixtures/vector.rs");
    let saved = saved_length();
    let chained = saved.replace(
        "let length = self.values.len();",
        "let first = self.values.len(); let second = first; let length = second;",
    );
    let conditional = conditional_length();
    let after_if = untouched_vector();
    let conditional_boolean = conditional_guard(
        "let allowed = if condition { self.values.len() < 2 } else { self.values.len() < 2 }; if allowed",
    );
    let conditional_limit =
        conditional_guard("if self.values.len() < (if condition { 2 } else { 2 })");
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    for (name, source) in [
        ("vector", source),
        ("vector_saved_length", saved.as_str()),
        ("vector_chained_length", chained.as_str()),
        ("vector_conditional_length", conditional.as_str()),
        ("vector_after_if", after_if.as_str()),
        ("vector_conditional_boolean", conditional_boolean.as_str()),
        ("vector_conditional_limit", conditional_limit.as_str()),
    ] {
        let module = Tla::emit(&Lift::parse("vector.rs", source)?)?;
        let result = umpire_tlc::Tlc::discover(&root)?.check_module(&module)?;
        fs::create_dir_all(root.join(".flow/tmp/tlc-controls"))?;
        fs::write(root.join(format!(".flow/tmp/tlc-controls/{name}.txt")), &result.output)?;
        assert_eq!(result.states, expected);
        assert_eq!(result.properties["Invariant_contents"].verdict, Verdict::Verified);
        assert_eq!(result.properties["Invariant_never_two"].verdict, Verdict::Refuted);
    }
    Ok(())
}

fn saved_length() -> String {
    include_str!("fixtures/vector.rs")
        .replace(
            "User::Push if self.values.len() < 2 => {",
            "User::Push => { let length = self.values.len(); if length < 2 {",
        )
        .replace("self.values.push_back(7);", "self.values.push_back(7); }")
}

fn conditional_length() -> String {
    saved_length().replace("let length = self.values.len();", "let condition = self.values.is_empty() && self.values.is_empty(); let length = if condition { self.values.len() } else { self.values.len() };")
}

fn untouched_vector() -> String {
    saved_length().replace(
        "let length = self.values.len();",
        "if self.values.is_empty() {} let length = self.values.len();",
    )
}

#[test]
fn unrelated_conditionals_preserve_vector_length_identity() -> Result<(), Box<dyn std::error::Error>>
{
    let source = untouched_vector();
    for source in [
        source.clone(),
        source.replace("let length = self.values.len(); if length < 2", "if self.values.len() < 2"),
        source.replace(
            "if self.values.is_empty() {}",
            "if self.values.is_empty() { if self.values.len() == 0 {} }",
        ),
        source
            .replace("pub values: Vector<u8>,", "pub values: Vector<u8>, pub count: u8,")
            .replace("if self.values.is_empty() {}", "if self.count == 0 { self.count += 1; }"),
    ] {
        Lift::parse("untouched_vector.rs", &source)?;
        let unbounded =
            source.replace("length < 2", "true").replace("self.values.len() < 2", "true");
        assert!(Lift::parse("unbounded_untouched_vector.rs", &unbounded).is_err());
    }
    Ok(())
}

fn conditional_guard(comparison: &str) -> String {
    saved_length().replace(
        "let length = self.values.len(); if length < 2",
        &format!("let condition = self.values.is_empty() && self.values.is_empty(); {comparison}"),
    )
}

#[test]
fn conditional_guards_preserve_common_bound_facts() -> Result<(), Box<dyn std::error::Error>> {
    for (positive, negative) in [
        (
            "let allowed = if condition { self.values.len() < 2 } else { self.values.len() < 2 }; if allowed",
            "let allowed = if condition { self.values.len() < 2 } else { true }; if allowed",
        ),
        (
            "if self.values.len() < (if condition { 2 } else { 2 })",
            "if self.values.len() < (if condition { 2 } else { self.values.len() })",
        ),
    ] {
        Lift::parse("conditional_bound.rs", &conditional_guard(positive))?;
        let error = Lift::parse("unknown_conditional_bound.rs", &conditional_guard(negative))
            .err()
            .ok_or("unknown conditional bound admitted")?;
        assert!(error.message.contains("bounded length guard"), "{error}");
    }
    Ok(())
}

#[test]
fn unsupported_vector_operations_and_unbounded_growth_refuse()
-> Result<(), Box<dyn std::error::Error>> {
    let source = include_str!("fixtures/vector.rs");
    for (source, reason) in [
        (conditional_length().replace("else { self.values.len() }", "else { 0 }"), "bounded"),
        (
            conditional_length()
                .replace(
                    "pub values: Vector<u8>,",
                    "pub values: Vector<u8>, pub other: Vector<u8>,",
                )
                .replace("else { self.values.len() }", "else { self.other.len() }"),
            "bounded",
        ),
        (
            untouched_vector()
                .replace(
                    "pub values: Vector<u8>,",
                    "pub values: Vector<u8>, pub other: Vector<u8>,",
                )
                .replace(
                    "if self.values.is_empty() {}",
                    "if self.values.is_empty() { self.values = self.other; }",
                ),
            "bounded",
        ),
        (source.replace("self.values.push_back(7);", "self.values.pop_back();"), "pop_back"),
        (source.replace("self.values.len() < 2", "true"), "bounded"),
        (
            source.replace(
                "self.values.get(0).copied().unwrap_or(7)",
                "self.values.get(0).copied().unwrap()",
            ),
            "unwrap",
        ),
        (saved_length().replace("length < 2", "true"), "bounded"),
        (
            saved_length().replace(
                "let length = self.values.len();",
                "self.values.push_back(7); let length = self.values.len();",
            ),
            "bounded",
        ),
        (
            saved_length()
                .replace(
                    "pub values: Vector<u8>,",
                    "pub values: Vector<u8>, pub other: Vector<u8>,",
                )
                .replace("let length = self.values.len();", "let length = self.other.len();"),
            "bounded",
        ),
    ] {
        let error = Lift::parse("unsupported_vector.rs", &source)
            .err()
            .ok_or("unsupported Vector behavior was admitted")?;
        assert!(error.message.contains(reason), "{error}");
        assert!(error.line > 1);
    }
    Ok(())
}

#[test]
fn vector_get_keeps_eager_default_failures_when_element_is_present()
-> Result<(), Box<dyn std::error::Error>> {
    let native = umpire_checker::Checker::explore(&fallback_authored::fallback::world(), 100)?;
    let report = native.verify_all()?.to_string();
    assert!(report.contains("overflow"), "{report}");
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let module = Tla::emit(&Lift::parse(
        "vector_fallback.rs",
        include_str!("fixtures/vector_fallback.rs"),
    )?)?;
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../..");
    let error = umpire_tlc::Tlc::discover(&root)?
        .check_module(&module)
        .err()
        .ok_or("present Vector element suppressed eager fallback failure")?;
    assert!(error.to_string().contains("overflow"), "{error}");
    std::fs::create_dir_all(root.join(".flow/tmp/tlc-controls"))?;
    std::fs::write(root.join(".flow/tmp/tlc-controls/vector_fallback.txt"), error.to_string())?;
    Ok(())
}

#[test]
fn shared_boolean_guards_keep_vector_admission_linear() -> Result<(), Box<dyn std::error::Error>> {
    use std::fmt::Write as _;
    let mut bindings = String::from("let a0 = self.flag;");
    for index in 1..=40 {
        write!(bindings, "let a{index} = a{} && a{};", index - 1, index - 1)?;
    }
    let source = include_str!("fixtures/vector.rs")
        .replace("pub values: Vector<u8>,", "pub values: Vector<u8>, pub flag: bool,")
        .replace(
            "User::Push if self.values.len() < 2 => {",
            &format!("User::Push => {{ {bindings} if a40 && self.values.len() < 2 {{"),
        )
        .replace("self.values.push_back(7);", "self.values.push_back(7); }");
    for saved in ["direct", "cached", "conditional"] {
        let source = if saved == "conditional" {
            source.replace("if a40 && self.values.len() < 2", "let condition = self.values.is_empty() && self.values.is_empty(); let length = if condition { self.values.len() } else { self.values.len() }; if a40 && length < 2")
        } else if saved == "cached" {
            source.replace(
                "if a40 && self.values.len() < 2",
                "let length = self.values.len(); if a40 && length < 2",
            )
        } else {
            source.clone()
        };
        Lift::parse("shared_vector.rs", &source)?;
        let unbounded =
            source.replace("self.values.len() < 2", "true").replace("length < 2", "true");
        let error = Lift::parse("unbounded_shared_vector.rs", &unbounded)
            .err()
            .ok_or("unbounded append admitted")?;
        assert!(error.message.contains("bounded length guard"), "{error}");
    }
    Ok(())
}
