//! New scalar and expectation behavior.
use umpire_catalog::*;

#[test]
fn loose_rejections_match_only_the_declared_status_scope() {
    for (expectation, code, confirmed) in [
        (Expectation::RejectsCode(Code::NotFound), 5, true),
        (Expectation::RejectsCode(Code::NotFound), 3, false),
        (Expectation::RejectsAnyStatus, 16, true),
        (Expectation::RejectsAnyStatus, 0, false),
    ] {
        let violation = Violation {
            path: "example.token".into(),
            constraint: "required".into(),
            fields: vec![],
            expected: Some(Expected { expectation }),
        };
        let result = ProbeResult::Rejected { code, message_matches: false, details_type: None };
        let entry = GapEntry::new(violation.clone(), result);
        assert_eq!(entry.outcome == GapOutcome::Confirmed, confirmed);
        assert_eq!(GapEntry::new(violation, ProbeResult::Accepted).outcome, GapOutcome::Accepted);
    }
}

#[test]
fn flag_and_number_have_typed_presence_and_constraints() {
    assert!(!false.present());
    assert!(true.present());
    assert_eq!(bool::generate(&Kind::Flag, &mut Rng::new(1)), Ok(true));
    let kind = Kind::Number(NumberRules { gte: Some(1), ..Default::default() });
    assert_eq!(1.5_f64.violations(&kind), [] as [&str; 0]);
    assert_eq!(0.5_f64.violations(&kind), ["number.gte"]);
    assert_eq!(f64::NAN.violations(&kind), ["number.finite"]);
}

#[test]
fn floating_generation_can_fill_a_fractional_interval() -> Result<(), GenerationError> {
    let kind = Kind::Number(NumberRules { gt: Some(0), lt: Some(1), ..Default::default() });
    let value = f64::generate(&kind, &mut Rng::new(1))?;
    assert!(value > 0.0 && value < 1.0);
    Ok(())
}

#[test]
fn required_number_generation_excludes_absent_zero() -> Result<(), Box<dyn std::error::Error>> {
    let expected = Expected { expectation: Expectation::Accepts };
    for rules in [
        NumberRules::default(),
        NumberRules { gte: Some(0), lt: Some(1), ..Default::default() },
        NumberRules { gt: Some(-1), lte: Some(0), ..Default::default() },
    ] {
        let catalog = Catalog::request("example.Number")
            .field(Field::new("value", Kind::Number(rules), |v: &f64| v, |v| v).expects(&expected))
            .rule(Rule {
                presence: Presence::Required(vec!["value".into()]),
                expected: Some(expected.clone()),
            })
            .build()?;
        let mut value = 0.0;
        fill(&mut value, &catalog, &mut Rng::new(3))?;
        assert!(value.present());
        assert_eq!(validate(&value, &catalog), []);
    }
    let only_zero = Kind::Number(NumberRules { gte: Some(0), lte: Some(0), ..Default::default() });
    assert_eq!(
        f64::generate_present(&only_zero, &mut Rng::new(3)),
        Err(GenerationError::ImpossibleConstraints)
    );
    Ok(())
}

#[test]
fn author_metadata_reaches_gap_reports_without_request_values() -> Result<(), AdmissionErrors> {
    let presence = Presence::Required(vec!["value".into()]);
    let catalog = Catalog::request("example.Request")
        .description("A request description")
        .field(
            Field::new("value", Kind::Text(StringRules::default()), |v: &String| v, |v| v)
                .description("A field description"),
        )
        .rule(Rule {
            presence: presence.clone(),
            expected: Some(Expected {
                expectation: Expectation::RejectsCode(Code::InvalidArgument),
            }),
        })
        .inherited_pattern(presence)
        .build()?;
    let case = catalog.probe_cases().remove(0);
    let entry = GapEntry::new(
        case.violation.clone(),
        ProbeResult::Rejected { code: 3, message_matches: false, details_type: None },
    )
    .metadata(&case);
    let rendered = GapReport { entries: vec![entry] }.to_string();
    assert!(rendered.contains("A request description"));
    assert!(rendered.contains("inherited pattern"));
    assert!(rendered.contains("Loose expectations: 1"));
    assert!(rendered.contains("Confirmed (code only)"));
    assert!(rendered.contains("actual status: 3; message redacted"));
    Ok(())
}
