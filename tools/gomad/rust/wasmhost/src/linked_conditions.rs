use std::cell::Cell;
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};
use wasmtime::{Error, Result};

#[derive(Clone, Copy, Default, PartialEq, Eq)]
pub enum Conditions {
    #[default]
    Ordinary,
    DiagnosticProfile,
    GuestProfile,
    DiagnosticFuelOff,
    GuestProfileFuelOff,
    StressReplay,
    DeterminismFixtureReplay,
}

thread_local! {
    static CURRENT: Cell<Option<Conditions>> = const { Cell::new(None) };
}

impl Conditions {
    #[must_use]
    pub fn compiled() -> Self {
        if cfg!(feature = "guest-profile") && cfg!(feature = "diagnostic-fuel-off") {
            Self::GuestProfileFuelOff
        } else if cfg!(feature = "guest-profile") {
            Self::GuestProfile
        } else if cfg!(feature = "diagnostic-fuel-off") {
            Self::DiagnosticFuelOff
        } else if cfg!(feature = "diagnostic-profile") {
            Self::DiagnosticProfile
        } else if cfg!(feature = "stress-jitter") {
            Self::StressReplay
        } else if cfg!(feature = "determinism-fixture") {
            Self::DeterminismFixtureReplay
        } else {
            Self::Ordinary
        }
    }

    #[must_use]
    pub fn stress_replay() -> Self {
        Self::StressReplay
    }

    #[must_use]
    pub fn determinism_fixture_replay() -> Self {
        Self::DeterminismFixtureReplay
    }

    #[must_use]
    pub fn configuration(self) -> String {
        let suffix = match self {
            Self::Ordinary => "",
            Self::DiagnosticProfile => "-diagnostic-profile",
            Self::GuestProfile => "-guest-profile",
            Self::DiagnosticFuelOff => "-diagnostic-fuel-off",
            Self::GuestProfileFuelOff => "-guest-profile-fuel-off",
            Self::StressReplay => "-stress-jitter",
            Self::DeterminismFixtureReplay => "-determinism-fixture",
        };
        format!("{}{suffix}", crate::linked::CONFIGURATION)
    }

    /// # Errors
    /// Refuses a diagnostic role absent from this identity-bound linked implementation.
    pub fn validate(self) -> Result<()> {
        if self != Self::compiled() {
            return Err(Error::msg(
                "WASI invalid: engine/module/import identity mismatch before authorization",
            ));
        }
        Ok(())
    }

    #[cfg(any(test, not(feature = "determinism-fixture")))]
    pub(crate) fn enter(self) -> Result<Scope> {
        self.attach(&self.new_session(), true)
    }

    fn new_session(self) -> Session {
        Session {
            conditions: self,
            #[cfg(feature = "diagnostic-profile")]
            profile: crate::diagnostic::LinkedSink::current(),
            closed: Arc::new(AtomicBool::new(false)),
            #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
            events: crate::event_order::LinkedSession::new(),
            #[cfg(feature = "determinism-fixture")]
            fixture: None,
        }
    }

    #[cfg(feature = "determinism-fixture")]
    pub(crate) fn enter_with_fault(
        self,
        kind: Option<crate::determinism_fixture::Kind>,
        owner: Option<crate::model::inspection::Handle>,
    ) -> Result<Scope> {
        if kind.is_some() && self != Self::DeterminismFixtureReplay {
            return Err(Error::msg(
                "fixture fault requires the compiled determinism fixture role",
            ));
        }
        let mut session = self.new_session();
        session.fixture =
            kind.map(|kind| crate::determinism_fixture::control::Control::new(kind, owner));
        self.attach(&session, true)
    }

    pub(crate) fn enter_shared(self, session: &Session) -> Result<Scope> {
        self.attach(session, false)
    }

