AXIS: correctness

## Quality Audit — Correctness follow-up: 47cf0beea6..5c59b5edc9

- Files changed: 9 · Remaining findings: 0 · New findings: 0 · Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship for this follow-up scope.

**walked:** Every nonblank CFG line now passes framing validation. Continuations refuse in main and isolated CFGs; retained constants undergo exact typed decoding, main-binding comparison and cross-family equality. Both previously reported bypasses are closed.

**cited:** Retained logs show the continuation mutation failing before the fix and passing afterward. The regression test covers all five CFG families while retaining replacement and coordinated-mutation controls.

No new correctness issues or weakened assertions found. No builds/tests ran during this audit; primary review and full-gate status remain with the parent.
