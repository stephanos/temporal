- use build tags to cut down on dependencies we don't need in the WASM module (e.g. elasticsearch)
- a data race detector
- can we record external data and replay it; eg for Java SDK

## Signed qualification receipts (from fn-232.10 review, 2026-10-22)
Qualification receipts are content-addressed, not signed. Someone who rewrites the execution records, the Result and the receipt together can still forge provenance and identity fields that don't feed the verdict, such as runtime, compiler, model and engine digests. Codex reviews 2, 5, 7, 8 and 9 on branch fn232-qualify flagged this. Signing receipts, or binding every identity to retained build preimages, would close the gap. fn-232.10 explicitly placed it out of scope.

## Backstop exit and partial stdout lines (from fn-232.1 closeout, 2026-10-22)
The deadline backstop exits through `std::process::exit`, which flushes buffered stdout. If stdout were a full, undrained pipe holding a partial line, that exit could block. gomad writes whole JSON lines, so this is unlikely, and the review didn't flag it. A possible follow-up is to use `libc::_exit` after the bounded diagnostic wait.
