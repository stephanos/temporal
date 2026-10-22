use crate::evidence::canonical;
use gomad_wasmhost::{linked, linked_inspection};
use serde::Deserialize;
use serde_json::{Value, json};
use std::{
    path::{Path, PathBuf},
    time::Duration,
};

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    module_path: PathBuf,
    module_sha256: String,
    choice_identity: linked::ChoiceIdentity,
    memory_bytes: u64,
    fuel: u64,
    wall_millis: u64,
    output_bytes: u64,
    evidence_bytes: u64,
    arguments: Vec<String>,
    environment: Vec<String>,
    model: linked::ModelConfig,
    namespace: linked::NamespaceImage,
    replay_trace_path: Option<PathBuf>,
    prefix: Option<linked::PrefixControl>,
    inspection: Option<Inspection>,
    compare_diagnostic_path: Option<PathBuf>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Inspection {
    metadata_path: PathBuf,
    decision: u64,
    dumps: Vec<String>,
    probes: Vec<String>,
}

fn read(path: &Path) -> Result<Vec<u8>, String> {
    std::fs::read(path).map_err(|error| format!("{}: {error}", path.display()))
}

fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    std::fs::write(path, bytes).map_err(|error| format!("{}: {error}", path.display()))
}

fn diagnostic_difference(expected: &[u8], actual: &[u8]) -> Result<Value, String> {
    if expected.len() != actual.len() || !expected.len().is_multiple_of(96) {
        return Err("diagnostic record count differs".into());
    }
    let fields = [
        (0, "ordinal"),
        (8, "allocations"),
        (16, "allocated_bytes"),
        (24, "gc_cycles"),
        (40, "runq_draws"),
        (48, "scheduler_draws"),
        (56, "select_draws"),
        (64, "runtime_rand_draws"),
        (72, "runtime_cheaprand_draws"),
        (80, "timer_draws"),
        (88, "clock_tick_draws"),
    ];
    for (ordinal, (left, right)) in expected
        .as_chunks::<96>()
        .0
        .iter()
        .zip(actual.as_chunks::<96>().0)
        .enumerate()
    {
        let differences: Vec<_> = fields
            .iter()
            .filter_map(|(offset, name)| {
                (left[*offset..*offset + 8] != right[*offset..*offset + 8]).then_some(*name)
            })
            .collect();
        if !differences.is_empty() {
            return Ok(json!({"ordinal":ordinal,"fields":differences}));
        }
    }
    Ok(Value::Null)
}

#[expect(
    clippy::too_many_lines,
    reason = "Fixture keeps a raw linked request and its evidence export together"
)]
pub(crate) fn run(input: &Path, root: &Path) -> Result<(), String> {
    if cfg!(debug_assertions) {
        return Err("live runtime roots require a release fixture build".into());
    }
    let mut input: Input =
        serde_json::from_slice(&read(input)?).map_err(|error| error.to_string())?;
    let module = read(&input.module_path)?;
    let replay_trace = input.replay_trace_path.as_deref().map(read).transpose()?;
    if let Some(trace) = &replay_trace {
        let mut plan = linked::project_choices(trace, &input.choice_identity)?;
        if let Some(prefix) = &input.prefix {
            plan = plan.prefix(prefix)?;
        }
        input.model.replay = Some(serde_json::from_value(json!({
            "schema":"gomad.replay-trace/v1", "execution_identity":input.choice_identity,
            "trace_sha256":canonical::digest(trace), "tape_sha256":canonical::digest(&plan.bytes),
            "decisions":plan.decisions.len(), "trace_bytes":trace.len(),
            "trace_frames":trace.len().div_ceil(65536),
            "prefix_control":input.prefix.as_ref().map(|prefix| json!({
                "cutoff":prefix.cutoff,
                "override":prefix.r#override.as_ref().map(|rank| json!({
                    "ordinal":rank.ordinal,"rank":rank.rank,"reconstruct":rank.reconstruct,
                })),
            })),
        })).map_err(|error| error.to_string())?);
    }
    let retained = input
        .inspection
        .as_ref()
        .map(|control| {
            linked_inspection::Evidence::decode(&read(&control.metadata_path)?)
                .map_err(|error| error.to_string())
        })
        .transpose()?;
    if let (Some(control), Some(evidence)) = (&input.inspection, &retained) {
        input.model.inspection = Some(
            linked_inspection::resolve(
                evidence,
                linked_inspection::Handle {
                    node: String::new(),
                    incarnation: 0,
                },
                &linked_inspection::Stop {
                    kind: "choice".into(),
                    value: control.decision,
                },
                &control.dumps,
                &control.probes,
                "",
            )
            .map_err(|error| error.to_string())?,
        );
    }
    let result = crate::execute::execute(linked::Request {
        module,
        module_sha256: input.module_sha256,
        limits: linked::Limits {
            memory_bytes: input.memory_bytes,
            fuel: input.fuel,
            output_bytes: input.output_bytes,
            evidence_bytes: input.evidence_bytes,
            wall_timeout: Duration::from_millis(input.wall_millis),
        },
        model: Some(linked::ModelInput {
            configuration: input.model,
            arguments: input.arguments,
            environment: input.environment,
            replay_trace,
        }),
        namespace: Some(input.namespace),
        storage: None,
    })
    .map_err(|outcome| outcome.message)?;
    std::fs::create_dir(root).map_err(|error| error.to_string())?;
    std::fs::create_dir(root.join("streams")).map_err(|error| error.to_string())?;
    for (name, bytes) in &result.evidence {
        write(&root.join("streams").join(name), bytes)?;
    }
    let mut report = json!({"execution_returned":true,"outcome":result.outcome});
    if result.outcome.termination == "exit" {
        report["choices"] = serde_json::to_value(linked::inspect_choices(
            result.evidence.get("choice").map_or(&[], Vec::as_slice),
            &input.choice_identity,
        )?)
        .map_err(|error| error.to_string())?;
    }
    if let Some(raw) = result.evidence.get("inspection") {
        let evidence =
            linked_inspection::Evidence::decode(raw).map_err(|error| error.to_string())?;
        report["inspection"] =
            serde_json::to_value(&evidence).map_err(|error| error.to_string())?;
        report["inspection"]["boundaries"] =
            serde_json::to_value(&evidence.boundaries).map_err(|error| error.to_string())?;
        if let (Some(control), Some(retained)) = (&input.inspection, &retained) {
            let expected = retained
                .boundaries
                .iter()
                .find(|boundary| boundary.decision == control.decision)
                .ok_or("missing inspection boundary")?;
            if result.outcome.termination == "inspected_prefix" {
                linked_inspection::verify_prefix(retained, raw, expected)
                    .map_err(|error| error.to_string())?;
                report["prefix_verified"] = true.into();
            }
            if let Some(debug) = result.evidence.get("debug") {
                let output = linked_inspection::debug_output(
                    debug,
                    &expected.handle,
                    &retained.incarnations[0].registry,
                    65536,
                )
                .map_err(|error| error.to_string())?;
                report["debug"] = output.output;
                if !output.heap.is_empty() {
                    write(&root.join("heap.pprof"), &output.heap)?;
                }
            }
        }
    }
    if let Some(path) = input.compare_diagnostic_path {
        report["diagnostic_difference"] = diagnostic_difference(
            &read(&path)?,
            result.evidence.get("diagnostic").map_or(&[], Vec::as_slice),
        )?;
    }
    write(&root.join("result.json"), &canonical::encode(&report)?)?;
    println!("{}", root.join("result.json").display());
    Ok(())
}
