import hashlib
import json
from pathlib import Path
import re

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
admission = json.loads((PROOF / 'root-admission.json').read_text())


def digest(data):
    return hashlib.sha256(data).hexdigest()


sources = {}
for relative in admission['source_paths']:
    actual = (ROOT / relative).read_text()
    restored = actual
    if relative.endswith('corpus.go'):
        restored = restored.replace('readSnapshot() (result Snapshot, retErr error)', 'readSnapshot() (Snapshot, error)')
        restored = restored.replace('validateEntry(entry Entry) (result artifact.SharedTarget, retErr error)', 'validateEntry(entry Entry) (artifact.SharedTarget, error)')
        for owner, zero in [('file', 'Snapshot{}'), ('opened', 'artifact.SharedTarget{}')]:
            closure = f'\tdefer func() {{\n\t\tif closeErr := {owner}.Close(); closeErr != nil {{\n\t\t\tresult = {zero}\n\t\t\tif retErr == nil {{\n\t\t\t\tretErr = closeErr\n\t\t\t}} else {{\n\t\t\t\tretErr = errors.Join(retErr, closeErr)\n\t\t\t}}\n\t\t}}\n\t}}()'
            assert restored.count(closure) == 1
            restored = restored.replace(closure, f'\tdefer {owner}.Close()')
    else:
        restored = restored.split('\nfunc TestCorpusReadSnapshotPreservesResultsAndErrorOrder(', 1)[0]
        for added_import in ['bytes', 'errors', 'reflect']:
            assert restored.count(f'\t"{added_import}"\n') == 1
            restored = restored.replace(f'\t"{added_import}"\n', '')
        for owner, count in [('reopened', 3), ('corpus', 1)]:
            closure = f'\tdefer func() {{\n\t\tif err := {owner}.Close(); err != nil {{\n\t\t\tt.Errorf("Close() = %v", err)\n\t\t}}\n\t}}()'
            assert restored.count(closure) == count
            restored = restored.replace(closure, f'\tdefer {owner}.Close()')
    assert digest(restored.encode()) == admission['source_before_sha256'][relative], relative
    assert actual.endswith('\n')
    assert not any(line.rstrip() != line for line in actual.splitlines())
    sources[relative] = {'sha256': digest(actual.encode()), 'original_reconstruction_sha256': digest(restored.encode()), 'original_reconstruction_matches_base': True, 'whitespace_check': 'pass'}
protected = {p: digest((ROOT / p).read_bytes()) for p in admission['protected_files']}
assert protected == admission['protected_files']
assert digest((ROOT / admission['owner_plan_path']).read_bytes()) == admission['owner_plan_sha256']
capture = (PROOF / 'controls-focused.log').read_text()
vector = re.search(r'BASE_CANONICAL_BYTES=(.*)', capture).group(1)
vector_hash = re.search(r'BASE_CANONICAL_HASH=(.*)', capture).group(1)
test = (ROOT / admission['source_paths'][1]).read_text()
assert f'const want = `{vector}`' in test
assert vector_hash == 'sha256:' + digest(vector.encode())
assert vector_hash in test
control_receipt = json.loads((PROOF / 'controls-focused.receipt.json').read_text())
literal_receipt = json.loads((PROOF / 'preservation-focused.receipt.json').read_text())
production = admission['source_paths'][0]
assert control_receipt['source_before']['sources'][production] == admission['source_before_sha256'][production]
assert literal_receipt['source_before']['sources'][production] == admission['source_before_sha256'][production]
lint = (PROOF / 'baseline-lint.log').read_text()
diagnostics = re.findall(r'^([^\n]+):([0-9]+):([0-9]+): (.+) \(([^)]+)\)$', lint, re.M)
assert len(diagnostics) == 6
assert (PROOF / 'final-lint.log').read_text().strip() == '0 issues.'
delta = {'baseline_count': 6, 'final_count': 0, 'introduced': [], 'residual': [], 'resolved': [{'path': p, 'line': int(line), 'column': int(column), 'message': message, 'analyzer': analyzer, 'repair': 'conditional named-result cleanup' if p.endswith('corpus.go') else 'same-position deferred nonfatal fixture cleanup'} for p, line, column, message, analyzer in diagnostics]}
(PROOF / 'lint-delta.json').write_text(json.dumps(delta, indent=2) + '\n')
receipts = []
for path in sorted(PROOF.glob('*.receipt.json')):
    receipt = json.loads(path.read_text())
    log = ROOT / receipt['log']
    assert digest(log.read_bytes()) == receipt['log_sha256']
    assert receipt['stability'] and receipt['source_before'] == receipt['source_after']
    count = len(re.findall(r'^--- PASS:', log.read_text(), re.M)) if 'go test ' in receipt['command'] else None
    if count is not None:
        assert count > 0
    receipts.append({'path': str(path.relative_to(ROOT)), 'sha256': digest(path.read_bytes()), 'command': receipt['command'], 'exit_code': receipt['exit_code'], 'elapsed_seconds': receipt['elapsed_seconds'], 'top_level_pass_count': count})
result = {'base_commit': admission['base_commit'], 'sources': sources, 'protected_count': len(protected), 'protected_unchanged': True, 'canonical_oracle': {'capture_log': str((PROOF / 'controls-focused.log').relative_to(ROOT)), 'captured_with_base_production': True, 'literal_verified_before_production_edit': True, 'bytes_sha256': vector_hash, 'snapshot_sha256': json.loads(vector)['snapshot_sha256'], 'entries': len(json.loads(vector)['entries'])}, 'generator_inspection': {'affected': False, 'inspected': ['tools/gomad3/Makefile:6-11', 'tools/gomad3/internal/gomadtool/generation/protocol/protocol.go', 'tools/gomad3/internal/gomadtool/generation/boundary/boundary_manifest.go', 'tools/gomad3/toolchain/version/version.json'], 'reason': 'The two corpus files are outside descriptor, schema/template, boundary, compatibility-request and qualification-generator inputs. No generated source or identity definition changes.', 'validate': 'not run; generator inputs unaffected'}, 'receipts': receipts}
(PROOF / 'source-check.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps({'protected_count': len(protected), 'source_reconstruction': 'pass', 'sources': {p: data['sha256'] for p, data in sources.items()}, 'counts': {Path(r['path']).name: r['top_level_pass_count'] for r in receipts}, 'lint_delta': '6 -> 0'}))
