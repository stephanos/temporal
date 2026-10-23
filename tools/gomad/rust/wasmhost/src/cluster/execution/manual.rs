use super::transport::{Broken, SharedInput, SharedOutput, TaggedInput, TaggedOutput, lock};
use super::{
    ClusterExecute, ClusterResult, Coordinator, InstanceConfig, InstanceSummary, Report,
    STORE_THREAD_STACK, read_instance_tape, refuse_instance, run_instance, validate_model,
};
use crate::cluster::network::{Endpoint, Failure};
use crate::engine::State;
use crate::protocol::{self, Outcome};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::Deserialize;
use serde_json::{Value, json};
use std::collections::BTreeMap;
use std::collections::BTreeSet;
use std::panic::{AssertUnwindSafe, catch_unwind};
use std::sync::Arc;
use std::thread::JoinHandle;
use std::time::Instant;
use wasmtime::{Error, InstancePre, Result};

fn include_module_metrics(cluster: &mut crate::metrics::Metrics, module: &crate::metrics::Metrics) {
    cluster.cache_hit &= module.cache_hit;
    cluster.compile_nanos += module.compile_nanos;
    cluster.cache_load_nanos += module.cache_load_nanos;
    cluster.cache_write_nanos += module.cache_write_nanos;
}

#[cfg(test)]
mod tests {
    #[test]
    fn every_distinct_module_must_hit_cache_for_a_warm_cluster() {
        let mut cluster = crate::metrics::Metrics {
            cache_hit: true,
            cache_load_nanos: 11,
            ..Default::default()
        };
        super::include_module_metrics(
            &mut cluster,
            &crate::metrics::Metrics {
                compile_nanos: 17,
                cache_write_nanos: 23,
                ..Default::default()
            },
        );
        super::include_module_metrics(
            &mut cluster,
            &crate::metrics::Metrics {
                cache_hit: true,
                cache_load_nanos: 29,
                ..Default::default()
            },
        );
        assert!(!cluster.cache_hit);
        assert_eq!(cluster.compile_nanos, 17);
        assert_eq!(cluster.cache_load_nanos, 40);
        assert_eq!(cluster.cache_write_nanos, 23);
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Boundary {
    node: String,
    incarnation: u64,
    marker_base64: String,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CrashGroup {
    nodes: Vec<String>,
    marker_base64: String,
}

fn marker(encoded: &str) -> Result<Vec<u8>> {
    let bytes = STANDARD.decode(encoded).map_err(Error::new)?;
    if bytes.is_empty() || bytes.len() > 4096 {
        return Err(Error::msg("invalid broker crash marker"));
    }
    Ok(bytes)
}

pub(super) fn validate(request: &ClusterExecute) -> Result<()> {
    if !request.manual_start
        && (!request.boots.is_empty()
            || !request.output_crashes.is_empty()
            || !request.output_stops.is_empty()
            || request.output_crash_group.is_some())
    {
        return Err(Error::msg("cluster lifecycle requires manual start"));
    }
    if request.manual_start && request.broker.is_none() {
        return Err(Error::msg("manual cluster requires broker"));
    }
    let mut boundaries = BTreeSet::new();
    for boundary in request.output_crashes.iter().chain(&request.output_stops) {
        if boundary.incarnation == 0
            || !request
                .instances
                .iter()
                .any(|instance| instance.id == boundary.node)
            || !boundaries.insert((boundary.node.clone(), boundary.incarnation))
        {
            return Err(Error::msg("invalid or duplicate cluster output boundary"));
        }
        marker(&boundary.marker_base64)?;
    }
    if let Some(group) = &request.output_crash_group {
        marker(&group.marker_base64)?;
        let mut ids = BTreeSet::new();
        if group.nodes.len() < 2 {
            return Err(Error::msg("invalid cluster output crash group"));
        }
        for id in &group.nodes {
            if !request.instances.iter().any(|instance| &instance.id == id)
                || !ids.insert(id)
                || boundaries.contains(&(id.clone(), 1))
            {
                return Err(Error::msg("invalid cluster output crash group node"));
            }
        }
    }
    for instance in request.instances.iter().chain(&request.boots) {
        let mut single = request.single(instance);
        single.local_model.clone_from(&instance.local_model);
        single.namespace.clone_from(&instance.namespace);
        single.storage.clone_from(&instance.storage);
        single.validate()?;
        if request.manual_start {
            let Some(model) = &instance.local_model else {
                return Err(Error::msg("cluster boot requires cooperative model"));
            };
            if model.split {
                return Err(Error::msg("cluster boot cannot forward model calls"));
            }
            match (&request.network, &instance.network_v2) {
                (Some(_), Some(network)) => {
                    validate_model(network).map_err(|failure| Error::msg(failure.message()))?;
                    if !network.faults.is_empty() {
                        return Err(Error::msg("broker node local fault schedule"));
                    }
                }
                (None, None) => {}
                _ => return Err(Error::msg("cluster boot network profile")),
            }
        }
        if instance
            .storage
            .as_ref()
            .is_some_and(|storage| storage.image.is_some())
        {
            return Err(Error::msg("cluster recovery images are lifecycle owned"));
        }
    }
    Ok(())
}

struct Boot {
    config: InstanceConfig,
    namespace_image: Option<crate::namespace::Image>,
    pre: Arc<InstancePre<State>>,
}

impl Boot {
    fn namespace(
        &self,
        storage: &mut Option<crate::storage_transport::StorageDescriptor>,
        image: Option<&[u8]>,
    ) -> Result<Option<crate::namespace::Namespace>> {
        let Some(initial) = &self.namespace_image else {
            return Ok(None);
        };
        let model = self.config.local_model.as_ref().unwrap();
        let mut namespace =
            crate::namespace::Namespace::new(initial.clone()).map_err(Error::new)?;
        if let Some(storage) = storage {
            if let Some(image) = image {
                use sha2::{Digest, Sha256};
                storage.image = Some(crate::storage_transport::ChunkedDescriptor {
                    bytes: image.len() as u64,
                    chunks: (image.len() as u64).div_ceil(64 * 1024),
                    sha256: format!("{:x}", Sha256::digest(image)),
                });
            }
            storage
                .attach(&mut namespace, image, model.limits.transcript_bytes)
                .map_err(crate::namespace::Stop::into_error)?;
        }
        Ok(Some(namespace))
    }
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Runtime {
    mode: u8,
    replay: Option<crate::model::replay::Request>,
    #[serde(default)]
    inspection: Option<crate::model::inspection::Request>,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Control {
    #[serde(rename = "type")]
    kind: String,
    id: u64,
    #[serde(default)]
    node: String,
    #[serde(default)]
    incarnation: u64,
    #[serde(default)]
    boot: String,
    #[serde(default)]
    runtime: Option<Runtime>,
    #[serde(default)]
    command: String,
    #[serde(default)]
    action: String,
    #[serde(default)]
    from: String,
    #[serde(default)]
    to: String,
    #[serde(default)]
    delay_nanos: u64,
    #[serde(default)]
    symmetric: bool,
}

type ThreadResult = std::result::Result<Report, (&'static str, String)>;

/// What the control loop waits for: Go's next control frame, or an
/// incarnation that ended in an instance or broker failure (fn-235).
enum Event {
    Control(Result<Vec<u8>>),
    Failed(&'static str, String),
}
struct Node {
    incarnation: u64,
    slot: u64,
    thread: Option<JoinHandle<ThreadResult>>,
    image: Option<Vec<u8>>,
}

fn join(
    node: &mut Node,
    id: &str,
    summaries: &mut Vec<InstanceSummary>,
    metrics: &mut crate::metrics::Metrics,
    scheduler: &mut crate::cluster::handoff::Metrics,
) -> std::result::Result<(), (&'static str, String)> {
    if let Some(thread) = node.thread.take() {
        let report = thread
            .join()
            .map_err(|_| ("infrastructure", "cluster store panicked".to_owned()))??;
        metrics.instantiate_nanos += report.instantiate_nanos;
        metrics.execute_nanos += report.execute_nanos;
        scheduler.host_nanos += report.host_nanos;
        scheduler.guest_nanos += report.guest_nanos;
        summaries.push(InstanceSummary {
            id: id.to_owned(),
            incarnation: node.incarnation,
            termination: report.termination,
        });
        node.image = report.image;
    }
    Ok(())
}

#[expect(
    clippy::needless_pass_by_value,
    reason = "Execution owns transport handles for the full incarnation lifecycle"
)]
#[expect(
    clippy::too_many_lines,
    reason = "The ordered model transition preserves Go admission, mutation and refusal order"
)]
pub(super) fn execute(
    mut request: ClusterExecute,
    input: SharedInput,
    output: SharedOutput,
) -> Result<()> {
    let transport_failure = Broken::default();
    let mut metrics = crate::metrics::Metrics::default();
    let mut summaries = Vec::new();
    let mut execution_metrics = crate::cluster::handoff::Metrics::default();
    let mut result = ClusterResult::new("exit", "");
    let phase = Instant::now();
    let prepared = (|| -> std::result::Result<_, Outcome> {
        let mut configurations = std::mem::take(&mut request.instances);
        let ids: Vec<_> = configurations
            .iter()
            .map(|instance| instance.id.clone())
            .collect();
        configurations.extend(std::mem::take(&mut request.boots));
        let mut sealed = Vec::new();
        for mut config in configurations {
            let mut tagged = TaggedInput::new(input.clone(), &config.id, transport_failure.clone());
            if let Some(model) = &mut config.local_model {
                model.read_inputs(&mut tagged).map_err(|error| {
                    Outcome::new(protocol::classification(&error), error.to_string())
                })?;
            }
            let namespace_image = match (&config.namespace, &config.local_model) {
                (Some(namespace), Some(model)) => Some(
                    crate::namespace::transport::read_image(
                        &mut tagged,
                        namespace,
                        config.output_bytes,
                        model.epoch_nanos,
                        &model.working_directory_sha256,
                    )
                    .map_err(|error| {
                        Outcome::new(protocol::classification(&error), error.to_string())
                    })?,
                ),
                _ => None,
            };
            if let (Some(storage), Some(namespace)) = (&mut config.storage, &namespace_image) {
                let maximum = crate::storage_transport::StorageDescriptor::image_limit(
                    namespace.limits.filesystem_bytes,
                );
                storage
                    .read_context(&mut tagged, maximum.saturating_mul(18))
                    .map_err(|error| {
                        Outcome::new(protocol::classification(&error), error.to_string())
                    })?;
            }
            sealed.push((config, namespace_image));
        }
        let primary = request.single(&sealed[0].0);
        let (engine, module, started) = crate::engine::prepare(&primary, &mut metrics)?;
        let mut modules: BTreeMap<String, (Arc<InstancePre<State>>, Value)> = BTreeMap::new();
        modules.insert(
            primary.module_sha256,
            (
                Arc::new(
                    crate::engine::linker(&engine, &module)
                        .and_then(|linker| linker.instantiate_pre(&module))
                        .map_err(|error| Outcome::new("infrastructure", error.to_string()))?,
                ),
                started.clone(),
            ),
        );
        let mut boots = BTreeMap::new();
        for (config, namespace_image) in sealed {
            let single = request.single(&config);
            if !modules.contains_key(&single.module_sha256) {
                let mut module_metrics = crate::metrics::Metrics::default();
                let (module, started) =
                    crate::engine::prepare_module(&single, &engine, &mut module_metrics)?;
                include_module_metrics(&mut metrics, &module_metrics);
                let pre = crate::engine::linker(&engine, &module)
                    .and_then(|linker| linker.instantiate_pre(&module))
                    .map_err(|error| Outcome::new("infrastructure", error.to_string()))?;
                modules.insert(single.module_sha256.clone(), (Arc::new(pre), started));
            }
            let (pre, mut admitted) = modules[&single.module_sha256].clone();
            let boot = Boot {
                config,
                namespace_image,
                pre,
            };
            let mut storage = boot.config.storage.clone();
            boot.namespace(&mut storage, None).map_err(|error| {
                Outcome::new(protocol::classification(&error), error.to_string())
            })?;
            admitted["type"] = "boot_started".into();
            admitted["boot"] = boot.config.id.clone().into();
            protocol::send_frame(&mut **lock(&output), &admitted)
                .map_err(|error| Outcome::new("infrastructure", error.to_string()))?;
            boots.insert(boot.config.id.clone(), boot);
        }
        Ok((engine, started, ids, boots))
    })();
    metrics.prepare_nanos = crate::metrics::elapsed(phase);
    let finish = |mut result: ClusterResult, mut metrics: crate::metrics::Metrics| {
        metrics.sample_process();
        result.metrics = Some(metrics);
        protocol::send_frame(&mut **lock(&output), &result)
    };
    let (engine, started, ids, boots) = match prepared {
        Ok(prepared) => prepared,
        Err(outcome) => {
            return finish(
                ClusterResult::new(outcome.termination, outcome.message),
                metrics,
            );
        }
    };
    protocol::send_frame(&mut **lock(&output), &started)?;
    if let Err(error) = crate::engine::authorize(
        &mut **lock(&input),
        started["module_sha256"].as_str().unwrap(),
    ) {
        return finish(
            ClusterResult::new(protocol::classification(&error), error.to_string()),
            metrics,
        );
    }
    let broker = Arc::new(
        crate::cluster::handoff::Broker::new(
            request.broker.as_ref().unwrap(),
            &ids,
            request.shared.clone(),
            None,
        )
        .map_err(Error::msg)?,
    );
    if let Some(group) = &request.output_crash_group {
        broker
            .barrier(&group.nodes, marker(&group.marker_base64)?)
            .map_err(|failure| Error::msg(failure.message()))?;
    }
    let coordinator = Coordinator::Broker(broker.clone());
    broker.defer_starts();
    let mut nodes: Vec<Node> = ids
        .iter()
        .enumerate()
        .map(|(index, _)| Node {
            incarnation: 0,
            slot: index as u64 + 1,
            thread: None,
            image: None,
        })
        .collect();
    broker.start();
    // Go's control frames are read on their own thread, one frame per
    // request, so a failed incarnation ends the run promptly: Go waits for
    // that incarnation's result and sends no further control frame (fn-235).
    let (events, received) = std::sync::mpsc::channel::<Event>();
    let request_frame = control_reader(input.clone(), events.clone())?;
    let mut reading = false;
    let mut failed_first = false;
    let mut command_id = 0;
    loop {
        if !reading {
            reading = request_frame.send(()).is_ok();
        }
        let command = match received.recv() {
            Ok(Event::Control(frame)) => {
                reading = false;
                frame
                    .and_then(|bytes| serde_json::from_slice::<Control>(&bytes).map_err(Error::new))
            }
            Ok(Event::Failed(..)) if broker.inspection_stopped() => continue,
            Ok(Event::Failed(termination, message)) => {
                result = ClusterResult::new(termination, message);
                failed_first = true;
                break;
            }
            Err(error) => Err(Error::new(error)),
        };
        let control = match command {
            Ok(control) if control.id == command_id + 1 => control,
            Ok(_) => {
                result = ClusterResult::new("invalid", "cluster control sequence");
                break;
            }
            Err(error) => {
                result = ClusterResult::new("invalid", format!("cluster control: {error}"));
                break;
            }
        };
        command_id = control.id;
        let ack = |failure: Option<(&str, String)>| {
            let (termination, message) = failure.unwrap_or(("", String::new()));
            protocol::send_frame(
                &mut **lock(&output),
                &json!({"type":"control_result","id":control.id,"termination":termination,"message":message}),
            )
        };
        let refuse = |error: &Error| {
            if let Some(preflight) = error.downcast_ref::<crate::model::replay::Preflight>() {
                protocol::send_frame(
                    &mut **lock(&output),
                    &json!({"type":"control_result","id":control.id,"termination":"invalid","message":error.to_string(),"error":"replay_preflight","check":preflight.check}),
                )
            } else {
                ack(Some((protocol::classification(error), error.to_string())))
            }
        };
        if control.kind == "finish" {
            let inspected = broker.inspection_stopped();
            if !inspected
                && nodes
                    .iter()
                    .enumerate()
                    .any(|(index, node)| node.incarnation != 0 && !broker.finished(index))
            {
                ack(Some((
                    "invalid",
                    "cluster finish requires terminal incarnations".into(),
                )))?;
                continue;
            }
            ack(None)?;
            for (index, node) in nodes.iter().enumerate() {
                if node.incarnation == 0 {
                    broker.finish(index);
                }
            }
            break;
        }
        let index = ids.iter().position(|id| id == &control.node);
        if control.kind == "run" {
            let mut wire = lock(&output);
            let changed = match index {
                Some(index) => broker.run(index, control.incarnation),
                None => Err(Failure::Invalid("unknown cluster node")),
            };
            let (termination, message) = changed.err().map_or(("", String::new()), |failure| {
                (failure.termination(), failure.message())
            });
            protocol::send_frame(
                &mut **wire,
                &json!({"type":"control_result","id":control.id,"termination":termination,"message":message}),
            )?;
        } else if matches!(control.kind.as_str(), "start" | "restart") {
            let Some(index) = index else {
                ack(Some(("invalid", "unknown cluster node".into())))?;
                continue;
            };
            let Some(boot) = boots.get(&control.boot) else {
                ack(Some(("invalid", "unknown cluster boot".into())))?;
                continue;
            };
            let node = &mut nodes[index];
            if control.incarnation == 0
                || node.incarnation.checked_add(1) != Some(control.incarnation)
                || (control.kind == "start") != (control.incarnation == 1)
            {
                ack(Some(("invalid", "cluster incarnation lifecycle".into())))?;
                continue;
            }
            if node.incarnation != 0 && !broker.finished(index) {
                ack(Some(("invalid", "broker restart lifecycle".into())))?;
                continue;
            }
            if let Err(failure) = join(
                node,
                &ids[index],
                &mut summaries,
                &mut metrics,
                &mut execution_metrics,
            ) {
                result = ClusterResult::new(failure.0, failure.1);
                break;
            }
            let mut config = boot.config.clone();
            config.id.clone_from(&ids[index]);
            if let Some(runtime) = control.runtime {
                let Some(model) = &mut config.local_model else {
                    ack(Some(("invalid", "cluster runtime without model".into())))?;
                    continue;
                };
                model.mode = runtime.mode;
                model.replay = runtime.replay;
                model.tape = None;
                model.inspection = runtime.inspection;
                if let Err(error) = model.validate() {
                    refuse(&error)?;
                    continue;
                }
            }
            if let Some(model) = &mut config.local_model {
                let mut tagged = TaggedInput::incarnation(
                    input.clone(),
                    &config.id,
                    control.incarnation,
                    transport_failure.clone(),
                );
                let read = read_instance_tape(model, &mut tagged, &transport_failure);
                if let Some(message) = transport_failure.get() {
                    result = ClusterResult::new("invalid", message);
                    break;
                }
                if let Err(error) = read {
                    refuse(&error)?;
                    continue;
                }
            }
            if control.incarnation > 1 && config.storage.is_some() && node.image.is_none() {
                ack(Some((
                    "invalid",
                    "cluster restart lacks recovery image".into(),
                )))?;
                continue;
            }
            let mut storage = config.storage.clone();
            let namespace = match boot.namespace(&mut storage, node.image.as_deref()) {
                Ok(namespace) => namespace,
                Err(error) => {
                    ack(Some((protocol::classification(&error), error.to_string())))?;
                    continue;
                }
            };
            if control.incarnation > 1 {
                match broker.restart(index, control.incarnation) {
                    Ok((_, slot)) => node.slot = slot,
                    Err(failure) => {
                        ack(Some((failure.termination(), failure.message())))?;
                        continue;
                    }
                }
            } else {
                broker
                    .activate(index)
                    .map_err(|failure| Error::msg(failure.message()))?;
            }
            for (kind, boundaries) in [
                ("crash", &request.output_crashes),
                ("stop", &request.output_stops),
            ] {
                for boundary in boundaries.iter().filter(|boundary| {
                    boundary.node == config.id && boundary.incarnation == control.incarnation
                }) {
                    broker
                        .boundary(
                            index,
                            control.incarnation,
                            kind,
                            marker(&boundary.marker_base64)?,
                        )
                        .map_err(|failure| Error::msg(failure.message()))?;
                }
            }
            ack(None)?;
            node.incarnation = control.incarnation;
            let (engine, pre, coordinator, network, transport_failure) = (
                engine.clone(),
                boot.pre.clone(),
                coordinator.clone(),
                request.shared.clone(),
                transport_failure.clone(),
            );
            let tagged_input = TaggedInput::incarnation(
                input.clone(),
                &config.id,
                control.incarnation,
                transport_failure.clone(),
            );
            let tagged_output =
                TaggedOutput::incarnation(output.clone(), &config.id, control.incarnation);
            let incarnation = control.incarnation;
            let slot = node.slot;
            let failed = events.clone();
            let broker = broker.clone();
            let spawned = std::thread::Builder::new()
                .name(format!("instance-{index}-{incarnation}"))
                .stack_size(STORE_THREAD_STACK)
                .spawn(move || {
                    if !coordinator.acquire(index) && !broker.terminal_failed() {
                        if coordinator.inspection_stopped() {
                            return refuse_instance(
                                Outcome::new("inspection_aborted", "cluster inspection stopped"),
                                tagged_output,
                            );
                        }
                        return Err(("infrastructure", "cluster aborted".into()));
                    }
                    let ran = catch_unwind(AssertUnwindSafe(|| {
                        run_instance(
                            &engine,
                            &pre,
                            config,
                            index,
                            network,
                            &coordinator,
                            tagged_input,
                            tagged_output,
                            &transport_failure,
                            incarnation,
                            slot,
                            namespace,
                            storage,
                            true,
                            Some(&|outcome| {
                                let Some(failure) = super::terminal_failure(outcome) else {
                                    return;
                                };
                                let _ = failed.send(Event::Failed(failure.0, failure.1.clone()));
                                broker.abort_terminal(index, failure.0, failure.1);
                            }),
                        )
                    }));
                    let report = match ran {
                        Ok(report) => report,
                        Err(_) => Err(("infrastructure", "cluster store panicked".into())),
                    };
                    match &report {
                        Ok(_) => coordinator.finish(index),
                        Err((kind, message)) => {
                            // Queued before the abort wakes the other
                            // incarnations, whose aborts only follow it.
                            let _ = failed.send(Event::Failed(kind, message.clone()));
                            broker.abort_terminal(index, kind, message.clone());
                            if broker.terminal_failed() {
                                coordinator.finish(index);
                            }
                        }
                    }
                    report
                });
            match spawned {
                Ok(thread) => node.thread = Some(thread),
                Err(error) => {
                    result = ClusterResult::new("infrastructure", error.to_string());
                    break;
                }
            }
        } else if control.kind == "control" {
            let changed = match control.command.as_str() {
                "crash" | "stop" => match index {
                    Some(index) => broker.command(crate::cluster::handoff::Command::Revoke {
                        index,
                        incarnation: control.incarnation,
                        kind: if control.command == "crash" {
                            "crash"
                        } else {
                            "stop"
                        },
                    }),
                    None => Err(Failure::Invalid("unknown broker node")),
                },
                "link" => broker.command(crate::cluster::handoff::Command::Link {
                    action: control.action,
                    from: control.from,
                    to: control.to,
                    delay: control.delay_nanos,
                    symmetric: control.symmetric,
                }),
                _ => Err(Failure::Invalid("unknown cluster control")),
            };
            ack(changed
                .err()
                .map(|failure| (failure.termination(), failure.message())))?;
        } else {
            ack(Some(("invalid", "unknown cluster control".into())))?;
        }
    }
    if result.termination != "exit" {
        coordinator.abort(result.termination, result.message.clone());
    }
    // A failed incarnation's failure is the run's; the instances its abort
    // stopped only report that abort.
    for (index, node) in nodes.iter_mut().enumerate() {
        if let Err(failure) = join(
            node,
            &ids[index],
            &mut summaries,
            &mut metrics,
            &mut execution_metrics,
        ) && !broker.inspection_stopped()
            && !failed_first
        {
            result = ClusterResult::new(failure.0, failure.1);
        }
    }
    if let Some((kind, message)) = broker.failure() {
        result = ClusterResult::new(kind, message);
    }
    if matches!(result.termination, "exit" | "trap" | "capacity")
        && let Some(shared) = &request.shared
    {
        let source = ids
            .iter()
            .filter_map(|id| boots.get(id))
            .max_by_key(|boot| {
                boot.config
                    .local_model
                    .as_ref()
                    .unwrap()
                    .limits
                    .transcript_bytes
            })
            .unwrap();
        let index = ids.iter().position(|id| id == &source.config.id).unwrap();
        let model = source.config.local_model.as_ref().unwrap();
        let endpoint = Endpoint::new(
            shared.clone(),
            &source.config.id,
            nodes[index].incarnation.max(1),
            nodes[index].slot,
            source.config.network_v2.as_ref().unwrap(),
            model.epoch_nanos,
            model.limits.transcript_bytes,
        );
        let evidence = endpoint.and_then(|endpoint| endpoint.evidence());
        match evidence {
            Ok(evidence) => {
                for chunk in evidence.chunks(64 * 1024) {
                    protocol::send_frame(
                        &mut **lock(&output),
                        &json!({"type":"cluster_network","source":source.config.id,"data_base64":STANDARD.encode(chunk)}),
                    )?;
                }
            }
            Err(failure) => result = ClusterResult::new(failure.termination(), failure.message()),
        }
    }
    if matches!(result.termination, "exit" | "trap" | "capacity") {
        match broker.world_wire() {
            Ok(wire) => {
                for chunk in wire.chunks(64 * 1024) {
                    protocol::send_frame(
                        &mut **lock(&output),
                        &json!({"type":"evidence","stream":"world","data_base64":STANDARD.encode(chunk)}),
                    )?;
                }
            }
            Err(failure) => result = ClusterResult::new(failure.termination(), failure.message()),
        }
    }
    summaries.sort_by(|a, b| a.id.cmp(&b.id).then(a.incarnation.cmp(&b.incarnation)));
    result.instances = summaries;
    result.virtual_elapsed_nanos = broker.current();
    let mut scheduler = broker.metrics();
    scheduler.host_nanos = execution_metrics.host_nanos;
    scheduler.guest_nanos = execution_metrics.guest_nanos;
    result.scheduler = Some(scheduler);
    super::inspection_evidence(&broker, &output)?;
    finish(result, metrics)
}

/// Reads one Go control frame per request into `events`, in pipe order.
fn control_reader(
    input: SharedInput,
    events: std::sync::mpsc::Sender<Event>,
) -> Result<std::sync::mpsc::Sender<()>> {
    let (request_frame, frame_requested) = std::sync::mpsc::channel::<()>();
    std::thread::Builder::new()
        .name("cluster-control".into())
        .spawn(move || {
            while frame_requested.recv().is_ok() {
                let frame = protocol::read_frame(&mut **lock(&input));
                let failed = frame.is_err();
                if events.send(Event::Control(frame)).is_err() || failed {
                    break;
                }
            }
        })?;
    Ok(request_frame)
}
