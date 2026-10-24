//! Behavioral controls over actual typed message fields.

use std::fmt::Write;
use umpire_catalog::*;

#[derive(Clone, Default)]
struct Message {
    name: String,
}

#[test]
fn typed_catalog_validates_the_actual_field() -> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::response("example.Message")
        .field(Field::new(
            "name",
            Kind::Text(StringRules { min_len: Some(2), ..Default::default() }),
            |m: &Message| &m.name,
            |m| &mut m.name,
        ))
        .build()?;
    assert_eq!(validate(&Message { name: "ok".into() }, &catalog), []);
    assert_eq!(validate(&Message::default(), &catalog)[0].constraint, "string.min_len");
    Ok(())
}

#[test]
fn string_byte_limits_are_distinct_from_unicode_scalar_limits()
-> Result<(), Box<dyn std::error::Error>> {
    let rules = StringRules {
        min_len: Some(1),
        max_len: Some(2),
        min_bytes: Some(2),
        max_bytes: Some(4),
        ..Default::default()
    };
    for kind in [Kind::Text(rules.clone()), Kind::Identifier(rules)] {
        let catalog = catalog::<String>(kind)?;
        for (value, expected) in [
            ("é", vec![]),
            ("🙂", vec![]),
            ("a", vec!["string.min_bytes"]),
            ("é🙂", vec!["string.max_bytes"]),
            ("abc", vec!["string.max_len"]),
        ] {
            assert_eq!(
                validate(&Value { value: value.into() }, &catalog)
                    .iter()
                    .map(|v| v.constraint.as_str())
                    .collect::<Vec<_>>(),
                expected,
                "{value}"
            );
        }
    }
    Ok(())
}

#[derive(Clone, Default)]
struct Value<T> {
    value: T,
}

fn catalog<T: FieldValue>(kind: Kind) -> Result<Catalog<Value<T>>, AdmissionErrors> {
    Catalog::response("example.Value")
        .field(Field::new("value", kind, |m: &Value<T>| &m.value, |m| &mut m.value))
        .build()
}

fn control<T: FieldValue>(
    kind: Kind,
    valid: T,
    invalid: T,
    expected: &[&str],
    output: &mut String,
) -> Result<(), Box<dyn std::error::Error>> {
    let catalog = catalog(kind)?;
    assert_eq!(validate(&Value { value: valid }, &catalog), []);
    let violations = validate(&Value { value: invalid }, &catalog);
    assert_eq!(violations.iter().map(|v| v.constraint.as_str()).collect::<Vec<_>>(), expected);
    for violation in violations {
        writeln!(output, "{violation}")?;
    }
    Ok(())
}

