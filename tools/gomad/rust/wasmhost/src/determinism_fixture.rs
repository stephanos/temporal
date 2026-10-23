pub(crate) mod control;

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Kind {
    None,
    Draw,
    Clock,
    Handoff,
    OutputClock,
}

pub struct Fault {
    kind: Kind,
    consumed: bool,
    control: Option<std::sync::Arc<control::Control>>,
}

impl Fault {
    fn new(name: &str) -> Self {
        Self {
            kind: match name {
                "draw" => Kind::Draw,
                "clock" => Kind::Clock,
                "handoff" => Kind::Handoff,
                "output-clock" => Kind::OutputClock,
                _ => Kind::None,
            },
            consumed: false,
            control: None,
        }
    }

    pub fn none() -> Self {
        Self::new("")
    }

    pub fn from_environment() -> Self {
        if crate::linked_conditions::current().is_some() {
            return Self::none();
        }
        Self::new(&std::env::var("GOMAD_WASM_DETERMINISM_FAULT").unwrap_or_default())
    }

    pub(crate) fn from_control(owner: &crate::model::inspection::Handle) -> Self {
        match control::current().filter(|control| control.matches(owner)) {
            Some(control) => Self {
                kind: control.kind,
                consumed: false,
                control: Some(control),
            },
            None => Self::none(),
        }
    }

    pub(crate) fn for_broker() -> Self {
        match control::current() {
            Some(control) => Self {
                kind: control.kind,
                consumed: false,
                control: Some(control),
            },
            None => Self::from_environment(),
        }
    }

    pub fn bootstrap_seed(&self, seed: u64) -> u64 {
        if self.kind == Kind::Draw {
            seed | (1 << 63)
        } else if self.kind == Kind::OutputClock {
            seed | (1 << 62)
        } else {
            seed
        }
    }

    pub fn clock_shift(&mut self, records: u64) -> u64 {
        let output_ready = self.kind == Kind::OutputClock
            && self.control.as_ref().map_or_else(
                || OUTPUT_COMMITTED.load(std::sync::atomic::Ordering::Relaxed),
                |control| control.output_committed(),
            );
        if (self.kind != Kind::Clock && !output_ready) || self.consumed || records < 8 {
            return 0;
        }
        self.consumed = true;
        1
    }

    pub fn handoff(
        &mut self,
        turns: u64,
        runnable: impl IntoIterator<Item = bool>,
    ) -> Option<usize> {
        if self.kind != Kind::Handoff || self.consumed || turns < 2 {
            return None;
        }
        let next = runnable
            .into_iter()
            .enumerate()
            .filter_map(|(index, ready)| ready.then_some(index))
            .nth(1)?;
        self.consumed = true;
        Some(next)
    }
}

static OUTPUT_COMMITTED: std::sync::atomic::AtomicBool = std::sync::atomic::AtomicBool::new(false);

pub fn output_committed() {
    if crate::linked_conditions::current().is_some() {
        control::commit_output();
        return;
    }
    OUTPUT_COMMITTED.store(true, std::sync::atomic::Ordering::Relaxed);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixture_draw_signal_preserves_declared_seed_bits() {
        let fault = Fault::new("draw");
        assert_eq!(fault.bootstrap_seed(7), (1 << 63) | 7);
        assert_eq!(Fault::new("").bootstrap_seed(7), 7);
    }

    #[test]
    fn fixture_clock_shift_occurs_once_after_a_retained_prefix() {
        let mut fault = Fault::new("clock");
        assert_eq!(fault.clock_shift(7), 0);
        assert_eq!(fault.clock_shift(8), 1);
        assert_eq!(fault.clock_shift(9), 0);
        assert_eq!(Fault::new("draw").clock_shift(8), 0);
    }

    #[test]
    fn fixture_handoff_reorders_two_runnable_members_once() {
        let mut fault = Fault::new("handoff");
        assert_eq!(fault.handoff(1, [true, true]), None);
        assert_eq!(fault.handoff(2, [false, true]), None);
        assert_eq!(fault.handoff(2, [true, false, true]), Some(2));
        assert_eq!(fault.handoff(3, [true, true]), None);
    }
}
