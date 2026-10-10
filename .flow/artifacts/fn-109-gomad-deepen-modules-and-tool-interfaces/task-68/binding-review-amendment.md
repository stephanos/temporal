The worker amended worker-summary.md, worker-evidence.json and evidence.jq to disclose the checker-binding limitation before the conductor's archive-before-amend instruction arrived. The exact original bytes were reconstructed by reversing only those known disclosures, archived under distinct names, and verified against the immutable packet-review-seal.sha256. This reconstruction is not an execution-time checker attestation.

| Original historical member | Exact-byte archive | SHA256 in historical seal and archive |
| --- | --- | --- |
| worker-summary.md | worker-summary-pre-binding-review.md | d23e52434938391a9c8fb87de3c59ce4ede4bc4dcf3ae9c9796b511d5bbd323c |
| worker-evidence.json | worker-evidence-pre-binding-review.json | 4f211655ba5b81f45d26eba012601253045438ea2b9056dd4aba6c8622b98f6f |
| evidence.jq | evidence-pre-binding-review.jq | f2c8a887f5cc42bdb1ba80e319379deb4ea7cf7b2766f0bd540ba1476f971c2b |

The historical packet-review-seal.sha256 remains unchanged with SHA256 1ff47a5d1336117ce541df4d2bfcf44264e44f59810ff1aa9417bd5f511b5484. It is a historical snapshot, not a current all-member check. Its three amended member entries map to the archives above; all other historical member bytes remain unchanged. The final packet-final-seal.sha256 binds current members, these archives, this disclosure and the old seal; it excludes only itself.

Original run-control source/tool manifests omitted the actual Perl executable and preserve.pl, compare-runner-lint.pl and reconcile-final-boundaries.pl bytes at checker execution. Their original terminal receipts therefore do not provide execution-time checker binding. Later script hashes and seals cannot retroactively repair that limitation. Original executed wrappers, checker scripts, receipt JSON and raw logs remain unchanged. Go source/tool/log bindings are unaffected.

The conductor's independent reviewer owns separate fresh read-only checker observations with Perl/checker/input hashes before and after execution; those observations are not worker Go reruns or revisions to historical receipts. The worker has released the execution lane, has no active command handles and claims no review verdict or aggregate acceptance.
