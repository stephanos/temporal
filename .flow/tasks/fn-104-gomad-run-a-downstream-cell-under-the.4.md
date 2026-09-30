---
satisfies: [R2]
---
# fn-104-gomad-run-a-downstream-cell-under-the.4 Add the exact adapter for the membership address library

## Description
The membership layer's address library imports os/exec (remain_unsupported) and stays live in linked mode. Add an exact, digest-anchored adapter in the shape of the fx/SDK/otel adapters: version pinned in toolchain/version/version.json, per-file source and replacement digests, original/replacement inventories, darwin/arm64 prepared source-set pin, and a negative test that fails the build on an upstream edit. The rewrite refuses the subprocess path deterministically.

## Acceptance
- the adapter activates for the pinned version and fails closed for any other
- an upstream-edit mutation fails the build
- make validate and the adapter tests pass


## Done summary
Added the exact adapter for github.com/hashicorp/go-sockaddr@v1.0.7, the membership address library whose default-route lookup runs /sbin/route or ip through os/exec (remain_unsupported, live in linked mode). The darwin (route_info_bsd.go) and linux (route_info_linux.go) route readers now refuse with an error, which the library already returns when no default route exists; interface enumeration and address parsing are unchanged. Rewrites are anchored to exact source digests with pinned replacement digests, original and replacement inventories, and a darwin/arm64 prepared source-set pin reviewed through a fixture module (testdata/sockaddr); a linux build fails closed until fn-105 D9. Tests: inventories and rewritten content, changed identity rejected, upstream drift rejected (shared anchor/digest checks), and the fixture-graph source-set review; modules outside the server graph are downloaded first so CI's cold cache works.

The new inventory entry moved the deterministic profile digests, so the four libc-bound packs record the new profile implementation digests (darwin requests rediscovered; the linux request carries the digest computed from the same inventory with the linux platform, a method that reproduced both previous digests), and the profile, doctor, SQLite-pack, and bootstrap-frame goldens were updated. CI confirmed the linux digest: fork run 36677813390 passed the linux host tier, linux pack qualification, core corpus, and closure, the darwin upgrade dossier, and the smoke gate. In the downstream measurement the adapter removed the address library's os/exec from the live blockers.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fdd5bc05d, f063a539a, 7e653470d, bb3eb6902
- Tests: go test ./deterministicio -run Rewritten|RewriteAdapter|DeterministicProfile, make compatibility-pack-qualification core-qualification-set (darwin), fork run 36677813390: core-linux steps 5-10 success, core dossier success, smoke success
- PRs: