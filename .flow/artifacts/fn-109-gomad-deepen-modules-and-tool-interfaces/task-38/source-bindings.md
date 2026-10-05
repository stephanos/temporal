# Source binding index

Every command receipt retains its exact command, environment, working directory,
start/end/elapsed time, command exit, before/after source-manifest hashes and raw
log hash. All command before/after manifests compared equal. Repeated manifests
have one retained copy per unique source state, mapped below. Duplicate files
remain recoverable in ignored `.flow/tmp/task38-duplicate-manifests/`. Raw logs
and receipts retain their original bytes and original generated manifest names.

| Commands | Retained manifest | SHA-256 |
| --- | --- | --- |
| baseline-lint, baseline-controls, baseline-architecture | baseline-lint.before.sha256 | 1e7d577bf1127e061bf59c1f32fe9c156d2a61493a7cfc2421be49bc80342996 |
| baseline-digest-controls | baseline-digest-controls.before.sha256 | 78485a0ed562cda56cc0f7c13cf4f11aa67d4f294b48de04423420361dcc9c0f |
| final-controls, final-lint, final-architecture (superseded Sprintf candidate) | final-lint.before.sha256 | 12449b3598149c3fe4290a270c27cc66c4706f8e45d9dfd9c680c3e874f10ad6 |
| final-appendf-controls, final-appendf-lint, final-appendf-errortype, final-appendf-format, final-appendf-architecture | final-appendf-controls.before.sha256 | a5eb12d3228b6fe5b829532d784229a2c8bdcfe7df61d53f81cba5646dbec7af |

The initial manifest binds 1,038 tracked inputs under tools/gomad3 and
tools/gomad3sim, the lint configuration and module go.mod/go.sum files. The
later manifests include the new digest test, giving 1,039 entries. Initial
versus final comparison changes exactly the six existing files authorized by
task38 and adds its digest test. The other 1,032 entries remain identical.
The comparison also preserves task39 sources, runtime overlays, compatibility
pins, source inventory, module files, profiles and generator inputs.

The seven final source hashes are:

```text
3b439959ea1d82ba86814d24fc906995c5c22fece5bf530e009fc6d6c161eca1  tools/gomad3/target/capability.go
432f5027aa4b76f624e5717c60814e97cd320fb5c202ab18a6fc9c45842372de  tools/gomad3/target/capability_collection.go
daceec8ebdbba116a3b7a546cb0796c3e7d2a4f8b7e4924fe922c43469cbe3af  tools/gomad3/target/capability_evaluation.go
b96dba8f5360fc6c6a375bb8df619a01f3af0ea68e549a9ecc1a2406f16c92c7  tools/gomad3/target/capability_golden_test.go
b4bdf4a35393b59557d7dd4a9edc115688a7bbe523c20014b386fe4174a7e00f  tools/gomad3/target/capability_review_test.go
13d96303a2eafe579d87f82f38c6c4c8f78416647dc42ea38de21bbc44d02c01  tools/gomad3/target/prepared_cache.go
55ea0b20188616f00520a2dd3bbd9e6d1372d72e3e0731a951d1695e1bdc2356  tools/gomad3/target/prepared_cache_digest_test.go
```

Pinned tools and configuration matched the admission plan before and after
source edits. The lint binary hash is
`acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc`;
the errortype binary hash is
`db481b4086fb85962e98ae625984f2560a883dce91f8b7d8c705414c207be8cc`;
`.github/.golangci.yml` hashes to
`2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43`.
