#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../../.."
probe="$PWD/.flow/tmp/fn253-spike"
[[ "$(git rev-parse HEAD)" == 02d4f55660e4da54612d703f16bd92170c742539 ]]
[[ "$PWD" == */.worktrees/fn2532-spike ]]
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
if [[ -z "$(printenv CARGO_TARGET_DIR)" ]]; then export CARGO_TARGET_DIR="$probe/output/target"; fi
mkdir -p "$probe/output"
python3 "$probe/prepare.py"
mise exec rust@1.99.1 -- cargo run --offline --locked --manifest-path "$probe/Cargo.toml" --bin countdown-admission-spike -- "$probe"/activity-{baseline,local,library,shared}.rs "$probe"/control.rs "$probe"/local-{arm,arm-initial,tick,generic,alias,trait}.rs "$probe"/unused-generic.rs "$probe"/receiver-args.rs "$probe"/library-{assignment,condition,within-law,descends-law}.rs "$probe"/project-library.rs "$probe"/shared-{ticks,remaining}.rs "$probe"/parse-error.rs | tee "$probe/output/admission.jsonl"
python3 "$probe/check-results.py" "$probe/output/admission.jsonl"
for variant in baseline local library shared; do
    COUNTDOWN_MODEL="$probe/native-$variant.rs" mise exec rust@1.99.1 -- cargo check --offline --locked --manifest-path "$probe/Cargo.toml" --bin native-model > "$probe/output/native-$variant.log" 2>&1
done
COUNTDOWN_MODEL="$probe/project-library.rs" mise exec rust@1.99.1 -- cargo check --offline --locked --manifest-path "$probe/Cargo.toml" --bin native-model > "$probe/output/native-project-library.log" 2>&1
mise exec rust@1.99.1 -- cargo run --offline --locked --manifest-path "$probe/Cargo.toml" --bin finite-example
mise exec rust@1.99.1 -- rustc --edition=2024 --test "$probe/countdown.rs" -o "$probe/output/countdown"
"$probe/output/countdown"
mise exec rust@1.99.1 -- rustc --edition=2024 --test "$probe/activity-example.rs" -o "$probe/output/activity-example"
"$probe/output/activity-example"
