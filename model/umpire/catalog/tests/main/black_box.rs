//! Descriptor-driven mutation and classification controls.

#![forbid(unsafe_code)]

use umpire_catalog::*;

fn field(kind: WireKind) -> WireField {
    WireField {
        path: "example.Request.ignored".into(),
        declaration: "example.Request.ignored".into(),
        kind,
        optional: false,
        refusal: None,
    }
}

fn rejection(code: i32) -> ProbeResult {
    ProbeResult::Rejected { code, message_matches: false, details_type: None }
}

#[test]
fn generic_cases_need_any_rejection_explanation_but_keep_unexplained_failures()
-> Result<(), GapError> {
    let case = wire_cases(&[field(WireKind::String)], &[8]).remove(0);
    let rejects = || Expectation::Rejects {
        code: 3,
        message_pattern: "^invalid$".into(),
        details_type: Some("example.Invalid".into()),
    };
    let matching = || ProbeResult::Rejected {
        code: 3,
        message_matches: true,
        details_type: Some("example.Invalid".into()),
    };
    let wrong_details = || ProbeResult::Rejected {
        code: 3,
        message_matches: true,
        details_type: Some("example.Other".into()),
    };
    let cases = [
        (
            vec![(Expectation::Accepts, rejection(3)), (rejects(), matching())],
            rejection(3),
            "Confirmed",
        ),
        (vec![(rejects(), rejection(3)), (rejects(), matching())], rejection(3), "Confirmed"),
        (
            vec![(Expectation::Accepts, rejection(3)), (rejects(), rejection(3))],
            rejection(3),
            "Mismatch",
        ),
        (vec![(rejects(), wrong_details())], rejection(3), "Mismatch"),
        (vec![(rejects(), rejection(5))], rejection(5), "Mismatch"),
        (vec![(rejects(), rejection(3)), (rejects(), wrong_details())], rejection(3), "Mismatch"),
        (vec![(Expectation::Accepts, rejection(3))], rejection(3), "Mismatch"),
        (vec![(Expectation::Unobservable, rejection(3))], rejection(3), "Unannotated"),
        (vec![], rejection(3), "Unannotated"),
        (
            vec![(Expectation::Accepts, ProbeResult::Accepted), (rejects(), ProbeResult::Accepted)],
            ProbeResult::Accepted,
            "Accepted",
        ),
        (vec![(Expectation::Accepts, ProbeResult::Accepted)], ProbeResult::Accepted, "Confirmed"),
        (
            vec![(Expectation::Unobservable, ProbeResult::Accepted)],
            ProbeResult::Accepted,
            "Confirmed",
        ),
        (vec![], ProbeResult::Accepted, "Confirmed"),
    ];
    for (predictions, actual, wanted) in cases {
        let predictions: Vec<_> = predictions
            .into_iter()
            .enumerate()
            .map(|(index, (expectation, result))| {
                (
                    Violation {
                        path: case.field.path.clone(),
                        constraint: format!("constraint_{index}"),
                        fields: vec![],
                        expected: Some(Expected { expectation }),
                    },
                    result,
                )
            })
            .collect();
        let entries = mutation_entries(&case, &predictions, actual)?;
        assert_eq!(entries.len(), 1, "{wanted}");
        let outcome = match entries[0].outcome {
            GapOutcome::Confirmed => "Confirmed",
            GapOutcome::Accepted => "Accepted",
            GapOutcome::Mismatch(_) => "Mismatch",
            GapOutcome::Unannotated(_) => "Unannotated",
            GapOutcome::Unverifiable => "Unverifiable",
        };
        assert_eq!(outcome, wanted);
        assert_eq!(entries[0].fails(), wanted != "Confirmed");
        if predictions.iter().all(|(v, _)| {
            matches!(
                v.expected.as_ref().map(|e| &e.expectation),
                None | Some(Expectation::Unobservable)
            )
        }) {
            assert!(entries[0].violation.expected.is_none());
        }
    }
    Ok(())
}

#[test]
fn crossed_actual_status_cannot_confirm_a_predicted_rejection() {
    let case = wire_cases(&[field(WireKind::String)], &[8]).remove(0);
    let prediction = Violation {
        path: case.field.path.clone(),
        constraint: "string.max_bytes".into(),
        fields: vec![],
        expected: Some(Expected { expectation: Expectation::RejectsCode(Code::InvalidArgument) }),
    };
    for (observed, actual) in [
        (rejection(3), ProbeResult::Accepted),
        (rejection(5), rejection(3)),
        (ProbeResult::Unverifiable, rejection(3)),
        (ProbeResult::Unverifiable, ProbeResult::Unverifiable),
    ] {
        assert_eq!(
            mutation_entries(&case, &[(prediction.clone(), observed)], actual),
            Err(GapError)
        );
    }
}

#[test]
fn independent_mutations_include_unannotated_fields_and_explicit_refusals() {
    let mut recursive = field(WireKind::Message);
    recursive.refusal = Some("cycle: example.Request".into());
    let cases = wire_cases(&[field(WireKind::Bool), recursive], &[8]);
    assert_eq!(cases.len(), 10);
    assert!(cases.iter().any(|case| case.mutation == WireMutation::Absent
        && case.applicability == MutationApplicability::Applicable));
    assert!(
        cases
            .iter()
            .any(|case| matches!(&case.applicability, MutationApplicability::NotApplicable(_)))
    );
    assert_eq!(cases.iter().filter(|case| matches!(&case.applicability, MutationApplicability::Refused(reason) if reason.contains("cycle"))).count(), 5);
    assert_eq!(wire_cases(&[field(WireKind::Bool)], &[8, 8]).len(), 5);
}

