mod declarations;

use super::model::{Isolation, Mode, Plan, PlanConfig, Semantic, Stage, StageContract};
use std::collections::BTreeSet;
use std::path::{Component, Path};

use declarations::{contracts, semantics};

fn validate_mapping(stages: &[StageContract], semantic: &[Semantic]) -> Result<(), String> {
    let labels: BTreeSet<_> = semantic.iter().map(|row| &row.label).collect();
    if semantic.len() != 96 || labels.len() != 96 {
        return Err("missing/duplicate semantic stage".into());
    }
    let dag: BTreeSet<_> = stages.iter().map(|row| &row.label).collect();
    if labels != dag || dag.len() != stages.len() {
        return Err("semantic stage/DAG mismatch".into());
    }
    Ok(())
}

fn absolute(path: &Path) -> Result<String, String> {
    if !path.is_absolute()
        || path
            .components()
            .any(|part| matches!(part, Component::ParentDir | Component::CurDir))
    {
        return Err("plan binding must be an absolute path without traversal".into());
    }
    path.to_str()
        .filter(|value| !value.contains('\0'))
        .map(str::to_owned)
        .ok_or_else(|| "plan binding must be a UTF-8 path without NUL".into())
}

pub(crate) fn derive(config: &PlanConfig, mode: Mode) -> Result<Plan, String> {
    let binary = absolute(&config.binary)?;
    let path = absolute(&config.config_path)?;
    if config.config_sha256.len() != 64
        || !config
            .config_sha256
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
    {
        return Err("plan requires the original configuration SHA256".into());
    }
    let mut stages = contracts();
    let semantic = semantics();
    validate_mapping(&stages, &semantic)?;
    let mut previous = None;
    let mut plan = Vec::with_capacity(stages.len());
    for contract in &mut stages {
        let row = semantic
            .iter()
            .find(|row| row.label == contract.label)
            .ok_or("semantic stage/DAG mismatch")?;
        contract
            .required_subgates
            .clone_from(&row.required_subgates);
        contract.repetitions = row.repetitions;
        let mut dependencies = contract.dependencies.clone();
        if mode == Mode::Serial {
            if let Some(label) = previous
                && !dependencies.contains(&label)
            {
                dependencies.push(label);
            }
            previous = Some(contract.label.clone());
        }
        let argv = vec![
            binary.clone(),
            "qualification-cycle".into(),
            "internal".into(),
            "driver".into(),
            "stage".into(),
            "--config".into(),
            path.clone(),
            "--config-sha256".into(),
            config.config_sha256.clone(),
            contract.label.clone(),
        ];
        plan.push(Stage {
            dependencies,
            isolation_class: if mode == Mode::Serial {
                Isolation::Solo
            } else {
                contract.isolation_class
            },
            argv,
            contract: contract.clone(),
        });
    }
    Ok(Plan { mode, stages: plan })
}

pub(crate) fn validate(config: &PlanConfig, plan: &Plan) -> Result<(), String> {
    if *plan != derive(config, plan.mode)? {
        return Err("changed stage contract or undeclared orchestration substitution".into());
    }
    Ok(())
}

pub(crate) fn decode(config: &PlanConfig, bytes: &[u8]) -> Result<Plan, String> {
    if bytes.len() > 1 << 20 {
        return Err("cycle plan exceeds byte bound".into());
    }
    let plan = crate::qualify::spec::strict(bytes)?;
    validate(config, &plan)?;
    Ok(plan)
}

#[cfg(test)]
mod tests;
