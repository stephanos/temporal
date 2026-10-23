use super::{
    AXES, ExploreOptions, Request, Spec, Supervision, canonical, cases, cell_status, directory,
    inputs, options, prepare, process, report, source, write,
};
use serde_json::{Value, json};
use std::{collections::BTreeMap, path::Path};

pub(super) struct Preparation {
    options: BTreeMap<String, ExploreOptions>,
    errors: BTreeMap<String, String>,
}

pub(super) struct Matrix {
    pub(super) cells: Vec<Value>,
    pub(super) references: Vec<Value>,
    pub(super) rejected: Vec<Value>,
}

pub(super) fn prepare(spec: &Spec) -> Result<Preparation, String> {
    let mut preparation_errors = BTreeMap::new();
    let mut prepared_options = BTreeMap::new();
    for workload in &spec.workloads {
        if !spec.only.is_empty() && spec.only != workload.name {
            continue;
        }
        let root = spec.root.join(format!("prepared-{}", workload.name));
        directory(&root)?;
        let preparation = (|| {
            let options = options(workload)?;
            let prepared = prepare::prepare(&options)?;
            inputs(&workload.name, &prepared)?;
            Ok::<_, String>((options, prepared))
        })();
        let (mut options, prepared) = match preparation {
            Ok(prepared) => prepared,
            Err(error) => {
                preparation_errors.insert(workload.name.clone(), error);
                continue;
            }
        };
        write(&root.join("module.wasm"), &prepared.module)?;
        write(&root.join("provenance.json"), &prepared.provenance)?;
        options.target_kind = "exec".into();
        options.source = root.join("module.wasm").to_string_lossy().into_owned();
        options.provenance = root.join("provenance.json").to_string_lossy().into_owned();
        prepared_options.insert(workload.name.clone(), options);
    }
    Ok(Preparation {
        options: prepared_options,
        errors: preparation_errors,
    })
}

fn perturbed(
    binary: &Path,
    spec: &Spec,
    (name, seed): (&str, u64),
    axis: &str,
    options: &ExploreOptions,
    reference: &Value,
    rejected: &mut Vec<Value>,
) -> Result<Value, String> {
    let fanout = if axis == "concurrent" {
        spec.parallel
    } else {
        1
    };
    let requested = spec.repetitions * fanout;
    let mut runs = Vec::new();
    let mut comparisons = Vec::new();
    let mut errors = Vec::new();
    if axis != "handoff" || name == "two-cluster" {
        let load = process::Load::new(if axis == "cpu" { spec.cpu_workers } else { 0 });
        for round in 0..spec.repetitions {
            let requests = (0..fanout)
                .map(|member| {
                    let root = spec
                        .root
                        .join(format!("{name}-{seed}-{axis}-{round}-{member}"));
                    directory(&root)?;
                    Ok(Request {
                        requested_root: spec.requested_root.clone(),
                        workload: name.into(),
                        seed,
                        axis: axis.into(),
                        options: options.clone(),
                        root,
                        source_sha256: spec.source_sha256.clone(),
                        target_pool: spec.root.join(".targets"),
                        supervision: Supervision {
                            cpu_workers: if axis == "cpu" { spec.cpu_workers } else { 0 },
                            cohort_size: fanout,
                            repetition: round,
                            member,
                        },
                    })
                })
                .collect::<Result<Vec<_>, String>>()?;
            for result in process::cohort(binary, requests, round) {
                match result {
                    Ok(run) => {
                        match report::compare(reference, &run) {
                            Ok(comparison) => {
                                if run["oracle_error"].is_null()
                                    && run["perturbation_error"].is_null()
                                {
                                    comparisons.push(comparison);
                                } else {
                                    comparisons.push(json!({"status":"oracle-failure","reason":run["oracle_error"],"perturbation_error":run["perturbation_error"],"comparison":comparison}));
                                }
                            }
                            Err(error) => {
                                rejected.push(
                                    json!({"workload":name,"seed":seed,"axis":axis,"reason":error}),
                                );
                                errors.push(error);
                            }
                        }
                        runs.push(run);
                    }
                    Err(error) => errors.push(error),
                }
            }
        }
        drop(load);
    } else {
        errors.push(
            "inapplicable: no cluster baton or remote peer replies in a standalone guest".into(),
        );
    }
    let status = if !errors.is_empty()
        && comparisons
            .iter()
            .all(|report| report["status"] == "identical")
    {
        "skipped"
    } else {
        cell_status(requested, runs.len(), &comparisons)
    };
    Ok(
        json!({"workload":name,"seed":seed,"axis":axis,"rounds":spec.repetitions,"fanout":fanout,"requested":requested,"completed":runs.len(),"status":status,"runs":runs,"comparisons":comparisons,"reasons":errors}),
    )
}

