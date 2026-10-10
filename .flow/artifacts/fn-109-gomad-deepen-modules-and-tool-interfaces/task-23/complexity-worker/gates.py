import sys
sys.dont_write_bytecode = True
import run as r

for name, argv, cwd in (
    ('final-helper-vet', [r.GO, 'vet', '-tags', 'test_dep', './cmd/tools/lintcode'], r.ROOT),
    ('final-helper-errortype', [r.GO, 'vet', '-tags', 'test_dep', '-vettool=' + str(r.TOOLS / 'errortype'), '-style-check=false', './cmd/tools/lintcode'], r.ROOT),
    ('final-gofmt', [str(r.GO_ROOT / 'bin/gofmt'), '-d', 'cmd/tools/lintcode/main.go', 'cmd/tools/lintcode/main_test.go'], r.ROOT),
    ('final-ownership', [r.GO, 'test', '-tags', 'test_dep', '-count=1', '-json', '-run', '^TestMakeTargetsMatchTheirOwnership$', '.'], r.ROOT / 'tools/gomad3'),
    ('final-architecture', [r.GO, 'test', '-tags', 'test_dep', '-count=1', '-json', '-run', '^TestPackageArchitecture$', '.'], r.ROOT / 'tools/gomad3'),
    ('final-validate', ['/usr/bin/make', '-C', 'tools/gomad3', 'validate', 'SHELL=/bin/sh'], r.ROOT),
):
    print(name, flush=True)
    status = r.run(name, argv, cwd)
    if status:
        sys.exit(status)
