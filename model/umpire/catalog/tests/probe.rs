//! Catalog-derived negative witnesses and fail-closed probe evidence.

use umpire_catalog::*;

#[derive(Clone, Default, Debug)]
struct Message {
    a: String,
    b: String,
}

fn expected() -> Expected {
    Expected { expectation: Expectation::Accepts }
}

#[test]
fn any_of_attributes_each_field_but_every_probe_omits_all_alternatives()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::request("example.Message")
        .field(Field::new(
            "a",
            Kind::Text(StringRules::default()),
            |m: &Message| &m.a,
            |m| &mut m.a,
        ))
        .field(Field::new(
            "b",
            Kind::Text(StringRules::default()),
            |m: &Message| &m.b,
            |m| &mut m.b,
        ))
        .rule(Rule {
            presence: Presence::AnyOf(vec!["a".into(), "b".into()]),
            expected: Some(expected()),
        })
        .build()?;
    let baseline = Message { a: "a".into(), b: "b".into() };
    let cases = catalog.probe_cases();
    assert_eq!(cases.len(), 2);
    for case in &cases {
        assert!(case.id.contains(":all_absent:missing="));
        let candidate = catalog.probe(&baseline, case, &mut Rng::new(1))?;
        assert!(candidate.message.a.is_empty() && candidate.message.b.is_empty());
        assert!(
            catalog
                .probe_with(&baseline, case, |m| {
                    m.a.clear();
                    Ok(())
                })
                .is_err()
        );
        assert!(
            catalog
                .probe_with(&baseline, case, |m| {
                    m.b.clear();
                    Ok(())
                })
                .is_err()
        );
    }
    Ok(())
}

fn catalog(ineffective: bool) -> Catalog<Message> {
    Catalog::request("example.Message")
        .field(
            Field::projected(
                "a",
                Kind::Text(StringRules { max_bytes: Some(8), ..Default::default() }),
                |m: &Message| m.a.clone(),
                move |m, value| {
                    if !ineffective {
                        m.a = value;
                    }
                },
            )
            .expects(&expected()),
        )
        .field(Field::new(
            "b",
            Kind::Text(StringRules::default()),
            |m: &Message| &m.b,
            |m| &mut m.b,
        ))
        .rule(Rule {
            presence: Presence::Required(vec!["a".into(), "b".into()]),
            expected: Some(expected()),
        })
        .build()
        .unwrap_or_else(|error| panic!("{error}"))
}

#[test]
fn enumerates_each_omission_and_verifies_real_negative_witnesses() {
    let catalog = catalog(false);
    let baseline = Message { a: "a".into(), b: "b".into() };
    let cases = catalog.probe_cases();
    assert_eq!(cases.len(), 3);
    assert_eq!(cases.iter().filter(|case| case.omitted_field.is_some()).count(), 2);
    for case in &cases {
        let probe = catalog
            .probe(&baseline, case, &mut Rng::new(1))
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(validate(&probe.message, &catalog), vec![probe.violation]);
    }
    assert_eq!(baseline.a, "a");
}

#[test]
fn refuses_invalid_baselines_forged_cases_and_ineffective_mutation() {
    let catalog = catalog(true);
    let baseline = Message { a: "a".into(), b: "b".into() };
    let case = catalog.probe_cases().remove(0);
    assert!(catalog.probe(&Message::default(), &case, &mut Rng::new(1)).is_err());
    assert!(catalog.probe(&baseline, &case, &mut Rng::new(1)).is_err());
    assert!(catalog.probe_with(&baseline, &case, |_| Ok(())).is_err());
    let mut forged = case.clone();
    forged.id.push('x');
    assert!(catalog.probe_with(&baseline, &forged, |_| Ok(())).is_err());
    let probe = catalog
        .probe_with(&baseline, &case, |m| {
            m.a = "x".repeat(9);
            Ok(())
        })
        .unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(probe.violation.constraint, "string.max_bytes");
    assert!(matches!(
        catalog.probe_with(&baseline, &case, |_| Err(GenerationError::UnsupportedType)),
        Err(ProbeError::Generation(GenerationError::UnsupportedType))
    ));
}

#[test]
fn coverage_requires_each_unique_identity_and_observable_results()
-> Result<(), Box<dyn std::error::Error>> {
    let cases = catalog(false).probe_cases();
    let ids: Vec<_> = cases.iter().map(|case| case.id.clone()).collect();
    verify_probe_coverage(&cases, &ids)?;
    let mut incomplete = ids.clone();
    incomplete[0].clone_from(&ids[1]);
    assert!(verify_probe_coverage(&cases, &incomplete).is_err());
    assert!(verify_probe_coverage(&[cases[0].clone(), cases[0].clone()], &ids[..1]).is_err());
    let mut extra = ids;
    extra.push("unknown".into());
    assert!(verify_probe_coverage(&cases, &extra).is_err());
    assert!(GapEntry::checked(cases[0].violation.clone(), ProbeResult::Unverifiable).is_err());
    let mut unobservable = cases[0].violation.clone();
    if let Some(expected) = unobservable.expected.as_mut() {
        expected.expectation = Expectation::Unobservable;
    }
    assert_eq!(
        GapEntry::checked(unobservable, ProbeResult::Unverifiable)?.outcome,
        GapOutcome::Unverifiable
    );
    Ok(())
}

