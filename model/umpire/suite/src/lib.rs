//! A Model's own tests, beside its `model.rs`: authored examples replayed through the Model.
//!
//! A test file writes `use umpire_suite::*;` and plain `#[test]` functions; `cargo test` runs
//! them. Nothing here explores on its own: an example replays through the same
//! `Machine::checked_step` the checker uses, and a search asks the checker.

mod refusal;
mod run;
mod search;

pub use refusal::Refusal;
pub use run::Run;
pub use umpire_dsl::path;

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
