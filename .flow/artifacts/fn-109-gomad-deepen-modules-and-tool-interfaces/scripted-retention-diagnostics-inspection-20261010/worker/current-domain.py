import json
from pathlib import Path
import sys
sys.dont_write_bytecode = True
import capture_v2 as r

historical = r.ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-70/manifest-2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
assert r.sha(historical) == '2a1b63fe97199753cc29e63535c34003238dc9547dad4040aa4228faa4a6bc74'
binding = json.loads((r.OUTPUT/'validate-binding.json').read_text())
receipt = json.loads((r.OUTPUT/'validate.json').read_text())
assert receipt['exit_code'] == 0 and not receipt['timed_out'] and not receipt['remaining_group_members']
assert r.sha(r.OUTPUT/'validate-binding.json') == receipt['binding_sha256']
assert r.sha(r.OUTPUT/'validate.log') == receipt['log_sha256']
assert binding['source_manifest'] == receipt['source_after']
assert binding['tools_manifest'] == receipt['tools_after']
retained = {}
for line in historical.read_text().splitlines():
    parts = line.split('  ', 1)
    if len(parts) == 2 and parts[1].startswith('tests/'):
        retained[parts[1]] = parts[0]
assert len(retained) == 132
top_tests = sorted(p for p in retained if Path(p).parent == Path('tests') and p.endswith('_test.go'))
discovered = sorted(str(p.relative_to(r.ROOT)) for p in (r.ROOT/'tests').glob('*_test.go') if p.is_file())
assert len(top_tests) == 113 and top_tests == discovered
assert all((r.ROOT/p).is_file() and r.sha(r.ROOT/p) == digest == binding['source_manifest'][str(r.ROOT/p)] for p, digest in retained.items())
assert binding['actual_go_settings']['GOMOD'] == str(r.ROOT/'go.mod')
assert binding['actual_go_settings']['GOCACHE'] == str(r.ROOT/'tools/gomad3/.toolchain/generator-cache')
print(json.dumps({'validation_execution': 'EXECUTED: current candidate numeric exit0', 'go_settings_probe': 'NOT_EXECUTED: consumes current validate capture', 'receipt_sha256': r.sha(r.OUTPUT/'validate.json'), 'binding_sha256': r.sha(r.OUTPUT/'validate-binding.json'), 'log_sha256': receipt['log_sha256'], 'argv': receipt['argv'], 'cwd': receipt['cwd'], 'materialized_domain_files': 132, 'discovered_top_level_tests': 113, 'all_domain_hashes_match': True, 'domain_files': retained, 'current_selected_file_sha256': {p: r.sha(r.ROOT/p) for p in r.SELECTED}, 'limitations': 'Outer make Go-env capture describes ROOT/go.mod. Validation child routes are retained in raw output and source-bound Makefiles; this observer executes no Go-env.'}, sort_keys=True, indent=2))
