//! Typed request witnesses must survive the actual protobuf projection.

use umpire_runtime::annotations::ProbeMatch;
use umpire_runtime::annotations::ProbeObservation;

#[test]
fn black_box_inventory_includes_ignored_wire_fields() -> Result<(), Box<dyn std::error::Error>> {
    use prost::Message;
    let inventory = probe(&ProbeRequest::default())?;
    assert!(
        inventory
            .cases
            .iter()
            .any(|case| case.generic && case.id.contains("namespace:mutation.empty"))
    );
    assert!(
        inventory.encoded_len() < 1024 * 1024,
        "inventory metadata must stay below one MiB without request values"
    );
    Ok(())
}

use umpire_temporal::foundation::dc::Limits;

use umpire_temporal::probes::probe;
use umpire_temporal_api::umpire::temporal::v1::{ProbeRequest, probe_request::Operation};

#[test]
fn large_generic_witnesses_use_transport_recipes() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal::foundation::dc::BLOB_SIZE_ERROR;
    let inventory = probe(&ProbeRequest::default())?;
    let limit = Limits::default().get(&BLOB_SIZE_ERROR);
    for field in [".namespace:", ".data:"] {
        let case = inventory
            .cases
            .iter()
            .find(|case| {
                case.generic
                    && case.id.contains("StartActivityExecution::")
                    && case.id.contains(field)
                    && case.id.ends_with(&format!("limit_plus_one({limit})"))
            })
            .ok_or("missing large generic field")?;
        let prepared = probe(&ProbeRequest {
            operation: Operation::Prepare as i32,
            case_id: case.id.clone(),
            namespace: "namespace".into(),
            activity_id: "activity".into(),
            task_queue: "queue".into(),
            ..Default::default()
        })?;
        assert_eq!(
            prepared.cases[0].applicability, 0,
            "large typed mutation must not be downgraded"
        );
        assert_eq!(prepared.message, Vec::<u8>::new());
        assert!(
            u64::from(
                prepared.recipe.as_ref().ok_or("missing large witness recipe")?.decoded_length
            ) > limit
        );
        assert!(prost::Message::encoded_len(&prepared) < 1024 * 1024 - 1024);
    }
    Ok(())
}

#[test]
fn generic_actual_witness_predictions_and_observation_admission()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_runtime::annotations::ProbeMatch;
    let inventory = probe(&ProbeRequest::default())?;
    let case = inventory
        .cases
        .iter()
        .find(|case| {
            case.generic
                && case.id.contains("StartActivityExecution::")
                && case.id.contains(".activity_id:mutation.empty")
        })
        .ok_or("missing activity mutation")?;
    let prepared = probe(&ProbeRequest {
        operation: Operation::Prepare as i32,
        case_id: case.id.clone(),
        namespace: "namespace".into(),
        activity_id: "activity".into(),
        task_queue: "queue".into(),
        ..Default::default()
    })?;
    let metadata = prepared.cases.first().ok_or("missing actual prepared metadata")?;
    assert!(!metadata.predictions.is_empty(), "actual mutated request must be validated");
    let observations = inventory.cases.iter().map(reference_observation).collect();
    let mut request =
        ProbeRequest { operation: Operation::Classify as i32, observations, ..Default::default() };
    let index = request
        .observations
        .iter()
        .position(|observation| observation.case_id == case.id)
        .ok_or("missing observation")?;
    request.observations[index].prediction_count = u32::try_from(metadata.predictions.len())?;
    request.observations[index].predictions = metadata
        .predictions
        .iter()
        .zip(0_u32..)
        .map(|(prediction, index)| ProbeMatch {
            index,
            prediction_id: prediction.id.clone(),
            message_matches: false,
            ..Default::default()
        })
        .collect();
    request.observations[index].accepted = true;
    request.observations[index].code = 0;
    assert!(probe(&request)?.fails, "acceptance despite a violated reject rule must fail");
    assert_invalid_generic_metadata(&request, index);
    let ignored = inventory
        .cases
        .iter()
        .find(|case| {
            case.generic
                && case.id.contains("StartActivityExecution::")
                && case.id.ends_with("StartActivityExecutionRequest.header:mutation.empty")
        })
        .ok_or("missing unreviewed ignored header")?;
    assert_eq!(ignored.predictions.len(), 0);
    assert!(dispositions::unannotated().iter().all(|(id, _)| id != &ignored.id));
    let ignored_index = request
        .observations
        .iter()
        .position(|observation| observation.case_id == ignored.id)
        .ok_or("missing ignored observation")?;
    request.observations[index].accepted = false;
    request.observations[index].code = metadata.predictions[0].code;
    request.observations[index].predictions[0].message_matches = true;
    assert!(!probe(&request)?.fails, "one matched rejection explains the generic outcome");
    request.observations[ignored_index].accepted = false;
    request.observations[ignored_index].code = 3;
    let result = probe(&request)?;
    assert!(
        result.fails && result.report.contains("Unannotated"),
        "unpredicted real rejection must surface"
    );
    Ok(())
}

