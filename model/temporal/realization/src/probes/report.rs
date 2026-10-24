use super::{Entry, Result, dispositions, generic};
use std::{collections::BTreeSet, fmt::Write};
use umpire_catalog::{Expectation, GapEntry, GapOutcome, GapReport, ProbeResult};
use umpire_runtime::annotations::ProbeResponse;
use umpire_temporal_api::umpire::temporal::v1::ProbeRequest;

pub(super) fn classify(
    request: &ProbeRequest,
    entries: &[Entry],
    generic_entries: &[generic::Entry],
) -> Result<ProbeResponse> {
    let expected: BTreeSet<_> = entries
        .iter()
        .map(Entry::id)
        .chain(generic_entries.iter().map(generic::Entry::id))
        .collect();
    let dispositions = dispositions::Dispositions::with_unannotated(
        expected.iter().map(String::as_str),
        crate::activity_api::accepted_probe_gaps(),
        dispositions::unannotated(),
    )?;
    let actual: BTreeSet<_> = request.observations.iter().map(|v| v.case_id.clone()).collect();
    if actual != expected || actual.len() != request.observations.len() {
        return Err("incomplete, duplicate or unknown probe observations".into());
    }
    let mut text = String::new();
    let mut fails = false;
    let mut outcomes = [0_usize; 4];
    let mut intentional_accepts = 0;
    let mut loose_expectations = 0;
    let cases = entries.len();
    for entry in entries {
        let observation = request
            .observations
            .iter()
            .find(|v| v.case_id == entry.id())
            .ok_or("missing observation")?;
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
            Expectation::Rejects { .. } => "Rejects",
            Expectation::RejectsCode(_) | Expectation::RejectsAnyStatus => {
                loose_expectations += 1;
                "Rejects"
            }
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
    fails |= classify_generic(request, entries, generic_entries, &dispositions, &mut text)?;
    writeln!(text, "Loose expectations: {loose_expectations}")?;
    Ok(ProbeResponse { report: text, fails, ..Default::default() })
}

fn classify_generic(
    request: &ProbeRequest,
    entries: &[Entry],
    generic_entries: &[generic::Entry],
    dispositions: &dispositions::Dispositions,
    text: &mut String,
) -> Result<bool> {
    let mut fails = false;
    let mut tested = 0;
    let mut local = 0;
    for entry in generic_entries {
        let observation = request
            .observations
            .iter()
            .find(|v| v.case_id == entry.id())
            .ok_or("missing generic observation")?;
        let mut rows = entry.classify(observation, entries)?;
        for row in &mut rows {
            dispositions.apply(&entry.id(), row);
            fails |= row.fails();
        }
        if observation.applicability == 0 {
            tested += 1;
        } else {
            local += 1;
        }
        writeln!(text, "Generic case: {}", entry.id())?;
        for prediction in &observation.predictions {
            writeln!(text, "Prediction: {}", prediction.prediction_id)?;
        }
        write!(text, "{}", GapReport { entries: rows })?;
    }
    writeln!(text, "Generic summary: tested={tested} local_not_tested={local}")?;
    Ok(fails)
}

fn observation_result(observation: &umpire_runtime::annotations::ProbeObservation) -> ProbeResult {
    if observation.accepted {
        ProbeResult::Accepted
    } else {
        ProbeResult::Rejected {
            code: observation.code,
            message_matches: observation.message_matches,
            details_type: (!observation.details_type.is_empty())
                .then(|| observation.details_type.clone()),
        }
    }
}
