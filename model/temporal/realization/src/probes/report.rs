use super::{Entry, Result, dispositions};
use std::fmt::Write;
use umpire_catalog::{
    Expectation, GapEntry, GapOutcome, GapReport, generic_probe::observation_result,
};
use umpire_runtime::annotations::ProbeObservation;
use umpire_runtime::annotations::ProbeResponse;

pub(super) fn classify(
    observations: &[ProbeObservation],
    entries: &[Entry],
    dispositions: &dispositions::Dispositions,
) -> Result<ProbeResponse> {
    let mut text = String::new();
    let mut fails = false;
    let mut outcomes = [0_usize; 4];
    let mut intentional_accepts = 0;

    let cases = entries.len();
    for entry in entries {
        let observation =
            observations.iter().find(|v| v.case_id == entry.id()).ok_or("missing observation")?;
        if observation.applicability != 0
            || !observation.refusal.is_empty()
            || observation.prediction_count != 0
            || !observation.predictions.is_empty()
            || (observation.accepted && observation.code != 0)
            || (!observation.accepted && !(1..=16).contains(&observation.code))
        {
            return Err("strict observation contains generic or inconsistent metadata".into());
        }
        let actual = observation_result(observation);
        let mut row =
            GapEntry::checked(entry.target.violation.clone(), actual)?.metadata(&entry.target);
        dispositions.apply(&entry.id(), &mut row);
        fails |= row.fails();
        let outcome = match &row.outcome {
            GapOutcome::Confirmed => 0,
            GapOutcome::Mismatch(_) => 1,
            GapOutcome::Accepted => 2,
            GapOutcome::Unverifiable => 3,
            GapOutcome::Unannotated(_) => {
                return Err("strict probe produced an unannotated outcome".into());
            }
        };
        outcomes[outcome] += 1;
        let expectation = match &entry
            .target
            .violation
            .expected
            .as_ref()
            .ok_or("missing expectation")?
            .expectation
        {
            Expectation::Accepts => {
                intentional_accepts += 1;
                "Accepts"
            }
            Expectation::Rejects { .. }
            | Expectation::RejectsCode(_)
            | Expectation::RejectsAnyStatus => "Rejects",
            Expectation::Unobservable => return Err("unobservable request expectation".into()),
        };
        write!(
            text,
            "Case: {} [expects: {expectation}]\n{}",
            entry.id(),
            GapReport { entries: vec![row] }
        )?;
        for companion in entry.companions()? {
            writeln!(
                text,
                "Companion: {companion} (Accepts; UUID syntax and excessive request-ID length cannot coexist)"
            )?;
        }
    }
    writeln!(
        text,
        "Summary: cases={cases} confirmed={} mismatch={} accepted={} unverifiable={} intentional_accepts={intentional_accepts} expected_rejects={}",
        outcomes[0],
        outcomes[1],
        outcomes[2],
        outcomes[3],
        cases - intentional_accepts
    )?;
    Ok(ProbeResponse { report: text, fails, ..Default::default() })
}