#[test]
fn kind_constraints_have_positive_and_negative_controls() -> Result<(), Box<dyn std::error::Error>>
{
    let mut report = String::new();
    control(
        Kind::Uuid,
        "a13cd9da-521c-48d4-ab9c-6ab174876f25".to_owned(),
        "not-uuid".to_owned(),
        &["string.uuid"],
        &mut report,
    )?;
    control(
        Kind::Identifier(StringRules::default()),
        "id".to_owned(),
        String::new(),
        &["string.min_len"],
        &mut report,
    )?;
    for (rules, valid, invalid, name) in [
        (StringRules { min_len: Some(2), ..Default::default() }, "é🙂", "é", "string.min_len"),
        (StringRules { max_len: Some(2), ..Default::default() }, "é🙂", "abc", "string.max_len"),
        (
            StringRules { prefix: Some("pre".into()), ..Default::default() },
            "prefix",
            "other",
            "string.prefix",
        ),
        (
            StringRules { suffix: Some("end".into()), ..Default::default() },
            "the end",
            "other",
            "string.suffix",
        ),
        (
            StringRules { contains: Some("mid".into()), ..Default::default() },
            "amidst",
            "other",
            "string.contains",
        ),
    ] {
        control(Kind::Text(rules), valid.to_owned(), invalid.to_owned(), &[name], &mut report)?;
    }
    for kind in [
        Kind::Bytes(BytesRules { min_len: Some(2), max_len: Some(3) }),
        Kind::Random(BytesRules { min_len: Some(2), max_len: Some(3) }),
    ] {
        control(kind.clone(), vec![1_u8, 2], vec![1_u8], &["bytes.min_len"], &mut report)?;
        control(kind, vec![1_u8, 2, 3], vec![1_u8, 2, 3, 4], &["bytes.max_len"], &mut report)?;
    }
    control(
        Kind::Enum(EnumRules { defined_only: true, values: vec![0, 2] }),
        2_i32,
        1_i32,
        &["enum.defined_only"],
        &mut report,
    )?;
    for (rules, valid, invalid, bound) in [
        (NumberRules { gt: Some(3), ..Default::default() }, 4, 3, "gt"),
        (NumberRules { gte: Some(3), ..Default::default() }, 3, 2, "gte"),
        (NumberRules { lt: Some(3), ..Default::default() }, 2, 3, "lt"),
        (NumberRules { lte: Some(3), ..Default::default() }, 3, 4, "lte"),
    ] {
        control(
            Kind::Counter(rules.clone()),
            valid,
            invalid,
            &[&format!("int32.{bound}")],
            &mut report,
        )?;
        control(
            Kind::Timestamp(rules.clone()),
            Timestamp { seconds: 0, nanos: valid },
            Timestamp { seconds: 0, nanos: invalid },
            &[&format!("timestamp.{bound}")],
            &mut report,
        )?;
        control(
            Kind::Duration(rules),
            Duration { seconds: 0, nanos: valid },
            Duration { seconds: 0, nanos: invalid },
            &[&format!("duration.{bound}")],
            &mut report,
        )?;
    }
    control(
        Kind::Timestamp(NumberRules::default()),
        Timestamp::default(),
        Timestamp { seconds: 0, nanos: -1 },
        &["timestamp.valid"],
        &mut report,
    )?;
    control(
        Kind::Duration(NumberRules::default()),
        Duration { seconds: -1, nanos: -1 },
        Duration { seconds: 1, nanos: -1 },
        &["duration.valid"],
        &mut report,
    )?;
    insta::assert_snapshot!("kind-violations", report);
    Ok(())
}

#[derive(Clone, Default, Debug, PartialEq, Eq)]
struct Pair {
    a: Option<String>,
    b: Option<String>,
}

fn pair_catalog(presence: Presence) -> Result<Catalog<Pair>, AdmissionErrors> {
    Catalog::response("example.Pair")
        .field(Field::new("a", Kind::Text(StringRules::default()), |m: &Pair| &m.a, |m| &mut m.a))
        .field(Field::new("b", Kind::Text(StringRules::default()), |m: &Pair| &m.b, |m| &mut m.b))
        .rule(Rule { presence, expected: None })
        .build()
}

#[test]
fn presence_operators_validate_and_guide_fill() -> Result<(), Box<dyn std::error::Error>> {
    let mut report = String::new();
    for (presence, truth_table) in [
        (Presence::Required(vec!["a".into(), "b".into()]), [false, false, false, true]),
        (Presence::AnyOf(vec!["a".into(), "b".into()]), [false, true, true, true]),
        (Presence::OneOf(vec!["a".into(), "b".into()]), [false, true, true, false]),
        (Presence::Requires("a".into(), "b".into()), [true, false, true, true]),
        (Presence::Excludes("a".into(), "b".into()), [true, true, true, false]),
    ] {
        let catalog = pair_catalog(presence)?;
        for (bits, valid) in truth_table.into_iter().enumerate() {
            let message =
                Pair { a: (bits & 1 != 0).then(String::new), b: (bits & 2 != 0).then(String::new) };
            let violations = validate(&message, &catalog);
            assert_eq!(violations.is_empty(), valid, "{bits}: {:?}", catalog.rules());
            for violation in violations {
                writeln!(report, "{bits}: {violation}")?;
            }
        }
        let mut message = Pair::default();
        fill(&mut message, &catalog, &mut Rng::new(5))?;
        assert_eq!(validate(&message, &catalog), []);
    }
    insta::assert_snapshot!("presence-violations", report);
    Ok(())
}

