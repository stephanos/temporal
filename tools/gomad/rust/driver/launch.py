#!/usr/bin/env python3
"""Execute a registered command after its parent releases the admission pipe."""
import os
import signal
import sys


def main():
    gate = int(sys.argv[1])
    try:
        admitted = os.read(gate, 1)
    finally:
        os.close(gate)
    if admitted != b'1':
        return 130
    signal.pthread_sigmask(signal.SIG_UNBLOCK, {signal.SIGTERM, signal.SIGHUP, signal.SIGINT})
    os.execvpe(sys.argv[2], sys.argv[2:], os.environ)


if __name__ == '__main__':
    raise SystemExit(main())
