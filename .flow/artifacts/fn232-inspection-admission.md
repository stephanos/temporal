# fn222 standalone inspection admission

Base source: `2f52fbb72492c016dd3fc29ac923f39c20d11054`.
Scope: source repair for fn232 reference preservation, no tracker lifecycle change.

The temporal-inspection primary stage precedes mandatory onebox qualification.
Its previous native admission opened that later Result, making a serial run fail
and a parallel run race. The primary now has a separate native entrypoint:

```
validate-inspection-driver RECEIPT RECEIPT_SHA DRIVER_CONFIG DRIVER_CONFIG_SHA EXECUTABLE EXECUTABLE_SHA
```

The caller authenticates the frozen support executable and passes the frozen
ordinary candidate executable separately. Native admission reads actual candidate
bytes and requires their digest/path to match the driver and its frozen files.
The driver must bind compiled host/helper identities, ordinary features, the
actual source manifest and every source member. It authenticates the selected
cooperative installation, compiler/runtime/cache paths, guest-configuration
admission and the complete frozen installation compiler-input inventory. Public
source/compiler files use bounded no-follow stable reads; evidence artifacts
retain their existing private store checks. The receipt must name this exact
private observation config.

The actual preparation Store is opened and authenticated. Its retained
provenance supplies the actual aliased compiler closure. Original overlay bytes
are reconstructed with the existing Go encoding and must match the frozen raw
hash. Each observed member must have an exact frozen counterpart; compiler,
runtime identity and overlay replacements must agree. Only then is the original
inventory projected to the exact observed closure. This matters because an
installation pins a larger compiler tree than an individual preparation uses
(read-only current metadata inspection found 1442 installed inputs versus 50
prepared inputs, with matching subset hashes). Full frozen inputs are still
validated, including unused members. An observed missing compiler, changed hash,
foreign member or replacement cannot pass. No Result or provenance is fabricated.

The resulting candidate runs the same existing `inspection::validate` checks for
raw preparation/plan/module/config, commands, policy, source/compiler detachment,
replays and debugging records. Existing Result-based `validate-inspection` is
unchanged and remains the final crossgate reread. Python routing/freezer changes
are separately owned by qualification_composition.

Focused support tests: 26 passed, zero failed, exit 0 in
`/home/agent/evidence/fn232-inspection-admission-verified.log`. Synthetic fixtures
exercise executable/source/installation/driver substitutions, public input modes,
actual authenticated Store envelopes, and a full frozen inventory with unused
members. The public command reaches an existing inspection semantic refusal
without creating/opening a Result. These binding fixtures are not actual Temporal
guests or a successful complete inspection qualification. Initial compile/mode
failures are retained in earlier logs. Strict gomad library/test Clippy exits 0
in `fn232-inspection-admission-clippy-final.log`. Installation regressions also pass: 21 tests, zero failures, exit 0 in
`fn232-inspection-admission-install.log` (1127.70s in debug mode, dominated by
repeated hashing of the 352 MiB test executable). Exact source/log hashes and
exit codes are retained in `fn232-inspection-admission-summary.json`.

Fresh independent gpt-6.1-sol/high review found no blocking findings, including
full-input authentication versus exact selected closure and preservation of
normal installed `current_exe` error precedence. No heavy guests, Go compiler,
full qualification, frozen inventory edits or global ledger changes were run.
Final composed source and actual primary/final rereads remain root-owned gates.
