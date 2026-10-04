# Early package/signature review

Fresh read-only `/root/architecture_package_early_review` reviewed implemented
inventory/public-signature logic, root integration and relevant tests/logs,
excluding unfinished effects and active production repairs. Requested
gpt-6.1-sol / high, same family as writer, actual metadata unobservable.
Judge once, unavailable (no_key). No tests/builds/package loading, edits,
artifacts, generation, bridges or Flow/Git mutation by the reviewer.

Strengths: complete ./... discovery, included error rejection, independent
hidden/underscore/testdata and nested-module inventory, exact boundaries and
future root-source rejection. Host vet covers all 55 observed host paths per
source set with target environment on children only. Typed traversal covers
containers, underlying types, aliases, generics, methods, embeddings and cycles;
actual RED identifies the three confirmed Gomad leaks.

Critical: none. Important findings:

1. architecture.go:70 checks required go.mod existence but not Go source in
   each classified module. Enclosing exclusions allow siblings to mask a source-
   empty cactusstatsd fixture. Validate source per module, respecting nested
   boundaries, and retain a Discover deletion regression.
2. architecture.go:69 rejects source-file links, but WalkDir and go list skip
   directory symlinks. A nonoptional linked source/module tree escapes coverage.
   Reject or explicitly inventory those links with a real checker regression.
3. program.go:95 recursively expands foreign public APIs and rejects existing
   json.RawMessage through public jsontext.Value.IsValid -> jsontext.Options
   alias -> private jsonopts/sealing marker. Its omitted options/public option
   constructors are usable by external consumers. Define foreign publicly
   nameable API treatment and add checker/compile positives without removing
   the seam or blanket internal/alias exemptions. Gomad aliases retaining private
   identities must remain rejected.

Minor: none separately. Source identities remained unchanged before/after review:
architecture.go bce019ade09ad36d8b52b1001b4a20b117f7c98cc5d9070a9d22f8e6bb0dd214;
program.go a33c389b54a29ba9af28dc2315b515058d0aa2c914eaefdcf06a488a481e829b;
root architecture_test.go 411ef3a2921eb24bd61e3eea2f9d65d5838673371df8566c6d0011e53e4b727b.

The writer receives actionable findings; final frozen review and gate verification
remain required. No SHIP, native acceptance, task completion or merge readiness.
