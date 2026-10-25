# mise provisioning feasibility

Installed mise 2026.10.11 can provision these artifacts without a custom plugin. Both HTTP and GitHub backends installed the exact TLC 1.7.4 JAR and refused an incorrect checksum. Keep runtime JAR admission. An isolated corruption experiment proves `mise install --locked` accepts an already-installed, corrupted JAR.

## Evidence and limits

All probes used independently isolated MISE_DATA_DIR, MISE_CACHE_DIR, MISE_STATE_DIR, MISE_CONFIG_DIR, global/system config files, ceiling paths and trusted paths under this report's directory. HOME was unchanged. `http-good/config.log` and `github-good/config.log` show only their scratch configs. No source, global tooling, active task cache or guest build was changed. `probe.py`, `probe-more.py`, and `probe-optional.py` reproduce the experiments. Each individual command has a `.log` and `.status` file.

| Probe | Result | Exact evidence relative to this directory |
| --- | --- | --- |
| Installed version | 2026.10.11, linux-arm64 | `http-good/version.log` |
| HTTP raw TLC, exact SHA | Exit 0; installed 2,274,532-byte JAR with repository's SHA | `http-good/install.log`, `http-good/install.status`, `http-good/original-jar.sha256`, `probe-summary.log` |
| HTTP raw TLC, all-zero SHA | Exit 1, checksum mismatch | `http-bad/install.log`, `http-bad/install.status` |
| GitHub raw TLC, exact SHA | Exit 0 | `github-good/install.log`, `github-good/install.status` |
| GitHub raw TLC, all-zero SHA | Exit 1, checksum mismatch | `github-bad/install.log`, `github-bad/install.status` |
| Tiny local AP-shaped tgz | Exit 0; preserves `lib/apalache.jar` with `strip_components=1`, `bin_path="bin"` | `archive-layout/mise.toml`, `archive-layout/install.log`, `probe-more-summary.log` |
| Lock then corrupt installed TLC | Lock succeeds; subsequent locked install exits 0/already installed; corrupted bytes remain | `http-good/lock.log`, `http-good/locked-corrupt.log`, `http-good/locked-corrupt.status`, `http-good/after-locked.sha256` |
| Optional AP named environment | Default tool set empty; AP appears only with `--env apalache` | `opt-in/default-ls.log`, `opt-in/opt-in-ls.log` |

The original TLC JAR bytes were restored after the corruption probe. The probe did not execute TLC or Java. AP's 192,038,424-byte official archive was not downloaded. Its release metadata reports the exact existing archive SHA, retained in `apalache-release.json`. The local layout fixture establishes generic extraction behavior, not successful admission of the real AP release. Exact install directories were ordinary directories; version aliases such as `1.7` were symlinks. Use exact versions and keep shared extraction disabled.

The installed release's own [HTTP docs](https://github.com/jdx/mise/blob/v2026.10.11/docs/dev-tools/backends/http.md), [GitHub docs](https://github.com/jdx/mise/blob/v2026.10.11/docs/dev-tools/backends/github.md), and [lock docs](https://github.com/jdx/mise/blob/v2026.10.11/docs/dev-tools/mise-lock.md) are retained as `*-installed.md`. The last explicitly distinguishes download verification from mutable installed trees. Current docs advertise 2026.10.19, so the tested 2026.10.11 behavior is the evidence for these options. These probes found no version blocker for raw JAR checksums or the demonstrated archive layout; they do not certify later checksum/security fixes as unnecessary.

## Minimum candidate configuration

Prefer HTTP for explicit fixed official URLs and exact SHA values. GitHub works too, with `asset_pattern="tla2tools.jar"`, `format="raw"`, `bin="tla2tools.jar"`; it adds release API lookup without benefit for these fixed URLs. Keep pins inside the framework. Candidate TLC declaration:

```toml
[tools."http:tlc"]
version = "1.7.4"
url = "https://github.com/tlaplus/tlaplus/releases/download/v1.7.4/tla2tools.jar"
format = "raw"
bin = "tla2tools.jar"
checksum = "sha256:936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"
```

Candidate AP declaration in framework-owned `mise.apalache.toml`, absent from the default tool set:

```toml
[tools."http:apalache"]
version = "0.62.4"
url = "https://github.com/apalache-mc/apalache/releases/download/v0.62.4/apalache-0.62.4.tgz"
format = "tar.gz"
strip_components = 1
bin_path = "bin"
checksum = "sha256:b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4"
```

Scope jar provisioning/lookup commands to the framework root and its owned mise data/cache/state directory; install only the named tool. AP provisioning explicitly selects `--env apalache`. Resolve the exact installed path with `mise where` or structured installed-tool metadata, then retain contained-path admission before Java execution. Keep these jar-specific directory overrides local to those commands so they do not silently move the Rust/Java installations used by ordinary framework commands. Reuse .13's admitted root resolver and independent framework pin ownership. Do not infer roots from Temporal parents or copy artifacts into a second custom cache.

Keep normal execution auto-install disabled and require an explicit provisioning command. With auto-install disabled, even opted-in `mise exec -- /bin/true` succeeded while AP was missing (`opt-in/opt-in-exec.log`). Therefore gate-required AP must explicitly require its installation and pass backend admission; mise exec success alone is insufficient. Lock generation should cover supported platforms if `--locked` is adopted; only linux-arm64 was tested here.

## Checks to retain or delegate

- Delegate download/retry, exact archive checksum, extraction, per-version installation coordination and normal temporary cleanup to mise. Remove the corresponding custom curl/archive plumbing once the replacement's focused controls pass. The installed source has incomplete-install markers and cleanup (`backend-mod.rs:5194-5235`) and HTTP sibling temporary extraction followed by rename (`http.rs:726-773`). Cleanup is sometimes best-effort, so this is not proof of identical old cleanup semantics.
- Keep the AP **JAR** SHA in addition to its archive SHA. Keep TLC's JAR SHA, exact Java pin and runtime version probe. `model/umpire/backends/process/src/jar.rs:20-40` already provides this admission; installed-file corruption bypassed mise's locked install in the experiment.
- Keep no-symlink/escape and regular-file admission (`jar.rs:46-74`). Scope install roots before invoking mise, and reject unexpected resolved locations. The old AP installer extracts only the selected JAR into private staging; mise extracts the package. Archive path safety, interrupted installs, concurrent installs and corrupt/partial trees need focused replacement controls before deleting the old equivalents. This investigation did not exercise hostile archives, concurrent writers or crashes. Preserve existing refusal behavior where required; do not claim generic mise cleanup is an identical security contract.

Place the simplification in existing fn-254.10 after .13 integrates. Leave .13's frozen driver, lockfiles and guest-reproduction proofs untouched now. This investigation supports removing custom provisioning machinery while retaining the existing runtime admission module. Vendoring the 192MB AP archive or its JAR adds repository distribution/update work and is unnecessary for this approach; an offline distribution requirement would warrant a separate choice.
