"""Reconstruct the selected host scope without invoking lint or changing sources."""
import json
import pathlib
import re
import subprocess

import capture

destination = capture.OUT / 'lint-scope.json'
assert not destination.exists(), 'lint scope already exists'
owner = 'tools/gomad3/'
classifier = capture.ROOT / (owner + 'internal/gomadtool/architecture/architecture.go')
source = classifier.read_text()
literal = re.search(r'var sourceExclusions = \[\]string\{([^}]+)\}', source).group(1)
exclusions = json.loads('[' + literal + ']')
paths = subprocess.check_output(['git', 'ls-files', '-z', '--cached', '--others', '--exclude-standard',
                                 '--', '*.go', 'go.mod', '**/go.mod'], cwd=capture.ROOT).decode().split('\0')
ordinary = []
for path in paths:
    if not path.startswith(owner) or not path.endswith('.go') or not (capture.ROOT / path).is_file():
        continue
    relative = path[len(owner):]
    if any(relative == excluded or relative.startswith(excluded + '/') for excluded in exclusions):
        continue
    assert not any(part.startswith(('.', '_')) or part == 'testdata' for part in pathlib.PurePosixPath(relative).parent.parts)
    ordinary.append(path)
packages = sorted({'.' if str(pathlib.PurePosixPath(path[len(owner):]).parent) == '.'
                   else './' + str(pathlib.PurePosixPath(path[len(owner):]).parent) for path in ordinary})
log = (capture.OUT / 'final-lint-fast.stdout').read_text()
assert 'Lint module tools/gomad3: 55 host packages' in log
assert len(packages) == 55
info = {'evidence': 'read-only reconstruction from unchanged driver classifier and frozen git source inventory; actual failed invocation reported 55 selected host packages',
        'classifier_sha256': capture.digest(classifier),
        'driver_sha256': capture.digest(capture.ROOT / 'cmd/tools/lintcode/main.go'),
        'source_exclusions': exclusions, 'ordinary_source_count': len(ordinary),
        'selected_host_packages': packages, 'selected_host_package_count': len(packages),
        'module': 'tools/gomad3', 'generated_host_wire_endpoints_included': True,
        'runtime_overlay_classified_separately': True,
        'retained_checkpoint_test_classified_as_evidence': True,
        'reported_issues_only_new_from_rev': '1147416b2e6465de7b631e1e4b695eba33ebf000',
        'full_lint_pass_claim': False}
destination.write_text(json.dumps(info, indent=2) + '\n')
print('selected host scope reconstructed:', len(packages), 'packages;', len(ordinary), 'ordinary Go sources')
print(' '.join(packages))
