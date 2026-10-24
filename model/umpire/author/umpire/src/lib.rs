//! Model authoring through one dependency: declarations, authored suites and catalog derives.

pub use umpire_catalog as catalog;
pub use umpire_catalog::{Catalog, annotate};
pub use umpire_dsl as dsl;
pub use umpire_dsl::*;
pub use umpire_suite as suite;
pub use umpire_suite::*;

#[doc(hidden)]
pub use umpire_runtime as runtime;
