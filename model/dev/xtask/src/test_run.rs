//! Temporal Cargo test and doctest execution, owned by the gate.
use std::{io, path::Path};

pub(crate) fn run(root: &Path) -> io::Result<i32> {
    crate::gate::stage(
        root,
        "tests",
        crate::gate::cargo(&["test", "--workspace", "--locked", "--quiet"]),
    )
}