#[test]
fn presence_admission_rejects_aliases_of_the_same_field() {
    for presence in [
        Presence::OneOf(vec!["a".into(), "example.Pair.a".into()]),
        Presence::Required(vec!["a".into(), "example.Pair.a".into()]),
        Presence::AnyOf(vec!["a".into(), "example.Pair.a".into()]),
        Presence::Requires("a".into(), "example.Pair.a".into()),
        Presence::Excludes("a".into(), "example.Pair.a".into()),
    ] {
        let name = presence.name();
        assert_eq!(
            pair_catalog(presence).err(),
            Some(AdmissionErrors(vec![AdmissionError::InvalidRule(name.into())]))
        );
    }
}

#[test]
fn distinct_presence_fields_accept_relative_and_qualified_paths()
-> Result<(), Box<dyn std::error::Error>> {
    for (a, b) in [("a", "b"), ("example.Pair.a", "example.Pair.b"), ("a", "example.Pair.b")] {
        let catalog = pair_catalog(Presence::OneOf(vec![a.into(), b.into()]))?;
        assert_eq!(validate(&Pair { a: Some("value".into()), b: None }, &catalog), []);
        let mut message = Pair::default();
        fill(&mut message, &catalog, &mut Rng::new(5))?;
        assert_eq!(validate(&message, &catalog), []);
    }
    Ok(())
}

#[test]
fn presence_fill_preserves_unrelated_generated_fields() -> Result<(), Box<dyn std::error::Error>> {
    for presence in [
        None,
        Some(Presence::OneOf(vec!["a".into(), "b".into()])),
        Some(Presence::Excludes("example.Many.a".into(), "b".into())),
    ] {
        let mut builder = Catalog::response("example.Many");
        for (index, name) in ('a'..='r').enumerate() {
            builder = builder.field(Field::projected(
                name.to_string(),
                Kind::Text(StringRules::default()),
                move |m: &Value<[String; 18]>| m.value[index].clone(),
                move |m, value| m.value[index] = value,
            ));
        }
        if let Some(presence) = presence {
            builder = builder.rule(Rule { presence, expected: None });
        }
        let catalog = builder.build()?;
        let mut message = Value::default();
        fill(&mut message, &catalog, &mut Rng::new(5))?;
        assert_eq!(validate(&message, &catalog), []);
        assert!(message.value[2..].iter().all(|value| !value.is_empty()));
        if catalog.rules().is_empty() {
            assert!(message.value.iter().all(|value| !value.is_empty()));
        } else {
            assert_ne!(message.value[0], "");
            assert_eq!(message.value[1], "");
        }
    }
    Ok(())
}

#[test]
fn presence_fill_ignores_unrelated_optional_fields() -> Result<(), Box<dyn std::error::Error>> {
    let mut builder = Catalog::response("example.Many");
    for (index, name) in ('a'..='q').enumerate() {
        builder = builder.field(Field::projected(
            name.to_string(),
            Kind::Text(StringRules::default()),
            move |m: &Value<[Option<String>; 17]>| m.value[index].clone(),
            move |m, value| m.value[index] = value,
        ));
    }
    let catalog = builder
        .rule(Rule { presence: Presence::OneOf(vec!["a".into(), "b".into()]), expected: None })
        .build()?;
    let mut message = Value::default();
    fill(&mut message, &catalog, &mut Rng::new(5))?;
    assert_eq!(validate(&message, &catalog), []);
    assert!(message.value[0].is_some());
    assert!(message.value[1].is_none());
    assert!(message.value[2..].iter().all(Option::is_some));
    Ok(())
}

#[test]
fn impossible_presence_rules_refuse_without_mutating_message_or_rng()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::response("example.Pair")
        .field(Field::new("a", Kind::Text(StringRules::default()), |m: &Pair| &m.a, |m| &mut m.a))
        .field(Field::new("b", Kind::Text(StringRules::default()), |m: &Pair| &m.b, |m| &mut m.b))
        .rule(Rule { presence: Presence::Required(vec!["a".into(), "b".into()]), expected: None })
        .rule(Rule { presence: Presence::Excludes("a".into(), "b".into()), expected: None })
        .build()?;
    let mut message = Pair::default();
    let before = message.clone();
    let mut rng = Rng::new(5);
    let mut before_rng = rng.clone();
    assert!(matches!(fill(&mut message, &catalog, &mut rng), Err(FillError::Invalid(_))));
    assert_eq!(message, before);
    assert_eq!(rng.next_u64(), before_rng.next_u64());
    Ok(())
}

