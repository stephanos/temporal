//! Current retained storage receiving. Wire admission belongs to wasmhost; this
//! module independently evaluates admitted persistence effects and run provenance.
#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle reader routing belongs to B15")
)]
use gomad_wasmhost::{linked, linked_storage_inspection as codec};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::{
    collections::{BTreeMap, BTreeSet},
    time::Duration,
};

mod effects;
#[cfg(test)]
mod testdata;
#[cfg(test)]
mod tests;

pub(crate) const STORAGE_LIMIT: usize = 64 << 20;

pub(crate) fn bounded(bytes: &[u8]) -> Result<(), String> {
    if bytes.len() > STORAGE_LIMIT {
        Err("storage receiving byte bound".into())
    } else {
        Ok(())
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum Case {
    Baseline,
    Selected(usize),
    Divergence(u8),
}

#[derive(Clone)]
pub(crate) struct EngineCase {
    pub(crate) fresh: u8,
    pub(crate) case: Case,
    pub(crate) input: Value,
    pub(crate) termination: String,
    pub(crate) message: String,
    pub(crate) exit_code: Option<u32>,
    pub(crate) peak_memory: u64,
    pub(crate) state: Vec<u8>,
    pub(crate) log: Vec<u8>,
    pub(crate) image: Option<Vec<u8>>,
}

pub(crate) struct Admitted<'a> {
    pub(crate) state: codec::StateObservation<'a>,
    pub(crate) image: Option<codec::RecoveryObservation>,
}

/// Accepts already independently reopened buffers; never reopens producer paths.
pub(crate) fn admit(row: &EngineCase) -> Result<Admitted<'_>, String> {
    bounded(&row.state)?;
    bounded(&row.log)?;
    let storage: linked::StorageInput =
        serde_json::from_value(row.input["storage"].clone()).map_err(|e| e.to_string())?;
    let state = codec::decode_state(&storage, &row.state, &row.log, STORAGE_LIMIT as u64)?;
    let image = row
        .image
        .as_ref()
        .map(|raw| {
            bounded(raw)?;
            let digest = format!("{:x}", Sha256::digest(raw));
            if state.image_digest() != Some(digest.as_str()) {
                return Err("storage state/image binding".into());
            }
            recovery(&storage, raw, &digest)
        })
        .transpose()?;
    if row.image.is_none() != state.failed_stage().is_some() {
        return Err("storage image presence".into());
    }
    Ok(Admitted { state, image })
}

pub(crate) fn recovery(
    input: &linked::StorageInput,
    raw: &[u8],
    expected_sha256: &str,
) -> Result<codec::RecoveryObservation, String> {
    bounded(raw)?;
    codec::recovery_image(input, raw, expected_sha256, STORAGE_LIMIT as u64)
}

pub(crate) fn validate_input(actual: &Value, expected: &Value) -> Result<(), String> {
    if actual == expected {
        Ok(())
    } else {
        Err("engine input changed".into())
    }
}

