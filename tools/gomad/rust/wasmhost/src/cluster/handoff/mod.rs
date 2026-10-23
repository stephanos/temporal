//! Owns continuation acquisition, idle transfer and incarnation finish.
mod baton;
mod broker;

use baton::{Baton, InOrder};
pub(crate) use broker::{Broker, Clockwork, Command, Member, Metrics};
pub use broker::{Config, WorldLimits};
use std::sync::Arc;

/// The cluster's continuation owner: the fn-178.6 baton without a broker, the
/// in-process broker (fn-178.7) with one.
#[derive(Clone)]
pub(super) enum Coordinator {
    Baton(Arc<Baton>),
    Broker(Arc<Broker>),
}

impl Coordinator {
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    pub(super) fn set_event_owners(&self, owners: Vec<crate::event_order::Context>) {
        if let Self::Baton(baton) = self {
            baton.set_event_owners(owners);
        }
    }
    pub(super) fn in_order(instances: usize) -> Self {
        Self::Baton(Arc::new(Baton::new(instances, Box::new(InOrder))))
    }

    pub(super) fn start(&self) {
        match self {
            Self::Baton(baton) => baton.start(),
            Self::Broker(broker) => broker.start(),
        }
    }

    pub(super) fn acquire(&self, me: usize) -> bool {
        match self {
            Self::Baton(baton) => baton.acquire(me),
            Self::Broker(broker) => broker.acquire(me),
        }
    }

    pub(super) fn inspection_stopped(&self) -> bool {
        match self {
            Self::Baton(_) => false,
            Self::Broker(broker) => broker.inspection_stopped(),
        }
    }

    pub(super) fn driven(&self) -> bool {
        match self {
            Self::Baton(_) => false,
            Self::Broker(broker) => broker.manual(),
        }
    }

    pub(super) fn finish(&self, me: usize) {
        match self {
            Self::Baton(baton) => baton.finish(me),
            Self::Broker(broker) => broker.finish(me),
        }
    }

    pub(super) fn abort(&self, termination: &'static str, message: String) {
        match self {
            Self::Baton(baton) => baton.abort(),
            Self::Broker(broker) => broker.abort(termination, message),
        }
    }

    pub(super) fn wait(&self) -> bool {
        match self {
            Self::Baton(baton) => baton.wait(),
            Self::Broker(broker) => broker.wait(),
        }
    }

    /// A whole-run failure the coordinator itself raised (a broker stall).
    pub(super) fn failure(&self) -> Option<(&'static str, String)> {
        match self {
            Self::Baton(_) => None,
            Self::Broker(broker) => broker.failure(),
        }
    }
}
