#!/usr/bin/env python3
"""Offline Make-policy fixture, never a compiler or guest acceptance producer."""
import json
import os
from pathlib import Path
import sys

args = sys.argv[1:]
with Path(os.environ['INTERCEPTION_MAKE_CAPTURE']).open('a') as capture:
    capture.write(json.dumps(dict(argv=args, cgo=os.environ.get('CGO_ENABLED'))) + '\n')
if args[0] == 'env':
    print({'GOOS': 'linux', 'GOARCH': 'arm64', 'GOPATH': '/offline/gopath'}[args[1]])
elif args[0] == 'clean':
    pass
elif args[0] == 'test':
    if '-race' in args:
        print('gomad: unsupported guest option: -race', file=sys.stderr)
        sys.exit(2)
    print('ok\t' + args[1] + '\t0.001s')
else:
    raise SystemExit('unexpected offline fixture command')
