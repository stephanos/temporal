//! Import candidates produced by the live Go tests into insta's review workflow.

#[test]
#[ignore = "requires explicitly exported live Go candidates"]
fn import_live_candidates() -> Result<(), Box<dyn std::error::Error>> {
    let candidates = std::path::PathBuf::from(
        std::env::var_os("UMPIRE_SNAPSHOT_CANDIDATES").ok_or("missing candidate directory")?,
    );
    let mut settings = insta::Settings::clone_current();
    settings.set_snapshot_path("../expected");
    settings.set_prepend_module_to_snapshot(false);
    settings.set_omit_expression(true);
    let _guard = settings.bind_to_scope();
    let mut imported = 0;
    for name in ["coverage.txt", "gaps.txt", "mutants.txt", "transcript.txt"] {
        let path = candidates.join(name);
        match std::fs::read_to_string(&path) {
            Ok(candidate) => {
                insta::assert_snapshot!(name, candidate);
                imported += 1;
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
            Err(error) => return Err(error.into()),
        }
    }
    if imported == 0 {
        return Err("candidate directory contains no registered live output".into());
    }
    Ok(())
}
