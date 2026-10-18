//! Missing fixture inputs and outputs must never disappear from discovery.

use std::{error::Error, fs};

#[test]
fn refuses_incomplete_fixtures() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-discovery-{}", std::process::id()));
    fs::create_dir_all(&root)?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        for missing in ["model.rs", "expected"] {
            let fixture = root.join(format!("missing_{}", missing.replace('.', "_")));
            fs::create_dir(&fixture)?;
            if missing == "model.rs" {
                fs::create_dir(fixture.join("expected"))?;
            } else {
                fs::write(fixture.join("model.rs"), "")?;
            }
            let error = umpire_testing::discovery::generate(&root, &root.join("generated.rs"))
                .err()
                .ok_or("incomplete fixture was silently accepted")?;
            let message = error.to_string();
            assert!(message.contains(missing), "{message}");
            assert!(message.contains(&*fixture.to_string_lossy()), "{message}");
            fs::remove_dir_all(fixture)?;
        }
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}
