use crate::evidence::canonical;

pub const POLICY: &str = "gomad-wasm.sqlite-busy-cooperative/v1";
pub const ORIGINAL_SHA256: &str =
    "85d9934324e945f1209c69eab06a81faa13f68ae157aee7a25f426736e1e3dd6";
pub const REPLACEMENT_SHA256: &str =
    "ebd9d8f39c7b349a5d5b4586be4cb889daaa78ed146692c76ac5a308d109a50a";
pub struct Dependency<'a> {
    pub path: &'a str,
    pub version: &'a str,
    pub replaced: bool,
}
#[derive(Debug)]
pub struct Adapter {
    pub policy: &'static str,
    pub original_sha256: String,
    pub replacement_sha256: String,
    pub source: Vec<u8>,
}
/// # Errors
/// Refuses unsupported dependency identities, source digests or replacement bytes.
pub fn prepare(dependency: &Dependency<'_>, source: &[u8]) -> Result<Adapter, String> {
    if dependency.path != "github.com/ncruces/go-sqlite3"
        || dependency.version != "v0.35.6"
        || dependency.replaced
    {
        return Err(
            "SQLite cooperative adapter requires unreplaced github.com/ncruces/go-sqlite3 v0.35.6"
                .into(),
        );
    }
    let original_sha256 = canonical::digest(source);
    if original_sha256 != ORIGINAL_SHA256 {
        return Err(format!(
            "SQLite cooperative adapter source identity mismatch: {original_sha256}"
        ));
    }
    let before = b"\t\t\ttime.Sleep(time.Duration(rand.Int63() & sleepIncrement))\n\t\t}";
    let after=b"\t\t\ttime.Sleep(time.Duration(rand.Int63() & sleepIncrement))\n\t\t} else {\n\t\t\truntime.Gosched()\n\t\t}";
    let positions: Vec<_> = source
        .windows(before.len())
        .enumerate()
        .filter_map(|(index, word)| (word == before).then_some(index))
        .collect();
    if positions.len() != 1 {
        return Err("SQLite cooperative adapter requires the pinned busy-timeout branch".into());
    }
    let position = positions[0];
    let mut replacement = source[..position].to_vec();
    replacement.extend_from_slice(after);
    replacement.extend_from_slice(&source[position + before.len()..]);
    Ok(Adapter {
        policy: POLICY,
        original_sha256,
        replacement_sha256: canonical::digest(&replacement),
        source: replacement,
    })
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn prepare_six_variants() {
        let source = include_bytes!("testdata/sqlite_conn.go.orig");
        let mut drift = source.to_vec();
        drift.push(b'\n');
        for (label, path, version, replaced, bytes, valid) in [
            (
                "pinned",
                "github.com/ncruces/go-sqlite3",
                "v0.35.6",
                false,
                source.as_slice(),
                true,
            ),
            (
                "module",
                "other",
                "v0.35.6",
                false,
                source.as_slice(),
                false,
            ),
            (
                "version",
                "github.com/ncruces/go-sqlite3",
                "v0.35.5",
                false,
                source.as_slice(),
                false,
            ),
            (
                "replacement",
                "github.com/ncruces/go-sqlite3",
                "v0.35.6",
                true,
                source.as_slice(),
                false,
            ),
            (
                "source-drift",
                "github.com/ncruces/go-sqlite3",
                "v0.35.6",
                false,
                drift.as_slice(),
                false,
            ),
            (
                "missing-source",
                "github.com/ncruces/go-sqlite3",
                "v0.35.6",
                false,
                &[],
                false,
            ),
        ] {
            let baseline = bytes.to_vec();
            let result = prepare(
                &Dependency {
                    path,
                    version,
                    replaced,
                },
                bytes,
            );
            assert_eq!(result.is_ok(), valid, "{label}: {result:?}");
            if let Ok(mut adapter) = result {
                assert_eq!(adapter.policy, POLICY);
                assert_eq!(adapter.original_sha256, ORIGINAL_SHA256);
                assert_eq!(adapter.replacement_sha256, REPLACEMENT_SHA256);
                assert_eq!(canonical::digest(&adapter.source), REPLACEMENT_SHA256);
                adapter.source[0] = 0;
                assert_eq!(bytes, baseline);
            }
        }
    }
}
