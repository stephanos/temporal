#!/usr/bin/env python3
"""Produce the final lightweight handoff and a complete local evidence manifest."""
import hashlib
import json
from pathlib import Path
import subprocess

HERE = Path(__file__).resolve().parent
child = subprocess.run(['python3', str(HERE/'verify_bound_evidence.py')], cwd=HERE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
(HERE/'verification.log').write_bytes(child.stdout)
assert child.returncode == 0, child.stdout
verified = json.loads(child.stdout)
execution = json.loads((HERE/'runs/execution-evidence.json').read_text())
handoff = dict(status='bounded developmental baseline evidence prepared; no current comparison or native qualification', terminal_session=79949, terminal_exit_code=0, cases_serialized=True, verification=verified, report='measurement.md', local_raw_outputs='runs/', scratch=execution['scratch'], environment_contract=execution['bindings'], identities=execution['hashes'], metadata_drift=execution['historical_input_metadata_drift'], original_and_historical_artifacts_unchanged=True, lightweight_checkpoint_candidates=['measurement.md', 'handoff.json', 'verification-result.json', 'verification.log', 'verify_bound_evidence.py', 'finalize_evidence.py', 'run_bound.py', 'analyze_profiles.py', 'observe_post_run_compiler_settings.py', 'post-run-compiler-settings.json', 'r19_measurement_test.go', 'r19_logical_policy_test.go', 'r19_artifact_payload_test.go', 'r19_transport_guard.go', 'descriptor_dup_linux.go', 'developmental-platform.patch', 'baseline-compilation-calls.patch', 'handoff-output.sha256'], bulk_files_not_for_checkpoint='runner.test binaries, campaign target/payload files, pprof profiles and raw/top pprof logs retained locally with hashes', requested_model='gpt-6.1-sol/high', actual_model_metadata='unknown', tier='session (jev-unavailable(no_key))')
(HERE/'handoff.json').write_text(json.dumps(handoff, indent=2)+'\n')
rows = []
for path in sorted(HERE.rglob('*')):
    assert not path.is_symlink(), path
    if path.is_file() and path != HERE/'handoff-output.sha256':
        rows.append((hashlib.sha256(path.read_bytes()).hexdigest(), str(path.relative_to(HERE))))
(HERE/'handoff-output.sha256').write_text(''.join(digest+'  '+relative+'\n' for digest, relative in rows))
for digest, relative in rows:
    assert hashlib.sha256((HERE/relative).read_bytes()).hexdigest() == digest
print(json.dumps(dict(passed=True, complete_handoff_files=len(rows), handoff_manifest_sha256=hashlib.sha256((HERE/'handoff-output.sha256').read_bytes()).hexdigest(), verification_exit=child.returncode), indent=2))