fn assert_invalid_generic_metadata(request: &ProbeRequest, index: usize) {
    for variant in 0..10 {
        let mut invalid = request.clone();
        let observation = &mut invalid.observations[index];
        match variant {
            0 => observation.prediction_count += 1,
            1 => observation.predictions[0].index = 256,
            2 => observation.predictions[0].prediction_id.push_str("stale"),
            3 => observation.applicability = 99,
            4 => {
                observation.applicability = 2;
                observation.refusal = "unreviewed local excuse".into();
            }
            5 => observation.message_matches = true,
            6 => {
                observation.predictions.push(observation.predictions[0].clone());
                observation.prediction_count += 1;
            }
            7 => {
                observation.predictions[0].prediction_id = observation.predictions[0]
                    .prediction_id
                    .replacen("activity_id:mutation.empty", "namespace:mutation.empty", 1);
            }
            _ => {
                observation.predictions.clear();
                observation.prediction_count = u32::from(variant == 8);
            }
        }
        assert!(probe(&invalid).is_err(), "invalid metadata variant {variant}");
    }
}

#[path = "../src/probes/dispositions.rs"]
mod dispositions;

#[test]
fn generic_indices_bind_declared_prediction_order() -> Result<(), Box<dyn std::error::Error>> {
    let inventory = probe(&ProbeRequest::default())?;
    let case = inventory
        .cases
        .iter()
        .find(|case| case.generic && case.predictions.len() > 1)
        .ok_or("missing multiple predictions")?;
    let mut request = ProbeRequest {
        operation: Operation::Classify as i32,
        observations: inventory.cases.iter().map(reference_observation).collect(),
        ..Default::default()
    };
    let observation = request
        .observations
        .iter_mut()
        .find(|observation| observation.case_id == case.id)
        .ok_or("missing observation")?;
    observation.predictions[0].index = 1;
    observation.predictions[1].index = 0;
    assert!(
        probe(&request).is_err(),
        "indices must bind the declared identity, not only be unique"
    );
    Ok(())
}

#[test]
fn generic_group_predictions_name_rules_not_strict_omission_witnesses()
-> Result<(), Box<dyn std::error::Error>> {
    let inventory = probe(&ProbeRequest::default())?;
    let case = inventory
        .cases
        .iter()
        .find(|case| {
            case.generic
                && case.id.contains("StartActivityExecution::")
                && case.id.contains(".task_queue:mutation.absent")
        })
        .ok_or("missing task queue absence")?;
    let prediction = case
        .predictions
        .iter()
        .find(|prediction| prediction.id.contains(":required:"))
        .ok_or("missing required group prediction")?;
    assert!(
        !prediction.id.contains(":omit="),
        "generic rule attribution must not borrow a different strict omission witness"
    );
    Ok(())
}