/// Validates a complete ordered same-candidate matrix against frozen input bytes.
pub(crate) fn validate_matrix(expected: &Value, rows: &[EngineCase]) -> Result<(), String> {
    validate_input(expected, &engine_input())?;
    let mut indexed = BTreeMap::new();
    for row in rows {
        if row.fresh > 1 || row.peak_memory == 0 || row.peak_memory > 65536 {
            return Err("engine fresh execution/memory".into());
        }
        if indexed.insert((row.fresh, row.case), row).is_some() {
            return Err("duplicate engine observation".into());
        }
    }
    let mut expected_keys = BTreeSet::new();
    let mut states_count = None;
    for fresh in 0..2 {
        let baseline = indexed
            .get(&(fresh, Case::Baseline))
            .ok_or("missing engine baseline")?;
        validate_input(&baseline.input, expected)?;
        success(baseline)?;
        let admitted = admit(baseline)?;
        let image = admitted.image.as_ref().ok_or("baseline image absent")?;
        let states = effects::baseline(image, &admitted.state)?;
        if states_count.is_some_and(|count| count != states.len()) {
            return Err("fresh enumeration count differs".into());
        }
        states_count = Some(states.len());
        expected_keys.insert((fresh, Case::Baseline));
        for (index, state) in states.iter().enumerate() {
            let key = (fresh, Case::Selected(index));
            expected_keys.insert(key);
            let selected = indexed.get(&key).ok_or("missing selected engine state")?;
            let mut input = expected.clone();
            configure_selection(
                &mut input,
                admitted.state.contexts().ok_or("baseline context absent")?,
                &state["SelectedOperations"],
            );
            validate_input(&selected.input, &input)?;
            success(selected)?;
            let selected_admitted = admit(selected)?;
            same_prior(
                baseline,
                &admitted.state,
                selected,
                &selected_admitted.state,
                true,
            )?;
            effects::selected(
                image,
                selected_admitted
                    .image
                    .as_ref()
                    .ok_or("selected image absent")?,
                state,
            )?;
        }
        validate_divergences(
            expected,
            fresh,
            baseline,
            &admitted.state,
            &states,
            &indexed,
            &mut expected_keys,
        )?;
    }
    if indexed.keys().copied().collect::<BTreeSet<_>>() != expected_keys {
        return Err("unexpected engine state index".into());
    }
    for case in
        std::iter::once(Case::Baseline).chain((0..states_count.unwrap_or(0)).map(Case::Selected))
    {
        let left = indexed[&(0, case)];
        let right = indexed[&(1, case)];
        if left.state != right.state || left.log != right.log || left.image != right.image {
            return Err("same-candidate storage evidence differs".into());
        }
    }
    Ok(())
}

fn validate_divergences(
    expected: &Value,
    fresh: u8,
    baseline: &EngineCase,
    state: &codec::StateObservation<'_>,
    states: &[Value],
    indexed: &BTreeMap<(u8, Case), &EngineCase>,
    expected_keys: &mut BTreeSet<(u8, Case)>,
) -> Result<(), String> {
    for variant in 0..3 {
        let key = (fresh, Case::Divergence(variant));
        expected_keys.insert(key);
        let divergence = indexed.get(&key).ok_or("missing engine divergence")?;
        let mut context = state.contexts().ok_or("baseline context absent")?.clone();
        match variant {
            0 => context["data"]["PendingSHA256"] = json!(format!("sha256:{}", "0".repeat(64))),
            1 => context["data"]["Operations"][0]["Kind"] = json!("forged"),
            _ => context["data"]["Operations"] = json!([]),
        }
        let mut input = expected.clone();
        let empty_selection = json!([]);
        configure_selection(
            &mut input,
            &context,
            if variant == 0 {
                &states.last().ok_or("empty enumeration")?["SelectedOperations"]
            } else {
                &empty_selection
            },
        );
        validate_input(&divergence.input, &input)?;
        if divergence.termination != "invalid"
            || divergence.exit_code.is_some()
            || divergence.image.is_some()
            || divergence.message != "storage crash context diverged from prior observation"
        {
            return Err("engine divergence outcome".into());
        }
        let diverged = admit(divergence)?;
        if diverged.state.failed_stage() != Some(2) {
            return Err("context divergence failed-stage identity".into());
        }
        same_prior(baseline, state, divergence, &diverged.state, false)?;
    }
    Ok(())
}

fn success(row: &EngineCase) -> Result<(), String> {
    if row.termination != "exit" || row.exit_code != Some(0) || !row.message.is_empty() {
        Err("engine storage execution failed".into())
    } else {
        Ok(())
    }
}

fn same_prior(
    left: &EngineCase,
    a: &codec::StateObservation<'_>,
    right: &EngineCase,
    b: &codec::StateObservation<'_>,
    context: bool,
) -> Result<(), String> {
    if left.log != right.log
        || [0, 1, 2]
            .iter()
            .any(|index| a.sections()[*index] != b.sections()[*index])
        || (context && a.contexts() != b.contexts())
    {
        return Err("selected engine prior observation changed".into());
    }
    Ok(())
}

pub(crate) fn configure_selection(input: &mut Value, context: &Value, selection: &Value) {
    let raw = serde_json::to_vec(context).expect("JSON value serializes");
    input["storage"]["descriptor"]["crash_selection"] = json!({"data":selection});
    input["storage"]["descriptor"]["crash_context"] = json!({"bytes":raw.len(),"chunks":raw.len().div_ceil(65536),"sha256":format!("{:x}",Sha256::digest(&raw))});
    input["storage"]["context"] = json!(raw);
}

