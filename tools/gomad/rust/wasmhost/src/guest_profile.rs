use serde::Deserialize;
use std::{
    fs::OpenOptions,
    sync::{Arc, Mutex, mpsc},
    thread::JoinHandle,
    time::{Duration, Instant},
};
use wasmtime::{CallHook, GuestProfiler, Module, Result, Store, UpdateDeadline};

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    pub path: String,
    pub interval_micros: u64,
    pub max_samples: u64,
}
impl Config {
    /// # Errors
    /// Refuses relative output paths and sampling bounds outside the supported range.
    pub fn validate(&self) -> Result<()> {
        if !std::path::Path::new(&self.path).is_absolute()
            || !(1..=1_000_000).contains(&self.interval_micros)
            || !(1..=100_000).contains(&self.max_samples)
        {
            return Err(wasmtime::Error::msg("invalid guest profile path or bounds"));
        }
        Ok(())
    }
}

struct Sampling {
    profiler: GuestProfiler,
    samples: u64,
    markers: u64,
    truncated: bool,
    markers_truncated: bool,
    previous: Instant,
}

pub struct Session {
    config: Config,
    sampling: Arc<Mutex<Option<Sampling>>>,
    stop: mpsc::Sender<()>,
    ticker: Option<JoinHandle<()>>,
}
impl Session {
    pub fn start<T: Send + 'static>(
        store: &mut Store<T>,
        module: &Module,
        config: &Config,
    ) -> Result<Self> {
        Self::start_interruptible(store, module, config, || Ok(()))
    }

    pub fn start_interruptible<T: Send + 'static>(
        store: &mut Store<T>,
        module: &Module,
        config: &Config,
        interrupt: impl Fn() -> Result<()> + Send + Sync + 'static,
    ) -> Result<Self> {
        config.validate()?;
        let interval = Duration::from_micros(config.interval_micros);
        let profiler = GuestProfiler::new(
            store.engine(),
            "gomad guest",
            interval,
            [("guest".into(), module.clone())],
        )?;
        let sampling = Arc::new(Mutex::new(Some(Sampling {
            profiler,
            samples: 0,
            markers: 0,
            truncated: false,
            markers_truncated: false,
            previous: Instant::now(),
        })));
        let samples = Arc::clone(&sampling);
        let max_samples = config.max_samples;
        store.set_epoch_deadline(1);
        store.epoch_deadline_callback(move |store| {
            interrupt()?;
            let mut guard = samples.lock().unwrap();
            let sample = guard.as_mut().unwrap();
            if sample.samples < max_samples {
                let now = Instant::now();
                sample
                    .profiler
                    .sample(&store, now.duration_since(sample.previous));
                sample.previous = now;
                sample.samples += 1;
                sample.truncated = sample.samples == max_samples;
            }
            Ok(UpdateDeadline::Continue(1))
        });
        let samples = Arc::clone(&sampling);
        store.call_hook(move |mut store, kind| {
            let mut guard = samples.lock().unwrap();
            let sample = guard.as_mut().unwrap();
            if sample.markers < max_samples * 2 {
                sample.profiler.call_hook(&store, kind);
                if matches!(kind, CallHook::CallingHost | CallHook::ReturningFromHost) {
                    sample.markers += 1;
                }
            } else {
                sample.markers_truncated = true;
            }
            if matches!(kind, CallHook::CallingWasm | CallHook::ReturningFromHost) {
                // An epoch reached inside host work must not sample the frame
                // that happens to resume afterwards, or charge it host time.
                store.set_epoch_deadline(1);
                sample.previous = Instant::now();
            }
            Ok(())
        });
        let (stop, stopped) = mpsc::channel();
        let engine = store.engine().clone();
        let samples = Arc::clone(&sampling);
        let ticker = std::thread::Builder::new()
            .name("gomad-guest-profile".into())
            .spawn(move || {
                while matches!(
                    stopped.recv_timeout(interval),
                    Err(mpsc::RecvTimeoutError::Timeout)
                ) {
                    if samples.lock().unwrap().as_ref().unwrap().truncated {
                        break;
                    }
                    engine.increment_epoch();
                }
            })?;
        Ok(Self {
            config: config.clone(),
            sampling,
            stop,
            ticker: Some(ticker),
        })
    }

    fn stop(&mut self) {
        let _ = self.stop.send(());
        if let Some(ticker) = self.ticker.take() {
            // The ticker touches no guest state; a panic must not replace the outcome.
            let _ = ticker.join();
        }
    }

    pub fn finish(mut self) -> Result<()> {
        use std::io::Write;
        self.stop();
        let sampling = self.sampling.lock().unwrap().take().unwrap();
        let mut bytes = Vec::new();
        sampling.profiler.finish(&mut bytes)?;
        let mut profile: serde_json::Value = serde_json::from_slice(&bytes)?;
        profile["meta"]["gomadGuestProfile"] = serde_json::json!({
            "intervalMicros": self.config.interval_micros,
            "maxSamples": self.config.max_samples,
            "samples": sampling.samples,
            "truncated": sampling.truncated,
            "markersTruncated": sampling.markers_truncated,
        });
        let mut options = OpenOptions::new();
        options.write(true).create_new(true);
        #[cfg(unix)]
        {
            use std::os::unix::fs::OpenOptionsExt;
            options.mode(0o600);
        }
        let mut file = options.open(&self.config.path)?;
        serde_json::to_writer(&mut file, &profile)?;
        file.flush()?;
        Ok(())
    }
}
impl Drop for Session {
    fn drop(&mut self) {
        self.stop();
    }
}

#[cfg(test)]
#[path = "guest_profile/preservation.rs"]
mod preservation;