#[test]
fn every_generic_attempt_checks_actual_context_against_canonical_predictions()
-> Result<(), Box<dyn std::error::Error>> {
    let inventory = probe(&ProbeRequest::default())?;
    let cases: Vec<_> = inventory.cases.iter().filter(|case| case.generic).collect();
    assert_eq!(cases.len(), 2238);
    for context in [false, true] {
        let slots = ProbeRequest {
            operation: Operation::Prepare as i32,
            namespace: if context { "second-namespace" } else { "first-namespace" }.into(),
            activity_id: if context { "12345678-1234-4234-8234-123456789abc" } else { "activity" }
                .into(),
            task_queue: if context { "second-queue" } else { "queue" }.into(),
            run_id: if context {
                "87654321-4321-4321-8321-cba987654321"
            } else {
                "12345678-1234-4234-8234-123456789abc"
            }
            .into(),
            task_token: if context { vec![7; 1001] } else { vec![8, 1] },
            ..Default::default()
        };
        for case in &cases {
            let prepared = probe(&ProbeRequest { case_id: case.id.clone(), ..slots.clone() })
                .map_err(|error| format!("{} context={context}: {error}", case.id))?;
            let actual = prepared.cases.first().ok_or("missing actual metadata")?;
            assert_eq!(actual.predictions, case.predictions, "{} context={context}", case.id);
            if actual.applicability == 3 {
                assert_eq!(actual.predictions.len(), 0);
            }
        }
    }
    Ok(())
}

#[test]
fn repeated_parallel_probes_materialize_and_validate_current_slots()
-> Result<(), Box<dyn std::error::Error>> {
    use prost::Message;
    use umpire_temporal_api::temporal::api::workflowservice::v1::DescribeActivityExecutionRequest;
    let inventory = probe(&ProbeRequest::default())?;
    let case = inventory
        .cases
        .iter()
        .find(|case| {
            case.generic
                && case.id.ends_with("DescribeActivityExecutionRequest.activity_id:mutation.empty")
        })
        .ok_or("missing activity mutation")?;
    std::thread::scope(|scope| -> Result<(), Box<dyn std::error::Error>> {
        let handles: Vec<_> = ["first", "second"]
            .into_iter()
            .map(|context| {
                let case_id = case.id.clone();
                scope.spawn(move || -> Result<(), String> {
                    let request = ProbeRequest {
                        operation: Operation::Prepare as i32,
                        case_id,
                        namespace: format!("{context}-namespace"),
                        activity_id: format!("{context}-activity"),
                        task_queue: format!("{context}-queue"),
                        run_id: "12345678-1234-4234-8234-123456789abc".into(),
                        ..Default::default()
                    };
                    let first = probe(&request).map_err(|error| error.to_string())?;
                    for _ in 0..3 {
                        assert_eq!(probe(&request).map_err(|error| error.to_string())?, first);
                        let actual =
                            DescribeActivityExecutionRequest::decode(first.message.as_slice())
                                .map_err(|error| error.to_string())?;
                        assert_eq!(actual.namespace, request.namespace);
                        assert_eq!(actual.run_id, request.run_id);
                        assert_eq!(actual.activity_id, "");
                    }
                    let invalid = ProbeRequest { activity_id: "x".repeat(1001), ..request.clone() };
                    assert_eq!(
                        probe(&invalid)
                            .err()
                            .ok_or("invalid current baseline admitted")?
                            .to_string(),
                        "invalid generic probe baseline"
                    );
                    let missing = ProbeRequest { namespace: String::new(), ..request };
                    assert_eq!(
                        probe(&missing).err().ok_or("missing current slots admitted")?.to_string(),
                        "missing namespace/activity/queue prerequisite"
                    );
                    Ok(())
                })
            })
            .collect();
        for handle in handles {
            handle.join().map_err(|_| "probe thread panicked")??;
        }
        Ok(())
    })
}

