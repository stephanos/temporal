mod checkpoint;
#[cfg(feature = "determinism-fixture")]
pub(crate) mod fixture;
pub(crate) mod ordering;
mod predicate;
mod proposals;
pub(crate) mod report;
pub(crate) mod source;

use crate::{
    campaign::exploration::simulation::Dimension,
    evidence::{
        canonical,
        inspection::{self, Details},
        record::Stream,
        store::{Opened, Publication, Store},
    },
    replay::preflight,
};
use checkpoint::{ArtifactReference, Binding, Workspace};
use predicate::Parent;
use proposals::State;
use serde_json::{Value, json};
use std::{os::unix::fs::DirBuilderExt as _, path::Path};

fn insert(publication: &mut Publication, name: &str, bytes: Vec<u8>) {
    if name != "record.json" {
        publication.record.streams.insert(
            name.into(),
            Stream {
                bytes: bytes.len() as u64,
                sha256: canonical::digest(&bytes),
                records: 1,
            },
        );
    }
    publication.payloads.insert(name.into(), bytes);
}

fn store(path: &Path, maximum_bytes: u64) -> Store {
    Store {
        root: path.into(),
        maximum_bytes,
        target_pool: None,
        #[cfg(feature = "determinism-fixture")]
        fault: None,
    }
}

fn metadata(parent: &Parent, state: &State) -> Result<inspection::Minimization, String> {
    let strip = |value: &str| value.strip_prefix("sha256:").unwrap_or(value).to_owned();
    Ok(inspection::Minimization {
        schema: "gomad.minimization-evidence/v1".into(),
        implementation_sha256: strip(&state.implementation()),
        parent_record_hash: parent.opened.manifest().record_sha256.clone(),
        parent_failure_signature: parent.opened.manifest().record.failure_signature()?,
        original_candidate_sha256: strip(&state.original.sha256),
        final_candidate_sha256: strip(&state.current.sha256),
        attempt_budget: state.attempt_budget,
        attempts: state.attempts,
        original_forced_decisions: state.original.overrides.len() as u64,
        final_forced_decisions: state.current.overrides.len() as u64,
        accepted: state
            .accepted
            .iter()
            .map(|reduction| inspection::Reduction {
                kind: reduction.kind.clone(),
                before_sha256: strip(&reduction.before_sha256),
                after_sha256: strip(&reduction.after_sha256),
                removed: reduction
                    .removed
                    .iter()
                    .map(|decision| inspection::Decision {
                        dimension: match decision.dimension {
                            Dimension::Runtime => "runtime",
                            Dimension::Scenario => "scenario",
                            Dimension::Network => "network",
                            Dimension::Storage => "storage",
                            Dimension::Fault => "fault",
                            Dimension::Crash => "crash",
                        }
                        .into(),
                        ordinal: decision.ordinal,
                        identity: strip(&decision.identity),
                    })
                    .collect(),
            })
            .collect(),
        predicate: inspection::Predicate {
            failure_signature: parent.original.failure_signature()?,
            domain: parent.original.outcome.class.clone(),
            reason: parent.original.outcome.reason.clone(),
            termination: parent.original.outcome.termination.clone(),
            replay_match: true,
            choice_replay: "exact".into(),
            simulation_replay: if state.config.is_some() {
                "exact"
            } else {
                "not_present"
            }
            .into(),
        },
    })
}

