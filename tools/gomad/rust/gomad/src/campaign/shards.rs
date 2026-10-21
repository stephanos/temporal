use super::{
    ledger::{Event, Execution, Job, Journal},
    retention::Retention,
    seeds::SeedSelection,
};
use crate::evidence::{
    canonical,
    store::{Opened, Publication},
};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    io::Write,
    path::{Path, PathBuf},
    time::Duration,
};

fn source_plan(path: &Path) -> Result<Journal, String> {
    let metadata = std::fs::symlink_metadata(path).map_err(|error| error.to_string())?;
    let root = if metadata.is_dir() {
        path
    } else if metadata.is_file() && path.file_name().is_some_and(|name| name == "plan.json") {
        path.parent()
            .filter(|parent| !parent.as_os_str().is_empty())
            .unwrap_or_else(|| Path::new("."))
    } else {
        return Err("campaign plan must name a campaign directory or its plan.json".into());
    };
    let journal = Journal::open(root, false)?;
    if journal.plan()["schema"] != "gomad.seed-campaign-plan/v2"
        || !journal.plan()["shard"].is_null()
        || !journal.plan()["plan_sha256"].is_null()
        || journal.plan()["on_failure"] != "all"
    {
        return Err("portable seed campaigns require an unsharded all-failures plan".into());
    }
    Ok(journal)
}

pub(super) fn execute_shard(
    plan: &Path,
    index: u64,
    count: u64,
    artifacts: &Path,
    json_output: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    if count == 0 || index >= count {
        return Err("campaign shard requires a positive count and index less than count".into());
    }
    let original = source_plan(plan)?;
    let prepared = super::restore(&original)?;
    let mut plan = original.plan().clone();
    plan["plan_sha256"] = canonical::digest(&canonical::encode(&plan)?).into();
    plan["shard"] = json!({"index": index, "count": count});
    let path = super::campaign_path(artifacts)?;
    super::retain_prepared(&path, &prepared, &mut plan)?;
    let bound = plan["journal_bytes"]
        .as_u64()
        .ok_or("campaign journal bound missing")?;
    let duration = Duration::from_millis(
        plan["overall_millis"]
            .as_u64()
            .ok_or("campaign overall bound missing")?,
    );
    let mut journal = Journal::create(&path, &plan, bound)?;
    super::run(&mut journal, &prepared, duration, json_output, output)
}

fn same_plan(source: &Value, original: &Value, digest: &str) -> Result<(), String> {
    if source["plan_sha256"].as_str() != Some(digest) || source["shard"].is_null() {
        return Err("campaign shard portable plan identity mismatch".into());
    }
    let mut normalized = source.clone();
    normalized
        .as_object_mut()
        .ok_or("invalid campaign shard plan")?
        .remove("plan_sha256");
    normalized["shard"] = Value::Null;
    normalized["prepared_artifact"] = original["prepared_artifact"].clone();
    if &normalized != original {
        return Err("campaign shard changed its immutable plan bounds or identity".into());
    }
    Ok(())
}

fn executions(journal: &Journal) -> Vec<Execution> {
    journal
        .events()
        .iter()
        .filter_map(|event| match event {
            Event::Outcome { executions, .. } => Some(executions.as_slice()),
            _ => None,
        })
        .flatten()
        .cloned()
        .collect()
}

fn missing(total: u64, ordinals: impl Iterator<Item = u64>) -> Result<Vec<Value>, String> {
    let mut next = 0;
    let mut missing = Vec::new();
    for ordinal in ordinals {
        if ordinal < next || ordinal >= total {
            return Err("campaign merge has duplicate or out-of-bounds ordinals".into());
        }
        if ordinal != next {
            missing.push(json!({"start": next, "end": ordinal - 1}));
        }
        next = ordinal
            .checked_add(1)
            .ok_or("campaign merge ordinal overflow")?;
    }
    if next != total {
        missing.push(json!({"start": next, "end": total - 1}));
    }
    Ok(missing)
}

fn copy_artifact(opened: &Opened, output: &Journal) -> Result<(String, u64), String> {
    let pinned = crate::evidence::store::open::pin(output.path())?;
    let actual = rustix::fs::fstat(output.root()).map_err(|error| error.to_string())?;
    let named = rustix::fs::fstat(&pinned).map_err(|error| error.to_string())?;
    if actual.st_dev != named.st_dev || actual.st_ino != named.st_ino {
        return Err("merged campaign root substituted before payload publication".into());
    }
    let payloads = opened
        .manifest()
        .files
        .iter()
        .map(|(name, entry)| {
            opened
                .payload(name, entry.bytes)
                .map(|bytes| (name.clone(), bytes))
        })
        .collect::<Result<BTreeMap<_, _>, _>>()?;
    let published =
        super::store(output.path().join("evidence"), u64::MAX).publish(Publication {
            record: opened.manifest().record.clone(),
            payloads,
        })?;
    let reference = published
        .path
        .strip_prefix(output.path())
        .map_err(|error| error.to_string())?
        .to_str()
        .ok_or("merged artifact path is not UTF-8")?
        .to_owned();
    Ok((reference, published.stored_bytes))
}

