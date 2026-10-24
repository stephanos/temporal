use super::*;

#[test]
fn wrong_java_version_vendor_and_failed_launch_are_refused() {
    for (success, output) in [
        (false, "openjdk version \"27\"\nTemurin-27+"),
        (true, "openjdk version \"21\"\nTemurin-21+"),
        (true, "openjdk version \"27\"\nOtherVendor-27+"),
    ] {
        assert!(validate_java(success, output).is_err());
    }
    assert!(
        validate_java(
            true,
            "openjdk version \"27\" 2026-09-15\nOpenJDK Runtime Environment Temurin-27+18"
        )
        .is_ok()
    );
}
