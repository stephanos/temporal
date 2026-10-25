//! Closure laws distinguish phase preservation from state and outcome preservation.

#![forbid(unsafe_code)]

use umpire_dsl::{Outcome, Phase, Phased, Rejection, Verdict, claim, closable};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
enum Status {
    Open,
    Finished,
    Aborted,
}

impl Phase for Status {
    fn closed(&self) -> bool {
        matches!(self, Self::Finished | Self::Aborted)
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
struct Record {
    status: Status,
    value: u8,
}

impl Phased for Record {
    type Phase = Status;

    fn phase(&self) -> Self::Phase {
        self.status
    }
}

#[test]
fn closure_checks_separate_phase_state_and_entity_specific_rejection() {
    use Status::{Aborted, Finished, Open};
    let rejected = Outcome::Rejected(Rejection::FailedPrecondition);
    for (before, after, outcome, terminal, uniform) in [
        ((Open, 0), (Finished, 1), Outcome::Accepted, Verdict::Holds, Verdict::Holds),
        ((Finished, 0), (Finished, 0), rejected, Verdict::Holds, Verdict::Holds),
        ((Finished, 0), (Finished, 0), Outcome::Accepted, Verdict::Holds, Verdict::Fails),
        (
            (Finished, 0),
            (Finished, 0),
            Outcome::Rejected(Rejection::NotFound),
            Verdict::Holds,
            Verdict::Fails,
        ),
        ((Finished, 0), (Finished, 1), rejected, Verdict::Holds, Verdict::Fails),
        ((Finished, 0), (Open, 0), rejected, Verdict::Fails, Verdict::Fails),
        ((Finished, 0), (Aborted, 0), rejected, Verdict::Fails, Verdict::Fails),
    ] {
        let before = Record { status: before.0, value: before.1 };
        let after = Record { status: after.0, value: after.1 };
        assert_eq!(before.closed(), before.status.closed());
        assert_eq!(
            claim::evaluate(|| closable::terminal_remains(&after, &before)).verdict,
            terminal,
            "{before:?} -> {after:?} ({outcome:?})"
        );
        assert_eq!(
            claim::evaluate(|| {
                closable::closed_rejects(&after, &before, &outcome, Rejection::FailedPrecondition);
            })
            .verdict,
            uniform,
            "{before:?} -> {after:?} ({outcome:?})"
        );
    }
}
