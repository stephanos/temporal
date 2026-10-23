//! The cluster baton (fn-178.1 design record items 1-2): every store runs on
//! its own OS thread, and only the thread holding the baton executes, so
//! instances are serialized by construction. A `Scheduler` picks the next
//! holder; swapping the policy (fn-178.7's broker order) or the mechanism
//! (fn-178.10's fiber revisit) stays behind this seam.
use crate::cluster::transport::lock;
use std::sync::{Condvar, Mutex};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Status {
    Runnable,
    Active,
    Finished,
}

pub trait Scheduler: Send {
    /// The next instance to hold the baton, or `None` when none can run.
    fn next(&mut self, status: &[Status]) -> Option<usize>;
}

/// fn-178.6 policy: run each instance to its terminal, in instance order.
pub struct InOrder;

impl Scheduler for InOrder {
    fn next(&mut self, status: &[Status]) -> Option<usize> {
        status.iter().position(|status| *status == Status::Runnable)
    }
}

struct State {
    status: Vec<Status>,
    active: Option<usize>,
    scheduler: Box<dyn Scheduler>,
    aborted: bool,
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    event_owners: Vec<crate::event_order::Context>,
}

pub struct Baton {
    state: Mutex<State>,
    wake: Vec<Condvar>,
    idle: Condvar,
}

impl Baton {
    pub fn new(instances: usize, scheduler: Box<dyn Scheduler>) -> Self {
        Self {
            state: Mutex::new(State {
                status: vec![Status::Runnable; instances],
                active: None,
                scheduler,
                aborted: false,
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                event_owners: Vec::new(),
            }),
            wake: (0..instances).map(|_| Condvar::new()).collect(),
            idle: Condvar::new(),
        }
    }

    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    pub fn set_event_owners(&self, owners: Vec<crate::event_order::Context>) {
        lock(&self.state).event_owners = owners;
    }

    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    fn event(state: &State, me: usize, action: &str) {
        if let Some(owner) = state.event_owners.get(me) {
            crate::event_order::emit(
                &crate::event_order::latest(owner),
                "world-order",
                0,
                serde_json::json!({"model":"baton","action":action}),
            );
        }
    }

    fn pass(&self, state: &mut State) {
        state.active = state.scheduler.next(&state.status);
        match state.active {
            Some(next) => {
                state.status[next] = Status::Active;
                #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
                Self::event(state, next, "grant");
                #[cfg(feature = "stress-jitter")]
                crate::stress_jitter::delay();
                self.wake[next].notify_one();
            }
            None => self.idle.notify_all(),
        }
    }

    /// Hands the baton to the first instance.
    pub fn start(&self) {
        let mut state = lock(&self.state);
        if state.active.is_none() && !state.aborted {
            self.pass(&mut state);
        }
    }

    /// Blocks until `me` holds the baton; false when the run was aborted.
    pub fn acquire(&self, me: usize) -> bool {
        let mut state = lock(&self.state);
        while state.active != Some(me) && !state.aborted {
            state = self.wake[me]
                .wait(state)
                .unwrap_or_else(std::sync::PoisonError::into_inner);
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        if !state.aborted {
            Self::event(&state, me, "acquire");
        }
        !state.aborted
    }

    /// `me` reached its terminal; the baton passes on.
    pub fn finish(&self, me: usize) {
        let mut state = lock(&self.state);
        state.status[me] = Status::Finished;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        Self::event(&state, me, "finish");
        if state.active == Some(me) {
            self.pass(&mut state);
        }
    }

    /// Ends the run: no instance acquires the baton again.
    pub fn abort(&self) {
        let mut state = lock(&self.state);
        state.aborted = true;
        state.active = None;
        for wake in &self.wake {
            wake.notify_all();
        }
        self.idle.notify_all();
    }

    /// Blocks the coordinator until no instance holds the baton; true when
    /// every instance finished, false when the run was aborted.
    pub fn wait(&self) -> bool {
        let mut state = lock(&self.state);
        while state.active.is_some() && !state.aborted {
            state = self
                .idle
                .wait(state)
                .unwrap_or_else(std::sync::PoisonError::into_inner);
        }
        !state.aborted
            && state
                .status
                .iter()
                .all(|status| *status == Status::Finished)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Arc;

    #[test]
    fn cluster_baton_serializes_in_instance_order() {
        let baton = Arc::new(Baton::new(4, Box::new(InOrder)));
        let order = Arc::new(Mutex::new(Vec::new()));
        // Spawn in reverse so thread start order cannot produce the result.
        let threads: Vec<_> = (0..4)
            .rev()
            .map(|me| {
                let (baton, order) = (baton.clone(), order.clone());
                std::thread::spawn(move || {
                    assert!(baton.acquire(me));
                    lock(&order).push(me);
                    baton.finish(me);
                })
            })
            .collect();
        baton.start();
        assert!(baton.wait());
        for thread in threads {
            thread.join().unwrap();
        }
        assert_eq!(*lock(&order), vec![0, 1, 2, 3]);
    }

    #[test]
    fn cluster_baton_abort_releases_waiters() {
        let baton = Arc::new(Baton::new(2, Box::new(InOrder)));
        let waiter = {
            let baton = baton.clone();
            std::thread::spawn(move || baton.acquire(1))
        };
        baton.start();
        assert!(baton.acquire(0));
        baton.abort();
        assert!(!waiter.join().unwrap());
        assert!(!baton.wait());
    }
}
