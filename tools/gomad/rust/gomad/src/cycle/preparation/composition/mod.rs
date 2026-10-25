#![cfg_attr(
    not(test),
    allow(dead_code, reason = "Cycle route integration belongs to B15")
)]
pub(crate) mod static_inputs;
use static_inputs::{StaticInputs, StaticPlan, static_plan};

use crate::cycle::{
    custody,
    model::{Adapter, Mode, Plan, PlanConfig, Stage},
    plan::{
        self,
        leaf::{Binding, LeafBindings, LeafPlan},
    },
};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, BTreeSet},
    path::{Path, PathBuf},
};

const SCHEMA: &str = "gomad.cycle-composition/v1";
const DRY_GROUPS: &[&str] = &[
    "final_profiles",
    "final_retained",
    "public_verbs",
    "private_observations",
    "live_corpus",
    "fn222",
    "terminal_controls_reference",
];
const LIVE_FAMILIES: &[&str] = &[
    "inspection",
    "observed",
    "backend-runtime",
    "choice",
    "minimize",
    "markers",
    "softassert",
    "oracles",
    "cluster",
    "runtime",
    "broker",
    "owners",
    "private35",
    "codec-oracles",
    "prepared-metadata",
];

#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct CompositionInputs {
    pub schema: String,
    pub leaves: LeafBindings,
    pub base: Value,
    pub runtime_build_receipt: PathBuf,
    pub sqlite_build_receipt: PathBuf,
    pub static_inputs: StaticInputs,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct FinalSection {
    pub required: Vec<String>,
    pub argv: Vec<String>,
}
#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct StageBindings {
    pub inputs: CompositionInputs,
    pub plan: Plan,
    pub stages: BTreeMap<String, LeafPlan>,
    pub final_sections: BTreeMap<String, FinalSection>,
    pub static_plan: StaticPlan,
    pub config: Value,
    pub files: BTreeMap<String, String>,
}
impl StageBindings {
    pub(crate) fn config(&self) -> &Value {
        &self.config
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        require(
            *self == compose(self.inputs.clone())?,
            "changed composed stage or immutable static plan",
        )
    }
    pub(crate) fn decode(bytes: &[u8]) -> Result<Self, String> {
        require(bytes.len() <= 8 << 20, "composition byte bound exceeded")?;
        let value = custody::strict_json(bytes)?;
        let result: Self = serde_json::from_value(value).map_err(|e| e.to_string())?;
        result.validate()?;
        Ok(result)
    }
}
fn require(ok: bool, error: &str) -> Result<(), String> {
    if ok { Ok(()) } else { Err(error.into()) }
}
fn path(b: &LeafBindings, key: Binding) -> Result<&str, String> {
    let text = b
        .paths
        .get(&key)
        .ok_or_else(|| format!("missing composition binding {key:?}"))?;
    custody::clean_absolute(Path::new(text))?;
    Ok(text)
}
fn endpoint(b: &LeafBindings, name: &str, tail: &[&str]) -> Result<Vec<String>, String> {
    require(
        plan::leaf::endpoint_catalog()
            .iter()
            .filter(|(_, endpoint)| *endpoint == name)
            .count()
            == 1,
        "missing/duplicate/unmapped orchestration endpoint",
    )?;
    let mut result = vec![
        path(b, Binding::Binary)?.into(),
        "qualification-cycle".into(),
        "internal".into(),
        name.into(),
    ];
    result.extend(tail.iter().map(|item| (*item).into()));
    Ok(result)
}
pub(crate) fn final_sections(b: &LeafBindings) -> Result<BTreeMap<String, FinalSection>, String> {
    let config = path(b, Binding::Config)?;
    let mut sections = BTreeMap::new();
    for (name, target, section, required) in [
        (
            "20-profiles",
            "final-profiles",
            None,
            vec![
                "terminal/current-host-admission-and-linked-attribution",
                "fn185-R4/runtime-cost-equivalent-release-behavior",
                "fn185-R4/qualified-workload-choice-wasi-network-stdout-stderr-preservation",
            ],
        ),
        (
            "40-retained",
            "final-retained",
            Some("retained"),
            vec!["retained/20-fresh-observed-replays"],
        ),
        (
            "50-operator",
            "final-retained",
            Some("operator"),
            vec!["operator/detached-installed-discover-inspect-replay"],
        ),
        (
            "60-inspection",
            "final-retained",
            Some("inspection"),
            vec!["inspection/rotated-three-node-ringpop-120s-fresh-exact-replay"],
        ),
        (
            "90-public-verbs",
            "final-public-verbs",
            None,
            vec!["public-verbs/fresh-differential-and-raw-golden-integrity"],
        ),
    ] {
        let mut tail = vec!["--config", config];
        if let Some(section) = section {
            tail.extend(["--section", section]);
            if section != "inspection" {
                tail.push("--read-only");
            }
        }
        sections.insert(
            name.into(),
            FinalSection {
                required: required.into_iter().map(str::to_owned).collect(),
                argv: endpoint(b, target, &tail)?,
            },
        );
    }
    let ids: Vec<_> = sections
        .values()
        .flat_map(|entry| &entry.required)
        .collect();
    require(
        ids.len() == 7 && ids.iter().collect::<BTreeSet<_>>().len() == ids.len(),
        "changed final endpoint partition",
    )?;
    Ok(sections)
}
fn seal(files: &mut BTreeMap<String, String>, p: &Path) -> Result<(), String> {
    let bytes = custody::read_file(p, 512 << 20)?;
    let name = p.to_str().ok_or("non UTF-8 composition file")?.to_owned();
    files.insert(name, custody::digest(&bytes));
    Ok(())
}
fn bind_stage(
    stage: &Stage,
    inputs: &CompositionInputs,
    files: &mut BTreeMap<String, String>,
) -> Result<(LeafPlan, Value), String> {
    let b = &inputs.leaves;
    let label = &stage.contract.label;
    let leaf = stage.leaf_commands(b)?;
    validate_endpoints(b, &leaf)?;
    let mut entry = json!({"adapter":stage.contract.adapter, "leaf":leaf});
    match stage.contract.adapter {
        Adapter::Runtime => {
            entry["harness"] = json!(path(b, Binding::RuntimeHarness)?);
            entry["harness_build_receipt"] = json!(inputs.runtime_build_receipt);
        }
        Adapter::Qualify => {
            let install = label
                .strip_prefix("workload-")
                .and_then(|v| {
                    v.strip_suffix("-success")
                        .or_else(|| v.strip_suffix("-canary"))
                })
                .ok_or("wrong cell label")?;
            let installation = b
                .installations
                .get(install)
                .ok_or("missing actual mandatory installation")?;
            let spec = b
                .specifications
                .get(label)
                .ok_or("missing mandatory scenario")?;
            seal(files, Path::new(installation))?;
            seal(files, Path::new(spec))?;
            entry["installation"] = json!(installation);
            entry["spec"] = json!(spec);
            entry["private_observations"] = json!(true);
            if install.starts_with("sqlite-") || install.starts_with("multi-instance-") {
                entry["checker"] = json!(endpoint(
                    b,
                    "cell-reader",
                    &["--config", path(b, Binding::Config)?, "--stage", label]
                )?);
            }
        }
        Adapter::CheckedCommand => {
            entry["timeout_s"] = json!(3600);
            entry["stop_at_deadline"] = json!(false);
        }
        Adapter::SqliteSelection => {
            entry["installation"] = json!(
                b.installations
                    .get("sqlite-cooperative")
                    .ok_or("missing actual SQLite installation")?
            );
        }
        _ => (),
    }
    Ok((leaf, entry))
}
fn configure_maps(config: &mut Value, inputs: &CompositionInputs) -> Result<(), String> {
    let b = &inputs.leaves;
    let installations = serde_json::to_value(&b.installations).map_err(|e| e.to_string())?;
    require(
        inputs
            .base
            .get("installations")
            .is_none_or(|v| v == &installations),
        "base/leaf installation bindings differ",
    )?;
    config["installations"] = installations;
    for (name, environment) in [
        ("environment", &b.environment),
        ("reference_environment", &b.reference_environment),
    ] {
        let environment = serde_json::to_value(environment).map_err(|e| e.to_string())?;
        require(
            inputs.base.get(name).is_none_or(|v| v == &environment),
            "base/leaf environment bindings differ",
        )?;
        config[name] = environment;
    }
    Ok(())
}
fn validate_base(
    inputs: &CompositionInputs,
    template: &PlanConfig,
    labels: &BTreeSet<&str>,
) -> Result<(), String> {
    let b = &inputs.leaves;
    let original = inputs.base["stages"]
        .as_object()
        .ok_or("missing original stage labels")?;
    require(
        original.keys().map(String::as_str).collect::<BTreeSet<_>>() == *labels,
        "missing/foreign original stage labels",
    )?;
    require(
        inputs.base["deadline_s"].as_u64() == Some(3600),
        "qualification deadline must be 3600 seconds",
    )?;
    require(
        inputs.base["sent_epoch"]
            .as_f64()
            .is_some_and(|n| n.is_finite() && n > 0.0),
        "send epoch must precede composition",
    )?;
    require(
        inputs.base.get("stress").is_none(),
        "stress matrix belongs to post-freeze window",
    )?;
    require(
        inputs.static_inputs.driver_config == template.config_path
            && inputs.static_inputs.repo == Path::new(path(b, Binding::Repo)?),
        "static/final driver config or checkout differs",
    )?;
    let identity = inputs.base["identity"]
        .as_object()
        .ok_or("unknown candidate host/helper identities")?;
    require(
        identity.keys().map(String::as_str).collect::<BTreeSet<_>>()
            == BTreeSet::from(["host", "helper"])
            && identity.values().all(|v| {
                v.as_str()
                    .is_some_and(|v| custody::valid_digest(v) && v != "0".repeat(64))
            }),
        "unknown candidate host/helper identities",
    )?;
    Ok(())
}
pub(crate) fn compose(inputs: CompositionInputs) -> Result<StageBindings, String> {
    require(
        serde_json::to_vec(&inputs)
            .map_err(|e| e.to_string())?
            .len()
            <= 8 << 20,
        "composition input byte bound exceeded",
    )?;
    require(inputs.schema == SCHEMA, "unknown composition schema")?;
    require(
        inputs.base["stages"]["sqlite-catalogue"]
            .get("oracle")
            .is_none(),
        "caller SQLite oracle argv forbidden",
    )?;
    let b = &inputs.leaves;
    let template = PlanConfig {
        binary: path(b, Binding::Binary)?.into(),
        config_path: path(b, Binding::Config)?.into(),
        config_sha256: "0".repeat(64),
    };
    let plan = plan::derive(&template, Mode::Parallel)?;
    let labels: BTreeSet<_> = plan
        .stages
        .iter()
        .map(|stage| stage.contract.label.as_str())
        .collect();
    validate_base(&inputs, &template, &labels)?;
    let static_plan = static_plan(&inputs.static_inputs, b)?;
    let mut files = static_plan.files.clone();
    for file in [&inputs.runtime_build_receipt, &inputs.sqlite_build_receipt] {
        seal(&mut files, file)?;
    }
    let mut stages = BTreeMap::new();
    let mut entries = serde_json::Map::new();
    for stage in &plan.stages {
        let label = &stage.contract.label;
        let (leaf, entry) = bind_stage(stage, &inputs, &mut files)?;
        stages.insert(label.clone(), leaf);
        entries.insert(label.clone(), entry);
    }
    seal(&mut files, Path::new(path(b, Binding::RuntimeHarness)?))?;
    seal(&mut files, Path::new(path(b, Binding::Binary)?))?;
    let final_sections = final_sections(b)?;
    let mut config = inputs.base.clone();
    config["repo"] = json!(path(b, Binding::Repo)?);
    config["root"] = json!(path(b, Binding::Root)?);
    config["binary"] = json!(path(b, Binding::Binary)?);
    configure_maps(&mut config, &inputs)?;
    config["stages"] = Value::Object(entries);
    config["final_sections"] = serde_json::to_value(&final_sections).map_err(|e| e.to_string())?;
    config["final_reader"] = json!(endpoint(
        b,
        "final",
        &["--config", path(b, Binding::Config)?]
    )?);
    config["sqlite_oracle"] = json!({"build_receipt":inputs.sqlite_build_receipt});
    configure_live(&mut config, &labels)?;
    configure_final(&mut config)?;
    config["inputs"] = json!(files.keys().collect::<Vec<_>>());
    Ok(StageBindings {
        inputs,
        plan,
        stages,
        final_sections,
        static_plan,
        config,
        files,
    })
}
pub(crate) fn compose_to(
    inputs: CompositionInputs,
    output: &Path,
) -> Result<StageBindings, String> {
    let bindings = compose(inputs)?;
    let bytes = crate::cycle::preparation::inputs::canonical(
        &serde_json::to_value(&bindings).map_err(|e| e.to_string())?,
    )?;
    require(bytes.len() <= 8 << 20, "composition byte bound exceeded")?;
    static_inputs::publish(output, &bytes)?;
    Ok(bindings)
}
fn validate_endpoints(b: &LeafBindings, leaf: &LeafPlan) -> Result<(), String> {
    for row in &leaf.commands {
        require(!row.argv.is_empty(), "empty leaf argv")?;
        if row.argv.get(1).map(String::as_str) == Some("qualification-cycle") {
            require(
                row.argv.first().map(String::as_str) == Some(path(b, Binding::Binary)?)
                    && row.argv.get(2).map(String::as_str) == Some("internal"),
                "foreign orchestration executable",
            )?;
            let name = row.argv.get(3).ok_or("missing orchestration endpoint")?;
            endpoint(b, name, &[])?;
        } else {
            require(
                !row.argv.get(1).is_some_and(|arg| {
                    arg.contains("/rust/driver/") || arg.contains("/rust/workload_tools/")
                }),
                "deleted or unmapped orchestration endpoint",
            )?;
        }
    }
    Ok(())
}
fn configure_live(config: &mut Value, labels: &BTreeSet<&str>) -> Result<(), String> {
    let dry = config["dry_run"] == true;
    let families = config["live_corpus"]["families"]
        .as_object()
        .cloned()
        .unwrap_or_default();
    let expected: BTreeSet<_> = LIVE_FAMILIES.iter().copied().collect();
    let actual: BTreeSet<_> = families.keys().map(String::as_str).collect();
    require(
        if dry {
            actual.is_subset(&expected)
        } else {
            actual == expected
        },
        "missing/foreign actual live receiver families",
    )?;
    for (name, entry) in families {
        let label = entry["stage"]
            .as_str()
            .ok_or("missing live receiver stage")?;
        require(labels.contains(label), "foreign live receiver stage")?;
        let target = &mut config["stages"][label]["live_families"];
        if target.is_null() {
            *target = json!([]);
        }
        target
            .as_array_mut()
            .ok_or("invalid live family list")?
            .push(json!(name));
    }
    Ok(())
}
fn configure_final(config: &mut Value) -> Result<(), String> {
    let dry = config["dry_run"] == true;
    let mut missing: Vec<String> = DRY_GROUPS
        .iter()
        .filter(|group| {
            config
                .get(**group)
                .is_none_or(|v| v.as_object().is_none_or(serde_json::Map::is_empty))
        })
        .map(|v| (*v).into())
        .collect();
    if config.get("final_profiles").is_some_and(|v| !v.is_null())
        && ["runtime_recipe", "qualified", "runtime_reference"]
            .iter()
            .any(|key| {
                config["final_profiles"]
                    .get(*key)
                    .is_none_or(Value::is_null)
            })
    {
        missing.push("final_profiles".into());
    }
    require(
        dry || [
            "final_profiles",
            "final_retained",
            "public_verbs",
            "private_observations",
        ]
        .iter()
        .all(|key| !missing.iter().any(|v| v == key)),
        "missing actual final input binding",
    )?;
    if dry {
        for name in LIVE_FAMILIES {
            if config["live_corpus"]["families"].get(*name).is_none() {
                missing.push(format!("live_corpus/{name}"));
            }
        }
        missing.sort();
        missing.dedup();
        config["dry_run_unexecuted"] = json!(missing);
    }
    Ok(())
}
#[cfg(test)]
pub(crate) mod tests;