    fn attach(self, session: &Session, owner: bool) -> Result<Scope> {
        self.validate()?;
        if self != session.conditions || session.closed.load(Ordering::Acquire) {
            return Err(Error::msg(
                "incompatible or closed linked execution session",
            ));
        }
        if current().is_some() {
            return Err(Error::msg("overlapping linked execution conditions"));
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        crate::event_order::start_linked(&session.events).map_err(Error::msg)?;
        CURRENT.with(|current| current.set(Some(self)));
        #[cfg(feature = "determinism-fixture")]
        crate::determinism_fixture::control::attach(session.fixture.clone());
        Ok(Scope {
            finished: false,
            owner,
            session: session.clone(),
            _thread: std::marker::PhantomData,
        })
    }
}

#[derive(Clone)]
pub(crate) struct Session {
    #[cfg(feature = "diagnostic-profile")]
    profile: Option<crate::diagnostic::LinkedSink>,
    conditions: Conditions,
    closed: Arc<AtomicBool>,
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    events: crate::event_order::LinkedSession,
    #[cfg(feature = "determinism-fixture")]
    fixture: Option<Arc<crate::determinism_fixture::control::Control>>,
}

#[cfg(feature = "diagnostic-profile")]
impl Session {
    pub(crate) fn profile(
        &self,
        slot: u64,
        incarnation: u64,
    ) -> Result<Option<crate::diagnostic::LinkedThread>> {
        self.profile
            .as_ref()
            .map(|sink| sink.enter(slot, incarnation).map_err(Error::msg))
            .transpose()
    }
}

pub(crate) struct Scope {
    finished: bool,
    owner: bool,
    session: Session,
    _thread: std::marker::PhantomData<std::rc::Rc<()>>,
}

impl Scope {
    pub(crate) fn session(&self) -> Session {
        self.session.clone()
    }

    pub(crate) fn finish(&mut self) -> Option<Vec<u8>> {
        if self.finished || !self.owner {
            return None;
        }
        self.finished = true;
        self.session.closed.store(true, Ordering::Release);
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        {
            crate::event_order::finish_linked(true)
        }
        #[cfg(not(any(feature = "determinism-fixture", feature = "stress-jitter")))]
        None
    }
}

impl Drop for Scope {
    fn drop(&mut self) {
        if self.owner {
            self.session.closed.store(true, Ordering::Release);
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        if !self.finished {
            if self.owner {
                crate::event_order::finish_linked(false);
            } else {
                crate::event_order::detach_linked();
            }
        }
        CURRENT.with(|current| current.set(None));
        #[cfg(feature = "determinism-fixture")]
        crate::determinism_fixture::control::attach(None);
    }
}

pub(crate) fn current() -> Option<Conditions> {
    CURRENT.with(Cell::get)
}

#[cfg(test)]
#[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
mod tests {
    use super::*;
    use crate::event_order;
    use serde_json::{Value, json};
    use std::sync::Barrier;

    fn context(instance: &str) -> event_order::Context {
        event_order::Context {
            instance: instance.into(),
            incarnation: 1,
            trace: 0,
            decision: 0,
            time_nanos: 1,
        }
    }

    fn rows(bytes: &[u8]) -> Vec<Value> {
        std::str::from_utf8(bytes)
            .unwrap()
            .lines()
            .map(|line| {
                serde_json::from_str(line.strip_prefix("gomad-wasm-event-order-v1 ").unwrap())
                    .unwrap()
            })
            .collect()
    }

    // The session token and thread attachment lifetime are internal capabilities unavailable to CLI fixtures.
    #[test]
    fn shared_workers_commit_to_one_session_and_closed_tokens_refuse() {
        let conditions = Conditions::compiled();
        let mut owner = conditions.enter().unwrap();
        let session = owner.session();
        let barrier = Barrier::new(2);
        std::thread::scope(|scope| {
            let mut workers = Vec::new();
            for name in ["left", "right"] {
                let session = session.clone();
                let barrier = &barrier;
                workers.push(scope.spawn(move || {
                    let _child = conditions.enter_shared(&session).unwrap();
                    assert!(conditions.enter_shared(&session).is_err());
                    let context = context(name);
                    let _context = event_order::enter(context.clone());
                    barrier.wait();
                    event_order::emit(&context, "fixture", 0, json!(name));
                }));
            }
            for worker in workers {
                worker.join().unwrap();
            }
        });
        let output = rows(&owner.finish().unwrap());
        drop(owner);
        assert_eq!(output.first().unwrap()["type"], "header");
        assert_eq!(output.last().unwrap()["type"], "complete");
        let events: Vec<_> = output.iter().filter(|row| row["type"] == "event").collect();
        assert_eq!(events.len(), 2);
        assert_eq!(events[0]["order"], 1);
        assert_eq!(events[1]["order"], 2);
        assert_ne!(events[0]["instance"], events[1]["instance"]);
        assert_eq!(output.last().unwrap()["events"], 2);
        assert!(conditions.enter_shared(&session).is_err());
        assert!(current().is_none());
    }

