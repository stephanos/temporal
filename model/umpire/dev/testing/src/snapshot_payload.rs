//! A std-only text snapshot reader shared with native control binaries.

/// Read the text body without changing whitespace or interpreting body delimiters.
///
/// # Errors
/// Refuses missing or malformed insta text snapshot headers.
pub fn snapshot_payload(snapshot: &str) -> Result<&str, &'static str> {
    let header = snapshot.strip_prefix("---\n").ok_or("missing snapshot header")?;
    let (metadata, payload) = header.split_once("\n---\n").ok_or("unterminated snapshot header")?;
    if !metadata.lines().any(|line| line.starts_with("source: "))
        || metadata.lines().any(|line| line == "snapshot_kind: binary")
    {
        return Err("invalid text snapshot header");
    }
    Ok(payload)
}

#[cfg(test)]
mod tests {
    use super::snapshot_payload;

    #[test]
    fn preserves_body_and_refuses_invalid_headers() {
        let body = "\nfirst\n---\n\nlast\n\n";
        assert_eq!(snapshot_payload(&format!("---\nsource: control.rs\n---\n{body}")), Ok(body));
        for invalid in [
            "plain text",
            "---\nsource: control.rs",
            "---\nwrong: field\n---\nbody",
            "---\nsource: control.rs\nsnapshot_kind: binary\n---\nbody",
        ] {
            assert!(snapshot_payload(invalid).is_err(), "{invalid}");
        }
    }
}