#[test]
fn typed_scalar_mutations_are_independent_of_catalog_constraints() -> Result<(), GenerationError> {
    let mut text = "baseline".to_owned();
    text.mutate(&WireKind::String, &WireMutation::LimitPlusOne { limit: 8 })?;
    assert_eq!(text, "xxxxxxxxx");
    let mut integer = 3_i32;
    integer
        .mutate(&WireKind::Enum { defined: vec![0, 1, i32::MAX] }, &WireMutation::UndefinedEnum)?;
    assert_eq!(integer, i32::MAX - 1);
    let mut number = 3_i64;
    number.mutate(&WireKind::Signed, &WireMutation::Negative)?;
    assert_eq!(number, -1);
    let before = text.clone();
    assert_eq!(
        text.mutate(&WireKind::String, &WireMutation::LimitPlusOne { limit: u64::MAX }),
        Err(GenerationError::SizeLimit)
    );
    assert_eq!(text, before);
    assert_eq!(
        text.mutate(&WireKind::String, &WireMutation::Negative),
        Err(GenerationError::InapplicableMutation)
    );
    assert_eq!(text, before);
    let mut bytes = vec![1_u8];
    bytes.mutate(&WireKind::Bytes, &WireMutation::LimitPlusOne { limit: 2 })?;
    assert_eq!(bytes, [b'x'; 3]);
    bytes.mutate(&WireKind::Bytes, &WireMutation::Empty)?;
    assert_eq!(bytes, []);
    let mut flag = true;
    flag.mutate(&WireKind::Bool, &WireMutation::Absent)?;
    assert!(!flag);
    let mut unsigned = 5_u32;
    assert_eq!(
        unsigned.mutate(&WireKind::Unsigned, &WireMutation::Negative),
        Err(GenerationError::InapplicableMutation)
    );
    assert_eq!(unsigned, 5);
    assert_eq!(
        unsigned.mutate(
            &WireKind::Unsigned,
            &WireMutation::LimitPlusOne { limit: u64::from(u32::MAX) }
        ),
        Err(GenerationError::SizeLimit)
    );
    assert_eq!(unsigned, 5);
    let mut float = 2.0_f64;
    float.mutate(&WireKind::Float, &WireMutation::Negative)?;
    assert_eq!(float, -1.0);
    Ok(())
}

#[test]
fn mutation_admission_preserves_distinct_diagnostic_matches_and_refusals() -> Result<(), GapError> {
    let case = wire_cases(&[field(WireKind::String)], &[8]).remove(0);
    let mut predictions = Vec::new();
    for details in ["example.First", "example.Second"] {
        let violation = Violation {
            path: case.field.path.clone(),
            constraint: details.into(),
            fields: vec![],
            expected: Some(Expected {
                expectation: Expectation::Rejects {
                    code: 3,
                    message_pattern: "^invalid$".into(),
                    details_type: Some(details.into()),
                },
            }),
        };
        let observation = ProbeResult::Rejected {
            code: 3,
            message_matches: true,
            details_type: Some(details.into()),
        };
        predictions.push((violation, observation));
    }
    let entries = mutation_entries(&case, &predictions, rejection(3))?;
    assert_eq!(entries.len(), 1);
    assert!(entries.iter().all(|entry| entry.outcome == GapOutcome::Confirmed));
    let mut unobservable = predictions[0].0.clone();
    unobservable.expected = Some(Expected { expectation: Expectation::Unobservable });
    assert!(
        mutation_entries(&case, &[(unobservable, rejection(3))], rejection(3))?
            .iter()
            .any(|entry| matches!(entry.outcome, GapOutcome::Unannotated(_)) && entry.fails())
    );
    let refused =
        WireCase { applicability: MutationApplicability::Refused("cycle".into()), ..case };
    assert_eq!(
        mutation_entries(&refused, &[], ProbeResult::Unverifiable)?[0].outcome,
        GapOutcome::Unverifiable
    );
    assert_eq!(mutation_entries(&refused, &[], ProbeResult::Accepted), Err(GapError));
    Ok(())
}

#[test]
fn only_actual_unpredicted_rejections_are_unannotated_failures() -> Result<(), GapError> {
    let case = wire_cases(&[field(WireKind::String)], &[8]).remove(0);
    let entry = mutation_entries(&case, &[], rejection(3))?.remove(0);
    assert!(matches!(entry.outcome, GapOutcome::Unannotated(_)));
    assert!(entry.fails());
    let rendered = GapReport { entries: vec![entry] }.to_string();
    assert!(rendered.contains("Unannotated"));
    assert!(!rendered.contains("covers"));
    assert_eq!(
        mutation_entries(&case, &[], ProbeResult::Accepted)?[0].outcome,
        GapOutcome::Confirmed
    );
    let finding = Violation {
        path: case.field.path.clone(),
        constraint: "string.max_bytes".into(),
        fields: vec![],
        expected: Some(Expected { expectation: Expectation::Accepts }),
    };
    assert!(matches!(
        mutation_entries(&case, &[(finding, rejection(3))], rejection(3))?[0].outcome,
        GapOutcome::Mismatch(_)
    ));
    Ok(())
}
