#!/usr/bin/env python3
"""Retain the evidence-only handover and a lean, explicit checkpoint selection.

This writes no lifecycle state and performs no staging. Bulk evidence remains
local and hash identified; the conductor owns the actual checkpoint decision.
"""
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import shlex
import subprocess

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[3]
ROOT = HERE.parent
CURRENT = HERE/'current-measurement'
BASE = '8604c07def0f97b63cbca3864b4c286d6803c4b1'
PREFIX = str(HERE.relative_to(REPO))

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def read(path):
    return json.loads(path.read_text())

assert subprocess.run(['git','rev-parse','HEAD'],cwd=REPO,check=True,stdout=subprocess.PIPE,text=True).stdout.strip() == BASE
execution = read(CURRENT/'runs/execution-evidence.json')
receipt = read(HERE/'preservation-audit/focused-preservation-receipt.json')
campaigns = [row for row in execution['commands'] if row['output'] in ['discard-10.log','discard-100.log','novel-10.log','novel-100.log']]
assert len(campaigns) == 4 and all(row['exit_code'] == 0 for row in campaigns)
assert all(row['exit_code'] == 0 for row in execution['commands'])

hash_inputs = [
    ROOT/'completion-matrix.md', ROOT/'qualification-evidence.md',
    REPO/'MILESTONES.md', HERE/'current-qualification-handover.md',
    HERE/'native-command-ledger.json', HERE/'measurement-independent-check.md',
    HERE/'lint-tooling-diagnostic.md', HERE/'baseline-validate.log',
    HERE/'baseline-flow-validate.log', HERE/'baseline-lint.log',
    HERE/'lint-code-fast-linux-development.log', HERE/'lint-tool-install-linux.log',
    HERE/'preservation-audit/report.md', HERE/'preservation-audit/outputs.sha256',
    HERE/'preservation-audit/focused-preservation-receipt.json',
    CURRENT/'measurement.md', CURRENT/'comparison.json', CURRENT/'run_current.py',
    CURRENT/'verify_current.py', CURRENT/'verification.log',
    CURRENT/'runs/execution-evidence.json', CURRENT/'runs/initial-source-inventory.json',
    CURRENT/'runs/shipped-source-before-overlay.json', CURRENT/'runs/retained-output.sha256',
    HERE/'bound-baseline-measurement/handoff-output.sha256',
]
evidence = {
    'task_id': 'fn-109-gomad-deepen-modules-and-tool-interfaces.21',
    'status': 'in_progress', 'base_commit': BASE, 'commits': [], 'prs': [],
    'acceptance_complete': False,
    'created_utc': datetime.now(timezone.utc).isoformat(),
    'tier': 'session (jev-unavailable(no_key)); worker pinned role gpt-6.1-sol/high requested, actual model metadata unknown unless evidenced',
    'delegated_agents': 3,
    'review': {'dispatched': False, 'verdict': None, 'owner': 'root conductor'},
    'lifecycle': {'mutated': False, 'owner': 'root conductor'},
    'tests': [
        'PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH GOENV=off GOTOOLCHAIN=local timeout 600 make -C tools/gomad3 validate',
        '/home/agent/.codex/scripts/flowctl validate --spec fn-109-gomad-deepen-modules-and-tool-interfaces',
        'PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH GOENV=off GOTOOLCHAIN=local timeout 600 make lint-code-fast GOLANGCI_LINT_FIX=false',
        'make lint-code-fast LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50 GOLANGCI_LINT_FIX=false',
        *[shlex.join(row['argv']) for row in campaigns],
        shlex.join(receipt['command']),
        'python3 '+PREFIX+'/current-measurement/verify_current.py',
        'python3 '+PREFIX+'/retain_qualification_ledger.py',
        'python3 '+PREFIX+'/verify_handover.py',
        'git diff --check',
    ],
    'baseline': {
        'result': 'red: unchanged lint-code-fast failed pre-edit',
        'generator_validate_exit': 0, 'flow_validate_exit': 0,
        'lint_exit': 2, 'elapsed_seconds': None,
        'duration_limit': 'Exact full-command monotonic durations were not captured for worker baseline checks; no duration is invented.',
        'green_handoff_claimed': False,
    },
    'bounded_measurement': {
        'completed': True, 'native_qualification': False,
        'host': 'linux/arm64', 'scratch': execution['scratch'],
        'source_files': 982, 'shipped_files': 978,
        'complete_source_inventories': 10, 'successful_child_commands': 98,
        'driver_session': 38361, 'driver_exit': 0,
        'driver_prelaunch_sha256': execution['driver_prelaunch_sha256'],
        'pinned_go_sha256': execution['hashes']['pinned-go-binary'],
        'bindings': execution['bindings'],
        'cases': [{'case': row['output'][:-4], 'exit_code': row['exit_code'], 'elapsed_seconds': row['elapsed_seconds']} for row in campaigns],
        'current_output_hashes': 174, 'immutable_baseline_handoff_hashes': 201,
        'logical_both_role_bytes': {'baseline': 4120, 'current': 4472, 'delta': 352},
        'scope': 'Matched paired injected fixture only; no actual child transport, target preparation or replay; private novelty cardinality and universal transient-copy absence unobserved.',
        'extra_publication_buffer': {'bytes': 65536, 'payload_read_bytes': 20, 'owner': 'fn-114-gomad-correct-search-path-defects-and.9', 'commit': 'bc2e970b5306aa09f594657a8d42c159cf4a1270', 'site': 'artifact.verifySharedPayload -> copyWithContext -> SHA256 hasher'},
        'receipt': PREFIX+'/current-measurement/runs/execution-evidence.json',
        'verification': PREFIX+'/current-measurement/verification.log',
        'failed_attempts_retained': ['current-measurement/failed-inventory-preflight', 'current-measurement/failed-policy-equality.log'],
    },
    'preservation': {
        'complete': False, 'requirement': 'R18',
        'source_surfaces': 8, 'source_platform_selections': ['darwin/arm64','linux/amd64'],
        'inventory_commands': 69, 'declaration_provenance_rows': 202,
        'fresh_current_test_exit': receipt['exit_code'],
        'fresh_current_test_seconds': receipt['elapsed_seconds'],
        'top_level_tests': 18, 'cases': 55, 'skips': 0,
        'known_fn109_inventory_gap': 'runner/campaign_options.go:109-175 two error types and six campaign helpers; introduced by WIP a3b9f80efab9356c0be2080779133337e2471ac0, owning task unresolved rather than attributed to task2 by assumption.',
        'independently_owned_migrations': ['fn112 diagnostics/soak','fn113 adapter/profile retirement','fn114 choice ordinal/guidance/minimizer/target pool/readiness/wire3/controller3'],
        'remaining': ['Reconcile complete intentional migration inventory with actual provenance.', 'Reconcile fn114.11 wire/profile-v3 and fn114.12 controller-v3 with the aggregate format/identity contract.', 'Nine existing public flag literals lack CLI.md inventory coverage.', 'Matched first-baseline fixed-identity canonical comparison remains incomplete beyond retained specific vectors.'],
        'original_scout_report_sha256': '0fc86685ce46a6a2f4b1f1d4550f93e307ca6086216bc1b8d8e881d57d3bfd33',
        'original_scout_manifest_sha256': '028ef71d3631dce74844522a1314f96c4376c7ed375820e9444168b14c515162',
    },
    'native_qualification': {
        'complete': False, 'required_platforms': ['darwin/arm64','linux/amd64'],
        'available_platforms': [], 'patched_toolchain_present': False,
        'command_ledger_rows': 89, 'native_commands_executed': 0,
        'ledger': PREFIX+'/native-command-ledger.json',
        'result': 'incomplete: required native hosts/toolchain unavailable, not passed or synthetically qualified',
        'dispositions_modified': False,
    },
    'lint': {
        'result': 'failed', 'original_exit': 2,
        'original_cause': 'Existing pinned executable is Mach-O ARM64 on Linux aarch64.',
        'conductor_isolated_pinned_linux_rerun': {'session': 66346, 'make_exit': 2, 'golangci_exit': 7, 'elapsed_seconds': 147.354353404, 'cause': 'Unchanged root target passes nested-module overlay/conformance paths to root module loader.'},
        'diagnostic': PREFIX+'/lint-tooling-diagnostic.md',
        'source_or_selection_changed': False,
    },
    'completion_matrix': {'finding_rows': 16, 'unique_accountable_owners': True, 'D1_D5_each_once': True, 'D3_owner': 'task6', 'D4_owner': 'task19', 'D5_owner': 'task20', 'transferred_fn105_3_4_5_closed': False},
    'remaining_formal': ['Task21 conductor independent/formal review.', 'Task19 formal review after predispatch failure with no verdict.', 'Task20 guidance SHIP does not close original fn102 R6 both-native integrated acceptance.'],
    'commands_still_running': [],
    'artifact_sha256': {str(path.relative_to(REPO)):digest(path) for path in hash_inputs},
}
(HERE/'current-qualification-evidence.json').write_text(json.dumps(evidence,indent=2)+'\n')

