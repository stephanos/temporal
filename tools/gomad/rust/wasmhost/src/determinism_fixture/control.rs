use super::Kind;
use crate::model::inspection::Handle;
use std::{
    cell::RefCell,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
};

pub(crate) struct Control {
    pub(super) kind: Kind,
    owner: Option<Handle>,
    output_committed: AtomicBool,
}

impl Control {
    pub(crate) fn new(kind: Kind, owner: Option<Handle>) -> Arc<Self> {
        Arc::new(Self {
            kind,
            owner,
            output_committed: AtomicBool::new(false),
        })
    }

    pub(super) fn matches(&self, owner: &Handle) -> bool {
        self.owner.as_ref().is_none_or(|expected| expected == owner)
    }

    pub(super) fn output_committed(&self) -> bool {
        self.output_committed.load(Ordering::Relaxed)
    }
}

thread_local! {
    static CURRENT: RefCell<Option<Arc<Control>>> = const { RefCell::new(None) };
}

pub(crate) fn attach(control: Option<Arc<Control>>) {
    CURRENT.with(|current| *current.borrow_mut() = control);
}

pub(super) fn current() -> Option<Arc<Control>> {
    CURRENT.with(|current| current.borrow().clone())
}

pub(super) fn commit_output() {
    if let Some(control) = current()
        && crate::event_order::current_owner()
            .as_ref()
            .is_some_and(|owner| control.matches(owner))
    {
        control.output_committed.store(true, Ordering::Relaxed);
    }
}

#[cfg(all(
    test,
    not(any(feature = "diagnostic-profile", feature = "stress-jitter"))
))]
#[path = "control_tests.rs"]
mod tests;
