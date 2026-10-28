//! Invocation-local payload admission; mise continues to own task eligibility.
use std::{fmt, io, path::Path, time::Duration};

#[derive(Clone, Debug, Eq, Ord, PartialEq, PartialOrd)]
pub struct StageId(String);

impl StageId {
    pub fn new(value: &str) -> io::Result<Self> {
        let parts: Vec<_> = value.split('/').collect();
        if value.len() > 128
            || parts.len() != 2
            || parts.iter().any(|part| {
                part.is_empty()
                    || !part.bytes().all(|byte| byte.is_ascii_alphanumeric() || byte == b'-')
            })
        {
            return Err(io::Error::other("invalid qualified budget stage"));
        }
        Ok(Self(value.into()))
    }

    pub fn as_str(&self) -> &str {
        &self.0
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FailureCause {
    pub stage: StageId,
    pub code: i32,
    pub reason: String,
}

#[derive(Debug)]
pub enum Error {
    Io(io::Error),
    Aborted(FailureCause),
}

impl Error {
    pub fn original_failure(&self) -> Option<&FailureCause> {
        match self {
            Self::Aborted(cause) => Some(cause),
            Self::Io(_) => None,
        }
    }
}

impl fmt::Display for Error {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(error) => error.fmt(out),
            Self::Aborted(cause) => {
                write!(out, "{}: exit {}: {}", cause.stage.as_str(), cause.code, cause.reason)
            }
        }
    }
}

impl std::error::Error for Error {}

impl From<io::Error> for Error {
    fn from(error: io::Error) -> Self {
        Self::Io(error)
    }
}

#[derive(Clone, Debug)]
pub struct Coordinator;

impl Coordinator {
    pub fn create(
        _directory: &Path,
        _invocation: &str,
        _stages: &[StageId],
        _drain: Duration,
    ) -> io::Result<Self> {
        Ok(Self)
    }

    pub fn open(
        _directory: &Path,
        _invocation: &str,
        _stages: &[StageId],
        _drain: Duration,
    ) -> io::Result<Self> {
        Err(io::Error::other("budget coordinator implementation pending"))
    }

    pub fn admit_shared(&self, _stage: &StageId) -> Result<Permit, Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }

    pub fn admit_exclusive_first(&self, _stage: &StageId) -> Result<Permit, Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }

    pub fn abort(&self, _cause: FailureCause) -> Result<(), Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }
}

#[derive(Debug)]
pub struct Permit;

impl Permit {
    pub fn finish_fast(self) -> Result<(), Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }

    pub fn finish_slow(self) -> Result<RetryToken, Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }

    pub fn abort(self, _cause: FailureCause) -> Result<(), Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }
}

#[derive(Debug)]
pub struct RetryToken;

impl RetryToken {
    pub fn wait(self) -> Result<Permit, Error> {
        Err(io::Error::other("budget coordinator implementation pending").into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{sync::mpsc, thread};

    fn stages() -> io::Result<Vec<StageId>> {
        ["framework/a", "framework/b", "temporal/c", "temporal/standalone"]
            .into_iter()
            .map(StageId::new)
            .collect()
    }

    #[test]
    fn two_slow_payloads_remove_themselves_before_bounded_drain()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = tempfile::tempdir()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(2))?;
        let a = coordinator.admit_shared(&ids[0])?;
        let b = coordinator.admit_shared(&ids[1])?;
        let competitor = coordinator.admit_shared(&ids[2])?;
        let first = a.finish_slow()?;
        let second = b.finish_slow()?;
        let (sender, receiver) = mpsc::channel();
        thread::scope(|scope| -> Result<(), Box<dyn std::error::Error>> {
            let sender2 = sender.clone();
            let a = scope.spawn(move || -> Result<(), Error> {
                let retry = first.wait()?;
                sender.send("a").map_err(io::Error::other)?;
                retry.finish_fast()
            });
            let b = scope.spawn(move || -> Result<(), Error> {
                let retry = second.wait()?;
                sender2.send("b").map_err(io::Error::other)?;
                retry.finish_fast()
            });
            assert!(receiver.recv_timeout(Duration::from_millis(25)).is_err());
            competitor.finish_fast()?;
            assert_eq!(receiver.recv_timeout(Duration::from_secs(2))?, "a");
            assert_eq!(receiver.recv_timeout(Duration::from_secs(2))?, "b");
            a.join().map_err(|_| io::Error::other("first retry panicked"))??;
            b.join().map_err(|_| io::Error::other("second retry panicked"))??;
            Ok(())
        })
    }

