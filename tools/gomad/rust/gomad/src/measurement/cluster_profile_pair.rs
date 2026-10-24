use super::super::evidence::NativeRun;
use crate::evidence::canonical;
use serde_json::{Value, json};

pub(super) struct Capture<'a> {
    pub raw: &'a [u8],
    pub crashed: bool,
    pub reaped: bool,
}

fn number(value: &Value, name: &str) -> Result<u64, String> {
    value[name]
        .as_u64()
        .ok_or_else(|| format!("cluster profile unsigned counter absent: {name}"))
}

pub(super) fn owner_split(owners: &Value) -> Result<Value, String> {
    let mut work = 0_u64;
    let mut waiting = 0_u64;
    for (name, owner) in owners.as_object().ok_or("cluster owner inventory absent")? {
        if !gomad_wasmhost::profile_inventory::owner_names().contains(&name.as_str()) {
            return Err(format!("cluster unknown owner: {name}"));
        }
        let total = if matches!(name.as_str(), "handoff_wait" | "instrumentation") {
            &mut waiting
        } else {
            &mut work
        };
        *total = total
            .checked_add(number(owner, "execute_exclusive_wall_nanos")?)
            .ok_or("cluster owner total overflow")?;
    }
    Ok(json!({"host_work_nanos":work,"waiting_nanos":waiting}))
}

fn add(sum: &mut Value, next: &Value) -> Result<(), String> {
    for category in ["phases", "abi", "nested", "residual"] {
        merge(&mut sum[category], &next[category])?;
    }
    Ok(())
}
fn merge(sum: &mut Value, next: &Value) -> Result<(), String> {
    for (name, value) in next.as_object().ok_or("cluster aggregate bucket absent")? {
        if value.is_object() {
            if sum[name].is_null() {
                sum[name] = json!({});
            }
            merge(&mut sum[name], value)?;
        } else {
            let value = value
                .as_u64()
                .ok_or("cluster aggregate unsigned counter required")?;
            let previous = sum[name].as_u64().unwrap_or(0);
            sum[name] = json!(if name == "peak_rss_bytes_high_water" {
                previous.max(value)
            } else {
                previous
                    .checked_add(value)
                    .ok_or("cluster diagnostic sum overflow")?
            });
        }
    }
    Ok(())
}

#[derive(Debug)]
pub(super) struct Failure {
    pub summary: Value,
    error: String,
}
impl std::fmt::Display for Failure {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            formatter,
            "{}; partial evidence: {}",
            self.error, self.summary
        )
    }
}
fn summary(instances: &[Value], measured: usize, complete: bool, sum: &Value) -> Value {
    json!({"instances":instances,"measured_captures":measured,"complete":complete,"measured_sum":sum})
}