    // Independent in-process sessions can overlap below the CLI's artifact/report boundary.
    #[test]
    fn overlapping_execution_sessions_never_share_recorders() {
        let barrier = Barrier::new(2);
        std::thread::scope(|scope| {
            let mut workers = Vec::new();
            for name in ["first", "second"] {
                let barrier = &barrier;
                workers.push(scope.spawn(move || {
                    let mut owner = Conditions::compiled().enter().unwrap();
                    let context = context(name);
                    let _context = event_order::enter(context.clone());
                    barrier.wait();
                    event_order::emit(&context, "fixture", 0, json!(name));
                    let output = rows(&owner.finish().unwrap());
                    let events: Vec<_> =
                        output.iter().filter(|row| row["type"] == "event").collect();
                    assert_eq!(events.len(), 1);
                    assert_eq!(events[0]["value"], name);
                    assert_eq!(output.last().unwrap()["type"], "complete");
                }));
            }
            for worker in workers {
                worker.join().unwrap();
            }
        });
    }

    // CLI fixtures cannot synthesize these exact internal diagnostic event counts or payloads.
    #[test]
    fn shared_session_event_and_byte_exhaustion_never_complete() {
        for byte_bound in [false, true] {
            let mut owner = Conditions::compiled().enter().unwrap();
            let context = context("bounded");
            let _context = event_order::enter(context.clone());
            if byte_bound {
                event_order::emit(&context, "fixture", 0, json!("x".repeat(32 * 1024 * 1024)));
            } else {
                for ordinal in 0..=131_072 {
                    event_order::emit(&context, "fixture", ordinal, json!(null));
                }
            }
            let output = rows(&owner.finish().unwrap());
            assert!(output.iter().any(|row| row["type"] == "incomplete"));
            assert!(!output.iter().any(|row| row["type"] == "complete"));
        }
    }
}

#[cfg(all(
    test,
    not(any(feature = "stress-jitter", feature = "determinism-fixture"))
))]
mod role_tests {
    use super::Conditions;

    #[cfg(feature = "diagnostic-profile")]
    #[test]
    fn diagnostic_helper_cannot_authorize_an_ordinary_role() {
        assert!(Conditions::compiled() != Conditions::Ordinary);
        assert!(Conditions::Ordinary.validate().is_err());
        assert!(Conditions::Ordinary.enter().is_err());
        assert!(super::current().is_none());
        assert!(Conditions::compiled().validate().is_ok());
        let expected = if cfg!(feature = "guest-profile") && cfg!(feature = "diagnostic-fuel-off") {
            "-guest-profile-fuel-off"
        } else if cfg!(feature = "guest-profile") {
            "-guest-profile"
        } else if cfg!(feature = "diagnostic-fuel-off") {
            "-diagnostic-fuel-off"
        } else {
            "-diagnostic-profile"
        };
        assert!(Conditions::compiled().configuration().ends_with(expected));
        for role in [
            Conditions::DiagnosticProfile,
            Conditions::GuestProfile,
            Conditions::DiagnosticFuelOff,
            Conditions::GuestProfileFuelOff,
        ] {
            assert_eq!(role.validate().is_ok(), role == Conditions::compiled());
        }
    }

    #[cfg(not(any(
        feature = "diagnostic-profile",
        feature = "stress-jitter",
        feature = "determinism-fixture"
    )))]
    #[test]
    fn empty_feature_helper_preserves_ordinary_role() {
        assert!(Conditions::compiled() == Conditions::Ordinary);
        assert!(Conditions::Ordinary.validate().is_ok());
        assert_eq!(
            Conditions::compiled().configuration(),
            crate::linked::CONFIGURATION
        );
    }
}

#[cfg(all(test, any(feature = "stress-jitter", feature = "determinism-fixture")))]
mod nonordinary_role_tests {
    use super::Conditions;

    #[test]
    fn existing_nonordinary_roles_cannot_authorize_ordinary() {
        assert!(Conditions::compiled() != Conditions::Ordinary);
        assert!(Conditions::Ordinary.validate().is_err());
        assert!(Conditions::compiled().validate().is_ok());
        let expected = if cfg!(feature = "stress-jitter") {
            "-stress-jitter"
        } else {
            "-determinism-fixture"
        };
        assert!(Conditions::compiled().configuration().ends_with(expected));
    }
}