#[expect(
    clippy::too_many_lines,
    reason = "Reduction commits and publication preserve one ordered checkpoint transaction"
)]
pub(crate) fn run(
    replay: preflight::Prepared,
    path: &Path,
    output: &Path,
    attempt_budget: u64,
    maximum_bytes: Option<u64>,
    resume: bool,
    admission: ordering::Admission,
) -> Result<Value, String> {
    if attempt_budget == 0 || attempt_budget > 1_000_000 || maximum_bytes == Some(0) {
        return Err("invalid minimizer bounds".into());
    }
    let parent = Parent::from_replay(replay)?;
    let initial = State::choice(
        parent.trace.clone(),
        parent.identity.clone(),
        attempt_budget,
    )?
    .with_ordering(admission.ordering.clone())?;
    let maximum_bytes = maximum_bytes.unwrap_or_else(|| {
        parent
            .opened
            .stored_bytes()
            .saturating_add(parent.trace.len() as u64)
            .saturating_add(1 << 20)
    });
    std::fs::DirBuilder::new()
        .recursive(true)
        .mode(0o700)
        .create(output)
        .map_err(|error| error.to_string())?;
    let binding = Binding {
        parent_record_hash: parent.opened.manifest().record_sha256.clone(),
        parent_failure_signature: parent.opened.manifest().record.failure_signature()?,
        implementation_sha256: preflight::implementation()?,
        attempt_budget,
        maximum_bytes,
        ordering: admission.ordering.clone(),
        original_predicate_identity: serde_json::to_value(&parent.original)
            .map_err(|error| error.to_string())?,
    };
    let mut workspace = Workspace::open(
        output,
        binding,
        serde_json::to_value(&initial).map_err(|error| error.to_string())?,
        resume,
    )?;
    let mut state: State = serde_json::from_value(workspace.checkpoint().state.clone())
        .map_err(|error| error.to_string())?;
    if !state.source_matches(&initial)? {
        return Err("minimizer checkpoint source differs from parent".into());
    }
    state.validate()?;
    if workspace.checkpoint().accepted.is_some() == state.accepted.is_empty()
        || workspace.checkpoint().published.is_some() && state.accepted.is_empty()
    {
        return Err("minimizer checkpoint artifact references differ from reductions".into());
    }
    if let Some(reference) = &workspace.checkpoint().accepted {
        let accepted = Opened::open(&workspace.accepted_root().join(&reference.directory))?;
        if !parent.matches(&predicate::behavioral_record(&accepted)?)? {
            return Err("accepted minimizer artifact changed the parent predicate".into());
        }
        if let Some(profile) = Details::read(&accepted)?.and_then(|details| details.simulation)
            && profile.candidate_sha256.trim_start_matches("sha256:")
                != state.current.sha256.trim_start_matches("sha256:")
        {
            return Err("accepted minimizer artifact candidate differs from checkpoint".into());
        }
    }
    if let Some(reference) = &workspace.checkpoint().published {
        let published = Opened::open(&output.join(&reference.directory))?;
        let retained = Details::read(&published)?
            .and_then(|details| details.minimization)
            .ok_or("published minimizer artifact metadata missing")?;
        if !parent.matches(&predicate::behavioral_record(&published)?)?
            || canonical::encode(&retained)? != canonical::encode(&metadata(&parent, &state)?)?
            || published.payload(predicate::ORIGINAL, 16 << 20)? != parent.original.bytes()?
        {
            return Err("published minimizer artifact differs from parent or checkpoint".into());
        }
    }
    while let Some(attempt) = state.next()? {
        #[cfg(feature = "determinism-fixture")]
        crate::live_backend::minimize_control(attempt.candidate.overrides.len() as u64)?;
        let trial = parent.choice(attempt.candidate.overrides.len() as u64, attempt.index)?;
        let behavioral = trial
            .publication
            .payloads
            .get("simulation/behavioral-record.json")
            .map(|bytes| crate::evidence::record::Record::decode(bytes))
            .transpose()?
            .unwrap_or_else(|| trial.publication.record.clone());
        let mut reference = workspace.checkpoint().accepted.clone();
        let mut accepted = false;
        if parent.matches(&behavioral)? && trial.execution.divergence.is_none() {
            let published =
                store(&workspace.accepted_root(), u64::MAX).publish(trial.publication)?;
            if published.stored_bytes > parent.opened.stored_bytes().saturating_add(1 << 20) {
                return Err("accepted minimizer artifact exceeds byte bound".into());
            }
            crate::campaign::execution::retain_backend_metrics(
                &workspace.observation_root(),
                &trial.execution,
                &trial.metrics,
            )?;
            accepted = predicate::exact(&published.path)?;
            if accepted {
                reference = Some(ArtifactReference {
                    directory: published
                        .path
                        .file_name()
                        .and_then(|name| name.to_str())
                        .ok_or("invalid accepted artifact path")?
                        .into(),
                    record_hash: published.record_sha256,
                });
            }
        }
        state = state.commit(&attempt, accepted)?;
        workspace.commit(
            serde_json::to_value(&state).map_err(|error| error.to_string())?,
            reference,
        )?;
        #[cfg(feature = "determinism-fixture")]
        if accepted {
            crate::live_backend::minimize_checkpoint("after accepted")?;
        }
    }
    let mut artifact_path = path.to_path_buf();
    let mut record_hash = parent.opened.manifest().record_sha256.clone();
    if let Some(reference) = &workspace.checkpoint().accepted {
        if let Some(published) = &workspace.checkpoint().published {
            artifact_path = output.join(&published.directory);
            record_hash.clone_from(&published.record_hash);
        } else {
            let accepted = Opened::open(&workspace.accepted_root().join(&reference.directory))?;
            if !parent.matches(&predicate::behavioral_record(&accepted)?)?
                || !predicate::exact(&workspace.accepted_root().join(&reference.directory))?
            {
                return Err(
                    "accepted minimizer artifact does not reproduce parent predicate".into(),
                );
            }
            let mut publication = predicate::publication(&accepted)?;
            let mut details = Details::read(&accepted)?.ok_or("accepted inspection missing")?;
            details.minimization = Some(metadata(&parent, &state)?);
            insert(
                &mut publication,
                predicate::ORIGINAL,
                parent.original.bytes()?,
            );
            insert(
                &mut publication,
                "evidence/inspection.json",
                canonical::encode(&details)?,
            );
            let parent_details =
                Details::read(&parent.opened)?.ok_or("parent inspection missing")?;
            let source = source::Source {
                raw: parent.trace.clone(),
                identity: parent.identity.clone(),
                parent_record_sha256: parent.opened.manifest().record_sha256.clone(),
                limit: parent_details
                    .choices
                    .as_ref()
                    .ok_or("parent choice evidence missing")?
                    .trace
                    .limit,
            };
            source.attach(&mut publication)?;
            if parent
                .opened
                .manifest()
                .files
                .contains_key(crate::evidence::search_context::FILE)
            {
                publication.payloads.insert(
                    crate::evidence::search_context::FILE.into(),
                    parent.opened.payload(
                        crate::evidence::search_context::FILE,
                        crate::evidence::search_context::MAX_BYTES,
                    )?,
                );
            }
            let record = publication.record.bytes()?;
            insert(&mut publication, "record.json", record);
            let staged = store(&workspace.accepted_root(), u64::MAX).publish(publication)?;
            if staged.stored_bytes > maximum_bytes {
                return Err("minimizer publication exceeds byte bound".into());
            }
            let opened = Opened::open(&staged.path)?;
            let published = store(output, u64::MAX).publish(predicate::publication(&opened)?)?;
            artifact_path = published.path;
            record_hash = published.record_sha256;
            workspace.publication(ArtifactReference {
                directory: artifact_path
                    .file_name()
                    .and_then(|name| name.to_str())
                    .ok_or("invalid published artifact path")?
                    .into(),
                record_hash: record_hash.clone(),
            })?;
        }
        #[cfg(feature = "determinism-fixture")]
        crate::live_backend::minimize_checkpoint("after publication")?;
        if !predicate::exact(&artifact_path)? {
            return Err("minimized artifact exact replay diverged".into());
        }
    }
    workspace.complete()?;
    let mut accepted = serde_json::to_value(&state.accepted).map_err(|error| error.to_string())?;
    for (row, reduction) in accepted
        .as_array_mut()
        .ok_or("accepted report is not an array")?
        .iter_mut()
        .zip(&state.accepted)
    {
        let inside = admission.ordering.as_ref().map_or(0, |ordering| {
            reduction
                .removed
                .iter()
                .filter(|decision| ordering.in_scope.contains(*decision))
                .count()
        });
        row["removed_in_scope"] = json!(inside);
        row["removed_out_of_scope"] = json!(reduction.removed.len() - inside);
    }
    let mut result = json!({"artifact_path":artifact_path,"record_hash":record_hash,"changed":!state.accepted.is_empty(),"attempts":state.attempts,"attempt_budget":state.attempt_budget,"accepted":accepted,"stop_reason":state.stop_reason});
    if let Some(scope) = admission.scope {
        result["site_scope"] = serde_json::to_value(scope).map_err(|error| error.to_string())?;
    }
    Ok(result)
}
