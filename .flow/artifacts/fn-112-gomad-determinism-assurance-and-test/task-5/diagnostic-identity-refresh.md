# Controller-bound diagnostic fixture on candidate 2ecdbd33

The focused `TestDiagnosticsOffPreservesExistingCanonicalIdentities` run failed only in its choices row after the runtime/protocol source changed the controller identity. `diagnostic-identity-red.log` retains the actual canonical JSON.

`diagnostic-identity-changes.json` records all seven differences against the preceding approved fixture: the three controller identity occurrences, tape digest, failure signature, record hash, and portable plan digest. No semantic field, schema, decision count, or plain fixture changed. The choices fixture was refreshed to the emitted canonical bytes; the test still compares the entire JSON byte for byte.

The same focused command passed afterwards (`diagnostic-identity-green.log`, exit 0). It used the candidate patched Go with `GOEXPERIMENT=nogreenteagc`, `-tags test_dep`, and `-count=1`. The full host gate will cover the fixture with the frozen batch.
