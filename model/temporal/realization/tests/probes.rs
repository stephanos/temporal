//! Typed request witnesses must survive the actual protobuf projection.

use umpire_temporal::foundation::dc::Limits;

use umpire_temporal::probes::probe;
use umpire_temporal_api::umpire::temporal::v1::{ProbeRequest, probe_request::Operation};

#[path = "../src/probes/dispositions.rs"]
mod dispositions;

#[test]
fn source_dispositions_validate_before_applying_a_dated_waiver()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_annotations::{
        AcceptedGap, Expectation, Expected, GapEntry, ProbeResult, Violation,
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
                covers: "validator".into(),
            }),
        },
        ProbeResult::Accepted,
    )?;
    assert!(row.fails());
    registry.apply("case", &mut row);
    assert!(!row.fails());
    assert_eq!(row.accepted_gap, Some(gap.clone()));
    let invalid = [
        vec![("case".into(), AcceptedGap { date: "2026-02-30".into(), ..gap.clone() })],
        vec![("case".into(), AcceptedGap { reason: " \t ".into(), ..gap.clone() })],
        vec![("stale".into(), gap.clone())],
        vec![("case".into(), gap.clone()), ("case".into(), gap)],
    ];
    for registry in invalid {
        assert!(dispositions::Dispositions::new(["case"], registry).is_err());
    }
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
    for case in inventory.cases {
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
    use umpire_temporal_api::umpire::temporal::v1::ProbeObservation;
    let inventory = probe(&ProbeRequest::default())?;
    let observations: Vec<_> = inventory
        .cases
        .iter()
        .map(|case| ProbeObservation {
            case_id: case.id.clone(),
            accepted: case.accepts,
            code: case.code,
            message_matches: true,
            details_type: case.details_type.clone(),
        })
        .collect();
    let request =
        ProbeRequest { operation: Operation::Classify as i32, observations, ..Default::default() };
    let result = probe(&request)?;
    assert!(!result.fails);
    assert!(!result.report.contains("Unverifiable"));
    assert!(result.report.contains("Companion:"));
    assert_eq!(result.report.matches("Case:").count(), inventory.cases.len());
    let accepts_count = inventory.cases.iter().filter(|case| case.accepts).count();
    assert_eq!(result.report.matches("[expects: Accepts]").count(), accepts_count);
    assert_eq!(
        result.report.matches("[expects: Rejects]").count(),
        inventory.cases.len() - accepts_count
    );
    assert!(result.report.contains(&format!("cases={} confirmed={} mismatch=0 accepted=0 unverifiable=0 intentional_accepts={accepts_count} expected_rejects={}", inventory.cases.len(), inventory.cases.len(), inventory.cases.len() - accepts_count)));
    let reject =
        inventory.cases.iter().position(|case| !case.accepts).ok_or("missing rejection")?;
    let accepts =
        inventory.cases.iter().position(|case| case.accepts).ok_or("missing acceptance")?;
    for variant in 0..4 {
        let mut changed = request.clone();
        match variant {
            0 => changed.observations[reject].code += 1,
            1 => changed.observations[reject].message_matches = false,
            2 => changed.observations[reject].accepted = true,
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
    use umpire_temporal_api::umpire::temporal::v1::ProbeObservation;
    let inventory = probe(&ProbeRequest::default())?;
    let mut observations: Vec<_> = inventory
        .cases
        .iter()
        .map(|case| ProbeObservation {
            case_id: case.id.clone(),
            accepted: case.accepts,
            code: case.code,
            message_matches: true,
            details_type: String::new(),
        })
        .collect();
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
        assert_eq!(
            case.covers,
            "go.temporal.io/server/chasm/lib/activity.handler.StartActivityExecution"
        );
    }
    Ok(())
}
