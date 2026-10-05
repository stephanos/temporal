#!/usr/bin/env bash
set -euo pipefail
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
out=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42
source=tools/gomad3/internal/compatibilitypack/policy.go
test_source=tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go
base=533aa074337bc1e21e52c306ddb5a19e93fdfdbe
scratch=$(mktemp -d /tmp/fn109-policy-proof.XXXXXXXX)
git show "$base:$source" > "$scratch/base.go"
sed '/^\t\t\tcase FactMalformedLinkname, FactNoReviewedGoSource:$/,+1d' "$source" > "$scratch/reconstructed.go"
cmp "$scratch/base.go" "$scratch/reconstructed.go"
sed '\|  tools/gomad3/internal/compatibilitypack/policy.go$|d' "$out/admission-base/source.sha256" > "$out/manifests/protected.sha256"
sha256sum -c "$out/manifests/protected.sha256" > "$out/raw/protected-check.log"
sed '/policy.go:196:4: missing cases/,+2d; /^8 issues:/,$d' "$out/raw/base-additive-lint-nil-selection.log" > "$scratch/base-residual.log"
sed '/^7 issues:/,$d' "$out/raw/final-lint-nil-selection.log" > "$scratch/final-residual.log"
cmp "$scratch/base-residual.log" "$scratch/final-residual.log"
cmp "$out/admission-base/lint.log" "$out/raw/base-additive-lint-nil-selection.log"
cmp <(jq -s '[.[] | select(.Action == "output" and (.Output | contains("decision=") or contains("loader_error="))) | {Test,Output}]' "$out/raw/base-characterization-nil-selection.log") <(jq -s '[.[] | select(.Action == "output" and (.Output | contains("decision=") or contains("loader_error="))) | {Test,Output}]' "$out/raw/final-characterization-nil-selection.log")
base_test_hash=$(sed -n '\|  tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go$|s/ .*//p' "$out/manifests/base-characterization-nil-selection-before.sha256")
test "$base_test_hash" = "$(sha256sum "$test_source" | cut -d ' ' -f1)"
git diff -- "$source" > "$scratch/production.diff"
set +e
git diff --no-index -- /dev/null "$test_source" > "$scratch/test.diff"
diff_rc=$?
set -e
test "$diff_rc" = 1
for pair in production test; do
  jq -Rs --arg sha256 "$(sha256sum "$scratch/$pair.diff" | cut -d ' ' -f1)" '{raw_diff:.,sha256:$sha256}' "$scratch/$pair.diff" > "$out/$pair-diff.json"
  cmp <(jq -jr '.raw_diff' "$out/$pair-diff.json") "$scratch/$pair.diff"
done
jq -n --arg base_commit "$base" --arg production_base_sha256 "$(sha256sum "$scratch/base.go" | cut -d ' ' -f1)" --arg production_final_sha256 "$(sha256sum "$source" | cut -d ' ' -f1)" --arg test_sha256 "$base_test_hash" --arg protected_manifest_sha256 "$(sha256sum "$out/manifests/protected.sha256" | cut -d ' ' -f1)" --argjson protected_entries "$(wc -l < "$out/manifests/protected.sha256")" --arg residual_sha256 "$(sha256sum "$scratch/final-residual.log" | cut -d ' ' -f1)" --arg production_diff_sha256 "$(sha256sum "$scratch/production.diff" | cut -d ' ' -f1)" --arg test_diff_sha256 "$(sha256sum "$scratch/test.diff" | cut -d ' ' -f1)" '{base_commit:$base_commit,production_base_sha256:$production_base_sha256,production_final_sha256:$production_final_sha256,added_bytes:"\t\t\tcase FactMalformedLinkname, FactNoReviewedGoSource:\n\t\t\t\tcontinue\n",exact_reconstruction:true,test_sha256:$test_sha256,literal_tests_unchanged_between_passing_base_and_final:true,literal_decision_output_identical:true,protected_manifest_sha256:$protected_manifest_sha256,protected_entries:$protected_entries,protected_all_unchanged:true,existing_tests_unchanged:true,base_additive_lint_byte_identical_to_root:true,residual_blocks_byte_identical:true,residual_sha256:$residual_sha256,base_lint:{exhaustive:1,gci:1,staticcheck:6,total:8},final_lint:{exhaustive:0,gci:1,staticcheck:6,total:7},diffs:{production:{path:"production-diff.json",raw_sha256:$production_diff_sha256,decoded_exact:true},test:{path:"test-diff.json",raw_sha256:$test_diff_sha256,decoded_exact:true}}}' > "$out/source-proof.json"
git diff --check
cat "$out/source-proof.json"
