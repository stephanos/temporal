# Exact private remainder finish: prepared, unexecuted

Root must read and authorize execution. `finish.py` SHA256 is `d721624c1b3ae7db09030f21a1e71d2c08b21fe6d1e118b8640b4614598ffbcc`; receipt schema is `fn25415-private-cache-finish-v1`. No actual Cargo, CAS verification or cleanup was executed by this preparation. Four tiny fake log/PID controls pass; the real prior four Cargo PIDs are absent and their eight log hashes were checked read-only. The earlier receipt and every sealed input remain unchanged.

The original run completed four exact tagged official Cargo cleans at exit0 and stopped before root-remainder removal on the unknown unreadable-process guard. This continuation admits only that exact recorded state, sealed by partial-receipt SHA `b98ad385f08dcfa8bcf68c2e5184f475ad80e7c118fae4cedaf35eefaa762d3a` and original cleanup SHA `8ccd5dfc80ace0263379ba09da2fa9584d10e61fdf0aedff0c9fd08040a5860f`. It does not repeat Cargo clean or invent a new full-membership baseline.

The script rechecks the exact four commands, terminal0 outcomes, absent PIDs and original log hashes; validates expected custody/classification/tool seals; re-verifies both CAS payloads; requires the original protected snapshot to remain equal; and hashes every remaining original member against the admitted subset, including original inode/size/mode. Fixed literal/parent/realpath guards remain unchanged. It refreshes ownership again after expensive hashes and immediately before the single exact-root symlink-resistant `rmtree`.

The unknown unreadable-process guard is unchanged. Only the original three explicitly acknowledged UID0 infrastructure daemons can remain unreadable; every new unknown or readable private handle refuses. Attribution of the earlier short-lived process remains an inference. Source/proof outside the target, all CAS nodes and active shared-parent directory identity are checked against the original snapshot. Parent contents may legitimately evolve under task12; the parent is never cleaned.

A new exclusive finish receipt preserves admission, action, before/after free/allocated/device observations and partial failure. Original partial receipt/log evidence is rechecked after the attempt, and custody plus protected snapshots are reverified after any mutation. No prior receipt is rewritten.

## Root-only recipe

```sh
PYTHONDONTWRITEBYTECODE=1 python3 .flow/tmp/fn25415-validation/cache-cleanup-preparation/finish.py \
  --expected-script-sha256 d721624c1b3ae7db09030f21a1e71d2c08b21fe6d1e118b8640b4614598ffbcc \
  --expected-prior-sha256 b98ad385f08dcfa8bcf68c2e5184f475ad80e7c118fae4cedaf35eefaa762d3a \
  --expected-cleanup-sha256 8ccd5dfc80ace0263379ba09da2fa9584d10e61fdf0aedff0c9fd08040a5860f \
  --expected-primary-sha256 68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610 \
  --expected-supplemental-sha256 e22efa29f106d7a2f8bf96374b080ee8d5c67b73b1dc4adc864daf7b9554a06a \
  --expected-classification-sha256 78fa352ec05730c873597c1b928d80734e779884861dbc1a2915181997d6e9a5 \
  --expected-tools-sha256 8b11813eda5a3a5482ca065f8603e57fdaf43b946b3db40acc67345128550de2 \
  --accept-recorded-daemon-visibility-limits \
  --receipt .flow/tmp/fn25415-validation/cache-cleanup-preparation/finish-execution.json \
  --execute
```

Use a new `finish-*.json` receipt name if that path already exists. Without `--execute`, no mutation or admission scan runs. Root owns execution and interpretation of any new guard refusal.
