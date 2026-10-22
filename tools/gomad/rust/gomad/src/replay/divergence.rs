use crate::evidence::{canonical, record::Record};
use gomad_wasmhost::linked;

#[must_use]
pub fn first(record: &Record, observed: &linked::Execution) -> Option<String> {
    if let Some(divergence) = &observed.outcome.divergence {
        return Some(format!(
            "choice_profile.divergence.ordinal[{}].{}",
            divergence.ordinal,
            divergence.reason_name()
        ));
    }
    let mut reason = observed.outcome.message.clone();
    if let Some(stream) = record.streams.get("evidence/soft-assertions.json") {
        let mut matched = false;
        for report_only in [false, true] {
            let report = crate::campaign::soft_assertions::observe(
                observed.evidence.get("stdout").map_or(&[], Vec::as_slice),
                observed.evidence.get("stderr").map_or(&[], Vec::as_slice),
                matches!(observed.outcome.termination, "exit" | "trap"),
                report_only,
            );
            if let Some(report) = report
                && canonical::encode(&report)
                    .is_ok_and(|bytes| canonical::digest(&bytes) == stream.sha256)
            {
                reason = crate::campaign::soft_assertions::reason(Some(&report), reason);
                matched = true;
                break;
            }
        }
        if !matched {
            return Some("soft_assertions.observation".into());
        }
    }
    for (different, path) in [
        (
            record.outcome.termination != observed.outcome.termination,
            "outcome.termination",
        ),
        (
            record.outcome.exit_code != observed.outcome.exit_code,
            "outcome.exit_code",
        ),
        (record.outcome.reason != reason, "outcome.reason"),
    ] {
        if different {
            return Some(path.into());
        }
    }
    for (name, path) in [
        ("cluster_summary", "backend.evidence.sha256"),
        ("cluster_oracles", "backend.evidence.sha256"),
        ("stdout", "stdout.full_sha256"),
        ("stderr", "stderr.full_sha256"),
        ("transcript", "io_profile.transcript.sha256"),
        ("choice", "choice_profile.trace.sha256"),
        ("diagnostic", "diagnostic_profile.trace.sha256"),
        ("network", "backend.evidence.sha256"),
        ("storage", "backend.evidence.sha256"),
        ("storage_state", "backend.evidence.sha256"),
        ("storage_image", "backend.evidence.sha256"),
    ] {
        let retained = format!(
            "{}/{name}",
            if matches!(name, "stdout" | "stderr") {
                "output"
            } else {
                "evidence"
            }
        );
        let expected = record
            .streams
            .get(&retained)
            .map_or_else(|| canonical::digest(&[]), |stream| stream.sha256.clone());
        if canonical::digest(observed.evidence.get(name).map_or(&[], Vec::as_slice)) != expected {
            return Some(path.into());
        }
    }
    let now = observed
        .outcome
        .local_model
        .as_deref()
        .and_then(|model| model["now"].as_u64())
        .unwrap_or(0);
    (record.virtual_time_nanos != now).then(|| "backend.evidence.sha256".into())
}