pub(super) fn run(binary: &Path, spec: &Spec, prepared: &Preparation) -> Result<Matrix, String> {
    let mut cells = Vec::new();
    let mut references = Vec::new();
    let mut rejected = Vec::new();
    let mut invalidated = None;
    for (name, seed) in cases() {
        if (!spec.only.is_empty() && spec.only != name)
            || (!spec.seed.is_empty() && spec.seed != seed.to_string())
        {
            for axis in AXES {
                cells.push(json!({"workload":name,"seed":seed,"axis":axis,"requested":spec.repetitions * if axis=="concurrent"{spec.parallel}else{1},"completed":0,"status":"skipped","reason":"excluded by --stress-only","runs":[],"comparisons":[]}));
            }
            continue;
        }
        if let Err(error) = source(spec) {
            invalidated = Some(error);
        }
        if let Some(error) = invalidated.as_ref().or_else(|| prepared.errors.get(name)) {
            for axis in AXES {
                cells.push(json!({"workload":name,"seed":seed,"axis":axis,"requested":spec.repetitions * if axis == "concurrent" { spec.parallel } else { 1 },"completed":0,"status":"skipped","reason":error,"runs":[],"comparisons":[]}));
            }
            rejected.push(json!({"workload":name,"seed":seed,"reason":error}));
            continue;
        }
        let options = &prepared.options[name];
        let reference_root = spec.root.join(format!("{name}-{seed}-reference"));
        directory(&reference_root)?;
        let reference = match process::execute(
            binary,
            &Request {
                requested_root: spec.requested_root.clone(),
                workload: name.into(),
                seed,
                axis: "reference".into(),
                options: options.clone(),
                root: reference_root,
                source_sha256: spec.source_sha256.clone(),
                target_pool: spec.root.join(".targets"),
                supervision: Supervision {
                    cpu_workers: 0,
                    cohort_size: 1,
                    repetition: 0,
                    member: 0,
                },
            },
            0,
        ) {
            Ok(reference) => reference,
            Err(error) => {
                for axis in AXES {
                    cells.push(json!({"workload":name,"seed":seed,"axis":axis,"requested":spec.repetitions * if axis == "concurrent" { spec.parallel } else { 1 },"completed":0,"status":"skipped","reason":format!("reference unavailable: {error}"),"runs":[],"comparisons":[]}));
                }
                continue;
            }
        };
        references.push(reference.clone());
        for axis in AXES {
            if !spec.axis.is_empty() && spec.axis != axis {
                cells.push(json!({"workload":name,"seed":seed,"axis":axis,"requested":spec.repetitions * if axis=="concurrent"{spec.parallel}else{1},"completed":0,"status":"skipped","reason":"excluded by --stress-axis","runs":[],"comparisons":[]}));
                continue;
            }
            if invalidated.is_some() {
                cells.push(json!({"workload":name,"seed":seed,"axis":axis,"requested":spec.repetitions * if axis == "concurrent" { spec.parallel } else { 1 },"completed":0,"status":"skipped","reason":invalidated,"runs":[],"comparisons":[]}));
                continue;
            }
            cells.push(perturbed(
                binary,
                spec,
                (name, seed),
                axis,
                options,
                &reference,
                &mut rejected,
            )?);
            if let Err(error) = source(spec) {
                rejected.push(json!({"workload":name,"seed":seed,"axis":axis,"reason":error}));
                invalidated = Some(error);
            }
            write(
                &spec.root.join(format!("checkpoint-{}.json", cells.len())),
                &canonical::encode(
                    &json!({"cells":cells,"references":references,"rejected":rejected}),
                )?,
            )?;
        }
    }
    Ok(Matrix {
        cells,
        references,
        rejected,
    })
}
