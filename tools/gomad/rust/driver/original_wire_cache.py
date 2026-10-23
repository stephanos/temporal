"""The six actual original wasi/reference/cluster test dependency modules, not build objects."""
from pathlib import Path

import installed_cluster_cache as cache
import live
from driver import require, sha
from native_observations import canonical, digest

MODULES = (
    ('github.com/davecgh/go-spew', 'v1.1.2-0.20180830191138-d8f796af33cc'),
    ('github.com/go-json-experiment/json', 'v0.0.0-20260820222146-c27c302e5fc3'),
    ('github.com/pmezard/go-difflib', 'v1.0.1-0.20181226105442-5d4384ee4fb2'),
    ('github.com/stretchr/testify', 'v1.11.1'),
    ('golang.org/x/sys', 'v0.48.0'),
    ('gopkg.in/yaml.v3', 'v3.0.1'),
)


def verify(root, go_sum):
    result = {}
    for module, version in MODULES:
        result.update(cache.verify(root, go_sum, module, version))
    require(live.inventory(root) == result, 'original wire module cache has unadmitted inputs')
    return result


def input_paths(entry):
    root = Path(entry['module_cache'])
    sums = Path(entry['derived']) / 'source/tools/gomad/go.sum'
    return {str(root / name) for name in verify(root, sums)}


def binding(config, entry):
    root = Path(entry['module_cache'])
    live.sealed_input(config, str(root))
    sums = Path(entry['derived']) / 'source/tools/gomad/go.sum'
    files = verify(root, sums)
    return dict(root=str(root), modules=[list(value) for value in MODULES], go_sum_sha256=sha(sums),
                files_sha256=digest(canonical(files)), file_count=len(files),
                bytes=sum(value['bytes'] for value in files.values()))


def environment(goroot, scratch, module_cache):
    return dict(PATH=str(goroot / 'bin') + ':/usr/bin:/bin', GOFLAGS='-p=1 -mod=readonly',
        GOCACHE=str(scratch / 'go-cache'), TMPDIR=str(scratch), GOTMPDIR=str(scratch),
        GOMODCACHE=module_cache, GO111MODULE='on', GOPROXY='off', GOSUMDB='off', GOVCS='*:off',
        GOPRIVATE='', GONOPROXY='', GONOSUMDB='', GOCACHEPROG='')
