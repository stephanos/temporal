pub fn validate(stderr: &[u8]) -> Result<(), String> {
    for diagnostic in [b"LLVM Profile Error".as_slice(), b"LLVM Profile Warning"] {
        if stderr
            .windows(diagnostic.len())
            .any(|window| window == diagnostic)
        {
            return Err(format!(
                "LLVM profile diagnostic in helper stderr: {}",
                String::from_utf8_lossy(stderr)
            ));
        }
    }
    Ok(())
}

#[test]
fn llvm_profile_diagnostics_are_rejected() {
    for (stderr, accepted) in [
        (b"".as_slice(), true),
        (
            b"gomad-wasm-guest-profile warning: output unavailable\n",
            true,
        ),
        (b"LLVM Profile Error: Failed to write file\n", false),
        (
            b"LLVM Profile Warning: Unable to merge profile data\n",
            false,
        ),
        (
            b"ordinary output\nLLVM Profile Warning: truncated profile\n",
            false,
        ),
        (b"\xffLLVM Profile Error: corrupt profile\n", false),
    ] {
        assert_eq!(validate(stderr).is_ok(), accepted, "{stderr:?}");
    }
}
