//! Repository receiving harness over the ordinary replay and output-consumer APIs.
use gomad::{
    evidence::canonical,
    replay::{divergence, preflight},
};
use gomad_wasmhost::linked::{self, OutputConsumer, OutputStream};
use serde_json::json;
use std::{
    collections::BTreeMap,
    fs,
    io::{self, Write as _},
    os::unix::fs::{DirBuilderExt as _, OpenOptionsExt as _},
    path::Path,
    sync::{Arc, Mutex, mpsc},
    time::{Duration, Instant},
};

const MARKER: &[u8] = b"gomad callback marker\n";
const OUTPUT_LIMIT: usize = 8 << 20;
const WAIT: Duration = Duration::from_secs(30);

enum Event {
    Marker(u64),
    Finished(Box<Result<linked::Execution, String>>),
}

#[derive(Default)]
struct Forwarded {
    stdout: Vec<u8>,
    stderr: Vec<u8>,
    marker_sent: bool,
    release_received_ns: Option<u64>,
}

struct Controlled {
    forwarded: Mutex<Forwarded>,
    events: mpsc::SyncSender<Event>,
    release: Mutex<mpsc::Receiver<()>>,
}

impl OutputConsumer for Controlled {
    fn write(&self, stream: OutputStream, bytes: &[u8]) -> io::Result<()> {
        let mut forwarded = self
            .forwarded
            .lock()
            .map_err(|error| io::Error::other(error.to_string()))?;
        let target = match stream {
            OutputStream::Stdout => &mut forwarded.stdout,
            OutputStream::Stderr => &mut forwarded.stderr,
        };
        if bytes.len() > OUTPUT_LIMIT.saturating_sub(target.len()) {
            return Err(io::Error::other("forwarded output exceeds original bound"));
        }
        target.extend_from_slice(bytes);
        let hold = stream == OutputStream::Stdout
            && !forwarded.marker_sent
            && forwarded.stdout.len() >= MARKER.len();
        if hold {
            if !forwarded.stdout.starts_with(MARKER) {
                return Err(io::Error::other(
                    "compiled Go guest did not write the registered marker",
                ));
            }
            forwarded.marker_sent = true;
        }
        drop(forwarded);
        if hold {
            self.events
                .send(Event::Marker(
                    linked::monotonic_nanos().map_err(io::Error::other)?,
                ))
                .map_err(|error| io::Error::other(error.to_string()))?;
            self.release
                .lock()
                .map_err(|error| io::Error::other(error.to_string()))?
                .recv_timeout(WAIT)
                .map_err(io::Error::other)?;
            self.forwarded
                .lock()
                .map_err(|error| io::Error::other(error.to_string()))?
                .release_received_ns = Some(linked::monotonic_nanos().map_err(io::Error::other)?);
        }
        Ok(())
    }
}

fn write(path: &Path, bytes: &[u8]) -> Result<(), String> {
    let mut file = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(bytes)
        .and_then(|()| file.sync_all())
        .map_err(|error| error.to_string())
}

fn prepare(artifact: &Path) -> Result<preflight::Prepared, String> {
    let prepared = preflight::open(artifact)?;
    if !prepared.exact || !prepared.available {
        return Err("controlled Go receiving requires complete exact replay".into());
    }
    let record = &prepared.opened.manifest().record;
    let provenance: serde_json::Value = serde_json::from_slice(
        &prepared
            .opened
            .payload("target/provenance.json", 128 << 20)?,
    )
    .map_err(|error| error.to_string())?;
    if provenance["inputs"]["kind"] != "go-run"
        || provenance["inputs"]["source"] != "./cmd/gomad/testdata/seedfree"
        || provenance["inputs"]["arguments"] != json!(["stream"])
        || record.outcome.termination != "exit"
        || record.outcome.exit_code != Some(0)
        || record.seed != 1
    {
        return Err(
            "controlled Go receiving requires the original successful stream fixture".into(),
        );
    }
    Ok(prepared)
}

struct ControlledResult {
    result: linked::Execution,
    controlled: Arc<Controlled>,
    callback_ns: u64,
    observed_ns: u64,
    release_written_ns: u64,
}

