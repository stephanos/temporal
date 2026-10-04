#!/usr/bin/env python3
"""Capture safe compiler settings now without claiming pre-run observation."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess

HERE = Path(__file__).resolve().parent
evidence = json.loads((HERE/'runs/execution-evidence.json').read_text())
bindings = evidence['bindings']
env = dict(os.environ, **bindings)
go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
keys = ['CC', 'CXX', 'AR', 'PKG_CONFIG', 'CGO_CFLAGS', 'CGO_CPPFLAGS', 'CGO_CXXFLAGS', 'CGO_FFLAGS', 'CGO_LDFLAGS', 'GOGCCFLAGS', 'GO_EXTLINK_ENABLED', 'GOTOOLDIR', 'GOENV', 'GOFLAGS', 'GOEXPERIMENT']
argv = [go, 'env', '-json', *keys]
child = subprocess.run(argv, cwd=evidence['scratch'], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
assert child.returncode == 0, child.stdout
record = dict(observation='post-run only; compiler variables not explicitly bound or captured before the campaigns; do not infer historical effective settings', observed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), argv=argv, cwd=evidence['scratch'], child_environment_subset=bindings, exit_code=child.returncode, effective_safe_go_env=json.loads(child.stdout), driver_snapshot_sha256=hashlib.sha256((HERE/'runs/executed-driver-post-run-snapshot.py').read_bytes()).hexdigest())
(HERE/'post-run-compiler-settings.json').write_text(json.dumps(record, indent=2)+'\n')
print(json.dumps(record, indent=2))
