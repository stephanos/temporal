//! Integration controls for backends/apalache.

#![forbid(unsafe_code)]

#[path = "main/controls.rs"]
mod controls;
#[path = "main/decode.rs"]
mod decode;
#[path = "main/replay.rs"]
mod replay;

#[path = "fixtures/arithmetic_prefix.rs"]
mod prefix_fixture;
#[path = "fixtures/replay.rs"]
mod replay_fixture;
#[allow(unreachable_pub, reason = "the canonical fixture is shared by public host crates")]
#[path = "../../../dev/fixtures/fixtures/sample_kettle/model.rs"]
mod selected_kettle;
