use crate::{cluster::handoff::Broker, linked::Execution, protocol::Outcome};
use std::{
    collections::BTreeMap,
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
    },
};
use wasmtime::{Engine, Error};

#[derive(Clone, Default)]
pub struct Cancellation(Arc<State>);

#[derive(Default)]
struct State {
    cancelled: AtomicBool,
    execution: Mutex<Option<(Engine, Option<Arc<Broker>>)>>,
}

impl Cancellation {
    pub fn cancel(&self) {
        self.0.cancelled.store(true, Ordering::Release);
        let execution = self
            .0
            .execution
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        if let Some((engine, broker)) = &*execution {
            engine.increment_epoch();
            if let Some(broker) = broker {
                broker.abort("cancelled", "seed execution was cancelled".into());
            }
        }
    }

    #[must_use]
    pub fn cancelled(&self) -> bool {
        self.0.cancelled.load(Ordering::Acquire)
    }

    pub(crate) fn attach(
        &self,
        engine: &Engine,
        broker: Option<Arc<Broker>>,
    ) -> Result<Registration, Error> {
        let mut execution = self
            .0
            .execution
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        if execution.is_some() {
            return Err(Error::msg(
                "cancellation handle already bound to an execution",
            ));
        }
        *execution = Some((engine.clone(), broker));
        Ok(Registration(self.clone()))
    }

    pub(crate) fn apply(&self, outcome: &mut Outcome) {
        if self.cancelled() {
            *outcome = Outcome::new("cancelled", "seed execution was cancelled");
        }
    }

    pub(crate) fn execution() -> Execution {
        Execution {
            terminal_monotonic_nanos: None,
            admission_terminal_to_return_nanos: None,
            timing: None,
            #[cfg(feature = "guest-profile")]
            guest_profile_error: None,
            #[cfg(feature = "diagnostic-profile")]
            external_owner_attempts: 0,
            authorization_to_terminal_nanos: None,
            outcome: Outcome::new("cancelled", "seed execution was cancelled"),
            evidence: BTreeMap::new(),
            event_order: None,
        }
    }
}

pub(crate) struct Registration(Cancellation);

impl Drop for Registration {
    fn drop(&mut self) {
        *self
            .0
            .0
            .execution
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner) = None;
    }
}

#[cfg(test)]
mod tests {
    use super::Cancellation;
    use std::{sync::mpsc, thread};
    use wasmtime::{Config, Engine, Instance, Module, Store};

    // The CLI cannot synchronize a cancellation exactly after epoch registration or inspect its release.
    #[test]
    fn cancellation_interrupts_only_its_registered_engine_and_releases_it() {
        let mut configuration = Config::new();
        configuration.epoch_interruption(true);
        let engine = Engine::new(&configuration).unwrap();
        let cancellation = Cancellation::default();
        let registration = cancellation.attach(&engine, None).unwrap();
        assert!(cancellation.attach(&engine, None).is_err());
        thread::scope(|scope| {
            let (ready, started) = mpsc::sync_channel(0);
            let engine = &engine;
            let running = scope.spawn(move || {
                let module =
                    Module::new(engine, "(module (func (export \"run\") (loop br 0)))").unwrap();
                let mut store = Store::new(engine, ());
                store.set_epoch_deadline(1);
                let instance = Instance::new(&mut store, &module, &[]).unwrap();
                let run = instance
                    .get_typed_func::<(), ()>(&mut store, "run")
                    .unwrap();
                ready.send(()).unwrap();
                run.call(&mut store, ())
            });
            started.recv().unwrap();
            cancellation.clone().cancel();
            assert!(running.join().unwrap().is_err());
        });
        assert!(cancellation.cancelled());
        assert!(!Cancellation::default().cancelled());
        drop(registration);
        assert!(cancellation.attach(&engine, None).is_ok());
    }
}
