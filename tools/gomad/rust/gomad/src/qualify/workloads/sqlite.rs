use super::{
    Workload, boots,
    native_crash::{self, Selection},
};
use crate::{
    campaign::{prepare::Prepared, scenario::Bound},
    evidence::{canonical, store::Opened},
    qualify::{report, result},
    replay::cluster,
};
use std::{collections::BTreeMap, path::Path};

const CAPSULE: &str = "evidence/primary-sqlite/";

fn source(
    path: &Path,
    selection: &Selection,
) -> Result<(result::QualificationResult, BTreeMap<String, Vec<u8>>), String> {
    selection.validate()?;
    let primary = result::open(path)?;
    let (receipt, _) = result::open_receipt(path, &primary)?;
    if primary.manifest.id != "sqlite-durable-recovery"
        || primary.manifest.cell.as_deref() != Some("success")
        || !primary.report.qualified
        || primary.native_crash_selection.is_some()
        || primary.native_crash_catalogues.len() != 1
        || canonical::digest(&result::encode(&primary)?) != selection.primary_result_sha256
    {
        return Err(
            "SQLite runtime selection requires its actual qualified primary Rust Result".into(),
        );
    }
    let catalogue = &primary.native_crash_catalogues[0];
    if catalogue.context != selection.context
        || !catalogue
            .enumeration
            .states
            .as_ref()
            .is_some_and(|states| states.contains(&selection.state))
    {
        return Err(
            "SQLite runtime selection differs from complete primary catalogue/context".into(),
        );
    }
    let execution = primary
        .report
        .executions
        .first()
        .ok_or("SQLite primary execution missing")?;
    let opened = Opened::open(Path::new(
        execution
            .artifact_path
            .as_deref()
            .ok_or("SQLite primary artifact missing")?,
    ))?;
    opened.authenticate_all()?;
    let inputs = cluster::inputs(&opened)?;
    cluster::bind(&opened, &inputs)?;
    if opened.manifest().record != execution.record
        || !matches!(
            inputs.configuration.workload,
            Some(Workload::SQLite { selection: None })
        )
    {
        return Err("SQLite primary artifact differs from its actual unselected Result".into());
    }
    let mut payloads = BTreeMap::from([
        ("result.json".into(), result::encode(&primary)?),
        ("receipt.json".into(), canonical::encode(&receipt)?),
        (
            "cluster-replay.json".into(),
            opened.payload(cluster::FILE, 16 << 20)?,
        ),
    ]);
    for (name, limit) in [("storage_state", 8 << 20), ("storage", 32 << 20)] {
        payloads.insert(
            name.into(),
            opened.payload(&format!("evidence/cluster/sqlite/2/{name}"), limit)?,
        );
    }
    Ok((primary, payloads))
}

pub(super) fn bind(
    prepared: &Prepared,
    bound: &mut Bound,
    path: &Path,
    selection: &Selection,
) -> Result<(), String> {
    let (primary, _) = source(path, selection)?;
    let execution = primary
        .report
        .executions
        .first()
        .ok_or("SQLite primary execution missing")?;
    if primary.manifest.module_sha256 != canonical::digest(&prepared.module)
        || primary.manifest.toolchain_sha256 != prepared.toolchain
        || primary.manifest.host_config_sha256
            != canonical::digest(&canonical::encode(&prepared.configuration)?)
        || execution.record.identity.configuration
            != canonical::digest(&canonical::encode(&bound.configuration)?)
    {
        return Err("SQLite primary belongs to another module, toolchain, candidate or base execution configuration".into());
    }
    boots::sqlite(bound, Some(selection))?;
    bound.configuration.workload = Some(Workload::SQLite {
        selection: Some(selection.clone()),
    });
    Ok(())
}

pub(super) fn payloads(
    path: &Path,
    selection: &Selection,
) -> Result<BTreeMap<String, Vec<u8>>, String> {
    Ok(source(path, selection)?
        .1
        .into_iter()
        .map(|(name, raw)| (format!("{CAPSULE}{name}"), raw))
        .collect())
}

pub(crate) fn authenticate(opened: &Opened, selection: &Selection) -> Result<(), String> {
    selection.validate()?;
    native_crash::selected_binding(opened, "sqlite", 2, &selection.state, &selection.context)?;
    let read = |name: &str, limit| opened.payload(&format!("{CAPSULE}{name}"), limit);
    let raw = read("result.json", 64 << 20)?;
    let primary: result::QualificationResult = crate::qualify::spec::strict(&raw)?;
    if result::encode(&primary)? != raw
        || canonical::digest(&raw) != selection.primary_result_sha256
        || primary.manifest.id != "sqlite-durable-recovery"
        || primary.manifest.cell.as_deref() != Some("success")
        || primary.native_crash_selection.is_some()
        || !primary.report.qualified
        || primary.manifest.module_sha256 != opened.manifest().record.identity.target
        || primary.manifest.toolchain_sha256 != opened.manifest().record.identity.toolchain
    {
        return Err("retained SQLite primary Result capsule identity changed".into());
    }
    result::bind_provenance(&primary.manifest, &primary.provenance)?;
    report::validate(&primary.report)?;
    let receipt: result::Receipt = crate::qualify::spec::strict(&read("receipt.json", 8 << 10)?)?;
    result::validate_receipt(&primary, &receipt, &selection.primary_result_sha256)?;
    let execution = primary
        .report
        .executions
        .first()
        .ok_or("retained SQLite primary execution missing")?;
    if execution.record.identity.implementation != opened.manifest().record.identity.implementation
        || execution.record.identity.platform != opened.manifest().record.identity.platform
    {
        return Err("retained SQLite primary belongs to another host or candidate".into());
    }
    let inputs_raw = read("cluster-replay.json", 16 << 20)?;
    let inputs: cluster::Inputs = crate::qualify::spec::strict(&inputs_raw)?;
    if canonical::encode(&inputs)? != inputs_raw
        || canonical::digest(&canonical::encode(&inputs.configuration)?)
            != execution.record.identity.configuration
        || execution
            .record
            .streams
            .get(cluster::FILE)
            .is_none_or(|stream| {
                stream.sha256 != canonical::digest(&inputs_raw)
                    || stream.bytes != inputs_raw.len() as u64
            })
        || !matches!(
            inputs.configuration.workload,
            Some(Workload::SQLite { selection: None })
        )
    {
        return Err("retained SQLite primary execution preimage changed".into());
    }
    let state = read("storage_state", 8 << 20)?;
    let log = read("storage", 32 << 20)?;
    for (name, raw) in [("storage_state", &state), ("storage", &log)] {
        if execution
            .record
            .streams
            .get(&format!("evidence/cluster/sqlite/2/{name}"))
            .is_none_or(|stream| {
                stream.sha256 != canonical::digest(raw) || stream.bytes != raw.len() as u64
            })
        {
            return Err("retained SQLite primary native payload binding changed".into());
        }
    }
    let catalogue = native_crash::capture(&state, &log)?;
    if primary.native_crash_catalogues != [catalogue.clone()]
        || catalogue.context != selection.context
        || !catalogue
            .enumeration
            .states
            .as_ref()
            .is_some_and(|states| states.contains(&selection.state))
    {
        return Err("retained SQLite primary catalogue or pending context changed".into());
    }
    Ok(())
}