# Selection, not staging: exclude heavyweight profiles/binaries/payloads, raw
# stack logs and full blame dumps. Every excluded measurement output already
# has a retained hash, and the immutable scout manifest still identifies blame.
selected = [ROOT/'completion-matrix.md',ROOT/'qualification-evidence.md',REPO/'MILESTONES.md']
selected += [path for path in HERE.iterdir() if path.is_file() and path.name in {
    'baseline-validate.log','baseline-flow-validate.log','baseline-lint.log',
    'current-qualification-handover.md','current-qualification-evidence.json',
    'lint-tooling-diagnostic.md','lint-tool-install-linux.log','lint-code-fast-linux-development.log',
    'measurement-independent-check.md','native-command-ledger.json',
    'retain_qualification_ledger.py','verify_handover.py','finalize_current_handover.py',
    'seal_current_checkpoint.py'}]
selected += [path for path in (HERE/'preservation-audit').iterdir() if path.is_file() and not path.name.startswith('blame-')]
selected += [path for path in CURRENT.iterdir() if path.is_file()]
lean_runs = {'execution-evidence.json','initial-source-inventory.json','initial-source.sha256',
    'shipped-source-before-overlay.json','bound-environment-before-launch.json',
    'effective-go-env-before-build.json','retained-output.sha256',
    'logical-policy-both-roles.json','logical-policy-both-roles.log','host.log',
    'pinned-go-version.log','developmental-overlay.log','original-source-precheck.log',
    'artifact-payload-alias.log','runner-constructor-alias.log','existing-capacity-controls.log',
    'profile-analysis.log'}
