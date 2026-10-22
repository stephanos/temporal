use super::{Request, directory, read, write};
use crate::evidence::canonical;
use serde_json::{Value, json};
use std::{
    fs,
    path::Path,
    process::{Command, Stdio},
    sync::{
        Arc, Barrier,
        atomic::{AtomicBool, Ordering},
    },
    thread,
    time::Instant,
};

pub(super) fn host(command: &str, arguments: &[&str]) -> Value {
    match Command::new(command).args(arguments).output() {
        Ok(output) => {
            json!({"command":command,"arguments":arguments,"status":output.status.code(),"stdout":String::from_utf8_lossy(&output.stdout),"stderr":String::from_utf8_lossy(&output.stderr)})
        }
        Err(error) => json!({"command":command,"error":error.to_string()}),
    }
}

pub(super) struct Load {
    stopped: Arc<AtomicBool>,
    workers: Vec<thread::JoinHandle<()>>,
}

impl Load {
    pub(super) fn new(workers: usize) -> Self {
        let stopped = Arc::new(AtomicBool::new(false));
        let workers = (0..workers)
            .map(|_| {
                let stopped = stopped.clone();
                thread::spawn(move || {
                    let mut value = 1_u64;
                    while !stopped.load(Ordering::Relaxed) {
                        for _ in 0..100_000 {
                            value = value
                                .wrapping_mul(6_364_136_223_846_793_005)
                                .wrapping_add(1);
                        }
                        std::hint::black_box(value);
                    }
                })
            })
            .collect();
        Self { stopped, workers }
    }
}

impl Drop for Load {
    fn drop(&mut self) {
        self.stopped.store(true, Ordering::Relaxed);
        for worker in self.workers.drain(..) {
            let _ = worker.join();
        }
    }
}

pub(super) fn cohort(
    binary: &Path,
    requests: Vec<Request>,
    round: usize,
) -> Vec<Result<Value, String>> {
    let barrier = Barrier::new(requests.len());
    thread::scope(|scope| {
        let workers = requests
            .into_iter()
            .map(|request| {
                let barrier = &barrier;
                scope.spawn(move || {
                    barrier.wait();
                    execute(binary, &request, round)
                })
            })
            .collect::<Vec<_>>();
        workers
            .into_iter()
            .map(|worker| {
                worker
                    .join()
                    .map_err(|_| "stress subprocess supervisor panicked".to_owned())
                    .and_then(|result| result)
            })
            .collect()
    })
}

pub(super) fn supervision_requests(request: &Request, root: &Path) -> Result<Vec<Request>, String> {
    let bounds = &request.supervision;
    if !root.is_absolute()
        || bounds.repetition >= 256
        || bounds.member >= bounds.cohort_size
        || !(1..=256).contains(&bounds.cohort_size)
        || match request.axis.as_str() {
            "cpu" => !(1..=256).contains(&bounds.cpu_workers) || bounds.cohort_size != 1,
            "concurrent" => bounds.cpu_workers != 0 || bounds.cohort_size < 2,
            "reference" | "layout" | "handoff" => {
                bounds.cpu_workers != 0 || bounds.cohort_size != 1
            }
            _ => true,
        }
    {
        return Err("invalid retained stress supervision bounds".into());
    }
    Ok((0..bounds.cohort_size)
        .map(|member| {
            let mut next = request.clone();
            next.root = root.join(format!("member-{member}"));
            next.supervision.member = member;
            next
        })
        .collect())
}

pub(super) fn supervise(binary: &Path, request: &Request, root: &Path) -> Result<Value, String> {
    let requests = supervision_requests(request, root)?;
    directory(root)?;
    for member in &requests {
        directory(&member.root)?;
    }
    let load = Load::new(request.supervision.cpu_workers);
    let mut runs = Vec::new();
    let mut errors = Vec::new();
    for result in cohort(binary, requests, request.supervision.repetition) {
        match result {
            Ok(run) => runs.push(run),
            Err(error) => errors.push(error),
        }
    }
    drop(load);
    let passed = errors.is_empty()
        && runs.len() == request.supervision.cohort_size
        && runs.iter().all(|run| {
            run["oracle_error"].is_null()
                && run["perturbation_error"].is_null()
                && run["event_order_error"].is_null()
        });
    let report = json!({"schema":"gomad.stress-rerun/v1","axis":request.axis,"seed":request.seed,
        "supervision":request.supervision,"runs":runs,"errors":errors,"passed":passed});
    write(
        &root.join("rerun-report.json"),
        &canonical::encode(&report)?,
    )?;
    Ok(report)
}

pub(super) fn execute(
    binary: &Path,
    request: &Request,
    repetition: usize,
) -> Result<Value, String> {
    let path = request.root.join("request.json");
    write(&path, &canonical::encode(request)?)?;
    let layout = request.axis == "layout";
    let directory_path = request.root.join(if layout {
        "p".repeat(1 + repetition % 4 * 40)
    } else {
        "process".into()
    });
    directory(&directory_path)?;
    let executable = directory_path.join("gomad");
    fs::hard_link(binary, &executable)
        .map_err(|error| format!("stress executable layout link: {error}"))?;
    let stdout = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(request.root.join("stdout.log"))
        .map_err(|error| error.to_string())?;
    let stderr = fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(request.root.join("stderr.log"))
        .map_err(|error| error.to_string())?;
    let mut command = Command::new(&executable);
    command
        .args(["stress", "--cell"])
        .arg(&path)
        .current_dir(&directory_path)
        .env("GOMAD_WASM_STRESS", "1")
        .env_remove("GOMAD_WASM_STRESS_JITTER")
        .env_remove("GOMAD_STRESS_PADDING")
        .stdin(Stdio::null())
        .stdout(stdout)
        .stderr(stderr);
    if layout {
        command
            .env(
                "GOMAD_STRESS_PADDING",
                "x".repeat(1 + repetition % 16 * 4096),
            )
            .env("TMPDIR", &directory_path);
    }
    if request.axis == "handoff" {
        command.env("GOMAD_WASM_STRESS_JITTER", "1");
    }
    let started = Instant::now();
    let mut child = command.spawn().map_err(|error| error.to_string())?;
    let pid = child.id();
    let result = loop {
        match child.try_wait() {
            Ok(Some(status)) => break Ok(status),
            Ok(None) if started.elapsed() <= request.options.overall_timeout => {
                thread::sleep(std::time::Duration::from_millis(10));
            }
            other => {
                let reason = match other {
                    Err(error) => error.to_string(),
                    _ => "stress subprocess exceeded overall bound".into(),
                };
                let kill = child.kill();
                let wait = child.wait();
                break Err(format!("{reason}; kill={kill:?}; wait={wait:?}"));
            }
        }
    };
    let status = result?;
    if !status.success() {
        return Err(format!(
            "stress child {pid} exited {status}; retain {}",
            request.root.display()
        ));
    }
    let observed: Value = read(&request.root.join("observation.json"))?;
    if observed["process_id"] != pid {
        return Err("stress child process identity mismatch".into());
    }
    Ok(observed)
}
