//! Shared fixture discovery and golden runner for Umpire's tests.

pub mod backend;
pub mod boundary;
pub mod discovery;

mod bound;
mod interpreted;
mod render;
mod runner;
mod snapshot_payload;
pub use snapshot_payload::snapshot_payload;

pub use bound::{Bound, Declared, bound};
#[doc(hidden)]
pub use interpreted::probe as __interpreted_probe;
pub use interpreted::{Agreement, compare, interpreted};

pub use render::{add_paths, render, render_rows};
pub use runner::{
    CHECKER, Consumer, ExternalConsumer, SUITE, run, run_at, run_at_skipping,
    run_at_skipping_with_external,
};

/// Deterministic relative golden paths and their payloads.
pub type Files = std::collections::BTreeMap<String, String>;
/// A fixture or test failure.
pub type TestResult = Result<(), Box<dyn std::error::Error>>;
/// The files produced by one fixture.
pub type RunResult = Result<Files, Box<dyn std::error::Error>>;
/// A named fixture producer discovered by its owning crate's build script.
pub type Fixture = (&'static str, fn() -> RunResult);

/// The emitted module of the Model `world`'s `#[machine]` embeds, read for `source`; a World
/// without one, or of another source, refuses at the source.
///
/// # Errors
/// Returns the lifter's diagnostic.
pub fn lift(
    embedded: Option<&umpire_dsl::__lifted::Embedded>,
    source: &std::path::Path,
) -> Result<Module, String> {
    umpire_tla::Lift::embedded(embedded, source)
        .and_then(|lifted| umpire_tla::Tla::emit(&lifted))
        .map_err(|diagnostic| diagnostic.to_string())
}

/// Checks the emitted module of `world` with TLC against its native exploration.
///
/// # Errors
/// Returns any lift, checker or backend failure, or a disagreement.
pub fn check<M: umpire_dsl::Machine>(
    world: &M,
    embedded: Option<&umpire_dsl::__lifted::Embedded>,
    folder: &std::path::Path,
    source: &std::path::Path,
    run: fn() -> RunResult,
) -> RunResult {
    let module = lift(embedded, source)?;
    backend::check_machine(world, &module, folder, &run()?)
}

#[doc(hidden)]
pub use umpire_tla::Module;

/// Finds the Model a concrete World's `#[machine]` embeds, if it has one.
#[doc(hidden)]
pub mod __probe {
    use std::marker::PhantomData;
    use umpire_dsl::__lifted::{Embedded, Lifts};

    pub struct Probe<M>(PhantomData<M>);
    pub const fn of<M>(_: &M) -> Probe<M> {
        Probe(PhantomData)
    }
    pub trait Embeds {
        fn embedded(self) -> Option<Embedded>;
    }
    impl<M: Lifts> Embeds for &Probe<M> {
        fn embedded(self) -> Option<Embedded> {
            Some(M::__umpire_lifted())
        }
    }
    pub trait Unembedded {
        fn embedded(self) -> Option<Embedded>;
    }
    impl<M> Unembedded for &&Probe<M> {
        fn embedded(self) -> Option<Embedded> {
            None
        }
    }
}

/// Define a fixture's lift hooks for `world`: `lifted(source)` emits the Model its
/// `#[machine]` embeds, and `check(folder, source, run)` checks it with TLC. The fixture
/// runner calls both; `fixture!` defines them for its World. Also defines `interpreted()`, the
/// differential check of [`interpreted`](crate::interpreted) under the default state budget.
#[macro_export]
macro_rules! lifted {
    ($world:expr) => {
        $crate::__lift_hooks!($world);
        $crate::__interpreted_hook!($world, $crate::Checker::BUDGET);
    };
}

#[doc(hidden)]
#[macro_export]
macro_rules! __lift_hooks {
    ($world:expr) => {
        #[allow(dead_code)]
        pub(crate) fn lifted(
            source: &::std::path::Path,
        ) -> ::core::result::Result<$crate::Module, ::std::string::String> {
            $crate::lift($crate::__embedded!($world).as_ref(), source)
        }
        #[allow(dead_code)]
        pub(crate) fn check(
            folder: &::std::path::Path,
            source: &::std::path::Path,
            run: fn() -> $crate::RunResult,
        ) -> $crate::RunResult {
            $crate::check(&$world, $crate::__embedded!($world).as_ref(), folder, source, run)
        }
    };
}

/// `interpreted()`: the differential check of `world` when its `#[machine]` module was
/// admitted, otherwise `None`.
#[doc(hidden)]
#[macro_export]
macro_rules! __interpreted_hook {
    ($world:expr, $budget:expr $(, $negative:expr)?) => {
        #[allow(dead_code)]
        pub(crate) fn interpreted() -> ::core::result::Result<
            ::core::option::Option<$crate::Agreement>,
            ::std::boxed::Box<dyn ::std::error::Error>,
        > {
            #[allow(unused_imports)]
            use $crate::__interpreted_probe::{Admitted as _, Unadmitted as _};
            (&$crate::__probe::of(&$world)).interpreted(
                || $world,
                $budget,
                None $(.or(Some(($negative, $negative))))?,
            )
        }
    };
}

#[doc(hidden)]
#[macro_export]
macro_rules! __embedded {
    ($world:expr) => {{
        #[allow(unused_imports)]
        use $crate::__probe::{Embeds as _, Unembedded as _};
        (&$crate::__probe::of(&$world)).embedded()
    }};
}

/// Define a golden producer, optionally with a negative claim and additional files.
/// `fixture!(world, budget, negative, extra = files)` evaluates `files` once after
/// the checker reports and refuses paths that would replace an existing output.
/// `fixture!(World = selected, budget, negative)` names a declared World's type and also
/// defines `bound()`, which hands that one World to the lifter as values, and
/// `bound_for(world)`, the same hook for another World of the Model. Either form defines the
/// lift hooks of [`lifted!`].
#[macro_export]
macro_rules! fixture {
    ($ty:ident = $world:expr, $budget:expr $(, $negative:expr)?) => {
        pub(crate) fn bound() -> $crate::Bound<$ty> {
            bound_for($world)
        }
        #[allow(dead_code)]
        pub(crate) fn bound_for(world: $ty) -> $crate::Bound<$ty> {
            $crate::bound(world, $budget, None $(.or(Some($negative)))?)
        }
        pub(crate) fn run() -> $crate::RunResult {
            bound().run()
        }
        $crate::__interpreted_hook!($world, $budget $(, $negative)?);
        #[allow(dead_code)]
        pub(crate) fn lifted(source: &::std::path::Path) -> ::core::result::Result<$crate::Module, ::std::string::String> {
            bound().lift(source)
        }
        #[allow(dead_code)]
        pub(crate) fn check(
            folder: &::std::path::Path,
            _source: &::std::path::Path,
            _run: fn() -> $crate::RunResult,
        ) -> $crate::RunResult {
            bound().check(folder)
        }
    };
    ($world:expr, $budget:expr) => {
        $crate::__lift_hooks!($world);
        $crate::__interpreted_hook!($world, $budget);
        pub(crate) fn run() -> $crate::RunResult {
            Ok($crate::render(&$crate::Checker::explore(&$world, $budget)?)?)
        }
    };
    ($world:expr, $budget:expr, $negative:expr) => {
        $crate::fixture!($world, $budget, $negative, extra = $crate::Files::new());
    };
    ($world:expr, $budget:expr, $negative:expr, extra = $extra:expr) => {
        $crate::__lift_hooks!($world);
        $crate::__interpreted_hook!($world, $budget, $negative);
        pub(crate) fn run() -> $crate::RunResult {
            let mut files = $crate::render(&$crate::Checker::explore(&$world, $budget)?)?;
            let negative = $crate::Checker::refuted(&$world, $negative)?;
            let report = negative.verify_all()?;
            files.insert("negative_report.txt".into(), report.to_string());
            $crate::add_paths(&mut files, report);
            let extra: $crate::Files = $extra;
            for (file, content) in extra {
                if files.contains_key(&file) {
                    return Err(format!("duplicate fixture output: {file}").into());
                }
                files.insert(file, content);
            }
            Ok(files)
        }
    };
}

#[doc(hidden)]
pub use umpire_checker::Checker;
