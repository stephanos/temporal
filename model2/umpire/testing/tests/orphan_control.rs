//! An empty compilation fixture must still refuse orphan goldens.

use std::{error::Error, fs};

#[test]
fn orphan_control() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-orphan-{}", std::process::id()));
    let fixture = root.join("tests/fixtures/orphan_control");
    fs::create_dir_all(fixture.join("expected"))?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        fs::write(fixture.join("model.rs"), "")?;
        fs::write(fixture.join("expected/.gitkeep"), "")?;
        fs::write(fixture.join("expected/orphan.txt"), "orphan\n")?;
        let manifest = root.to_str().ok_or("invalid manifest path")?;
        let error = umpire_testing::run(
            manifest,
            vec![("orphan_control", || Ok(umpire_testing::Files::new()))],
        )
        .err()
        .ok_or("orphan golden was silently accepted")?;
        let message = error.to_string();
        assert!(message.contains("orphan.txt"), "{message}");
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}
