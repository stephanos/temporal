use super::{error, sys};
use std::{
    collections::BTreeSet,
    path::PathBuf,
    sync::{
        Arc, Mutex, MutexGuard,
        atomic::{AtomicBool, Ordering},
    },
    time::{Duration, Instant},
};

static CLAIMED: AtomicBool = AtomicBool::new(false);

/// Claims this dedicated process and every future child until process exit.
/// Call once before creating threads or children; all cycle Runners share it.
/// The caller must not run unrelated child-producing work in this process.
pub(crate) struct CycleOwner {
    pub(super) domain: Arc<Domain>,
}
pub(super) struct Domain {
    pid: u32,
    state: Mutex<State>,
}
pub(super) struct State {
    aborted: Option<String>,
    pub supervisors: BTreeSet<i32>,
}
impl State {
    pub fn check(&self) -> Result<(), String> {
        self.aborted
            .as_ref()
            .map_or(Ok(()), |reason| Err(reason.clone()))
    }
}
impl CycleOwner {
    pub fn initialize_exclusive() -> Result<Self, String> {
        let pid = std::process::id();
        let threads = std::fs::read_dir(format!("/proc/{pid}/task"))
            .map_err(error)?
            .collect::<Result<Vec<_>, _>>()
            .map_err(error)?
            .len();
        if threads != 1 || !children(pid)?.is_empty() || sys::subreaper()? {
            return Err(
                "cycle owner requires one thread, no children, and no prior subreaper".into(),
            );
        }
        if CLAIMED
            .compare_exchange(false, true, Ordering::AcqRel, Ordering::Acquire)
            .is_err()
        {
            return Err("cycle process ownership already claimed".into());
        }
        sys::claim_subreaper()?;
        Ok(Self {
            domain: Arc::new(Domain {
                pid,
                state: Mutex::new(State {
                    aborted: None,
                    supervisors: BTreeSet::new(),
                }),
            }),
        })
    }
}
impl Drop for CycleOwner {
    fn drop(&mut self) {
        if let Ok(mut state) = self.domain.lock() {
            self.domain.abort(&mut state, "cycle owner released");
        }
    }
}
impl Domain {
    pub fn lock(&self) -> Result<MutexGuard<'_, State>, String> {
        if self.pid != std::process::id() || !sys::subreaper()? {
            return Err("cycle owner identity or subreaper role changed".into());
        }
        self.state
            .lock()
            .map_err(|_| "cycle process ownership poisoned".into())
    }
    /// A fatal observation loses per-command attribution. The exclusive cycle
    /// owner instead drains every child, including newly adopted escaped sessions.
    pub fn abort(&self, state: &mut State, reason: &str) -> String {
        let primary = state
            .aborted
            .get_or_insert_with(|| format!("cycle process domain aborted: {reason}"))
            .clone();
        let deadline = Instant::now() + Duration::from_secs(5);
        loop {
            for &pid in &state.supervisors {
                sys::kill_owned(pid);
            }
            let observed = children(self.pid);
            if let Ok(children) = &observed {
                for &pid in children {
                    sys::kill_owned(pid);
                }
            }
            match sys::reap_domain_child() {
                Ok(Some(pid)) => {
                    state.supervisors.remove(&pid);
                }
                Ok(None) => {}
                Err(10) => {
                    if observed.as_ref().is_ok_and(BTreeSet::is_empty) {
                        state.supervisors.clear();
                        return primary;
                    }
                }
                Err(4) => continue,
                Err(code) => return format!("{primary}; domain wait4 failed errno {code}"),
            }
            if Instant::now() >= deadline {
                return format!("{primary}; whole-domain cleanup inconclusive: {observed:?}");
            }
            std::thread::park_timeout(Duration::from_millis(2));
        }
    }
}
fn children(owner: u32) -> Result<BTreeSet<i32>, String> {
    let mut children = BTreeSet::new();
    // Children belong to the spawning thread; enumerate every thread rather
    // than assuming the caller is the one that forked each supervisor.
    for task in std::fs::read_dir(format!("/proc/{owner}/task")).map_err(error)? {
        let task = task.map_err(error)?;
        let path: PathBuf = task.path().join("children");
        let text = match std::fs::read_to_string(path) {
            Ok(text) => text,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => continue,
            Err(failure) => return Err(error(failure)),
        };
        for field in text.split_ascii_whitespace() {
            let pid: i32 = field.parse().map_err(error)?;
            if pid <= 0 || !children.insert(pid) || children.len() > 65536 {
                return Err("ambiguous cycle child ownership snapshot".into());
            }
        }
    }
    Ok(children)
}
