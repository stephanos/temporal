//! Descriptor probe inventory, observation correlation and catalog coverage controls.

type Result<T = ()> = std::result::Result<T, Box<dyn std::error::Error>>;
use umpire_reference_kettle::{probes, wire::umpire::reference::kettle::v1::ProbeRequest};

#[test]
fn inventory_and_preparation_are_deterministic() -> Result {
    let request = ProbeRequest::default();
    let inventory = probes::run(&request)?;
    assert_ne!(inventory.cases, []);
    assert_eq!(inventory, probes::run(&request)?);
    for case in inventory.cases {
        let prepared = probes::run(&ProbeRequest {
            operation: 1,
            case_id: case.id.clone(),
            kettle_id: "room".into(),
            ..Default::default()
        })?;
        assert_eq!(prepared.cases, vec![case]);
    }
    Ok(())
}

#[test]
fn observations_require_exact_inventory() {
    assert!(probes::run(&ProbeRequest { operation: 2, ..Default::default() }).is_err());
    assert!(
        probes::run(&ProbeRequest {
            operation: 1,
            case_id: "unknown".into(),
            ..Default::default()
        })
        .is_err()
    );
}

#[test]
fn classification_requires_live_evidence_and_correlated_predictions() -> Result {
    use umpire_runtime::annotations::{ProbeMatch, ProbeObservation};
    let inventory = probes::run(&ProbeRequest::default())?;
    // Synthetic server evidence exercises classification only, never golden generation.
    let observations: Vec<_> = inventory
        .cases
        .iter()
        .map(|case| ProbeObservation {
            case_id: case.id.clone(),
            accepted: case.applicability == 0,
            applicability: case.applicability,
            refusal: case.refusal.clone(),
            prediction_count: u32::try_from(case.predictions.len())
                .unwrap_or_else(|_| panic!("too many predictions")),
            predictions: case
                .predictions
                .iter()
                .enumerate()
                .map(|(index, prediction)| ProbeMatch {
                    index: u32::try_from(index)
                        .unwrap_or_else(|_| panic!("prediction index overflow")),
                    prediction_id: prediction.id.clone(),
                    ..Default::default()
                })
                .collect(),
            ..Default::default()
        })
        .collect();
    let request = ProbeRequest { operation: 2, observations, ..Default::default() };
    let classified = probes::run(&request)?;
    assert!(classified.fails);
    assert!(classified.report.contains(": Accepted"));
    assert!(classified.report.contains("Unverifiable"));

    let index = request
        .observations
        .iter()
        .position(|v| !v.predictions.is_empty())
        .ok_or("missing predictions")?;
    for code in [1, 2, 4, 14] {
        let mut transport = request.clone();
        transport.observations[index].accepted = false;
        transport.observations[index].code = code;
        assert!(probes::run(&transport).is_err(), "transport status {code} became a finding");
    }
    let mut crossed = request;
    crossed.observations[index].predictions[0].prediction_id.push_str("foreign");
    assert!(probes::run(&crossed).is_err());
    Ok(())
}

#[test]
fn coverage_matches_catalog_golden() -> Result {
    let coverage = probes::run(&ProbeRequest { operation: 3, ..Default::default() })?.report;
    assert!(coverage.contains("legacy_note"));
    assert!(coverage.contains("Ignored"));
    let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("expected/coverage.snap");
    if std::env::var_os("UMPIRE_UPDATE_KETTLE_COVERAGE").is_some() {
        std::fs::write(&path, &coverage)?;
    }
    assert_eq!(coverage, std::fs::read_to_string(path)?);
    Ok(())
}