fn expected(expectation: Expectation) -> Expected {
    Expected { expectation }
}

fn rejects() -> Expected {
    expected(Expectation::Rejects {
        code: 3,
        message_pattern: "invalid identifier".into(),
        details_type: Some("example.Invalid".into()),
    })
}

#[test]
fn request_admission_retains_typed_diagnostics() {
    let errors = Catalog::request("example.Message")
        .field(
            Field::new("name", Kind::Uuid, |m: &Message| &m.name, |m| &mut m.name)
                .tag(&Consequence::Decided)
                .tag(&Consequence::Ignored(String::new())),
        )
        .field(Field::new(
            "count",
            Kind::Counter(NumberRules::default()),
            |m: &Message| &m.name,
            |m| &mut m.name,
        ))
        .rule(Rule {
            presence: Presence::Required(vec!["name".into(), "missing".into()]),
            expected: None,
        })
        .build()
        .err()
        .unwrap_or_else(|| panic!("catalog should be refused"));
    assert!(errors.0.iter().any(|e| matches!(e, AdmissionError::UnsupportedKind { path, rust_type, .. } if path == "count" && *rust_type == "alloc::string::String")));
    assert!(errors.0.iter().any(|e| matches!(e, AdmissionError::ConflictingTags { .. })));
    assert!(errors.0.iter().any(|e| matches!(e, AdmissionError::MissingIgnoreReason { .. })));
    assert!(errors.0.iter().any(|e| matches!(e, AdmissionError::IgnoredRuleField(_))));
    assert!(errors.0.iter().any(|e| matches!(e, AdmissionError::UnknownRuleField(_))));
    assert!(errors.0.iter().any(|e| matches!(e, AdmissionError::MissingExpectation { constraint, .. } if constraint == "required")));
    let no_expectation = Catalog::request("example.Message")
        .field(Field::new("name", Kind::Uuid, |m: &Message| &m.name, |m| &mut m.name))
        .build()
        .err();
    assert!(
        matches!(no_expectation, Some(AdmissionErrors(errors)) if errors.iter().any(|e| matches!(e, AdmissionError::MissingExpectation { constraint, .. } if constraint == "string.uuid")))
    );
    insta::assert_snapshot!("admission", errors.to_string());
}

#[test]
fn tags_admit_only_compatible_consequences() -> Result<(), Box<dyn std::error::Error>> {
    for (kind, tag) in [
        (Kind::Secret, Consequence::Observed),
        (Kind::Random(BytesRules::default()), Consequence::Decided),
        (Kind::Text(StringRules::default()), Consequence::Monotone),
        (Kind::Uuid, Consequence::Cached { ttl: std::time::Duration::ZERO }),
        (Kind::Uuid, Consequence::EventuallyConsistent { polls: 0 }),
    ] {
        assert!(
            Catalog::response("example.Message")
                .field(Field::new("name", kind, |m: &Message| &m.name, |m| &mut m.name).tag(&tag))
                .build()
                .is_err()
        );
    }
    let catalog = Catalog::response("example.Message")
        .field(
            Field::new("name", Kind::Uuid, |m: &Message| &m.name, |m| &mut m.name)
                .tag(&Consequence::Immutable)
                .tag(&Consequence::IdempotencyKey)
                .tag(&Consequence::Cached { ttl: std::time::Duration::from_secs(1) })
                .tag(&Consequence::EventuallyConsistent { polls: 3 })
                .tag(&Consequence::Decided),
        )
        .build()?;
    assert_eq!(catalog.fields().next().map(|field| field.tags.len()), Some(5));
    Ok(())
}

