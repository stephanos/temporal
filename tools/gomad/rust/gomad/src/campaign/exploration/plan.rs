use super::round::{RoundJournal, RoundPlan, RoundStrategy};
use crate::{
    campaign::prepare::Prepared, cli::exploration::ExplorationOptions, evidence::canonical,
};
use serde_json::json;
use std::{path::Path, time::Duration};

pub(crate) fn create(
    options: &ExplorationOptions,
    prepared: &Prepared,
    strategy: RoundStrategy,
) -> Result<RoundJournal, String> {
    let trace = prepared
        .configuration
        .model
        .as_ref()
        .and_then(|model| model["limit"].as_u64())
        .unwrap_or(options.base.choice_bytes);
    let per_round = options
        .max_exploration_result_bytes
        .max(trace)
        .checked_mul(6)
        .and_then(|bytes| bytes.checked_add(16 << 20))
        .ok_or("capacity: exploration journal capacity overflow")?;
    let journal_bytes = options
        .max_executions
        .checked_mul(per_round)
        .ok_or("capacity: exploration journal capacity overflow")?;
    let path = crate::campaign::campaign_path(&options.base.artifacts)?;
    let overall_nanos = u64::try_from(options.base.overall_timeout.as_nanos())
        .map_err(|_| "overall timeout overflow")?;
    let mut metadata = json!({
        "schema":"gomad.exploration-preparation/v1","journal_bytes":journal_bytes,
        "simulation_exploration":matches!(&strategy, RoundStrategy::Simulation(_)),
        "overall_millis":overall_nanos.div_ceil(1_000_000),"overall_nanos":overall_nanos,
        "terminate_grace_nanos":u64::try_from(options.base.terminate_grace.as_nanos()).map_err(|_|"termination grace overflow")?,
        "keep_successes":options.base.keep_successes,"success_limit":options.base.success_limit,
        "success_bytes":options.base.success_bytes,"coverage":options.base.coverage,
        "module_sha256":canonical::digest(&prepared.module),"toolchain":prepared.toolchain,
        "configuration":prepared.configuration,"choice_identity":prepared.choice_identity,
        "host_sha256":env!("GOMAD_HOST_IMPLEMENTATION_SHA256"),
        "helper_sha256":gomad_wasmhost::linked::IMPLEMENTATION_SHA256,
        "scenario":prepared.scenario,"network":prepared.network,
        "soft_assertion_report_only":prepared.soft_assertion_report_only
    });
    crate::campaign::retain_prepared(&path, prepared, &mut metadata)?;
    RoundJournal::create(
        &path,
        &RoundPlan {
            prepared: metadata,
            strategy,
        },
        journal_bytes,
    )
}

pub(crate) fn restore(journal: &RoundJournal) -> Result<Prepared, String> {
    let metadata = &journal.plan().prepared;
    if metadata["host_sha256"] != env!("GOMAD_HOST_IMPLEMENTATION_SHA256")
        || metadata["helper_sha256"] != gomad_wasmhost::linked::IMPLEMENTATION_SHA256
    {
        return Err("exploration implementation identity mismatch".into());
    }
    let opened = journal.open_artifact(
        metadata["prepared_artifact"]
            .as_str()
            .ok_or("exploration prepared artifact missing")?,
    )?;
    let mut prepared = crate::campaign::restore_from_plan(metadata, &opened)?;
    prepared.auxiliary = crate::campaign::prepare::open_observations(
        journal.path(),
        &prepared,
        &opened.manifest().record_sha256,
    )?;
    Ok(prepared)
}

pub(crate) fn duration(journal: &RoundJournal) -> Result<Duration, String> {
    duration_metadata(&journal.plan().prepared)
}

pub(super) fn duration_metadata(metadata: &serde_json::Value) -> Result<Duration, String> {
    let nanos = metadata["overall_nanos"]
        .as_u64()
        .filter(|value| *value > 0)
        .ok_or("exploration overall bound missing")?;
    if metadata["overall_millis"].as_u64() != Some(nanos.div_ceil(1_000_000)) {
        return Err("exploration overall bounds differ".into());
    }
    Ok(Duration::from_nanos(nanos))
}

pub(crate) fn is_round_campaign(path: &Path) -> Result<bool, String> {
    let root = crate::evidence::store::open::pin(path)?;
    let storage = crate::evidence::store::open::read(&root, "storage.json", 4096, false)?;
    let value: serde_json::Value =
        serde_json::from_slice(&storage).map_err(|error| error.to_string())?;
    Ok(value["format"] == "gomad.exploration-round-storage/v1")
}