pub(super) fn decode(captures: &[Capture<'_>]) -> Result<Value, Failure> {
    if captures.is_empty() {
        return Err(Failure {
            summary: summary(&[], 0, false, &Value::Null),
            error: "cluster capture cardinality".into(),
        });
    }
    let mut instances = Vec::new();
    let mut measured = 0;
    let mut sum = Value::Null;
    let mut identity = None;
    let mut failures = Vec::new();
    for capture in captures {
        let decoded = crate::profile::decode(capture.raw);
        match decoded {
            Ok(profile) => {
                let host = (
                    profile["host_os"].clone(),
                    profile["host_arch"].clone(),
                    profile["backend"].clone(),
                );
                if identity.as_ref().is_some_and(|value| value != &host) {
                    return Err(Failure {
                        summary: summary(&instances, measured, false, &Value::Null),
                        error: "cluster diagnostic host mismatch".into(),
                    });
                }
                identity = Some(host);
                if sum.is_null() {
                    sum = json!({"phases":{},"abi":{},"nested":{},"residual":{}});
                }
                let split = owner_split(&profile["owners"]).map_err(|error| Failure {
                    summary: summary(&instances, measured, false, &Value::Null),
                    error,
                })?;
                measured += 1;
                instances.push(json!({"profile":profile,"owner_split":split}));
                if let Err(error) = add(&mut sum, &profile) {
                    return Err(Failure {
                        summary: summary(&instances, measured, false, &Value::Null),
                        error,
                    });
                }
            }
            Err(error) => {
                instances.push(json!({"profile":null,"unavailable":error}));
                if !capture.crashed || !capture.reaped || capture.raw.len() > 16 << 10 {
                    failures.push(error);
                }
            }
        }
    }
    if !failures.is_empty() {
        return Err(Failure {
            summary: summary(&instances, measured, false, &sum),
            error: format!(
                "cluster successful/unreaped profile unavailable: {}",
                failures.join("; ")
            ),
        });
    }
    Ok(summary(
        &instances,
        measured,
        measured == captures.len(),
        &sum,
    ))
}

pub(super) fn summarize(run: &NativeRun) -> Result<Value, String> {
    if run.profile.is_null() {
        return Ok(
            json!({"capture_scope":"ordinary process; diagnostic attribution unavailable","complete":false,"measured_captures":0,"measured_sum":null}),
        );
    }
    let raw = canonical::encode(&run.profile)?;
    let mut summary = decode(&[Capture {
        raw: &raw,
        crashed: false,
        reaped: true,
    }])
    .map_err(|failure| failure.to_string())?;
    let declarations = run.request["incarnations"]
        .as_array()
        .ok_or("cluster store declarations absent")?;
    let threads = run.profile["threads"]
        .as_array()
        .ok_or("cluster profile thread rows absent")?;
    if declarations.len() != run.incarnations.len() || threads.len() != declarations.len() {
        return Err("cluster measured store cardinality differs".into());
    }
    let mut stores = Vec::new();
    let nodes = run.request["nodes"]
        .as_array()
        .ok_or("cluster profile nodes absent")?;
    for declaration in declarations {
        let handle = &declaration["handle"];
        let node = nodes
            .iter()
            .position(|node| node == &handle["node"])
            .ok_or("cluster profile node absent")?;
        let generation = handle["incarnation"]
            .as_u64()
            .and_then(|value| value.checked_sub(1))
            .ok_or("cluster profile incarnation absent")?;
        let slot = generation
            .checked_mul(nodes.len() as u64)
            .and_then(|value| value.checked_add(node as u64 + 1))
            .ok_or("cluster profile slot overflow")?;
        let thread = threads
            .iter()
            .find(|thread| {
                thread["slot"].as_u64() == Some(slot)
                    && thread["incarnation"] == handle["incarnation"]
            })
            .ok_or("cluster measured store identity differs")?;
        if !run
            .incarnations
            .iter()
            .any(|store| store.handle.as_ref() == Some(handle))
        {
            return Err("cluster measured store has no joined result".into());
        }
        stores.push(json!({"handle":handle,"thread":thread}));
    }
    summary["stores"] = json!(stores);
    summary["capture_scope"] = json!(
        "one linked process capture; CPU, RSS, ABI and nested buckets remain process-global; thread-time is per store"
    );
    Ok(summary)
}

#[cfg(test)]
#[path = "cluster_profile_pair_tests.rs"]
mod tests;

pub(super) fn retain_failed_capture(
    root: &std::path::Path,
    name: &str,
    diagnostic: bool,
    execution_error: &str,
) -> Result<(), String> {
    use crate::evidence::store::open;
    let evidence = open::pin(&root.join("evidence"))?;
    let journal: super::super::build::CommandObservation = crate::qualify::spec::strict(
        &open::read(&evidence, &format!("{name}-driver.json"), 1 << 20, false)?,
    )?;
    if journal.schema != "gomad.native-process-observation/v1" || journal.pid.is_none() {
        return Err("cluster failed capture lacks actual process custody".into());
    }
    let mut summary = if diagnostic {
        let path = root.join("evidence").join(name).join("profile.json");
        let raw = match std::fs::symlink_metadata(&path) {
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => Vec::new(),
            Err(error) => return Err(error.to_string()),
            Ok(_) => open::read(
                &open::child(&evidence, name, true, false)?,
                "profile.json",
                64 << 10,
                false,
            )?,
        };
        let capture = Capture {
            raw: &raw,
            crashed: journal.signal.is_some(),
            reaped: journal.reaped_monotonic_nanos.is_some() && journal.cleanup_errors.is_empty(),
        };
        match decode(&[capture]) {
            Ok(summary) => summary,
            Err(failure) => {
                let mut summary = failure.summary;
                summary["capture_failure"] = json!(failure.error);
                summary
            }
        }
    } else {
        json!({"complete":false,"measured_captures":0,"measured_sum":null,"unavailable":"ordinary process has no diagnostic capture"})
    };
    summary["schema"] = json!("gomad.failed-cluster-profile-pair/v1");
    summary["capture_scope"] =
        json!("one actual native process capture; not per-store helper aliases");
    summary["execution_error"] = json!(execution_error);
    summary["process_id"] = json!(journal.pid);
    summary["passed"] = json!(false);
    super::write(
        &root.join(format!("{name}-profile-pair-failure.json")),
        &canonical::encode(&summary)?,
    )
}
