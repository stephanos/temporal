//! Native responses for comparison with actual Wasm checker defects.

#[path = "../../../../tests/support/panic.rs"]
mod control;

#[path = "../../../../../../dev/testing/src/snapshot_payload.rs"]
#[allow(unreachable_pub, reason = "the shared reader is public when reexported by umpire-testing")]
mod snapshot_payload;

use prost::Message;
use std::{error::Error, fs, path::PathBuf};
use umpire_runtime::protocol;

fn main() -> Result<(), Box<dyn Error>> {
    let output = PathBuf::from(std::env::args_os().nth(1).ok_or("missing output directory")?);
    fs::create_dir_all(&output)?;
    for (name, mode, golden) in [
        (
            "effect_panic",
            control::support::Mode::EffectPanic,
            include_str!(
                "../../../../../../backends/native/tests/fixtures/effect_panic/expected/report.txt.snap"
            ),
        ),
        (
            "effect_overflow",
            control::support::Mode::Overflow,
            include_str!(
                "../../../../../../backends/native/tests/fixtures/effect_overflow/expected/report.txt.snap"
            ),
        ),
    ] {
        let mut model = control::model(mode);
        let response = model.dispatch(1, &protocol::ExploreRequest { budget: 3 }.encode_to_vec());
        let Some(protocol::response::Outcome::Explore(report)) = &response.outcome else {
            return Err(format!("{name}: exploration refused: {response:?}").into());
        };
        assert_eq!(
            report.report,
            snapshot_payload::snapshot_payload(golden)?,
            "{name}: native report changed"
        );
        fs::write(output.join(format!("{name}.pb")), response.encode_to_vec())?;
    }
    Ok(())
}