fn reference_observation(case: &umpire_runtime::annotations::ProbeCase) -> ProbeObservation {
    if case.generic {
        let rejects = case.predictions.iter().find(|prediction| !prediction.accepts);
        ProbeObservation {
            case_id: case.id.clone(),
            accepted: case.applicability == 0 && rejects.is_none(),
            code: rejects
                .map_or(0, |prediction| if prediction.code == 0 { 3 } else { prediction.code }),
            applicability: case.applicability,
            refusal: case.refusal.clone(),
            prediction_count: u32::try_from(case.predictions.len()).unwrap_or(u32::MAX),
            predictions: case
                .predictions
                .iter()
                .zip(0_u32..)
                .map(|(prediction, index)| ProbeMatch {
                    index,
                    prediction_id: prediction.id.clone(),
                    message_matches: true,
                    details_type: prediction.details_type.clone(),
                })
                .collect(),
            ..Default::default()
        }
    } else {
        ProbeObservation {
            case_id: case.id.clone(),
            accepted: case.accepts,
            code: case.code,
            message_matches: true,
            details_type: case.details_type.clone(),
            ..Default::default()
        }
    }
}

#[test]
fn source_dispositions_validate_before_applying_a_dated_waiver()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_annotations::{
        AcceptedGap, Expectation, Expected, GapEntry, GapOutcome, ProbeResult, Violation,
    };
    let gap =
        AcceptedGap { date: "2024-02-29".into(), reason: "Source-reviewed discrepancy".into() };
    let registry = dispositions::Dispositions::new(["case"], vec![("case".into(), gap.clone())])?;
    let mut row = GapEntry::checked(
        Violation {
            path: "request.field".into(),
            constraint: "string.uuid".into(),
            fields: Vec::new(),
            expected: Some(Expected {
                expectation: Expectation::Rejects {
                    code: 3,
                    message_pattern: "invalid".into(),
                    details_type: None,
                },
            }),
        },
        ProbeResult::Accepted,
    )?;
    assert!(row.fails());
    let mut general_row = row.clone();
    registry.apply("case", &mut general_row);
    assert!(!general_row.fails(), "general dated dispositions must retain Accepted support");
    assert_eq!(general_row.accepted_gap, Some(gap.clone()));
    general_row.outcome = GapOutcome::Mismatch(ProbeResult::Rejected {
        code: 5,
        message_matches: false,
        details_type: None,
    });
    general_row.accepted_gap = None;
    registry.apply("case", &mut general_row);
    assert!(!general_row.fails(), "general dated dispositions retain Mismatch support");
    let registry = dispositions::Dispositions::with_unannotated(
        ["case"],
        Vec::new(),
        vec![("case".into(), gap.clone())],
    )?;
    registry.apply("case", &mut row);
    assert!(row.fails(), "a reviewed Unannotated rejection cannot waive Accepted");
    assert_eq!(row.accepted_gap, None);
    let rejected =
        |code| ProbeResult::Rejected { code, message_matches: false, details_type: None };
    for (id, outcome, allowed) in [
        ("case", GapOutcome::Unannotated(rejected(3)), true),
        ("case", GapOutcome::Unannotated(rejected(5)), false),
        ("case", GapOutcome::Mismatch(rejected(3)), false),
        ("case", GapOutcome::Accepted, false),
        ("new-case", GapOutcome::Unannotated(rejected(3)), false),
    ] {
        let mut candidate = row.clone();
        candidate.outcome = outcome;
        registry.apply(id, &mut candidate);
        assert_eq!(candidate.accepted_gap.is_some(), allowed, "{id}: {:?}", candidate.outcome);
        assert_eq!(candidate.fails(), !allowed);
    }
    let invalid = [
        vec![("case".into(), AcceptedGap { date: "2026-02-30".into(), ..gap.clone() })],
        vec![("case".into(), AcceptedGap { reason: " \t ".into(), ..gap.clone() })],
        vec![("stale".into(), gap.clone())],
        vec![("case".into(), gap.clone()), ("case".into(), gap.clone())],
    ];
    for registry in invalid {
        assert!(dispositions::Dispositions::new(["case"], registry.clone()).is_err());
        assert!(
            dispositions::Dispositions::with_unannotated(["case"], Vec::new(), registry).is_err()
        );
    }
    assert!(
        dispositions::Dispositions::with_unannotated(
            ["case"],
            vec![("case".into(), gap.clone())],
            vec![("case".into(), gap)],
        )
        .is_err()
    );
    let known = dispositions::unannotated();
    assert_eq!(known.len(), 217, "only the explicitly reviewed observations are scoped");
    assert!(known.iter().all(|(_, gap)| gap.date == "2026-10-21"));
    let inventory = probe(&ProbeRequest::default())?;
    dispositions::Dispositions::with_unannotated(
        inventory.cases.iter().map(|case| case.id.as_str()),
        Vec::new(),
        known,
    )?;
    Ok(())
}

