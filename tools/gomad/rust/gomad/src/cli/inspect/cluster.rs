use crate::evidence::store::Opened;
use serde::Serialize;
use std::{
    collections::{BTreeMap, BTreeSet},
    io::Write,
};

#[derive(Serialize)]
pub(super) struct Cluster {
    virtual_elapsed_nanos: u64,
    stdout_bytes: u64,
    stderr_bytes: u64,
    offered_output_bytes: u64,
    discarded_output_bytes: u64,
    truncated: bool,
    exhausted: Option<String>,
    instances: Vec<Instance>,
}

#[derive(Serialize)]
struct Instance {
    node: String,
    incarnation: u64,
    interrupted: bool,
    stdout_bytes: u64,
    stderr_bytes: u64,
    offered_output_bytes: u64,
    discarded_output_bytes: u64,
    truncated: bool,
    stdout_tail: String,
}

impl Cluster {
    pub(super) fn append_text(&self, output: &mut Vec<u8>) -> Result<(), String> {
        writeln!(
            output,
            "cluster: virtual-elapsed-nanos={}",
            self.virtual_elapsed_nanos
        )
        .map_err(|error| error.to_string())?;
        if let Some(bound) = &self.exhausted {
            writeln!(output, "exhausted={}", bound.escape_default())
                .map_err(|error| error.to_string())?;
        }
        for instance in &self.instances {
            let tail =
                serde_json::to_string(&instance.stdout_tail).map_err(|error| error.to_string())?;
            writeln!(output, "cluster-instance: node={} incarnation={} stdout-bytes={} stderr-bytes={} stdout-tail={tail}",
                instance.node.escape_default(), instance.incarnation, instance.stdout_bytes, instance.stderr_bytes)
                .map_err(|error| error.to_string())?;
        }
        Ok(())
    }
}

pub(super) fn read(opened: &Opened) -> Result<Option<Cluster>, String> {
    if !opened
        .manifest()
        .files
        .contains_key("evidence/cluster_summary")
    {
        if opened.manifest().files.keys().any(|name| {
            name == crate::replay::cluster::FILE || name.starts_with("evidence/cluster/")
        }) {
            return Err("missing cluster observation".into());
        }
        return Ok(None);
    }
    let raw = opened.payload("evidence/cluster_summary", 128 << 20)?;
    let summary: serde_json::Value = crate::qualify::spec::strict(&raw)?;
    if summary["schema"] != "gomad.cluster-observation/v1" {
        return Err("unsupported cluster observation".into());
    }
    let mut cluster = Cluster {
        virtual_elapsed_nanos: summary["virtual_elapsed_nanos"]
            .as_u64()
            .ok_or("missing cluster elapsed time")?,
        stdout_bytes: 0,
        stderr_bytes: 0,
        offered_output_bytes: 0,
        discarded_output_bytes: 0,
        truncated: false,
        exhausted: (opened.manifest().record.outcome.termination == "capacity")
            .then(|| {
                opened
                    .manifest()
                    .record
                    .outcome
                    .reason
                    .strip_prefix("WASI capacity: ")
                    .map(str::to_owned)
            })
            .flatten(),
        instances: Vec::new(),
    };
    let mut owners = BTreeMap::new();
    let mut expected_files = BTreeSet::new();
    for row in summary["incarnations"]
        .as_array()
        .ok_or("missing cluster instances")?
    {
        let (instance, files) = instance(opened, row)?;
        if owners
            .insert(
                (instance.node.clone(), instance.incarnation),
                instance.interrupted,
            )
            .is_some()
        {
            return Err("duplicate cluster instance".into());
        }
        expected_files.extend(files);
        cluster.stdout_bytes = cluster
            .stdout_bytes
            .checked_add(instance.stdout_bytes)
            .ok_or("cluster stdout bytes overflow")?;
        cluster.stderr_bytes = cluster
            .stderr_bytes
            .checked_add(instance.stderr_bytes)
            .ok_or("cluster stderr bytes overflow")?;
        cluster.offered_output_bytes = cluster
            .offered_output_bytes
            .checked_add(instance.offered_output_bytes)
            .ok_or("cluster offered output bytes overflow")?;
        cluster.instances.push(instance);
    }
    if opened
        .manifest()
        .files
        .keys()
        .filter(|name| name.starts_with("evidence/cluster/"))
        .any(|name| !expected_files.contains(name))
    {
        return Err("orphan cluster instance payload".into());
    }
    validate_owners(opened, &owners)?;
    let retained = cluster
        .stdout_bytes
        .checked_add(cluster.stderr_bytes)
        .ok_or("cluster retained output bytes overflow")?;
    cluster.discarded_output_bytes = cluster
        .offered_output_bytes
        .checked_sub(retained)
        .ok_or("cluster output accounting underflow")?;
    cluster.truncated = cluster.discarded_output_bytes != 0;
    Ok(Some(cluster))
}