fn controlled_execution(
    mut prepared: preflight::Prepared,
    output: &Path,
    parent: &str,
    deadline: Instant,
) -> Result<ControlledResult, String> {
    let (events, observed) = mpsc::sync_channel(2);
    let (release, released) = mpsc::sync_channel(1);
    let controlled = Arc::new(Controlled {
        forwarded: Mutex::new(Forwarded::default()),
        events: events.clone(),
        release: Mutex::new(released),
    });
    match &mut prepared.execution {
        preflight::Execution::Standalone(execution) => execution.output(controlled.clone()),
        preflight::Execution::Cluster(_) => {
            return Err("controlled Go receiving requires standalone execution".into());
        }
    }
    let (result, callback_ns, observed_ns, release_written_ns) =
        std::thread::scope(|scope| -> Result<_, String> {
            scope.spawn(move || {
                let executed = preflight::execute(prepared.execution);
                let _ = events.send(Event::Finished(Box::new(executed)));
            });
            let remaining = || {
                deadline
                    .checked_duration_since(Instant::now())
                    .ok_or("original overall deadline exceeded".to_owned())
            };
            let callback_ns = match observed
                .recv_timeout(remaining()?)
                .map_err(|error| error.to_string())?
            {
                Event::Marker(at) => at,
                Event::Finished(result) => {
                    return Err(format!(
                        "guest completed before controlled marker: {}",
                        (*result).err().unwrap_or_default()
                    ));
                }
            };
            let observed_ns = linked::monotonic_nanos()?;
            if !matches!(observed.try_recv(), Err(mpsc::TryRecvError::Empty)) {
                return Err("guest completed before controller release".into());
            }
            write(&output.join("marker"), MARKER)?;
            write(
                &output.join("controller-release.json"),
                &canonical::encode(&json!({
                    "schema":"gomad.interception-streaming-release/v1", "pid":std::process::id(),
                    "parent_record_sha256":parent, "marker_sha256":canonical::digest(MARKER),
                    "observed_ns":observed_ns, "action_started_ns":linked::monotonic_nanos()?
                }))?,
            )?;
            fs::File::open(output)
                .and_then(|directory| directory.sync_all())
                .map_err(|error| error.to_string())?;
            let release_written_ns = linked::monotonic_nanos()?;
            release.send(()).map_err(|error| error.to_string())?;
            let executed = match observed
                .recv_timeout(remaining()?)
                .map_err(|error| error.to_string())?
            {
                Event::Finished(result) => (*result)?,
                Event::Marker(_) => return Err("duplicate controlled marker".into()),
            };
            Ok((executed, callback_ns, observed_ns, release_written_ns))
        })?;
    Ok(ControlledResult {
        result,
        controlled,
        callback_ns,
        observed_ns,
        release_written_ns,
    })
}

fn run(artifact: &Path, output: &Path) -> Result<(), String> {
    let identity = gomad::runtime_qualification::identity();
    if identity["release_role"] != true {
        return Err("controlled Go receiving requires ordinary release host and helper".into());
    }
    let deadline = Instant::now() + Duration::from_secs(600);
    fs::DirBuilder::new()
        .mode(0o700)
        .create(output)
        .map_err(|error| error.to_string())?;
    let prepared = prepare(artifact)?;
    let record = prepared.opened.manifest().record.clone();
    let parent = prepared.opened.manifest().record_sha256.clone();
    let ControlledResult {
        result,
        controlled,
        callback_ns,
        observed_ns,
        release_written_ns,
    } = controlled_execution(prepared, output, &parent, deadline)?;
    if let Some(reason) = divergence::first(&record, &result) {
        return Err(format!("controlled Go replay diverged: {reason}"));
    }
    let forwarded = controlled
        .forwarded
        .lock()
        .map_err(|error| error.to_string())?;
    for (name, bytes) in [("stdout", &forwarded.stdout), ("stderr", &forwarded.stderr)] {
        if result.evidence.get(name).map_or(&[][..], Vec::as_slice) != bytes.as_slice() {
            return Err(format!("forwarded {name} differs from retained execution"));
        }
    }
    let mut streams = BTreeMap::new();
    for name in ["stdout", "stderr", "transcript", "choice"] {
        let bytes = result.evidence.get(name).map_or(&[][..], Vec::as_slice);
        write(&output.join(name), bytes)?;
        streams.insert(name, canonical::digest(bytes));
    }
    let release_received_ns = forwarded
        .release_received_ns
        .ok_or("controller did not release the actual callback")?;
    let terminal_ns = result
        .terminal_monotonic_nanos
        .ok_or("missing actual terminal clock")?;
    if !(callback_ns <= observed_ns
        && observed_ns < release_written_ns
        && release_written_ns <= release_received_ns
        && release_received_ns <= terminal_ns)
        || result.outcome.termination != "exit"
        || result.outcome.exit_code != Some(0)
        || Instant::now() >= deadline
    {
        return Err("controlled Go marker/release/terminal ordering failed".into());
    }
    write(
        &output.join("report.json"),
        &canonical::encode(&json!({
            "schema":"gomad.interception-streaming-proof/v1", "pid":std::process::id(),
            "parent_record_sha256":parent, "module_sha256":record.identity.target,
            "identity":identity, "not_complete_before_release":true, "exact":true,
            "marker_callback_ns":callback_ns, "marker_observed_ns":observed_ns,
            "release_written_ns":release_written_ns, "release_received_ns":release_received_ns,
            "terminal_ns":terminal_ns, "outcome":result.outcome, "streams":streams
        }))?,
    )
}

fn main() {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    let result = if arguments.len() == 2 {
        run(Path::new(&arguments[0]), Path::new(&arguments[1]))
    } else {
        Err("usage: interception-streaming-replay ARTIFACT FRESH-OUTPUT".into())
    };
    if let Err(error) = result {
        eprintln!("{error}");
        std::process::exit(2);
    }
}
