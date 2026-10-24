use super::{Command, Completed, Environment, Runner, Running};
use std::{
    collections::BTreeMap,
    fs::File,
    io::Read,
    path::Path,
    sync::atomic::AtomicBool,
    time::{Duration, Instant},
};
pub(crate) struct BusyWorkers {
    workers: Vec<Running>,
}
impl BusyWorkers {
    pub fn start(runner: &mut Runner, label: &str, cancelled: &AtomicBool) -> Result<Self, String> {
        let mut busy = Self {
            workers: Vec::new(),
        };
        if !["busy-boundaries", "busy-grouped"].contains(&label) {
            return Ok(busy);
        }
        for index in 0..12 {
            let name = format!("{label}-busy-{index}");
            let command = Command {
                argv: vec![
                    "/bin/sh".into(),
                    "-c".into(),
                    "echo ready; while :; do :; done".into(),
                ],
                cwd: Path::new("/").into(),
                environment: Environment::Exact(BTreeMap::new()),
                accepted: vec![0],
                timeout: Duration::from_secs(3600),
                private_context: None,
            };
            let running = runner
                .start(&name, command, cancelled, |_| Ok(()))
                .map_err(|e| e.to_string())?;
            busy.workers.push(running);
            let deadline = Instant::now() + Duration::from_secs(3);
            loop {
                let mut ready = [0u8; 6];
                let mut stdout = File::open(runner.registry.path.join(format!("{name}.stdout")))
                    .map_err(|e| e.to_string())?;
                if stdout.read(&mut ready).map_err(|e| e.to_string())? == 6 && ready == *b"ready\n"
                {
                    break;
                }
                if cancelled.load(std::sync::atomic::Ordering::Acquire)
                    || Instant::now() >= deadline
                {
                    return Err("busy worker readiness interrupted or timed out".into());
                }
                if busy
                    .workers
                    .last_mut()
                    .ok_or("busy worker missing")?
                    .poll(cancelled)?
                    .is_some()
                {
                    return Err("busy worker exited before workload".into());
                }
                std::thread::park_timeout(Duration::from_millis(5));
            }
        }
        Ok(busy)
    }
    pub fn pids(&self) -> Vec<u32> {
        self.workers.iter().map(Running::pid).collect()
    }
    pub fn finish(mut self) -> Result<Vec<Completed>, String> {
        super::cleanup::terminate_all(std::mem::take(&mut self.workers))
    }
}