#[test]
fn repeated_constraints_and_seeded_filling() -> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::response("example.Value")
        .field(
            Field::new(
                "value",
                Kind::Counter(NumberRules { gte: Some(1), lte: Some(9), ..Default::default() }),
                |m: &Value<Vec<i32>>| &m.value,
                |m| &mut m.value,
            )
            .repeated(RepeatedRules {
                min_items: Some(2),
                max_items: Some(3),
                unique: true,
            }),
        )
        .build()?;
    assert_eq!(validate(&Value { value: vec![1, 2] }, &catalog), []);
    for (values, constraint) in [
        (vec![1], "repeated.min_items"),
        (vec![1, 2, 3, 4], "repeated.max_items"),
        (vec![1, 1], "repeated.unique"),
    ] {
        assert_eq!(validate(&Value { value: values }, &catalog)[0].constraint, constraint);
    }
    let mut first = Value::default();
    let mut second = Value::default();
    fill(&mut first, &catalog, &mut Rng::new(18))?;
    fill(&mut second, &catalog, &mut Rng::new(18))?;
    assert_eq!(first.value, second.value);
    assert_eq!(validate(&first, &catalog), []);
    Ok(())
}

#[test]
fn fills_every_generative_kind_and_refuses_impossible_constraints()
-> Result<(), Box<dyn std::error::Error>> {
    fn generated<T: FieldValue>(kind: Kind) -> Result<(), Box<dyn std::error::Error>> {
        let catalog = catalog::<T>(kind)?;
        let mut a = Value::default();
        let mut b = Value::default();
        fill(&mut a, &catalog, &mut Rng::new(123))?;
        fill(&mut b, &catalog, &mut Rng::new(123))?;
        assert!(a.value == b.value);
        assert_eq!(validate(&a, &catalog), []);
        Ok(())
    }
    generated::<String>(Kind::Uuid)?;
    generated::<String>(Kind::Identifier(StringRules::default()))?;
    generated::<String>(Kind::Text(StringRules {
        min_len: Some(4),
        max_len: Some(4),
        prefix: Some("abc".into()),
        contains: Some("bcd".into()),
        suffix: Some("cd".into()),
        ..Default::default()
    }))?;
    generated::<Vec<u8>>(Kind::Bytes(BytesRules { min_len: None, max_len: Some(0) }))?;
    generated::<i32>(Kind::Enum(EnumRules { defined_only: false, values: Vec::new() }))?;
    generated::<String>(Kind::Text(StringRules {
        min_len: Some(8),
        max_len: Some(12),
        prefix: Some("α".into()),
        contains: Some("mid".into()),
        suffix: Some("ω".into()),
        ..Default::default()
    }))?;
    generated::<String>(Kind::Text(StringRules {
        min_len: Some(5),
        max_len: Some(5),
        prefix: Some("abc".into()),
        suffix: Some("bc".into()),
        ..Default::default()
    }))?;
    generated::<String>(Kind::Random(BytesRules { min_len: Some(4), max_len: Some(5) }))?;
    generated::<Vec<u8>>(Kind::Bytes(BytesRules { min_len: Some(4), max_len: Some(5) }))?;
    generated::<i64>(Kind::Counter(NumberRules {
        gt: Some(-5),
        lt: Some(5),
        ..Default::default()
    }))?;
    generated::<u64>(Kind::Counter(NumberRules {
        gte: Some(i128::from(u64::MAX)),
        ..Default::default()
    }))?;
    generated::<i32>(Kind::Enum(EnumRules { defined_only: true, values: vec![1, 3] }))?;
    generated::<Timestamp>(Kind::Timestamp(NumberRules { gt: Some(0), ..Default::default() }))?;
    generated::<Duration>(Kind::Duration(NumberRules {
        gte: Some(-1),
        lte: Some(-1),
        ..Default::default()
    }))?;
    for kind in [
        Kind::Text(StringRules { min_len: Some(4), max_len: Some(2), ..Default::default() }),
        Kind::Random(BytesRules { min_len: Some(4), max_len: Some(2) }),
    ] {
        let catalog = catalog::<String>(kind)?;
        let mut message = Value::default();
        assert!(matches!(
            fill(&mut message, &catalog, &mut Rng::new(1)),
            Err(FillError::Field { reason: GenerationError::ImpossibleConstraints, .. })
        ));
        assert_eq!(message.value, "");
    }
    let catalog =
        catalog::<i64>(Kind::Counter(NumberRules { gt: Some(i128::MAX), ..Default::default() }))?;
    assert!(fill(&mut Value::default(), &catalog, &mut Rng::new(1)).is_err());
    Ok(())
}