#[expect(
    clippy::too_many_lines,
    reason = "Merge validates every source before publishing any aggregate state"
)]
pub(super) fn merge(
    plan_path: &Path,
    shard_paths: &[PathBuf],
    output_path: &Path,
    partial: bool,
    json_output: bool,
    output: &mut dyn Write,
) -> Result<i32, String> {
    if shard_paths.is_empty() {
        return Err("campaign merge requires at least one shard".into());
    }
    let original = source_plan(plan_path)?;
    let plan = original.plan();
    let digest = canonical::digest(&canonical::encode(plan)?);
    let selection = SeedSelection::parse(
        plan["selection"]
            .as_str()
            .ok_or("campaign selection missing")?,
    )?;
    if plan["selection_count"].as_u64() != Some(selection.count()) {
        return Err("campaign selection count mismatch".into());
    }
    let mut sources = Vec::new();
    let mut assignments = BTreeSet::new();
    let mut count = None;
    let mut ordered = BTreeMap::new();
    let mut reports = Vec::new();
    for path in shard_paths {
        let source = Journal::open(path, false)?;
        same_plan(source.plan(), plan, &digest)?;
        let (index, source_count) = super::shard(source.plan())?;
        if !assignments.insert(index) || count.is_some_and(|count| count != source_count) {
            return Err("campaign merge duplicates a shard or mixes shard counts".into());
        }
        count = Some(source_count);
        let mut report = source.fingerprints()?;
        report["index"] = index.into();
        report["count"] = source_count.into();
        reports.push((index, report));
        let complete = matches!(source.events().last(), Some(Event::Complete { stop_reason }) if stop_reason == "seeds_exhausted");
        if !complete && !partial {
            return Err("campaign merge requires complete source shards unless --partial".into());
        }
        let source_index = sources.len();
        for execution in executions(&source) {
            if selection.seed_at(execution.ordinal) != Some(execution.seed)
                || ordered
                    .insert(execution.ordinal, (source_index, execution))
                    .is_some()
            {
                return Err(
                    "campaign merge execution overlaps or differs from the plan selection".into(),
                );
            }
        }
        sources.push(source);
    }
    let missing = missing(selection.count(), ordered.keys().copied())?;
    if !missing.is_empty() && !partial {
        return Err(format!(
            "campaign merge is missing {} ordinal ranges",
            missing.len()
        ));
    }
    let prepared = super::restore(&original)?;
    let mut merged_plan = plan.clone();
    reports.sort_unstable_by_key(|(index, _)| *index);
    merged_plan["schema"] = "gomad.merged-seed-campaign-plan/v2".into();
    merged_plan["plan_sha256"] = digest.into();
    merged_plan["merge_missing"] = missing.into();
    merged_plan["merge_sources"] = reports
        .into_iter()
        .map(|(_, report)| report)
        .collect::<Vec<_>>()
        .into();
    super::retain_prepared(output_path, &prepared, &mut merged_plan)?;
    let bound = merged_plan["journal_bytes"]
        .as_u64()
        .ok_or("campaign journal bound missing")?;
    let mut merged = Journal::create(output_path, &merged_plan, bound)?;
    let mut committed = Vec::new();
    let mut failures = BTreeMap::<String, (String, u64)>::new();
    let mut retention = Retention::default();
    for (_, (source_index, mut execution)) in ordered {
        let source = &sources[source_index];
        let failure = matches!(
            execution.record.outcome.class.as_str(),
            "target_failure" | "watchdog"
        );
        let keep = failure
            || retention.keep(
                plan["keep_successes"]
                    .as_str()
                    .ok_or("campaign retention policy missing")?,
                &execution,
            );
        if keep {
            let signature = execution.record.failure_signature()?;
            let retained = if failure {
                failures.get(&signature).cloned()
            } else {
                None
            };
            let (reference, bytes) = if let Some(retained) = retained {
                retained
            } else {
                let reference = execution
                    .artifact
                    .as_deref()
                    .ok_or("source shard omitted evidence required by merged retention")?;
                let opened = source.open_artifact(reference)?;
                copy_artifact(&opened, &merged)?
            };
            if failure {
                failures.insert(signature, (reference.clone(), bytes));
            } else {
                let maximum = retention.remaining(
                    plan["success_limit"]
                        .as_u64()
                        .ok_or("campaign success bound missing")?,
                    plan["success_bytes"]
                        .as_u64()
                        .ok_or("campaign success byte bound missing")?,
                )?;
                if bytes > maximum {
                    return Err("capacity: merged success evidence exceeds the plan bound".into());
                }
            }
            execution.artifact = Some(reference);
            execution.stored_bytes = bytes;
        } else {
            execution.artifact = None;
            execution.stored_bytes = 0;
        }
        let job = Job {
            ordinal: execution.ordinal,
            seed: execution.seed,
        };
        merged.append(Event::Admit { jobs: vec![job] })?;
        merged.append(Event::Receipt {
            execution: execution.clone(),
        })?;
        merged.append(Event::Outcome {
            executions: vec![execution.clone()],
            admitted: Vec::new(),
            stop_reason: None,
        })?;
        retention.retained(&execution)?;
        committed.push(execution);
    }
    for source in &sources {
        let (index, _) = super::shard(source.plan())?;
        let mut actual = source.fingerprints()?;
        actual["index"] = index.into();
        actual["count"] = count.ok_or("campaign shard count missing")?.into();
        if !merged.plan()["merge_sources"]
            .as_array()
            .ok_or("merged source reports missing")?
            .iter()
            .any(|report| report == &actual)
        {
            return Err("campaign source changed during merge publication".into());
        }
    }
    merged.append(Event::Complete {
        stop_reason: "seeds_exhausted".into(),
    })?;
    super::report(&merged, &committed, "seeds_exhausted", json_output, output)
}
