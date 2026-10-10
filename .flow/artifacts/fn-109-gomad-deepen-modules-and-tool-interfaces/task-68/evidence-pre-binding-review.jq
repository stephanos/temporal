[inputs | select(has("argv")) | . + {receipt_path: input_filename}] as $receipts |
{
  commits: $commits,
  base_commit: $base,
  tests: [$receipts[] | .argv | @sh],
  prs: [],
  status: "in_progress",
  workspace: $workspace,
  frozen_product: {path: "tools/gomad3/runner/runner_test.go", sha256: $product, insertions: 3, deletions: 0},
  baseline: "red: three original tests fail at unsupported-host preparation; six inherited unfiltered Runner lint findings; retained controls green",
  focused_outcomes: {top_level_pass: 34, named_pass: 65, fail: 0, skip: 0},
  runner_lint: {baseline_findings: 6, final_findings: 6, introduced: 0, removed: 0, full_diagnostic_blocks_sha256: $lint_blocks},
  command_receipts: [$receipts[] | {receipt_path, exit, elapsed_seconds, source_manifest_sha256, post_source_manifest_sha256, tools_manifest_sha256, environment_sha256, wrapper_sha256, raw_log_sha256, post_source_match_exit}],
  artifact_script_hashes: $script_hashes,
  authoritative_spec: {path: "/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md", sha256: "851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c"},
  environment_limits: [
    "Per-command environment files capture selected exported variables, TZ=UTC, the supplied TMPDIR and complete effective outer go env; explicit supported-source-set overrides are retained in argv.",
    "The materialized successor wrapper adds tests/mixedbrain/go.mod and go.sum to its actual source manifest. Earlier receipts cover the earlier sparse checkout and do not attest those absent inputs.",
    "Corrected repository fast lint analyzed 55 nested host packages and diff-filtered 50 inherited findings to zero; visible missing proto/internal and chasm/lib find warnings remain. This is no full-root or unfiltered lint pass.",
    "Tool manifests hash executed Go/gofmt/lint/errortype and command/C binaries, not complete tool installations, headers, libc or reused cache contents.",
    "Stock linux/arm64 source tests and cross-platform vet supply no supported-native execution or full native test-host pass."
  ],
  inconclusive_observations: [
    "final-fast-lint.json exit2 stopped before lint analysis on the missing tests/mixedbrain/go.mod; its raw bytes remain unchanged. Root subsequently materialized the sparse cone before the separately named successor.",
    "initial-boundary-diagnostic.md retains a read-only exit255 against task66's older manifest after task67 changed runtime_repeatability.go.",
    "final-boundary-reconciliation.json exit255 discovered that task67's narrower manifest omits two qualification generator inputs. Corrected reconciliation binds those exact unchanged inputs to retained task65 validation and the other552 paths to task67."
  ],
  boundary_reconciliation: {
    receipt: "final-boundary-reconciliation-corrected.json",
    current_task67_inputs: 552,
    unchanged_task65_qualification_inputs: 2,
    task66_drift: "Only task67's admitted internal/gomadtool/conformance/runtime_repeatability.go correction differs from the older inventory.",
    claim: "Exact actual consumed-input reconciliation only. Current supported-source-set Runner vet covers the changed test body; prior generator/private-API/public-consumer evidence is reused only for its unchanged consumed inputs."
  },
  remaining_requirements: [
    "Root-owned frozen 673-name ordinary comparison and complete original-base RED50 lint-block comparison.",
    "Fresh independent handover/integrated review, commit authorization, integration and lifecycle updates remain conductor-owned.",
    "Six unfiltered affected lint findings keep task-owned source acceptance open. Fast diff lint and standalone errortype do not make the aggregate gate green.",
    "Original fn-109.63/fn-112.10 source acceptance and unaffected retained obligations remain open wherever unproved; fn-128/fn-149 native qualification stays deferred and unverified."
  ],
  handles_terminal: true,
  lane_released: true,
  review: "Conductor-owned. The worker issued no review verdict.",
  commit_authority: "Await conductor approval after independent final evidence review."
}