    #[test]
    fn queued_exclusive_first_cannot_overtake_requested_retry()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = tempfile::tempdir()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(2))?;
        let active = coordinator.admit_shared(&ids[0])?;
        let (sender, receiver) = mpsc::channel();
        thread::scope(|scope| -> Result<(), Box<dyn std::error::Error>> {
            let queued = scope.spawn(|| -> Result<(), Error> {
                let first = coordinator.admit_exclusive_first(&ids[3])?;
                sender.send("exclusive").map_err(io::Error::other)?;
                first.finish_fast()
            });
            assert!(receiver.recv_timeout(Duration::from_millis(25)).is_err());
            let retry = active.finish_slow()?.wait()?;
            assert!(receiver.recv_timeout(Duration::from_millis(25)).is_err());
            retry.finish_fast()?;
            assert_eq!(receiver.recv_timeout(Duration::from_secs(2))?, "exclusive");
            queued.join().map_err(|_| io::Error::other("exclusive first panicked"))??;
            Ok(())
        })
    }

    #[test]
    fn first_real_failure_and_cancellation_status_survive_other_readers()
    -> Result<(), Box<dyn std::error::Error>> {
        for code in [37, 143] {
            let root = tempfile::tempdir()?;
            let ids = stages()?;
            let coordinator =
                Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(1))?;
            let active = coordinator.admit_shared(&ids[0])?;
            let cause =
                FailureCause { stage: ids[0].clone(), code, reason: "original failure".into() };
            active.abort(cause.clone())?;
            let reader =
                Coordinator::open(root.path(), "invocation", &ids, Duration::from_secs(1))?;
            let failure = reader.admit_shared(&ids[1]).expect_err("aborted invocation reopened");
            assert_eq!(failure.original_failure(), Some(&cause));
        }
        Ok(())
    }

    #[test]
    fn second_slow_and_incomplete_permit_abort_without_reopening()
    -> Result<(), Box<dyn std::error::Error>> {
        for second_slow in [true, false] {
            let root = tempfile::tempdir()?;
            let ids = stages()?;
            let coordinator =
                Coordinator::create(root.path(), "invocation", &ids, Duration::from_secs(1))?;
            let active = coordinator.admit_shared(&ids[0])?;
            if second_slow {
                let retry = active.finish_slow()?.wait()?;
                assert!(retry.finish_slow().is_err(), "second retry was manufactured");
            } else {
                drop(active);
            }
            assert!(coordinator.admit_shared(&ids[1]).is_err());
        }
        Ok(())
    }

    #[test]
    fn duplicate_stage_scope_mismatch_and_partial_state_fail_closed()
    -> Result<(), Box<dyn std::error::Error>> {
        let root = tempfile::tempdir()?;
        let ids = stages()?;
        let coordinator =
            Coordinator::create(root.path(), "invocation", &ids, Duration::from_millis(50))?;
        let active = coordinator.admit_shared(&ids[0])?;
        assert!(coordinator.admit_shared(&ids[0]).is_err());
        assert!(
            Coordinator::open(root.path(), "crossed", &ids, Duration::from_millis(50)).is_err()
        );
        active.finish_fast()?;
        std::fs::write(root.path().join("state.json"), b"{\"partial\":")?;
        assert!(
            Coordinator::open(root.path(), "invocation", &ids, Duration::from_millis(50)).is_err()
        );
        Ok(())
    }

    #[test]
    fn unrelated_invocation_admission_remains_independent() -> Result<(), Box<dyn std::error::Error>>
    {
        let first = tempfile::tempdir()?;
        let second = tempfile::tempdir()?;
        let ids = stages()?;
        let a = Coordinator::create(first.path(), "first", &ids, Duration::from_secs(1))?;
        let b = Coordinator::create(second.path(), "second", &ids, Duration::from_secs(1))?;
        let retry = a.admit_shared(&ids[0])?.finish_slow()?;
        b.admit_shared(&ids[0])?.finish_fast()?;
        retry.wait()?.finish_fast()?;
        Ok(())
    }
}