#[test]
fn live_result_checks_each_expected_diagnostic_independently()
-> Result<(), Box<dyn std::error::Error>> {
    let mut violation = catalog(false).probe_cases().remove(0).violation;
    violation.expected = Some(Expected {
        expectation: Expectation::Rejects {
            code: 3,
            message_pattern: "^invalid$".into(),
            details_type: Some("example.Invalid".into()),
        },
    });
    for (code, message_matches, details_type, fails) in [
        (3, true, Some("example.Invalid"), false),
        (5, true, Some("example.Invalid"), true),
        (3, false, Some("example.Invalid"), true),
        (3, true, Some("example.Other"), true),
        (3, true, None, true),
    ] {
        let entry = GapEntry::checked(
            violation.clone(),
            ProbeResult::Rejected {
                code,
                message_matches,
                details_type: details_type.map(str::to_owned),
            },
        )?;
        assert_eq!(entry.fails(), fails);
        if !fails {
            assert_eq!(entry.outcome, GapOutcome::Confirmed);
        }
    }
    assert!(GapEntry::checked(violation.clone(), ProbeResult::Accepted)?.fails());
    if let Some(expected) = violation.expected.as_mut() {
        expected.expectation = Expectation::Accepts;
    }
    assert_eq!(GapEntry::checked(violation, ProbeResult::Accepted)?.outcome, GapOutcome::Confirmed);
    Ok(())
}

fn negative<T: FieldValue>(
    value: T,
    kind: Kind,
    repeated: Option<RepeatedRules>,
) -> Result<(), Box<dyn std::error::Error>> {
    #[derive(Clone)]
    struct Value<T> {
        value: T,
    }
    let mut field = Field::new("value", kind, |m: &Value<T>| &m.value, |m| &mut m.value);
    if let Some(rules) = repeated {
        field = field.repeated(rules);
    }
    let catalog = Catalog::request("example.Value").field(field.expects(&expected())).build()?;
    let baseline = Value { value };
    for case in catalog.probe_cases() {
        let probe = catalog.probe(&baseline, &case, &mut Rng::new(17))?;
        assert_eq!(validate(&probe.message, &catalog), [probe.violation]);
    }
    Ok(())
}

#[test]
fn witnesses_cover_typed_kinds_and_repeated_constraints() -> Result<(), Box<dyn std::error::Error>>
{
    negative("00000000-0000-0000-0000-000000000000".to_owned(), Kind::Uuid, None)?;
    negative(
        "valid".to_owned(),
        Kind::Identifier(StringRules { max_bytes: Some(8), ..Default::default() }),
        None,
    )?;
    negative(
        "valid".to_owned(),
        Kind::Text(StringRules { min_len: Some(2), max_len: Some(8), ..Default::default() }),
        None,
    )?;
    negative(
        "valid".to_owned(),
        Kind::Text(StringRules { min_bytes: Some(2), max_bytes: Some(8), ..Default::default() }),
        None,
    )?;
    negative(vec![1_u8], Kind::Bytes(BytesRules { min_len: Some(1), max_len: Some(8) }), None)?;
    negative(
        1_i32,
        Kind::Counter(NumberRules { gte: Some(0), lte: Some(4), ..Default::default() }),
        None,
    )?;
    negative(1_i32, Kind::Enum(EnumRules { defined_only: true, values: vec![0, 1] }), None)?;
    negative(
        Some(Duration { seconds: 1, nanos: 0 }),
        Kind::Duration(NumberRules { gte: Some(0), ..Default::default() }),
        None,
    )?;
    negative(
        Timestamp { seconds: 1, nanos: 0 },
        Kind::Timestamp(NumberRules { gte: Some(0), ..Default::default() }),
        None,
    )?;
    negative(
        vec![1_i32, 2],
        Kind::Counter(NumberRules::default()),
        Some(RepeatedRules { min_items: Some(1), max_items: Some(3), unique: true }),
    )?;
    Ok(())
}

#[test]
fn witnesses_cover_presence_algebra_and_required_variant() -> Result<(), Box<dyn std::error::Error>>
{
    for (presence, a, b) in [
        (Presence::AnyOf(vec!["a".into(), "b".into()]), "a", ""),
        (Presence::OneOf(vec!["a".into(), "b".into()]), "a", ""),
        (Presence::Requires("a".into(), "b".into()), "", ""),
        (Presence::Excludes("a".into(), "b".into()), "a", ""),
    ] {
        let catalog = Catalog::request("example.Message")
            .field(Field::new(
                "a",
                Kind::Text(StringRules::default()),
                |m: &Message| &m.a,
                |m| &mut m.a,
            ))
            .field(Field::new(
                "b",
                Kind::Text(StringRules::default()),
                |m: &Message| &m.b,
                |m| &mut m.b,
            ))
            .rule(Rule { presence, expected: Some(expected()) })
            .build()?;
        catalog.probe(
            &Message { a: a.into(), b: b.into() },
            &catalog.probe_cases()[0],
            &mut Rng::new(1),
        )?;
    }
    let catalog = catalog(false);
    let case = catalog
        .probe_cases()
        .into_iter()
        .find(|case| case.omitted_field.as_deref() == Some("example.Message.a"))
        .ok_or("missing omission case")?;
    assert!(
        catalog
            .probe_with(&Message { a: "a".into(), b: "b".into() }, &case, |m| {
                m.b.clear();
                Ok(())
            })
            .is_err()
    );
    Ok(())
}