fn instance(opened: &Opened, row: &serde_json::Value) -> Result<(Instance, Vec<String>), String> {
    let node = row["handle"]["node"]
        .as_str()
        .ok_or("missing cluster node")?;
    crate::evidence::store::open::components(node)?;
    if node.contains('/') {
        return Err("invalid cluster node".into());
    }
    let incarnation = row["handle"]["incarnation"]
        .as_u64()
        .filter(|value| *value > 0)
        .ok_or("missing cluster incarnation")?;
    let interrupted = row["interrupted"]
        .as_bool()
        .ok_or("missing cluster interruption state")?;
    let (counts, tail) = cluster_output(opened, node, incarnation, &row["streams"])?;
    let mut expected_files = Vec::new();
    for name in row["streams"]
        .as_object()
        .ok_or("missing cluster instance streams")?
        .keys()
    {
        expected_files.push(format!("evidence/cluster/{node}/{incarnation}/{name}"));
    }
    let offered = row["offered_output_bytes"]
        .as_u64()
        .ok_or("missing offered cluster output")?;
    let retained = counts[0]
        .checked_add(counts[1])
        .ok_or("cluster output bytes overflow")?;
    if offered < retained {
        return Err("offered cluster output is smaller than retained output".into());
    }
    Ok((
        Instance {
            node: node.into(),
            incarnation,
            interrupted,
            stdout_bytes: counts[0],
            stderr_bytes: counts[1],
            offered_output_bytes: offered,
            discarded_output_bytes: offered - retained,
            truncated: offered != retained,
            stdout_tail: String::from_utf8_lossy(&tail).into_owned(),
        },
        expected_files,
    ))
}

fn validate_owners(opened: &Opened, owners: &BTreeMap<(String, u64), bool>) -> Result<(), String> {
    if opened
        .manifest()
        .files
        .contains_key(crate::replay::cluster::FILE)
    {
        let raw = opened.payload(crate::replay::cluster::FILE, 128 << 20)?;
        let inputs: crate::replay::cluster::Inputs = crate::qualify::spec::strict(&raw)?;
        let configured: BTreeSet<_> = inputs
            .configuration
            .incarnations
            .iter()
            .map(|item| (item.handle.node.clone(), item.handle.incarnation))
            .collect();
        if configured.len() != inputs.configuration.incarnations.len() {
            return Err("duplicate configured cluster instance".into());
        }
        let mut retained = BTreeMap::new();
        for item in inputs.retained {
            let handle = (item.handle.node, item.handle.incarnation);
            if !configured.contains(&handle) || retained.insert(handle, item.interrupted).is_some()
            {
                return Err("invalid retained cluster instance ownership".into());
            }
        }
        if &retained != owners {
            return Err("cluster summary differs from retained instance ownership".into());
        }
    }
    Ok(())
}

fn cluster_output(
    opened: &Opened,
    node: &str,
    incarnation: u64,
    streams: &serde_json::Value,
) -> Result<([u64; 2], Vec<u8>), String> {
    let streams = streams
        .as_object()
        .ok_or("missing cluster instance streams")?;
    let prefix = format!("evidence/cluster/{node}/{incarnation}/");
    let limits = &opened.manifest().record.limits;
    let mut evidence_bytes = 0_u64;
    let mut output_bytes = 0_u64;
    let mut counts = [0; 2];
    // run() already authenticated every manifest payload with the shared streaming verifier.
    for (name, descriptor) in streams {
        crate::evidence::store::open::components(name)?;
        let bytes = descriptor["bytes"]
            .as_u64()
            .ok_or("missing cluster stream byte count")?;
        evidence_bytes = evidence_bytes
            .checked_add(bytes)
            .ok_or("cluster evidence bytes overflow")?;
        if evidence_bytes > limits.evidence_bytes {
            return Err("cluster evidence exceeds retained evidence budget".into());
        }
        let output = match name.as_str() {
            "stdout" => Some(0),
            "stderr" => Some(1),
            _ => None,
        };
        let limit = if let Some(index) = output {
            output_bytes = output_bytes
                .checked_add(bytes)
                .ok_or("cluster output bytes overflow")?;
            if output_bytes > limits.output_bytes {
                return Err("cluster output exceeds retained output budget".into());
            }
            counts[index] = bytes;
            limits.output_bytes
        } else {
            limits.evidence_bytes
        };
        crate::replay::cluster::Payload {
            file: format!("{prefix}{name}"),
            bytes,
            sha256: descriptor["sha256"]
                .as_str()
                .ok_or("missing cluster stream digest")?
                .to_owned(),
            limit,
        }
        .bind(opened, limit)?;
    }
    if opened
        .manifest()
        .files
        .keys()
        .filter_map(|name| name.strip_prefix(&prefix))
        .any(|name| !streams.contains_key(name))
    {
        return Err("cluster instance output descriptor/payload mismatch".into());
    }
    let tail = if streams.contains_key("stdout") {
        opened.payload_tail(&format!("{prefix}stdout"), limits.output_bytes, 4096)?
    } else {
        Vec::new()
    };
    Ok((counts, tail))
}
