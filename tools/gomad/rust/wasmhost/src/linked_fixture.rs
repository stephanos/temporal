use crate::engine::State;
use wasmtime::{Caller, Error, Linker, Result};

pub(crate) fn signature(
    module: &str,
    name: &str,
) -> Option<(&'static [&'static str], &'static [&'static str])> {
    (module == "gomad_fixture_v1" && name == "choice_mode").then_some((&[], &["i32"]))
}

pub(crate) fn link(linker: &mut Linker<State>) -> Result<()> {
    linker.func_wrap(
        "gomad_fixture_v1",
        "choice_mode",
        |caller: Caller<'_, State>| -> Result<i32> {
            caller
                .data()
                .model
                .as_ref()
                .map(|model| i32::from(model.fixture_choice_mode()))
                .ok_or_else(|| Error::msg("choice mode fixture requires a local model"))
        },
    )?;
    Ok(())
}