#[test]
fn declaration_universe_includes_nested_response_expectations()
-> Result<(), Box<dyn std::error::Error>> {
    let declared =
        umpire_temporal::activity_api::catalog::expected_probe_cases(&Limits::default())?;
    assert!(
        declared
            .iter()
            .any(|case| case.violation.path.ends_with("TimeoutFailureInfo.timeout_type"))
    );
    let mut declared = declared;
    declared.extend(
        umpire_temporal::activity_api::worker_failure_presence(&Limits::default())?.probe_cases(),
    );
    let registered: std::collections::BTreeSet<_> = probe(&ProbeRequest::default())?
        .cases
        .into_iter()
        .filter(|case| !case.generic)
        .map(|case| {
            case.id.split_once("::").map(|(_, target)| target.to_owned()).ok_or("missing context")
        })
        .collect::<Result<_, _>>()?;
    let mut registered: Vec<_> = registered.into_iter().collect();
    umpire_annotations::verify_probe_coverage(&declared, &registered)?;
    registered.pop();
    assert!(umpire_annotations::verify_probe_coverage(&declared, &registered).is_err());
    Ok(())
}

#[test]
fn every_inventory_case_has_an_isolated_decoded_witness() -> Result<(), Box<dyn std::error::Error>>
{
    let inventory = probe(&ProbeRequest::default())?;
    assert_ne!(inventory.cases.len(), 0);
    for case in inventory.cases.into_iter().filter(|case| !case.generic) {
        let request = ProbeRequest {
            operation: Operation::Prepare as i32,
            case_id: case.id.clone(),
            namespace: "probe-namespace".into(),
            activity_id: "probe-activity".into(),
            task_queue: "probe-queue".into(),
            run_id: "12345678-1234-4234-8234-123456789abc".into(),
            task_token: vec![8, 1],
            ..Default::default()
        };
        let baseline = probe(&ProbeRequest { baseline: true, ..request.clone() })?;
        assert!(!baseline.message.is_empty(), "{}", case.id);
        let witness = probe(&request).map_err(|error| format!("{}: {error}", case.id))?;
        assert!(!witness.message.is_empty() || witness.recipe.is_some(), "{}", case.id);
    }
    assert!(
        probe(&ProbeRequest { operation: Operation::Classify as i32, ..Default::default() })
            .is_err()
    );
    assert!(
        probe(&ProbeRequest {
            operation: Operation::Prepare as i32,
            case_id: "unknown".into(),
            ..Default::default()
        })
        .is_err()
    );
    Ok(())
}

