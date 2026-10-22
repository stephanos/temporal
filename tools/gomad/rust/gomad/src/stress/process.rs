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
