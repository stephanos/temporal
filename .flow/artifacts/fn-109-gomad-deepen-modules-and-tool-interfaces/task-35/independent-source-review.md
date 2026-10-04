# Independent corpus source review

Permit a source-progress checkpoint for task 35. No Critical, Important or Minor source findings were identified. This verdict covers the frozen product candidate only. Formal impl-review, SHIP and task completion remain open; root metadata integration still requires its later recheck.

BASE and HEAD both equal a80ad9b9d1a4195c4aeb2fe135557f71e6e6552a. The exact candidate source hashes are retained in independent-source-review.json and every fresh receipt.

## Strengths

- corpus.go:274 and :354 attempt each existing Close once at its original defer. Successful Close performs no named-return assignment, preserving result and concrete error identity. Failed Close clears the result and returns the sole raw error or errors.Join(primary, cleanup). The new branches cannot join a nil cleanup error. validateEntry finishes its inner release before readSnapshot releases the enclosing file.
- corpus.go:156 checks validation before publication. The unchanged writeAtomic, snapshot assignment and cleanupCases sequence retains its existing committed true,error at :177. Reconstructing the complete original source proves all earlier operations and comments remain byte-identical.
- corpus_test.go:60, :300, :403 and :459 check their original Close through same-position nonfatal t.Errorf wrappers. Complete original test reconstruction matches Git BASE, including every assertion, comment and body. Four appended real-file controls at :508 onward preserve ordinary helper success, error precedence, zero failed results, memory and publication behavior.
- The one-entry canonical literal at corpus_test.go:645 is bound to the original production hash during capture and literal preservation, then to the final hash. Raw bytes hash e243d461e78c4a4a1a7f9832091709dc8bd2f8ade34e61d318914b8269fecf2f and snapshot b465cb3046d07998b31810e6a10167507ead9c68d33cade81d212552d9f57833 remain unchanged. Its synthetic fixture metadata supplies no native qualification.

## Fresh serial verification

| Check | Result |
| --- | --- |
| Full corpus package | 24/24, exit 0 |
| Preservation controls | 14/14, exit 0 |
| Five actual nested-root boundaries | 5/5, exit 0 |
| Complete configured corpus lint | 0 issues, exit 0 |
| errortype | Empty output, exit 0 |
| gofmt | Empty diff, exit 0 |

review-*.receipt.json binds exact argv, effective Go controls, cwd, source/tool/config hashes before and after, UTC/timing, exit and raw-log SHA. The pinned Go1.27.1 runs on developmental linux/arm64. Each check used the admitted offline environment; tests used test_dep and count=1. The five boundaries actually executed architecture, exported-alias, public-signature, request external-module and Runner external-consumer tests.

The independent read-only audit verified all 1,043 protected inputs, the entire worker freeze and all 16 archived receipt/raw-log bindings. It compared regenerated audit JSON with the archive in memory. It bound __file__ to source_audit.py, suppressed its two writes through equality checks and restricted receipt interpretation to the original 15 archived source-check names; audit-source's sixteenth receipt was checked separately. No worker proof changed. The original unfiltered analyzer log contains exactly two production and four fixture errcheck findings; archived final and fresh configured lint contain zero. lint-delta.json matches those six sites, with no residual or introduced corpus findings.

Makefile and protocol/boundary generation inputs exclude these corpus files. Protected generator inputs and identities remain unchanged; validate was unnecessary for this source scope. No helpers, injection seam, resource framework, second-Close experiment or suppression appeared.

## Findings and remaining evidence

Critical, Important and Minor findings are empty. Genuine first-Close and simultaneous operation-and-Close failure execution remain unproved. Sole and combined cleanup branches have source evidence only; ordinary file tests establish successful-close preservation and existing validation/publication behavior.

Original R3/R13/R18/R19, shared fn108 assessment/retention, task12/relevant predecessors/task21, matched original first-baseline fixed identities, full/completion/formal/affected-consumer and native darwin/arm64 plus linux/amd64 gates remain required and open. These gaps do not block this source checkpoint and are not waived by it. Historical whole-Gomad 419 findings supply no fresh whole-scope count.

Requested reviewer and writer tiers were both gpt-6.1-sol/high. Actual executing model metadata is unavailable and recorded null. Requested family matches; actual family is unknown. Root's supplied judge result remains unavailable/no_key, spawn null, implementer gpt-6.1-sol. The explicit native pin was preserved without another judge call, bridge or delegation.

All command handles are terminal, including sessions 94031 and 52778. The only writes were 17 new reviewer-owned files listed in the JSON, using apply_patch. Product source, worker proof, Git/index/branch and Flow remained read-only. Root will edit Flow/docs after this review; final metadata is not yet claimed frozen.