const MODULE: &str = r#"(module
    (import "wasi_snapshot_preview1" "path_create_directory" (func $mkdir (param i32 i32 i32) (result i32)))
    (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
    (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
    (memory (export "memory") 1)
    (data (i32.const 0) "db")
    (func (export "_start")
        i32.const 512 i32.const 16 call $config drop
        i32.const 4 i32.const 0 i32.const 2 call $mkdir if unreachable end
        i32.const 995 i32.const 1 i32.store8
        i32.const 992 i32.const 16 call $finish drop))"#;

/// Frozen current mkdir workload and resource contract, independent of reported outputs.
pub(crate) fn engine_input() -> Value {
    let model_bytes = br#"{"volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1048576}],"limits":{"PendingOperations":1000,"Transitions":10000}}"#;
    let model: Value = serde_json::from_slice(model_bytes).unwrap();
    json!({"module":MODULE.as_bytes(),"module_sha256":format!("{:x}",Sha256::digest(MODULE.as_bytes())),
        "limits":{"memory_bytes":65536,"fuel":1_000_000,"output_bytes":1024,"evidence_bytes":8<<20,"wall_timeout_millis":5000},
        "configuration":{"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/workspace")),
            "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
            "seed":7,"mode":1,"limit":1_048_576,"diagnostics":false,"epoch_nanos":946_684_800_000_000_000_u64,
            "read_step_nanos":0,"now":1,"limits":{"calls":1_048_576,"transcript_bytes":1_048_576,"pending_events":1024},
            "digest_only":false,"split":false,"inspection":null,"replay":null},
        "arguments":[],"environment":["PWD=/workspace"],"replay_trace":null,
        "namespace":{"entries":[{"path":b"/workspace","directory":true,"readonly":false,"data":[]}],
            "writable_directories":[b"/workspace"],"preopens":[b"/",b"/workspace"],"working_directory":b"/workspace",
            "stdin":[],"epoch_nanos":946_684_800_000_000_000_u64,"limits":{"files":100,"descriptors":16,"filesystem_bytes":1<<20,"output_bytes":1024}},
        "storage":{"descriptor":{"model":model,"model_identity":format!("sha256:{:x}",Sha256::digest(model_bytes)),
            "return_image":true,"crash_limits":{"States":16,"Operations":8,"Depth":8,"Bytes":1<<20,"Work":256}},"context":null,"image":null}})
}

/// Executes exactly the accepted input object; callers retain this object with the output.
pub(crate) fn request(input: &Value) -> Result<linked::Request, String> {
    let decode = |key: &str| {
        input
            .get(key)
            .cloned()
            .ok_or_else(|| format!("missing engine {key}"))
    };
    let module: Vec<u8> = serde_json::from_value(decode("module")?).map_err(|e| e.to_string())?;
    let number = |key: &str| {
        input["limits"][key]
            .as_u64()
            .ok_or_else(|| format!("invalid engine limit {key}"))
    };
    let module_sha256 = input["module_sha256"]
        .as_str()
        .ok_or("missing module identity")?
        .to_owned();
    Ok(linked::Request {
        module,
        module_sha256,
        limits: linked::Limits {
            memory_bytes: number("memory_bytes")?,
            fuel: number("fuel")?,
            output_bytes: number("output_bytes")?,
            evidence_bytes: number("evidence_bytes")?,
            wall_timeout: Duration::from_millis(number("wall_timeout_millis")?),
        },
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(decode("configuration")?)
                .map_err(|e| e.to_string())?,
            arguments: serde_json::from_value(decode("arguments")?).map_err(|e| e.to_string())?,
            environment: serde_json::from_value(decode("environment")?)
                .map_err(|e| e.to_string())?,
            replay_trace: serde_json::from_value(decode("replay_trace")?)
                .map_err(|e| e.to_string())?,
        }),
        namespace: Some(serde_json::from_value(decode("namespace")?).map_err(|e| e.to_string())?),
        storage: Some(serde_json::from_value(decode("storage")?).map_err(|e| e.to_string())?),
    })
}

pub(crate) fn validate_recovered_state(
    state: &codec::StateObservation<'_>,
    image: &codec::RecoveryObservation,
) -> Result<(), String> {
    effects::recovered(image, state)
}