#[derive(Clone, Default, Debug, PartialEq, Eq)]
struct Roles {
    decided: String,
    observed: String,
    ignored: String,
    secret: String,
    random: String,
}

#[test]
fn fill_preserves_roles_and_is_atomic_on_missing_external_secrets()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::response("example.Roles")
        .field(
            Field::new(
                "decided",
                Kind::Text(StringRules::default()),
                |m: &Roles| &m.decided,
                |m| &mut m.decided,
            )
            .tag(&Consequence::Decided),
        )
        .field(
            Field::new(
                "observed",
                Kind::Text(StringRules::default()),
                |m: &Roles| &m.observed,
                |m| &mut m.observed,
            )
            .tag(&Consequence::Observed),
        )
        .field(
            Field::new("ignored", Kind::Uuid, |m: &Roles| &m.ignored, |m| &mut m.ignored)
                .tag(&Consequence::Ignored("outside this model".into())),
        )
        .field(Field::new("secret", Kind::Secret, |m: &Roles| &m.secret, |m| &mut m.secret))
        .field(Field::new(
            "random",
            Kind::Random(BytesRules::default()),
            |m: &Roles| &m.random,
            |m| &mut m.random,
        ))
        .build()?;
    let mut message = Roles {
        decided: "model choice".into(),
        observed: "server value".into(),
        ignored: "not a uuid".into(),
        ..Default::default()
    };
    let before = message.clone();
    let mut rng = Rng::new(42);
    let mut untouched_rng = rng.clone();
    assert!(
        matches!(fill(&mut message, &catalog, &mut rng), Err(FillError::Field { path, reason: GenerationError::ExternalSecret }) if path == "example.Roles.secret")
    );
    assert_eq!(message, before);
    assert_eq!(rng.next_u64(), untouched_rng.next_u64());
    message.secret = "sensitive-external-token".into();
    fill(&mut message, &catalog, &mut Rng::new(42))?;
    assert_eq!(message.decided, before.decided);
    assert_eq!(message.observed, before.observed);
    assert_eq!(message.ignored, before.ignored);
    assert_ne!(message.random, "");
    let once = message.clone();
    fill(&mut message, &catalog, &mut Rng::new(99))?;
    assert_eq!(message, once);
    assert_eq!(catalog.redact(&mut message), Ok(()));
    assert_eq!(message.secret, "");
    Ok(())
}

#[test]
fn gap_goldens_are_derived_from_validation_and_redact_secret_values()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::request("example.Roles")
        .field(Field::new("secret", Kind::Secret, |m: &Roles| &m.secret, |m| &mut m.secret))
        .field(
            Field::new("decided", Kind::Uuid, |m: &Roles| &m.decided, |m| &mut m.decided)
                .expects(&rejects()),
        )
        .rule(Rule {
            presence: Presence::Required(vec!["secret".into()]),
            expected: Some(rejects()),
        })
        .build()?;
    let message = Roles { secret: "sensitive-external-token".into(), ..Default::default() };
    let violations = validate(&message, &catalog);
    assert_eq!(violations.len(), 1);
    assert!(!format!("{violations:?}").contains(&message.secret));
    let mut report = GapReport::default();
    for actual in [
        ProbeResult::Rejected {
            code: 3,
            message_matches: true,
            details_type: Some("example.Invalid".into()),
        },
        ProbeResult::Rejected { code: 5, message_matches: false, details_type: None },
        ProbeResult::Accepted,
        ProbeResult::Unverifiable,
    ] {
        report.entries.push(GapEntry::new(violations[0].clone(), actual));
    }
    assert!(report.fails());
    assert!(!report.to_string().contains(&message.secret));
    insta::assert_snapshot!("gaps", report.to_string());
    let forward = report.to_string();
    report.entries.reverse();
    assert_eq!(report.to_string(), forward);
    for entry in &mut report.entries {
        entry.accepted_gap = Some(AcceptedGap {
            date: "2026-10-20".into(),
            reason: "tracked server discrepancy".into(),
        });
    }
    assert!(!report.fails());
    if let Some(gap) = report.entries[1].accepted_gap.as_mut() {
        gap.reason.clear();
    }
    assert!(report.fails());
    Ok(())
}
