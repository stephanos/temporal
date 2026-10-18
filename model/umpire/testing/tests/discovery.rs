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

#[test]
fn compiles_distinct_non_identifier_folder_names() -> Result<(), Box<dyn Error>> {
    let root = std::env::temp_dir().join(format!("umpire-discovery-names-{}", std::process::id()));
    let fixtures = root.join("fixtures");
    fs::create_dir_all(&fixtures)?;
    let result = (|| -> Result<(), Box<dyn Error>> {
        for name in ["kettle-cut", "kettle_cut", "kettle_2dcut", "Kettle", "4", "λ"] {
            let folder = fixtures.join(name);
            fs::create_dir_all(folder.join("expected"))?;
            fs::write(folder.join("model.rs"), "pub(crate) fn run() {}\n")?;
        }
        let generated = root.join("generated.rs");
        umpire_testing::discovery::generate(&fixtures, &generated)?;
        let source = root.join("lib.rs");
        fs::write(
            &source,
            "#![allow(dead_code)]\nmod umpire_testing { pub type Fixture = (&'static str, fn()); }\ninclude!(\"generated.rs\");\n",
        )?;
        let output = std::process::Command::new("rustc")
            .args(["--edition=2024", "--crate-type=lib", "--emit=metadata", "--out-dir"])
            .arg(&root)
            .arg(&source)
            .output()?;
        assert!(output.status.success(), "{}", String::from_utf8_lossy(&output.stderr));
        Ok(())
    })();
    fs::remove_dir_all(root)?;
    result
}