#[test]
fn classification_requires_complete_unique_real_observations()
-> Result<(), Box<dyn std::error::Error>> {
    let inventory = probe(&ProbeRequest::default())?;
    let observations = inventory.cases.iter().map(reference_observation).collect();
    let request =
        ProbeRequest { operation: Operation::Classify as i32, observations, ..Default::default() };
    let result = probe(&request)?;
    assert!(prost::Message::encoded_len(&result) < 1024 * 1024 - 1024);
    assert!(prost::Message::encoded_len(&request) < 1024 * 1024 - 1024);
    assert!(!result.fails);
    assert!(result.report.ends_with("Loose expectations: 0\n"));
    assert!(
        !result
            .report
            .split("Generic case:")
            .next()
            .ok_or("missing strict report")?
            .contains("Unverifiable")
    );
    assert!(result.report.contains("Companion:"));
    let strict_count = inventory.cases.iter().filter(|case| !case.generic).count();
    assert_eq!(result.report.matches("Case:").count(), strict_count);
    let accepts_count = inventory.cases.iter().filter(|case| case.accepts).count();
    assert_eq!(result.report.matches("[expects: Accepts]").count(), accepts_count);
    assert_eq!(result.report.matches("[expects: Rejects]").count(), strict_count - accepts_count);
    assert!(result.report.contains(&format!("cases={} confirmed={} mismatch=0 accepted=0 unverifiable=0 intentional_accepts={accepts_count} expected_rejects={}", strict_count, strict_count, strict_count - accepts_count)));
    let reject =
        inventory.cases.iter().position(|case| !case.accepts).ok_or("missing rejection")?;
    let accepts =
        inventory.cases.iter().position(|case| case.accepts).ok_or("missing acceptance")?;
    for code in [0, 17] {
        let mut invalid = request.clone();
        invalid.observations[reject].code = code;
        assert!(probe(&invalid).is_err(), "invalid strict status {code}");
    }
    for variant in 0..4 {
        let mut changed = request.clone();
        match variant {
            0 => changed.observations[reject].code += 1,
            1 => changed.observations[reject].message_matches = false,
            2 => {
                changed.observations[reject].accepted = true;
                changed.observations[reject].code = 0;
            }
            _ => {
                changed.observations[accepts].accepted = false;
                changed.observations[accepts].code = 3;
            }
        }
        assert!(probe(&changed)?.fails);
    }
    let mut incomplete = request.clone();
    incomplete.observations.pop();
    assert!(probe(&incomplete).is_err());
    let mut duplicated = request.clone();
    duplicated.observations[0] = duplicated.observations[1].clone();
    assert!(probe(&duplicated).is_err());
    let mut unknown = request;
    unknown.observations[0].case_id = "unknown".into();
    assert!(probe(&unknown).is_err());
    Ok(())
}

#[test]
fn unknown_start_policies_match_the_handler_rejections() -> Result<(), Box<dyn std::error::Error>> {
    let inventory = probe(&ProbeRequest::default())?;
    let mut observations: Vec<_> = inventory.cases.iter().map(reference_observation).collect();
    for field in ["id_reuse_policy", "id_conflict_policy"] {
        let index = inventory
            .cases
            .iter()
            .position(|case| case.id.ends_with(&format!(".{field}:enum.defined_only")))
            .ok_or("missing policy probe")?;
        observations[index].accepted = false;
        observations[index].code = 3;
    }
    let result = probe(&ProbeRequest {
        operation: Operation::Classify as i32,
        observations,
        ..Default::default()
    })?;
    assert!(!result.fails, "{}", result.report);
    for (field, pattern) in [
        ("id_reuse_policy", "unsupported ID reuse policy"),
        ("id_conflict_policy", "unsupported ID conflict policy"),
    ] {
        let case = inventory
            .cases
            .iter()
            .find(|case| case.id.ends_with(&format!(".{field}:enum.defined_only")))
            .ok_or("missing policy probe")?;
        assert!(!case.accepts);
        assert_eq!(case.code, 3);
        assert_eq!(case.message_pattern, pattern);
    }
    Ok(())
}
