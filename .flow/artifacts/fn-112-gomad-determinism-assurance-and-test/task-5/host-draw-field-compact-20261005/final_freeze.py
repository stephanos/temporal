"""Bind final product sources before developmental checks; never replace baseline."""
import json
import pathlib

import capture

destination = capture.OUT / 'final-freeze.json'
assert not destination.exists(), 'final freeze already exists'
baseline = json.loads((capture.OUT / 'freeze.json').read_text())
current = capture.sources()
changed = sorted(name for name in set(baseline['sources']) | set(current)
                 if baseline['sources'].get(name) != current.get(name))
expected = sorted('tools/gomad3/' + path for path in [
    'toolchain/runtime/go1.27.1.patch', 'toolchain/runtime/overlay/src/runtime/gomad.go',
    'choice/internal/wire/wire_generated.go', 'target/internal/livecap/protocol_generated.go',
    'toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go',
    'toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go',
    'runner/testdata/diagnostic-identity-choices.json'])
assert changed == expected, ('unexpected final product paths', changed)
tools = {path: capture.digest(pathlib.Path(path)) for path in baseline['tools']}
users = {path: capture.digest(capture.ROOT / path) for path in baseline['user_files']}
assert tools == baseline['tools'], 'pinned tool/archive changed'
assert users == baseline['user_files'], 'user-owned file changed'
info = {'base_commit': baseline['base_commit'], 'sources': current,
        'environment': capture.recorded_environment(capture.environment()),
        'tools': tools, 'user_files': users, 'changed_product_paths': changed}
destination.write_text(json.dumps(info, indent=2) + '\n')
print('final source freeze:', len(current), 'tracked product files;', len(changed), 'admitted changes')
print('pinned tools/archive and both user-owned file hashes unchanged')
