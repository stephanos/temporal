//! A fixture must select at least one consumer through its expected files.

use std::{error::Error, fs};

#[test]
fn rejects_fixture_without_owned_outputs() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-empty-{}", std::process::id()));
    let fixture = root.join("tests/fixtures/empty_control");
    fs::create_dir_all(fixture.join("expected"))?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        fs::write(fixture.join("model.rs"), "")?;
        let manifest = root.to_str().ok_or("invalid manifest path")?;
        let error = umpire_testing::run(
            manifest,
            vec![("empty_control", || Ok(umpire_testing::Files::new()))],
        )
        .err()
        .ok_or("empty fixture was silently accepted")?;
        assert!(error.to_string().contains(&*fixture.to_string_lossy()), "{error}");
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}
