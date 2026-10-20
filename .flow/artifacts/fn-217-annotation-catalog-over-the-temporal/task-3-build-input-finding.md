# Obsolete bootstrap input invalidates clean-checkout builds

After the failure-field fix, worker source `5c9d19cca65f6920ceb8fd988d107f4eb866618f`
passed focused tests, Clippy, the live slice (six scenarios and six controls),
default and required-TLC full gates. Root integrated gate session 30346 also
terminated with exit 0. These checks precede the build-input fix below.

Read-only research identified an introduced performance defect in
`model/temporal/api/build.rs`: `catalog_inputs` emits `rerun-if-changed` before
checking existence, and its bootstrap `annotate/catalogs` input no longer exists
after declarations moved into the feature catalog. Both saved root trybuild
fingerprints retain this missing path. Every per-fixture Cargo invocation can
therefore invalidate the API build unit and repeat linking, coverage checks and
code generation. No tests or checks failed; this is a build-freshness defect.

The conductor independently verified that the directory is absent in the
integrated root but still present in the worker checkout. This filesystem
difference explains why a clean-checkout regression was not exposed by the
worker's warm timing. The research did not isolate the exact compilation cost
or prove that the missing input is the only freshness invalidator.

Disposition: fix the obsolete bootstrap scan, retaining recursive feature
catalog watches, descriptor coverage and every compiler fixture. Verify with
the obsolete directory absent and consecutive unchanged builds; creating an
empty local directory is not a repository-level remedy. The worker waited for
both running gates to terminate before editing. Final-source full gates,
timing and single-session implementation re-review remain required.

Detailed active evidence: `.flow/tmp/fn2173-root-ui-build-prep.md`, original
review-fix gate logs and saved Cargo fingerprints. No gate weakening, scope
waiver, task completion or budget-pass claim is made here.
