//! A Model's own tests, beside its `model.rs`: authored examples replayed through the Model and
//! explorations over a space of its states.
//!
//! A test file writes `use umpire_suite::*;` and plain `#[test]` functions; `cargo test` runs
//! them. Nothing here explores on its own: an example replays through the same
//! `Machine::checked_step` the checker uses, and every exploration and search asks the checker.

#![forbid(unsafe_code)]

mod coverage;
mod covered;
mod refusal;
mod run;
mod search;
mod space;

pub use coverage::{CatalogEntry, ClaimCoverage, ClaimKind, Coverage};
pub use covered::{Cases, Covered, cover, sample};
pub use refusal::Refusal;
pub use run::Run;
#[doc(hidden)]
pub use space::SpaceBinding;
pub use space::{Space, find, refuted, verify_all};
pub use umpire_dsl::path;

/// A path over a state's registered world: what `find`, `cover`, `sample` and `Run::along`
/// search for. `path!` infers its action and outcome types from the call.
pub type WorldPath<S> = umpire_dsl::Path<
    S,
    <<S as umpire_dsl::Associated>::World as umpire_dsl::Machine>::Action,
    <<S as umpire_dsl::Associated>::World as umpire_dsl::Machine>::Outcome,
>;

/// Evaluates each line in order and fails the test at the first false one, with that line's
/// source text and the state it was judged at.
///
/// ```ignore
/// expect! { k = run.state();
///     k.heating_on;
///     !k.beeping;
/// }
/// ```
#[macro_export]
macro_rules! expect {
    ($state:ident = $at:expr; $($line:expr);+ $(;)?) => {{
        let $state = $at;
        $(
            if !$line {
                ::core::panic!("expected `{}` at {}", ::core::stringify!($line), $state);
            }
        )+
    }};
}
