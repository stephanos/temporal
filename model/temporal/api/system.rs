//! The Temporal API catalog system, shared by the API build and the realization macros.

use std::{io, path::Path, process::Command};

use umpire_catalog_build::System;

/// The workflow service, generated under `umpire_temporal_api` and covered by the catalogs in
/// `model/temporal` of the `repository` checkout.
pub(crate) fn system(repository: &Path) -> System<'static> {
    System {
        service: "temporal.api.workflowservice.v1.WorkflowService",
        prefix: "workflow",
        rust_root: "::umpire_temporal_api",
        request_roots: &[
            "StartActivityExecution",
            "DescribeActivityExecution",
            "PollActivityExecution",
            "RequestCancelActivityExecution",
            "PollActivityTaskQueue",
            "RecordActivityTaskHeartbeat",
            "RespondActivityTaskCompleted",
            "RespondActivityTaskFailed",
            "RespondActivityTaskCanceled",
        ],
        catalog_root: repository.join("model/temporal"),
    }
}

/// Build the upstream descriptor set and link its API closure into `out`, emitting the
/// Cargo inputs of both steps.
#[allow(clippy::print_stdout, reason = "Cargo consumes build input directives on stdout")]
pub(crate) fn link_descriptors(repository: &Path, out: &Path) -> io::Result<()> {
    for input in
        ["Makefile", "go.mod", "go.sum", "cmd/tools/getproto", "proto/internal", "proto/api.binpb"]
    {
        println!("cargo:rerun-if-changed={}", repository.join(input).display());
    }
    if !Command::new("make").current_dir(repository).arg("proto/api.binpb").status()?.success() {
        return Err(io::Error::other("make proto/api.binpb failed"));
    }
    if !Command::new("sh")
        .current_dir(repository)
        .arg("cmd/tools/getproto/run.sh")
        .arg("--linked-api")
        .arg(repository.join("proto/api.binpb"))
        .arg("--out")
        .arg(out)
        .status()?
        .success()
    {
        return Err(io::Error::other("linking Temporal API descriptors failed"));
    }
    Ok(())
}