selected += [path for path in (CURRENT/'runs').iterdir() if path.is_file() and (path.name in lean_runs or path.name in {case+'.log' for case in ['discard-10','discard-100','novel-10','novel-100']} or path.name.endswith('-binary-build-metadata.log'))]
for case in ['discard-10','discard-100','novel-10','novel-100']:
    selected.append(CURRENT/'runs'/case/'measurement.json')
for path in (CURRENT/'failed-inventory-preflight').rglob('*'):
    if path.is_file() and (path.name.endswith('.log') or path.name == 'execution-evidence.json'):
        selected.append(path)
rows = [dict(path=str(path.relative_to(REPO)),sha256=digest(path),bytes=path.stat().st_size) for path in sorted(set(selected))]
output = dict(purpose='Conductor checkpoint selection only; no staging performed.',files=rows,total_bytes=sum(row['bytes'] for row in rows),omissions='All prior artifacts unchanged; raw pprof/top/stack logs, duplicated inventories, binaries/payloads, 13MiB profile-attribution.json and full preservation blame dumps remain local and manifest-hash identified. handover-verification.json is a later generated receipt, retained separately.')
(HERE/'current-checkpoint-selection.json').write_text(json.dumps(output,indent=2)+'\n')
print(json.dumps(dict(evidence=str(HERE/'current-qualification-evidence.json'),selected_files=len(rows),selected_bytes=output['total_bytes'],acceptance_complete=False),indent=2))
